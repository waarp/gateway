// Package s3 provides a filesystem implementation for the S3 cloud storage
// service.
package s3

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/filesystems"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var ErrMissingBucket = errors.New(`the S3 "bucket" parameter is missing`)

type Options struct {
	Bucket   string `json:"bucket"`
	Region   string `json:"region"`
	Endpoint string `json:"endpoint"`
}

const (
	statTimeout   = 10 * time.Second
	mkdirTimeout  = 10 * time.Second
	listTimeout   = 30 * time.Second
	deleteTimeout = 10 * time.Second
)

//nolint:gochecknoinits //init is required here by design
func init() {
	filesystems.FileSystems.Store("s3",
		func(key, secret string, options map[string]any) (fs.FS, error) {
			return newS3FS(key, secret, options)
		},
	)
}

func newS3FS(key, secret string, optionsMap map[string]any) (*filesystem, error) {
	var options Options
	if err := utils.JSONConvert(optionsMap, &options); err != nil {
		return nil, fmt.Errorf("failed to parse S3 options: %w", err)
	}

	if options.Bucket == "" {
		return nil, ErrMissingBucket
	}

	setCredentialsProviderFn := func(o *config.LoadOptions) error {
		if key != "" && secret != "" {
			o.Credentials = credentials.NewStaticCredentialsProvider(key, secret, "")
		}

		return nil
	}

	conf, confErr := config.LoadDefaultConfig(context.Background(),
		setCredentialsProviderFn,
		config.WithRegion(options.Region),
	)
	if confErr != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", confErr)
	}

	if _, err := conf.Credentials.Retrieve(context.Background()); err != nil {
		return nil, fmt.Errorf("invalid AWS credentials: %w", err)
	}

	setEndPointFn := func(o *s3.Options) {
		if options.Endpoint != "" {
			if !strings.HasPrefix(options.Endpoint, "https://") {
				options.Endpoint = "https://" + options.Endpoint
			}

			o.BaseEndpoint = &options.Endpoint
		}
	}

	client := s3.NewFromConfig(conf,
		setEndPointFn)

	return &filesystem{client: client, bucket: options.Bucket}, nil
}

type filesystem struct {
	client *s3.Client
	bucket string

	// This should only be used in tests.
	root string
}

func (f *filesystem) fullPath(name string) string {
	switch {
	case f.root == "":
		return name
	case name == ".":
		return f.root
	default:
		// do not use path.Join, we DON'T want the path to be cleaned
		return f.root + "/" + name
	}
}

func (f *filesystem) OpenFile(name string, flag int, _ fs.FileMode) (fs.File, error) {
	file, err := f.openFile(f.fullPath(name), flag)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}

	return file, nil
}

func (f *filesystem) openFile(name string, flag int) (fs.File, error) {
	// S3 does not support modifying existing files. Thus, the Append flag
	// is not allowed.
	if flag&fs.FlagAppend != 0 {
		return nil, fs.ErrNotImplemented
	}

	if err := checkValidPath(name); err != nil {
		return nil, err
	}

	// S3 does not allow to read a file while its being written. So, in effect,
	// the ReadWrite flags functions identically to the WriteOnly flag, and trying
	// to read from the file will result in an error.
	if flag&fs.FlagWOnly == 0 && flag&fs.FlagRW == 0 && flag&fs.FlagCreate == 0 {
		info, statErr := statObject(f.client, f.bucket, name)
		if statErr != nil {
			return nil, statErr
		}

		if info.IsDir() {
			return newDir(f.client, f.bucket, name), nil
		}

		return newReadFile(f.client, f.bucket, name)
	}

	if err := writeFileFlagsCheck(f.client, f.bucket, name, flag); err != nil {
		return nil, err
	}

	return newWriteFile(f.client, f.bucket, name), nil
}

func (f *filesystem) Open(name string) (fs.File, error) {
	return f.OpenFile(name, fs.FlagROnly, 0)
}

func (f *filesystem) Create(name string) (fs.File, error) {
	//nolint:mnd //magic number is required here to mimic os.Create
	return f.OpenFile(name, fs.FlagWOnly|fs.FlagCreate|fs.FlagTruncate, 0o666)
}

func (f *filesystem) Stat(name string) (fs.FileInfo, error) {
	info, err := f.stat(f.fullPath(name))
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: err}
	}

	return info, nil
}

func (f *filesystem) stat(name string) (fs.FileInfo, error) {
	if err := checkValidPath(name); err != nil {
		return nil, err
	}

	info, statErr := statObject(f.client, f.bucket, name)
	if statErr != nil {
		return nil, statErr
	}

	return info, nil
}

func (f *filesystem) Remove(name string) error {
	if err := f.remove(f.fullPath(name)); err != nil {
		return &fs.PathError{Op: "remove", Path: name, Err: err}
	}

	return nil
}

func (f *filesystem) remove(name string) error {
	if err := checkValidPath(name); err != nil {
		return err
	}

	info, statErr := statObject(f.client, f.bucket, name)
	if statErr != nil {
		return statErr
	}

	if info.IsDir() {
		// check that dir is empty
		children, listErr := f.ReadDir(strings.TrimPrefix(name, f.root+"/"))
		if listErr != nil {
			return listErr
		}

		if len(children) > 0 {
			return fs.ErrNotEmpty
		}

		name += "/"
	}

	input := &s3.DeleteObjectInput{Bucket: &f.bucket, Key: &name}

	ctx, cancel := context.WithTimeout(context.Background(), deleteTimeout)
	defer cancel()

	if _, err := f.client.DeleteObject(ctx, input); err != nil {
		return wrapS3Error("failed to delete object", err)
	}

	return nil
}

func (f *filesystem) ReadDir(name string) ([]fs.DirEntry, error) {
	file, openErr := f.Open(name)
	if openErr != nil {
		return nil, openErr
	}

	entries, err := f.readDir(file)
	if err != nil {
		return entries, &fs.PathError{Op: "readdir", Path: name, Err: err}
	}

	return entries, nil
}

func (f *filesystem) readDir(file fs.File) ([]fs.DirEntry, error) {
	dir, canReadDir := file.(*directory)
	if !canReadDir {
		return nil, fs.ErrNotDir
	}

	entries, readErr := dir.readDir()
	if readErr != nil {
		return entries, readErr
	}

	return entries, nil
}

func (f *filesystem) Mkdir(name string, _ fs.FileMode) error {
	if err := f.mkdir(f.fullPath(name)); err != nil {
		return &fs.PathError{Op: "mkdir", Path: name, Err: err}
	}

	return nil
}

func (f *filesystem) mkdir(name string) error {
	if err := checkValidPath(name); err != nil {
		return err
	}

	// Check if object already exists.
	if _, err := statObject(f.client, f.bucket, name); err == nil {
		// object already exists, return an error
		return fs.ErrExist
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	// Check if parent dir exists.
	if parent := path.Dir(name); parent != "." {
		parentInfo, parErr := statObject(f.client, f.bucket, parent+"/")
		if parErr != nil {
			return parErr
		}

		if !parentInfo.IsDir() {
			return fs.ErrNotDir
		}
	}

	key := aws.String(name + "/")
	if path.Clean(name) == "." {
		key = aws.String("/")
	}

	input := &s3.PutObjectInput{
		Bucket: &f.bucket,
		Key:    key,
	}

	ctx, cancel := context.WithTimeout(context.Background(), mkdirTimeout)
	defer cancel()

	if _, err := f.client.PutObject(ctx, input); err != nil {
		return wrapS3Error("failed to create the object", err)
	}

	return nil
}

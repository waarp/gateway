package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

const (
	filePerm = 0o666
	dirPerm  = 0o777
)

func checkValidPath(name string) error {
	if fs.ValidPath(name) {
		return nil
	}

	return fs.ErrInvalid
}

func getFileMode(name string) fs.FileMode {
	if strings.HasSuffix(name, "/") {
		return dirPerm | fs.ModeDir
	}

	return filePerm
}

func statObject(client *s3.Client, bucket, name string) (*fs.GenericFileInfo, error) {
	if path.Clean(name) == "." {
		return &fs.GenericFileInfo{
			FileName:    name,
			FileSize:    0,
			FileMode:    fs.ModeDir,
			LastModTime: time.Time{},
			DataSource:  nil,
		}, nil
	}

	input := &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    aws.String(name),
	}

	ctx, cancel := context.WithTimeout(context.Background(), statTimeout)
	defer cancel()

	resp, getErr := client.HeadObject(ctx, input)
	if getErr != nil {
		if isNotFound(getErr) && !strings.HasSuffix(name, "/") {
			// If file does not exist, try to find a directory instead.
			return statObject(client, bucket, name+"/")
		}

		return nil, wrapS3Error("failed to retrieve the object's info", getErr)
	}

	return &fs.GenericFileInfo{
		FileName:    name,
		FileSize:    aws.ToInt64(resp.ContentLength),
		FileMode:    getFileMode(name),
		LastModTime: aws.ToTime(resp.LastModified),
		DataSource:  nil,
	}, nil
}

func isNotFound(err error) bool {
	var (
		nskErr *types.NoSuchKey
		nfErr  *types.NotFound
	)

	return errors.As(err, &nskErr) || errors.As(err, &nfErr)
}

func wrapS3Error(msg string, err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, io.EOF) {
		return io.EOF
	}

	if isNotFound(err) {
		return fs.ErrNotExist
	}

	return fmt.Errorf("%s: %w", msg, err)
}

func writeFileFlagsCheck(client *s3.Client, bucket, name string, flag int) error {
	info, statErr := statObject(client, bucket, name)
	if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
		return statErr
	}

	// If file does not exist
	if statErr != nil {
		// If no Create flag, return an error
		if flag&fs.FlagCreate == 0 {
			return fs.ErrNotExist
		}

		// If parent directory does not exist, return an error
		parent := path.Dir(name)
		if parent != "." {
			if _, err := statObject(client, bucket, parent+"/"); err != nil {
				return err
			}
		}
	} else { // If file does exist
		if flag&fs.FlagExclusive != 0 {
			// If Exclusive flag is present, return an error
			return fs.ErrExist
		}

		// Cannot open an existing directory in write mode.
		if info.IsDir() {
			return fs.ErrIsDir
		}
	}

	return nil
}

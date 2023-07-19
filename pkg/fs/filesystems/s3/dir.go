package s3

import (
	"context"
	"errors"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

var ErrReadOnDir = errors.New("cannot call Read() on directories")

type directory struct {
	client *s3.Client

	bucket, name string

	closed chan bool
}

func newDir(client *s3.Client, bucket, dirName string) *directory {
	return &directory{
		client: client,
		bucket: bucket,
		name:   dirName,
		closed: make(chan bool),
	}
}

func (d *directory) Stat() (fs.FileInfo, error) {
	return statObject(d.client, d.bucket, path.Clean(d.name+"/"))
}

func (d *directory) Read([]byte) (int, error) { return 0, ErrReadOnDir }
func (d *directory) Close() error {
	select {
	case <-d.closed:
		return fs.ErrClosed
	default:
		close(d.closed)
	}

	return nil
}

func (d *directory) readDir() ([]fs.DirEntry, error) {
	prefix := aws.String(d.name + "/")
	if path.Clean(d.name) == "." {
		prefix = nil
	}

	input := &s3.ListObjectsV2Input{
		Bucket:    &d.bucket,
		Prefix:    prefix,
		Delimiter: aws.String("/"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()

	resp, listErr := d.client.ListObjectsV2(ctx, input)
	if listErr != nil {
		return nil, wrapS3Error("failed to list the objects", listErr)
	}

	var entries []fs.DirEntry

	for _, object := range resp.Contents {
		objName := path.Clean(aws.ToString(object.Key))
		if objName == d.name {
			continue
		}

		info, statErr := statObject(d.client, d.bucket, *object.Key)
		if statErr != nil {
			return entries, statErr
		}

		entries = append(entries, &fs.GenericDirEntry{GenericFileInfo: info})
	}

	for _, dir := range resp.CommonPrefixes {
		dirName := path.Clean(aws.ToString(dir.Prefix))
		if dirName == d.name {
			continue
		}

		info, statErr := statObject(d.client, d.bucket, *dir.Prefix)
		if statErr != nil {
			return entries, statErr
		}

		entries = append(entries, &fs.GenericDirEntry{GenericFileInfo: info})
	}

	fs.SortDirEntries(entries)

	return entries, nil
}

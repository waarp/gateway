package s3

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

type writeFile struct {
	client      *s3.Client
	bucket, key string

	writer *io.PipeWriter
	errors chan error
	cancel context.CancelFunc

	closed chan bool
}

func newWriteFile(client *s3.Client, bucket, key string) fs.File {
	uploader := manager.NewUploader(client)
	reader, writer := io.Pipe()

	input := &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   reader,
	}

	ctx, cancel := context.WithCancel(context.Background())

	file := &writeFile{
		client: client,
		bucket: bucket,
		key:    key,
		writer: writer,
		errors: make(chan error),
		cancel: cancel,
		closed: make(chan bool),
	}

	go func() {
		if _, err := uploader.Upload(ctx, input); err != nil {
			file.errors <- err
		}

		close(file.errors)
	}()

	return file
}

func (f *writeFile) Stat() (fs.FileInfo, error) {
	return statObject(f.client, f.bucket, f.key)
}

func (f *writeFile) Read([]byte) (int, error) {
	return 0, fs.ErrInvalid
}

func (f *writeFile) Write(p []byte) (int, error) {
	//nolint:wrapcheck //wrapping the error here adds nothing
	return f.writer.Write(p)
}

func (f *writeFile) Close() error {
	select {
	case <-f.closed:
		return fs.ErrClosed
	default:
		close(f.closed)
	}

	defer func() {
		<-f.errors
		f.cancel()
	}()

	//nolint:wrapcheck //never returns an error
	return f.writer.Close()
}

package s3

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type readFile struct {
	client *s3.Client

	bucket, key string

	offset int64
	cancel func()
	body   io.ReadCloser

	closed chan bool
}

func newReadFile(client *s3.Client, bucket, key string) (fs.File, error) {
	file := &readFile{
		client: client,
		bucket: bucket,
		key:    key,
		closed: make(chan bool),
	}

	var err error
	if file.body, file.cancel, err = file.makeRequest(0, -1); err != nil {
		return nil, err
	}

	return file, nil
}

func (f *readFile) makeRequest(off int64, size int) (io.ReadCloser, func(), error) {
	input := &s3.GetObjectInput{
		Bucket: &f.bucket,
		Key:    &f.key,
	}

	if off > 0 || size >= 0 {
		input.Range = aws.String(fmt.Sprintf("bytes=%d-%s", off,
			utils.If(size < 0, "", utils.FormatInt(off+int64(size)))))
	}

	ctx, cancel := context.WithCancel(context.Background())

	resp, getErr := f.client.GetObject(ctx, input)
	if getErr != nil {
		cancel()

		return nil, nil, wrapS3Error("failed to retrieve the object", getErr)
	}

	return resp.Body, cancel, nil
}

func (f *readFile) Stat() (fs.FileInfo, error) {
	return statObject(f.client, f.bucket, f.key)
}

func (f *readFile) Read(p []byte) (n int, err error) {
	//nolint:wrapcheck //wrapping here adds nothing
	return f.body.Read(p)
}

func (f *readFile) ReadAt(p []byte, off int64) (n int, err error) {
	body, cancel, getErr := f.makeRequest(off, len(p))
	if getErr != nil {
		return 0, getErr
	}

	defer cancel()
	defer body.Close() //nolint:errcheck //error is irrelevant

	n, err = io.ReadFull(body, p)
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return n, io.EOF
	}

	return n, wrapS3Error("failed to read the response body", err)
}

func (f *readFile) CurrentOffset() int64 { return f.offset }

func (f *readFile) Seek(offset int64, whence int) (int64, error) {
	n, err := f.seek(offset, whence)
	if err != nil {
		return n, &fs.PathError{Op: "seek", Path: f.key, Err: err}
	}

	return n, nil
}

func (f *readFile) seek(offset int64, whence int) (int64, error) {
	newOffset, offErr := fs.GetSeekNewOffset(f, offset, whence)
	if offErr != nil {
		return 0, offErr //nolint:wrapcheck //wrapping here breaks tests
	}

	f.cancel()

	if err := f.body.Close(); err != nil {
		return 0, wrapS3Error("failed to close the response body", err)
	}

	var reqErr error
	if f.body, f.cancel, reqErr = f.makeRequest(newOffset, -1); reqErr != nil {
		return 0, reqErr
	}

	f.offset = newOffset

	return f.offset, nil
}

func (f *readFile) Close() error {
	select {
	case <-f.closed:
		return fs.ErrClosed
	default:
		close(f.closed)
	}

	defer f.cancel()

	//nolint:wrapcheck //wrapping here adds nothing
	return f.body.Close()
}

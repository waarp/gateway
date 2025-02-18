package gwtesting

import (
	"errors"
	"hash"
	"io"
	"slices"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

var (
	ErrHashMismatch  = errors.New("hash mismatch")
	ErrInvalidWhence = errors.New("invalid whence")
)

func SendFile(content string) protocol.SendFile {
	return dummySendFile{r: strings.NewReader(content)}
}

func ReceiveFile() protocol.ReceiveFile {
	return &dummyReceiveFile{}
}

type dummySendFile struct {
	r *strings.Reader
}

//nolint:wrapcheck // wrapping adds nothing here
func (d dummySendFile) Read(p []byte) (int, error) { return d.r.Read(p) }

//nolint:wrapcheck // wrapping adds nothing here
func (d dummySendFile) ReadAt(p []byte, off int64) (int, error) { return d.r.ReadAt(p, off) }

//nolint:wrapcheck // wrapping adds nothing here
func (d dummySendFile) Seek(offset int64, whence int) (int64, error) { return d.r.Seek(offset, whence) }

type dummyReceiveFile struct {
	w    []byte
	wOff int64
}

func (d *dummyReceiveFile) Write(p []byte) (n int, _ error) {
	defer func() { d.wOff += int64(n) }()

	d.w = append(d.w[:d.wOff], p...)

	return len(p), nil
}

func (d *dummyReceiveFile) WriteAt(p []byte, off int64) (int, error) {
	d.w = append(d.w[:off], p...)

	return len(p), nil
}

func (d *dummyReceiveFile) CheckHash(hasher hash.Hash, expected []byte) error {
	if slices.Equal(hasher.Sum(d.w), expected) {
		return nil
	}

	return ErrHashMismatch
}

func (d *dummyReceiveFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		d.wOff = offset
	case io.SeekCurrent:
		d.wOff += offset
	case io.SeekEnd:
		d.wOff = int64(len(d.w)) + offset
	default:
		return 0, ErrInvalidWhence
	}

	return d.wOff, nil
}

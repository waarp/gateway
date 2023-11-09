package fs

import "bytes"

type Buffer struct {
	*bytes.Reader
}

func NewBuffer(p []byte) *Buffer {
	return &Buffer{Reader: bytes.NewReader(p)}
}

func (b *Buffer) Close() error { return nil }

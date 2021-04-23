package testhelpers

import (
	"crypto/rand"
	"io"

	"github.com/smartystreets/goconvey/convey"
)

const TestBuffSize = 1000000

type TestStream struct {
	content []byte
	rOff    int
}

func NewSrcStream(c convey.C) *TestStream {
	cont := make([]byte, TestBuffSize)
	_, err := rand.Read(cont)
	c.So(err, convey.ShouldBeNil)
	return &TestStream{content: cont}
}

func NewDstStream(_ convey.C) *TestStream {
	return &TestStream{content: make([]byte, 0, TestBuffSize)}
}

func (t *TestStream) Read(p []byte) (int, error) {
	if t.rOff >= len(t.content) {
		return 0, io.EOF
	}
	n := copy(p, t.content[t.rOff:])
	t.rOff += n
	if t.rOff >= len(t.content) {
		return n, io.EOF
	}
	return n, nil
}

func (t *TestStream) Write(p []byte) (int, error) {
	t.content = append(t.content, p...)
	return len(p), nil
}

func (t *TestStream) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(t.content)) {
		return 0, io.EOF
	}
	n := copy(p, t.content[off:])
	if int(off)+n >= len(t.content) {
		return n, io.EOF
	}
	return n, nil
}

func (t *TestStream) WriteAt(p []byte, off int64) (int, error) {
	if off > int64(len(t.content)) {
		t.content = t.content[:off]
	}
	t.content = append(t.content, p...)
	return len(p), nil
}

func (t *TestStream) Content() []byte {
	cop := make([]byte, len(t.content))
	copy(cop, t.content)
	return cop
}

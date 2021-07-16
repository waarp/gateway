package testhelpers

import (
	"bytes"
	"crypto/rand"

	"github.com/smartystreets/goconvey/convey"
)

const TestStreamSize int64 = 1000000 // 1MB

type TestReader struct {
	cont []byte
	*bytes.Buffer
}

// NewTestReader returns a new buffered reader which can be used in tests.
func NewTestReader(c convey.C) *TestReader {
	r := &TestReader{}
	r.cont = make([]byte, TestStreamSize)
	_, err := rand.Read(r.cont)
	c.So(err, convey.ShouldBeNil)
	r.Buffer = bytes.NewBuffer(r.cont)
	return r
}

func (t *TestReader) Content() []byte {
	return t.cont
}

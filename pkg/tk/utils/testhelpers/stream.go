package testhelpers

import (
	"bytes"
	"crypto/rand"

	"github.com/smartystreets/goconvey/convey"
)

// TestStreamSize defines the size of the TestReader returned by the NewTestReader
// function.
const TestStreamSize int64 = 1000000 // 1MB

// TestReader is a wrapper for bytes.Buffer which can be used for testing. When
// initialized, the reader is filled with random data, which can then be retrieved
// with the Content function to check that the data has been transmitted correctly.
type TestReader struct {
	cont []byte
	*bytes.Buffer
}

// NewTestReader returns a new TestReader which can be used in tests. The buffer
// will be filled with random data, which can then be retrieved with the
// TestReader.Content function.
func NewTestReader(c convey.C) *TestReader {
	r := &TestReader{}
	r.cont = make([]byte, TestStreamSize)
	_, err := rand.Read(r.cont)

	c.So(err, convey.ShouldBeNil)

	r.Buffer = bytes.NewBuffer(r.cont)

	return r
}

// Content returns the full content of the buffer. Note that this does not affect
// the state of the reader in any way.
func (t *TestReader) Content() []byte {
	return t.cont
}

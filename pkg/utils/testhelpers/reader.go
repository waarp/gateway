package testhelpers

import (
	"crypto/rand"
	"time"
)

const slowReaderDelay = 100 * time.Millisecond

func NewSlowReader() *slowReader {
	return &slowReader{tick: time.NewTicker(slowReaderDelay)}
}

type slowReader struct {
	tick *time.Ticker
}

func (l *slowReader) Read(b []byte) (int, error) {
	<-l.tick.C

	return rand.Read(b) //nolint:wrapcheck //useless here, only used for tests
}

func (l *slowReader) ReadAt(b []byte, _ int64) (int, error) {
	<-l.tick.C

	return rand.Read(b) //nolint:wrapcheck //useless here, only used for tests
}

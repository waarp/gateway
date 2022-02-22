package testhelpers

import (
	"crypto/rand"
	"time"
)

func NewLimitedReader(lim int) *limitedReader {
	return &limitedReader{lim: lim, tick: time.NewTicker(time.Second)}
}

type limitedReader struct {
	lim  int
	tick *time.Ticker
}

func (l *limitedReader) Read(b []byte) (int, error) {
	<-l.tick.C

	return rand.Read(b[:l.lim]) //nolint:wrapcheck //useless here, only used for tests
}

func (l *limitedReader) ReadAt(b []byte, _ int64) (int, error) {
	<-l.tick.C

	return rand.Read(b[:l.lim]) //nolint:wrapcheck //useless here, only used for tests
}

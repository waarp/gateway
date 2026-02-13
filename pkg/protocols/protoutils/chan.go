package protoutils

import (
	"sync"

	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

type ChanOnce[T any] struct {
	once sync.Once
	c    chan T
}

func NewChanOnce[T any]() *ChanOnce[T] {
	return &ChanOnce[T]{
		c: make(chan T, 1),
	}
}

func (c *ChanOnce[T]) Send(val T) {
	c.once.Do(func() {
		c.c <- val
		close(c.c)
	})
}

func (c *ChanOnce[T]) Close() {
	c.once.Do(func() { close(c.c) })
}

func (c *ChanOnce[T]) Receive() <-chan T {
	return c.c
}

type ErrChan = ChanOnce[*pipeline.Error]

func NewErrChan() *ErrChan {
	return NewChanOnce[*pipeline.Error]()
}

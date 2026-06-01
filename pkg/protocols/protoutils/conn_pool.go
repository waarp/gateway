package protoutils

import (
	"errors"
	"io"
	"sync/atomic"
	"time"

	"github.com/puzpuzpuz/xsync/v4"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

const DefaultConnGracePeriod = 5 * time.Second

var ErrConnectionPoolClosed = errors.New("connection pool is closed")

type OpenConnFunc[T io.Closer] func(*pipeline.Pipeline, *TraceDialer) (T, error)

type counter[T io.Closer] struct {
	conn  T
	count uint
	grace *time.Timer
}

func newCounter[T io.Closer](conn T) *counter[T] {
	return &counter[T]{conn: conn, count: 1}
}

func (c *counter[T]) inc() {
	c.count++
	if c.grace != nil {
		c.grace.Stop()
	}
}

func (c *counter[T]) dec() {
	c.count--
}

func (c *counter[T]) close(loaded bool) (*counter[T], xsync.ComputeOp) {
	if !loaded || c == nil || c.count > 0 {
		return c, xsync.CancelOp
	}

	_ = c.conn.Close()

	return c, xsync.DeleteOp
}

func (c *counter[T]) forceClose() error {
	if c.grace != nil {
		c.grace.Stop()
	}

	//nolint:wrapcheck //no need to wrap here
	return c.conn.Close()
}

type ConnPool[T io.Closer] struct {
	pool        *xsync.Map[int64, *counter[T]]
	dialer      *TraceDialer
	openConn    OpenConnFunc[T]
	gracePeriod time.Duration
	closed      atomic.Bool
	// exclusive controls whether a pooled connection can be shared between
	// concurrent users. When false (default), multiple Connect() calls for
	// the same key return the same connection (suitable for R66/SFTP which
	// support concurrent sessions). When true, if a connection is already
	// in use (count > 0), a new standalone connection is created instead
	// (suitable for PeSIT which is sequential per-connection).
	exclusive bool
}

func NewConnPool[T io.Closer](dialer *TraceDialer, openConn OpenConnFunc[T]) *ConnPool[T] {
	return &ConnPool[T]{
		pool:        xsync.NewMap[int64, *counter[T]](),
		dialer:      dialer,
		openConn:    openConn,
		gracePeriod: DefaultConnGracePeriod,
	}
}

func (c *ConnPool[T]) SetGracePeriod(duration time.Duration) {
	c.gracePeriod = duration
}

// SetExclusive enables exclusive mode where each Connect() gets its own
// connection. When a pooled connection is already in use, a new standalone
// connection is created. This is needed for protocols like PeSIT where a
// single connection only supports one transfer at a time.
// When the connection is not in use (count == 0, during grace period),
// it is reused normally (sequential reuse).
func (c *ConnPool[T]) SetExclusive(exclusive bool) {
	c.exclusive = exclusive
}

func (c *ConnPool[T]) Connect(pip *pipeline.Pipeline) (T, error) {
	if c.isClosed() {
		return *new(T), ErrConnectionPoolClosed
	}

	key := pip.TransCtx.RemoteAccount.ID

	// In exclusive mode, check if the connection is currently in use.
	// If so, create a standalone (non-pooled) connection instead.
	if c.exclusive {
		var inUse bool

		c.pool.Compute(key, func(info *counter[T], loaded bool) (*counter[T], xsync.ComputeOp) {
			if loaded && info.count > 0 {
				inUse = true
			}

			return info, xsync.CancelOp // read-only check
		})

		if inUse {
			conn, err := c.openConn(pip, c.dialer)
			if err != nil {
				return *new(T), err
			}

			return conn, nil
		}
	}

	var err error
	info, _ := c.pool.Compute(key, func(info *counter[T], loaded bool) (*counter[T], xsync.ComputeOp) {
		if c.isClosed() {
			err = ErrConnectionPoolClosed

			return info, xsync.CancelOp
		}

		if loaded {
			info.inc()

			return info, xsync.UpdateOp
		}

		var conn T
		conn, err = c.openConn(pip, c.dialer)
		if err != nil {
			return info, xsync.CancelOp
		}

		return newCounter(conn), xsync.UpdateOp
	})

	if err != nil {
		return *new(T), err
	}

	return info.conn, nil
}

func (c *ConnPool[T]) CloseConn(pip *pipeline.Pipeline) {
	c.CloseConnFor(pip.TransCtx.RemoteAccount)
}

func (c *ConnPool[T]) CloseConnFor(account *model.RemoteAccount) {
	key := account.ID

	c.pool.Compute(key, func(info *counter[T], loaded bool) (*counter[T], xsync.ComputeOp) {
		if c.isClosed() || !loaded || info == nil || info.count <= 0 {
			return info, xsync.CancelOp
		}

		info.dec()

		if info.count > 0 {
			return info, xsync.UpdateOp
		}

		if c.gracePeriod == 0 {
			info.close(loaded)
		} else {
			info.grace = time.AfterFunc(c.gracePeriod, func() {
				c.pool.Compute(key, (*counter[T]).close)
			})
		}

		return info, xsync.UpdateOp
	})
}

// ForceDisconnect immediately removes and closes the connection for the
// given account from the pool, bypassing the grace period. Use this when
// a connection is known to be in an invalid state (e.g., after an ABORT).
func (c *ConnPool[T]) ForceDisconnect(account *model.RemoteAccount) {
	key := account.ID

	c.pool.Compute(key, func(info *counter[T], loaded bool) (*counter[T], xsync.ComputeOp) {
		if !loaded || info == nil {
			return info, xsync.CancelOp
		}

		_ = info.forceClose()

		return nil, xsync.DeleteOp
	})
}

// Evict removes the connection for the given account from the pool WITHOUT
// closing it. Use this when the connection has already been closed externally
// (e.g., after sending an ABORT) and you just need to clean up the pool entry.
func (c *ConnPool[T]) Evict(account *model.RemoteAccount) {
	key := account.ID

	c.pool.Compute(key, func(info *counter[T], loaded bool) (*counter[T], xsync.ComputeOp) {
		if !loaded || info == nil {
			return info, xsync.CancelOp
		}

		if info.grace != nil {
			info.grace.Stop()
		}

		return nil, xsync.DeleteOp
	})
}

func (c *ConnPool[T]) Stop() error {
	c.closed.Store(true)

	errs := make([]error, 0, c.pool.Size())
	c.pool.Range(func(key int64, _ *counter[T]) bool {
		c.pool.Compute(key, func(value *counter[T], loaded bool) (*counter[T], xsync.ComputeOp) {
			if loaded {
				errs = append(errs, value.forceClose())
			}

			return nil, xsync.DeleteOp
		})

		return true
	})

	return errors.Join(errs...)
}

func (c *ConnPool[T]) Exists(account *model.RemoteAccount) bool {
	_, ok := c.pool.Load(account.ID)

	return ok
}

func (c *ConnPool[T]) isClosed() bool {
	return c.closed.Load()
}

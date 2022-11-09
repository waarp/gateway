package internal

import (
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"code.waarp.fr/lib/log"
	"code.waarp.fr/lib/r66"
)

const DefaultConnectionGracePeriod = 5 * time.Second

type connInfo struct {
	conn *r66.Client // the connection
	num  uint64      // the number of sessions open on the connection
}

// ConnPool is a pool of r66 client connections used for multiplexing.
type ConnPool struct {
	m               map[string]*connInfo
	mux             sync.Mutex
	closed          chan bool
	connGracePeriod time.Duration
}

// NewConnPool initiates and returns a new ConnPool instance.
func NewConnPool() *ConnPool {
	return &ConnPool{
		m:               map[string]*connInfo{},
		closed:          make(chan bool),
		connGracePeriod: DefaultConnectionGracePeriod,
	}
}

// Exists returns whether a connection to the given address exists in the pool.
func (c *ConnPool) Exists(addr string) bool {
	c.mux.Lock()
	defer c.mux.Unlock()

	_, ok := c.m[addr]

	return ok
}

// Add returns an R66 connection to the given address. If a connection to that
// address already exists in the connection pool, it returns that connection and
// increments its usage counter. If no connection exists to the given address,
// it opens a new one and adds it to the pool with a counter of 1.
func (c *ConnPool) Add(addr string, tlsConf *tls.Config, logger *log.Logger) (*r66.Client, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	info, ok := c.m[addr]
	if ok {
		info.num++

		return info.conn, nil
	}

	var err error

	var cli *r66.Client

	if tlsConf == nil {
		cli, err = r66.Dial(addr, logger.AsStdLogger(log.LevelTrace))
	} else {
		cli, err = r66.DialTLS(addr, tlsConf, logger.AsStdLogger(log.LevelTrace))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote host: %w", err)
	}

	c.m[addr] = &connInfo{conn: cli, num: 1}

	return cli, nil
}

// Done informs the connection pool that the client has finished using connection
// to the given address by decrementing its usage counter. If the counter reaches
// 0, the connection  is closed and removed from the pool. Otherwise, it stays in
// the pool until the other clients have finished using it.
func (c *ConnPool) Done(addr string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	info, ok := c.m[addr]
	if !ok {
		return
	}

	info.num--
	if info.num <= 0 {
		go c.waitClose(addr)
	}
}

func (c *ConnPool) waitClose(addr string) {
	timer := time.NewTimer(c.connGracePeriod)
	select {
	case <-timer.C:
	case <-c.closed:
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	info, ok := c.m[addr]
	if !ok || info.num > 0 {
		return
	}

	info.conn.Close()
	delete(c.m, addr)
}

func (c *ConnPool) ForceClose() {
	c.mux.Lock()
	defer c.mux.Unlock()

	for addr, info := range c.m {
		info.conn.Close()
		delete(c.m, addr)
	}

	close(c.closed)
}

package internal

import (
	"crypto/tls"
	"fmt"
	"sync"

	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/log"
)

type connInfo struct {
	conn *r66.Client // the connection
	num  uint64      // the number of sessions open on the connection
}

// ConnPool is a pool of r66 client connections used for multiplexing.
type ConnPool struct {
	m   map[string]*connInfo
	mux sync.Mutex
}

// NewConnPool initiates and returns a new ConnPool instance.
func NewConnPool() *ConnPool {
	return &ConnPool{m: map[string]*connInfo{}}
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
		cli, err = r66.Dial(addr, logger.AsStdLog(logging.DEBUG))
	} else {
		cli, err = r66.DialTLS(addr, tlsConf, logger.AsStdLog(logging.DEBUG))
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

	if info.num <= 1 {
		info.conn.Close()
		delete(c.m, addr)
	} else {
		info.num--
	}
}

package internal

import (
	"crypto/tls"
	"sync"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"

	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/waarp-r66/r66"
)

type connInfo struct {
	conn *r66.Client // the connection
	num  uint64      // the number of sessions open on the connection
}

type ConnPool struct {
	m   map[string]*connInfo
	mux sync.Mutex
}

func NewConnPool() *ConnPool {
	return &ConnPool{m: map[string]*connInfo{}}
}

func (c *ConnPool) Add(addr string, tlsConf *tls.Config, logger *log.Logger) (*r66.Client, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	info, ok := c.m[addr]
	if ok {
		info.num++
		return info.conn, nil
	}

	var cli *r66.Client
	var err error
	if tlsConf == nil {
		cli, err = r66.Dial(addr, logger.AsStdLog(logging.DEBUG))
	} else {
		cli, err = r66.DialTLS(addr, tlsConf, logger.AsStdLog(logging.DEBUG))
	}
	if err != nil {
		return nil, err
	}

	c.m[addr] = &connInfo{conn: cli, num: 1}
	return cli, nil
}

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

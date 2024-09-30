package protoutils

import (
	"context"
	"net"
	"sync"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
)

type TraceListener struct {
	net.Listener
}

//nolint:wrapcheck //no need to wrap here
func (l *TraceListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return conn, err
	}

	analytics.AddIncomingConnection()

	return &TraceServerConn{Conn: conn}, nil
}

type TraceDialer struct {
	*net.Dialer
}

func (d *TraceDialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

//nolint:wrapcheck //no need to wrap here
func (d *TraceDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return conn, err
	}

	analytics.AddOutgoingConnection()

	return &TraceClientConn{Conn: conn}, err
}

type TraceServerConn struct {
	net.Conn
	once sync.Once
}

func (c *TraceServerConn) Close() error {
	defer c.once.Do(analytics.SubIncomingConnection)

	//nolint:wrapcheck //no need to wrap here
	return c.Conn.Close()
}

type TraceClientConn struct {
	net.Conn
	once sync.Once
}

func (c *TraceClientConn) Close() error {
	defer c.once.Do(analytics.SubOutgoingConnection)

	//nolint:wrapcheck //no need to wrap here
	return c.Conn.Close()
}

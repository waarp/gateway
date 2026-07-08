package protoutils

import (
	"context"
	"fmt"
	"net"
	"sync"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
)

type TraceDialer struct {
	*net.Dialer
}

func NewDialerFor(localAddr string) (*TraceDialer, error) {
	dialer := &net.Dialer{}
	if localAddr != "" {
		tcpAddr, err := net.ResolveTCPAddr("tcp", localAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the client's local address %q: %w",
				localAddr, err)
		}

		dialer.LocalAddr = tcpAddr
	}

	return &TraceDialer{Dialer: dialer}, nil
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

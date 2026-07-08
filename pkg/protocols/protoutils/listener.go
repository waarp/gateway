package protoutils

import (
	"crypto/tls"
	"net"
	"sync"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
)

type listener struct {
	net.Listener
	close     func() error
	tlsConfig *tls.Config
}

func Listen(network, address string) (net.Listener, error) {
	list, err := net.Listen(network, address)
	if err != nil {
		return nil, err //nolint:wrapcheck //wrapping adds nothing here
	}

	return wrapListener(list, nil), nil
}

func ListenTLS(network, address string, tlsConfig *tls.Config) (net.Listener, error) {
	list, err := net.Listen(network, address)
	if err != nil {
		return nil, err //nolint:wrapcheck //wrapping adds nothing here
	}

	return wrapListener(list, tlsConfig), nil
}

func wrapListener(l net.Listener, tlsCon *tls.Config) net.Listener {
	return &listener{
		Listener:  l,
		close:     sync.OnceValue(l.Close),
		tlsConfig: tlsCon,
	}
}

//nolint:wrapcheck //no need to wrap here
func (l *listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return conn, err
	}

	analytics.AddIncomingConnection()
	conn = &TraceServerConn{Conn: conn}

	if l.tlsConfig != nil {
		conn = tls.Server(conn, l.tlsConfig)
	}

	return conn, nil
}

func (l *listener) Close() error { return l.close() }

package httptransport

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

func NewTransport(overHttps bool, localAddr string) (Transporter, error) {
	dialer := &net.Dialer{}
	if localAddr != "" {
		tcpAddr, err := net.ResolveTCPAddr("tcp", localAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the AS2 client's local address %q: %w",
				localAddr, err)
		}

		dialer.LocalAddr = tcpAddr
	}

	tracer := &protoutils.TraceDialer{Dialer: dialer}

	if overHttps {
		return &HttpsTransport{
			pool: protoutils.NewConnPool(tracer, newHttpsConn),
		}, nil
	}

	return &HttpTransport{
		t: &http.Transport{
			DialContext: tracer.DialContext,
		},
	}, nil
}

type Transporter interface {
	Get(pip *pipeline.Pipeline) (*http.Transport, error)
	Return(pip *pipeline.Pipeline)
}

type HttpTransport struct {
	t *http.Transport
}

func (h *HttpTransport) Get(*pipeline.Pipeline) (*http.Transport, error) { return h.t, nil }
func (h *HttpTransport) Return(*pipeline.Pipeline)                       {}

type httpsTransport struct{ *http.Transport }

func (h *httpsTransport) Close() error { return nil }

type HttpsTransport struct {
	pool *protoutils.ConnPool[*httpsTransport]
}

func (h *HttpsTransport) Get(pip *pipeline.Pipeline) (*http.Transport, error) {
	t, err := h.pool.Connect(pip)
	if err != nil {
		return nil, err
	}

	return t.Transport, nil
}

func (h *HttpsTransport) Return(pip *pipeline.Pipeline) {
	h.pool.CloseConnFor(pip.TransCtx.RemoteAccount)
}

//nolint:wrapcheck //no need to wrap here
func newHttpsConn(pip *pipeline.Pipeline, dialer *protoutils.TraceDialer) (*httpsTransport, error) {
	minVersion := getMinTLSVersion(pip.TransCtx)
	tlsConfig, tlsErr := protoutils.GetClientTLSConfig(pip.TransCtx, pip.Logger, minVersion)
	if tlsErr != nil {
		return nil, tlsErr
	}

	return &httpsTransport{
		Transport: &http.Transport{
			ForceAttemptHTTP2: true,
			DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				conn, err := dialer.DialContext(ctx, network, addr)
				if err != nil {
					return nil, err
				}

				return tls.Client(conn, tlsConfig), nil
			},
		},
	}, nil
}

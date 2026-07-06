package httptransport

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:gochecknoglobals //needs to be a variable for tests
var idleConnTimeout = protoutils.DefaultConnGracePeriod

//nolint:mnd //magic number is for tests
func SetTestIdleConnTimeout() { idleConnTimeout = 100 * time.Millisecond }

type Transporter struct {
	transport *http.Transport
	overrides *conf.ConfigOverride
}

func NewTransporter(overHttps bool, localAddr string, overrides *conf.ConfigOverride) (*Transporter, error) {
	if localAddr != "" {
		host, port, err := net.SplitHostPort(localAddr)
		if err != nil {
			return nil, err //nolint:wrapcheck //wrapping adds nothing here
		}

		localAddr = overrides.GetRealAddress(host, port)
	}

	dialer, err := protoutils.NewDialerFor(localAddr)
	if err != nil {
		return nil, err
	}

	if !overHttps {
		return &Transporter{transport: &http.Transport{
			IdleConnTimeout: idleConnTimeout,
			DialContext:     dialer.DialContext,
		}}, nil
	}

	return &Transporter{transport: &http.Transport{
		IdleConnTimeout: idleConnTimeout,
		DialTLSContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			pip, ok := ctx.Value(pipKey).(*pipeline.Pipeline)
			if !ok {
				return nil, errNoPipeline
			}

			return dialHttps(ctx, dialer, overrides, pip)
		},
	}}, nil
}

func (t *Transporter) Connect(pip *pipeline.Pipeline) http.RoundTripper {
	return &rTripper{
		transport: t.transport,
		pip:       pip,
		overrides: t.overrides,
	}
}

func (t *Transporter) Close() {
	t.transport.CloseIdleConnections()
}

type pipKeyType string

const pipKey pipKeyType = "pipeline"

var errNoPipeline = errors.New("pipeline not found in context")

type rTripper struct {
	transport *http.Transport
	pip       *pipeline.Pipeline
	overrides *conf.ConfigOverride
}

func (rt *rTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := context.WithValue(req.Context(), pipKey, rt.pip)
	*req = *req.WithContext(ctx)

	//nolint:wrapcheck //wrapping adds nothing here
	return rt.transport.RoundTrip(req)
}

func dialHttps(ctx context.Context, dialer *protoutils.TraceDialer,
	ovrd *conf.ConfigOverride, pip *pipeline.Pipeline,
) (net.Conn, error) {
	addr := ovrd.GetRealAddress(pip.TransCtx.RemoteAgent.Address.Host,
		utils.FormatUint(pip.TransCtx.RemoteAgent.Address.Port))

	tlsConfig, confErr := protoutils.GetClientTLSConfig(pip.TransCtx, pip.Logger)
	if confErr != nil {
		return nil, confErr
	}

	tcpConn, tcpErr := dialer.DialContext(ctx, "tcp", addr)
	if tcpErr != nil {
		return nil, tcpErr
	}

	return tls.Client(tcpConn, tlsConfig), nil
}

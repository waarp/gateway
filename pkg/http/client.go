package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

var (
	errPause    = types.NewTransferError(types.TeStopped, "transfer paused by remote host")
	errShutdown = types.NewTransferError(types.TeShuttingDown, "remote host is shutting down")
	errCancel   = types.NewTransferError(types.TeCanceled, "transfer canceled by remote host")
)

//nolint:gochecknoinits // init is used by design
func init() {
	pipeline.ClientConstructors["http"] = newHTTPClient
	pipeline.ClientConstructors["https"] = newHTTPSClient
}

func newHTTPClient(pip *pipeline.Pipeline) (pipeline.Client, *types.TransferError) {
	return newClient(pip, nil), nil
}

func newHTTPSClient(pip *pipeline.Pipeline) (pipeline.Client, *types.TransferError) {
	tlsConf, err := makeTLSConf(pip.TransCtx)
	if err != nil {
		pip.Logger.Errorf("failed to make TLS configuration: %s", err)

		return nil, types.NewTransferError(types.TeInternal, "failed to make TLS configuration")
	}

	return newClient(pip, tlsConf), nil
}

func newClient(pip *pipeline.Pipeline, tlsConf *tls.Config) pipeline.Client {
	if pip.TransCtx.Rule.IsSend {
		return &postClient{
			pip:       pip,
			transport: &http.Transport{TLSClientConfig: tlsConf},
			reqErr:    make(chan error),
			resp:      make(chan *http.Response),
		}
	}

	return &getClient{
		pip:       pip,
		transport: &http.Transport{TLSClientConfig: tlsConf},
	}
}

func makeTLSConf(transCtx *model.TransferContext) (*tls.Config, error) {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		rootCAs = x509.NewCertPool()
	}

	for _, ca := range transCtx.RemoteAgentCryptos {
		if !rootCAs.AppendCertsFromPEM([]byte(ca.Certificate)) {
			//nolint:wrapcheck,goerr113 // this is a base error
			return nil, fmt.Errorf("failed to add certificate %s to cert pool", ca.Name)
		}
	}

	var certs []tls.Certificate

	for _, ce := range transCtx.RemoteAccountCryptos {
		cert, err := tls.X509KeyPair([]byte(ce.Certificate), []byte(ce.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse client certificate %s: %w", ce.Name, err)
		}

		certs = append(certs, cert)
	}

	tlsConf := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		RootCAs:      rootCAs,
		Certificates: certs,
	}

	for _, c := range transCtx.RemoteAccountCryptos {
		cert, err := tls.X509KeyPair([]byte(c.Certificate), []byte(c.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse TLS certificate: %w", err)
		}

		tlsConf.Certificates = append(tlsConf.Certificates, cert)
	}

	return tlsConf, nil
}

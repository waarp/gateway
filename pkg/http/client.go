package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
)

var (
	errPause    = types.NewTransferError(types.TeStopped, "transfer paused by remote host")
	errShutdown = types.NewTransferError(types.TeShuttingDown, "remote host is shutting down")
	errCancel   = types.NewTransferError(types.TeCanceled, "transfer cancelled by remote host")
)

func init() {
	pipeline.ClientConstructors["http"] = newHTTPClient
	pipeline.ClientConstructors["https"] = newHTTPSClient
}

func newHTTPClient(pip *pipeline.Pipeline) (pipeline.Client, *types.TransferError) {
	return newClient(pip, nil)
}

func newHTTPSClient(pip *pipeline.Pipeline) (pipeline.Client, *types.TransferError) {
	tlsConf, err := makeTLSConf(pip.TransCtx)
	if err != nil {
		pip.Logger.Errorf("failed to make TLS configuration: %s", err)
		return nil, types.NewTransferError(types.TeInternal, "failed to make TLS configuration")
	}
	return newClient(pip, tlsConf)
}

func newClient(pip *pipeline.Pipeline, tlsConf *tls.Config) (pipeline.Client, *types.TransferError) {
	if pip.TransCtx.Rule.IsSend {
		return &postClient{
			pip:       pip,
			transport: &http.Transport{TLSClientConfig: tlsConf},
			reqErr:    make(chan error),
			resp:      make(chan *http.Response),
		}, nil
	}
	return &getClient{
		pip:       pip,
		transport: &http.Transport{TLSClientConfig: tlsConf},
	}, nil
}

func makeTLSConf(transCtx *model.TransferContext) (*tls.Config, error) {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		rootCAs = x509.NewCertPool()
	}
	for _, ca := range transCtx.RemoteAgentCryptos {
		if !rootCAs.AppendCertsFromPEM([]byte(ca.Certificate)) {
			return nil, fmt.Errorf("failed to add certificate %s to cert pool", ca.Name)
		}
	}

	var certs []tls.Certificate
	for _, ce := range transCtx.RemoteAccountCryptos {
		cert, err := tls.X509KeyPair([]byte(ce.Certificate), []byte(ce.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse client certificate %s", ce.Name)
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
			return nil, err
		}
		tlsConf.Certificates = append(tlsConf.Certificates, cert)
	}

	return tlsConf, nil
}

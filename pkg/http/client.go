package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
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
	tlsConf, err := makeTLSConf(pip)
	if err != nil {
		pip.Logger.Error("failed to make TLS configuration: %s", err)

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

func makeTLSConf(pip *pipeline.Pipeline) (*tls.Config, error) {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		rootCAs = x509.NewCertPool()
	}

	for _, crypto := range pip.TransCtx.RemoteAgentCryptos {
		certChain, parseErr := utils.ParsePEMCertChain(crypto.Certificate)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse the certificate chain: %w", parseErr)
		}

		rootCAs.AddCert(certChain[0])
	}

	tlsConf := &tls.Config{
		MinVersion:       tls.VersionTLS12,
		RootCAs:          rootCAs,
		VerifyConnection: compatibility.LogSha1(pip.Logger),
	}

	for _, crypto := range pip.TransCtx.RemoteAccountCryptos {
		cert, err := tls.X509KeyPair([]byte(crypto.Certificate), []byte(crypto.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse TLS certificate: %w", err)
		}

		tlsConf.Certificates = append(tlsConf.Certificates, cert)
	}

	return tlsConf, nil
}

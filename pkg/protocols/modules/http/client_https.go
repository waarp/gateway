package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

const schemeHTTPS = "https://"

type httpsClient struct {
	*httpClient
}

func (h *httpsClient) Start() error {
	if err := h.httpClient.Start(); err != nil {
		return err
	}

	h.transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}

	return nil
}

func (h *httpsClient) InitTransfer(pip *pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	transport, err := h.makeTransport(pip)
	if err != nil {
		return nil, pipeline.NewErrorWith(types.TeInternal, "failed to initialize HTTPS client", err)
	}

	return newTransferClient(pip, transport, true), nil
}

func (h *httpsClient) makeTransport(pip *pipeline.Pipeline) (*http.Transport, error) {
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

	var certs []tls.Certificate

	for _, ce := range pip.TransCtx.RemoteAccountCryptos {
		cert, err := tls.X509KeyPair([]byte(ce.Certificate), []byte(ce.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse client certificate %s: %w", ce.Name, err)
		}

		certs = append(certs, cert)
	}

	transport := h.transport.Clone()
	transport.TLSClientConfig.RootCAs = rootCAs
	transport.TLSClientConfig.Certificates = certs
	transport.TLSClientConfig.VerifyConnection = compatibility.LogSha1(pip.Logger)

	return transport, nil
}

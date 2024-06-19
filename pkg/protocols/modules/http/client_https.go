package http

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
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
	rootCAs := utils.TLSCertPool()

	for _, cred := range pip.TransCtx.RemoteAgentCreds {
		if cred.Type != auth.TLSTrustedCertificate {
			continue
		}

		certChain, parseErr := utils.ParsePEMCertChain(cred.Value)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse the certificate chain: %w", parseErr)
		}

		rootCAs.AddCert(certChain[0])
	}

	var certs []tls.Certificate

	for _, ce := range pip.TransCtx.RemoteAccountCreds {
		if ce.Type != auth.TLSCertificate {
			continue
		}

		cert, err := utils.X509KeyPair(ce.Value, ce.Value2)
		if err != nil {
			return nil, fmt.Errorf("failed to parse client certificate %s: %w", ce.Name, err)
		}

		certs = append(certs, cert)
	}

	transport := h.transport.Clone()
	transport.TLSClientConfig.ServerName = pip.TransCtx.RemoteAgent.Address.Host
	transport.TLSClientConfig.RootCAs = rootCAs
	transport.TLSClientConfig.Certificates = certs
	transport.TLSClientConfig.VerifyConnection = compatibility.LogSha1(pip.Logger)

	if err := auth.AddTLSAuthorities(pip.DB, transport.TLSClientConfig); err != nil {
		return nil, fmt.Errorf("failed to setup the TLS authorities: %w", err)
	}

	return transport, nil
}

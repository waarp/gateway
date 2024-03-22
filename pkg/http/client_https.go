package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

type httpsClient struct {
	*httpClient
}

func NewHTTPSClient(dbClient *model.Client) (pipeline.Client, error) {
	var clientConf config.HTTPClientProtoConfig
	if err := utils.JSONConvert(dbClient.ProtoConfig, &clientConf); err != nil {
		return nil, fmt.Errorf("failed to parse the HTTPS client's proto config: %w", err)
	}

	return &httpsClient{
		httpClient: &httpClient{
			transfers: service.NewTransferMap(),
			client:    dbClient,
			transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			},
		},
	}, nil
}

func (h *httpsClient) InitTransfer(pip *pipeline.Pipeline) (pipeline.TransferClient, *types.TransferError) {
	transport, err := h.makeTransport(pip)
	if err != nil {
		return nil, types.NewTransferError(types.TeInternal, err.Error())
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

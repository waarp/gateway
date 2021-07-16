package http

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
)

var (
	errPause    = types.NewTransferError(types.TeStopped, "transfer paused by remote host")
	errShutdown = types.NewTransferError(types.TeShuttingDown, "remote host is shutting down")
	errCancel   = types.NewTransferError(types.TeCanceled, "transfer cancelled by remote host")
)

func init() {
	pipeline.ClientConstructors["http"] = NewClient
}

// NewClient initializes and returns a new HTTP transfer client.
func NewClient(pip *pipeline.Pipeline) (pipeline.Client, *types.TransferError) {
	var conf config.HTTPProtoConfig
	if err := json.Unmarshal(pip.TransCtx.RemoteAgent.ProtoConfig, &conf); err != nil {
		pip.Logger.Errorf("failed to parse R66 partner configuration: %s", err)
		return nil, types.NewTransferError(types.TeInternal, "failed to parse partner configuration")
	}
	tlsConf, err := makeTLSConf(pip.TransCtx, &conf)
	if err != nil {
		pip.Logger.Errorf("failed to make TLS configuration: %s", err)
		return nil, types.NewTransferError(types.TeInternal, "failed to make TLS configuration")
	}

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

func makeTLSConf(transCtx *model.TransferContext, conf *config.HTTPProtoConfig) (*tls.Config, error) {
	if !conf.UseHTTPS {
		return nil, nil
	}

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

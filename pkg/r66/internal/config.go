// Package internal regroups internal utility functions for the pipeline module.
package internal

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

// MakeClientTLSConfig takes a client R66 transfer context and returns a TLS
// configuration suited for making that transfer. If the partner does not use
// TLS, the returned configuration will be nil.
func MakeClientTLSConfig(pip *pipeline.Pipeline) (*tls.Config, error) {
	tlsCerts := make([]tls.Certificate, len(pip.TransCtx.RemoteAccountCryptos))

	for i, cert := range pip.TransCtx.RemoteAccountCryptos {
		var err error

		tlsCerts[i], err = tls.X509KeyPair([]byte(cert.Certificate), []byte(cert.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse TLS certificate: %w", err)
		}
	}

	conf := &tls.Config{
		Certificates: tlsCerts,
		MinVersion:   tls.VersionTLS12,
	}

	if model.IsLegacyR66CertificateAllowed {
		conf.InsecureSkipVerify = true
		conf.VerifyConnection = func(state tls.ConnectionState) error {
			//nolint:errcheck //never returns an error
			_ = compatibility.LogSha1(pip.Logger)(state)

			//nolint:wrapcheck //error is returned as-is for better readability
			return pip.TransCtx.RemoteAgentCryptos.CheckServerAuthent(state.ServerName,
				state.PeerCertificates)
		}

		return conf, nil
	} else {
		conf.VerifyConnection = compatibility.LogSha1(pip.Logger)
	}

	var caPool *x509.CertPool
	for _, cert := range pip.TransCtx.RemoteAgentCryptos {
		if caPool == nil {
			caPool = x509.NewCertPool()
		}

		caPool.AppendCertsFromPEM([]byte(cert.Certificate))
	}

	conf.RootCAs = caPool

	return conf, nil
}

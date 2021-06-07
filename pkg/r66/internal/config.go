package internal

import (
	"crypto/tls"
	"crypto/x509"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func MakeClientTLSConfig(info *model.TransferContext) (*tls.Config, error) {
	tlsCerts := make([]tls.Certificate, len(info.RemoteAccountCryptos))
	for i, cert := range info.RemoteAccountCryptos {
		var err error
		tlsCerts[i], err = tls.X509KeyPair([]byte(cert.Certificate), []byte(cert.PrivateKey))
		if err != nil {
			return nil, err
		}
	}

	var caPool *x509.CertPool
	for _, cert := range info.RemoteAgentCryptos {
		if caPool == nil {
			caPool = x509.NewCertPool()
		}
		caPool.AppendCertsFromPEM([]byte(cert.Certificate))
	}

	return &tls.Config{
		Certificates: tlsCerts,
		MinVersion:   tls.VersionTLS12,
		RootCAs:      caPool,
	}, nil
}

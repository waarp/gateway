package gwtesting

import (
	"crypto/tls"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	ServerCertPEM = testhelpers.LocalhostCert
	ServerKeyPEM  = testhelpers.LocalhostKey

	ClientCertPEM = testhelpers.ClientFooCert
	ClientKeyPEM  = testhelpers.ClientFooKey
)

//nolint:gochecknoglobals //global var needed for tests
var (
	ServerCert tls.Certificate
	ClientCert tls.Certificate
)

//nolint:gochecknoinits //init is needed here
func init() {
	var err error
	if ServerCert, err = tls.X509KeyPair([]byte(ServerCertPEM), []byte(ServerKeyPEM)); err != nil {
		panic(err)
	}

	if ClientCert, err = tls.X509KeyPair([]byte(ClientCertPEM), []byte(ClientKeyPEM)); err != nil {
		panic(err)
	}
}

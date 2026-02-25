package gwtesting

import (
	"crypto/tls"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	LocalhostCertPEM = testhelpers.LocalhostCert
	LocalhostKeyPEM  = testhelpers.LocalhostKey
)

//nolint:gochecknoglobals //global var needed for tests
var LocalhostCert tls.Certificate

//nolint:gochecknoinits //init is needed here
func init() {
	var err error
	if LocalhostCert, err = tls.X509KeyPair([]byte(LocalhostCertPEM), []byte(LocalhostKeyPEM)); err != nil {
		panic(err)
	}
}

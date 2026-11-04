package auth

import (
	"crypto/x509"
	"testing"
)

// TestMain gives the handlers a single root of the system, the test authority
// publicRootCertPEM: the chains it signs pass the way those of a public
// authority do, and no test of the package reaches the network (#844).
func TestMain(m *testing.M) {
	systemRoots = func() *x509.CertPool {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(publicRootCertPEM)) {
			panic("the public root certificate of the tests is not a PEM certificate")
		}

		return pool
	}

	m.Run()
}

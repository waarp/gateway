package auth

import "crypto/x509"

// systemRoots gives the certification authorities the system trusts, the
// roots of the public authorities. The tests of the package replace it with
// a pool they fill, so that the public-authority path runs off the network.
//
//nolint:gochecknoglobals //the tests of the package replace it
var systemRoots = x509.NewCertPool()

//nolint:gochecknoinits //init is required here
func init() {
	if sysRoots, err := x509.SystemCertPool(); err == nil {
		systemRoots = sysRoots
	}
}

func SystemRoots() *x509.CertPool {
	return systemRoots.Clone()
}

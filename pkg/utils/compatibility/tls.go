// Package compatibility regroups utilities used to maintain backwards-compatibility
// with deprecated features.
package compatibility

import (
	"crypto/tls"
	"crypto/x509"
	"os"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:gochecknoinits //must use init here
func init() {
	if err := os.Setenv("GODEBUG", "x509sha1=1"); err != nil {
		logging.NewLogger("TLS").Warningf(
			"Failed to set the SHA1 environment variable, SHA1 signed certificates will not be accepted: %v", err)
	}
}

// LogSha1 takes a client tls.Config instance and adds a trigger which
// logs a deprecation warning when a remote server uses a certificate signed
// with SHA-1, which is deprecated. This function can then be assigned to the
// VerifyConnection field of a tls.Config.
//
// Once SHA-1 certificates are definitely phased out from the x509 library,
// this function can be changed to a noop (or just straight up removed).
func LogSha1(logger *log.Logger) func(tls.ConnectionState) error {
	return func(state tls.ConnectionState) error {
		if len(state.PeerCertificates) > 0 {
			cert := state.PeerCertificates[0]
			switch cert.SignatureAlgorithm {
			case x509.SHA1WithRSA, x509.DSAWithSHA1, x509.ECDSAWithSHA1:
				name := cert.Subject.CommonName
				if len(cert.DNSNames) > 0 {
					name = cert.DNSNames[0]
				}

				logger.Warningf("The certificate of partner %q is signed using "+
					"SHA-1 which is deprecated. All SHA-1 based signature "+
					"algorithms will be disallowed out shortly.", name)
			default:
			}
		}

		return nil
	}
}

// CheckSHA1 checks if the given pem certificate chain contains any certificate
// signed using SHA1. If it does, it returns a message warning that SHA1 based
// algorithms are deprecated and will be disallowed shortly.
func CheckSHA1(pem string) string {
	certChain, err := utils.ParsePEMCertChain(pem)
	if err != nil && len(certChain) == 0 {
		return ""
	}

	var warn string

	for _, cert := range certChain {
		switch cert.SignatureAlgorithm {
		case x509.SHA1WithRSA, x509.DSAWithSHA1, x509.ECDSAWithSHA1:
			warn = "This certificate is signed using SHA-1 which is deprecated. " +
				"All SHA-1 based signature algorithms will be disallowed out shortly."
		default:
		}
	}

	return warn
}

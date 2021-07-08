package utils

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
)

// MakeSignature takes a x509 certificate and return the base64 encoded sha256
// checksum of the certificate.
func MakeSignature(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.Raw)
	return base64.StdEncoding.EncodeToString(sum[:])
}

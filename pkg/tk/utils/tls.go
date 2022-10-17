package utils

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

var (
	errInvalidPEM           = fmt.Errorf("certificate input is not a valid PEM block")
	errCertParse            = fmt.Errorf("failed to parse certificate")
	errNoCertInPEM          = fmt.Errorf("no certificate found in PEM block")
	errVerifyEmptyCertChain = fmt.Errorf("cannot verify an empty certificate chain")
)

// ParsePEMCertChain takes a certification chain in PEM format, parses it, and
// returns it as a slice of x509.Certificate. If the decoding or the parsing
// fails, an error is returned.
func ParsePEMCertChain(pemCert string) ([]*x509.Certificate, error) {
	var certChain []*x509.Certificate

	rest := []byte(pemCert)

	for len(rest) > 0 {
		var block *pem.Block

		block, rest = pem.Decode(rest)
		if block == nil || block.Type != "CERTIFICATE" {
			return nil, errInvalidPEM
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, errCertParse
		}

		certChain = append(certChain, cert)
	}

	if len(certChain) == 0 {
		return nil, errNoCertInPEM
	}

	return certChain, nil
}

// CheckCertChain takes a certification chain, in the form of a slice of
// x509.Certificate (with the leaf first) and verifies if the chain is valid.
// An optional hostname can also be given to check if the certificate covers
// that domain.
func CheckCertChain(certChain []*x509.Certificate, isServer bool,
	host string,
) error {
	if len(certChain) == 0 {
		return errVerifyEmptyCertChain
	}

	options := x509.VerifyOptions{
		Roots:         x509.NewCertPool(),
		Intermediates: x509.NewCertPool(),
		DNSName:       host,
	}

	if isServer {
		options.KeyUsages = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	} else {
		options.KeyUsages = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}

	for i := 1; i < len(certChain)-1; i++ {
		options.Intermediates.AddCert(certChain[i])
	}

	options.Roots.AddCert(certChain[len(certChain)-1])

	if _, err := certChain[0].Verify(options); err != nil {
		return fmt.Errorf("certificate is invalid: %w", err)
	}

	return nil
}

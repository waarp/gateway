package utils

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
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
			return nil, fmt.Errorf("certificate input is not a valid PEM block")
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate")
		}

		certChain = append(certChain, cert)
	}
	if len(certChain) == 0 {
		return nil, fmt.Errorf("no certificate found in PEM block")
	}

	return certChain, nil
}

// CheckCertChain takes a certification chain, in the form of a slice of
// x509.Certificate (with the leaf first) and and verifies if the chain is valid.
// An optional hostname can also be given to check if the certificate covers
// that domain.
func CheckCertChain(certChain []*x509.Certificate, hostname string) error {
	if len(certChain) == 0 {
		return fmt.Errorf("cannot verify an empty certificate chain")
	}

	options := x509.VerifyOptions{
		DNSName:       hostname,
		Intermediates: x509.NewCertPool(),
	}
	if hostname == "" {
		options.KeyUsages = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	} else {
		options.KeyUsages = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	}
	roots := x509.NewCertPool()

	for i := 1; i < len(certChain)-1; i++ {
		if certChain[i].Issuer.CommonName == certChain[i].Subject.CommonName {
			roots.AddCert(certChain[i])
		} else {
			options.Intermediates.AddCert(certChain[i])
		}
	}

	if certChain[0].Issuer.CommonName == certChain[0].Subject.CommonName {
		roots.AddCert(certChain[0])
	}

	if len(roots.Subjects()) > 0 {
		options.Roots = roots
	}

	if _, err := certChain[0].Verify(options); err != nil {
		return err
	}
	return nil
}

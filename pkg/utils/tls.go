package utils

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

var (
	errInvalidPEM           = errors.New("certificate input is not a valid PEM block")
	errCertParse            = errors.New("failed to parse certificate")
	errNoCertInPEM          = errors.New("no certificate found in PEM block")
	errVerifyEmptyCertChain = errors.New("cannot verify an empty certificate chain")
)

func TLSCertPool() *x509.CertPool {
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}

	return pool
}

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

// KeyUsageToStrings returns a slice containing string representations of the
// usages contained in the given x509.KeyUsage.
func KeyUsageToStrings(keyUsage x509.KeyUsage) []string {
	var usages []string

	if keyUsage&x509.KeyUsageDigitalSignature != 0 {
		usages = append(usages, "Digital Signature")
	}

	if keyUsage&x509.KeyUsageContentCommitment != 0 {
		usages = append(usages, "Content Commitment")
	}

	if keyUsage&x509.KeyUsageKeyEncipherment != 0 {
		usages = append(usages, "Key Encipherment")
	}

	if keyUsage&x509.KeyUsageDataEncipherment != 0 {
		usages = append(usages, "Data Encipherment")
	}

	if keyUsage&x509.KeyUsageKeyAgreement != 0 {
		usages = append(usages, "Key Agreement")
	}

	if keyUsage&x509.KeyUsageCertSign != 0 {
		usages = append(usages, "Certificate Sign")
	}

	if keyUsage&x509.KeyUsageCRLSign != 0 {
		usages = append(usages, "CRL Sign")
	}

	if keyUsage&x509.KeyUsageEncipherOnly != 0 {
		usages = append(usages, "Encipher Only")
	}

	if keyUsage&x509.KeyUsageDecipherOnly != 0 {
		usages = append(usages, "Decipher Only")
	}

	return usages
}

// ExtKeyUsageToString returns a slice of string representations of the given
// list of x509.ExtKeyUsage.
func ExtKeyUsageToString(keyUsage x509.ExtKeyUsage) string {
	switch keyUsage {
	case x509.ExtKeyUsageServerAuth:
		return "Server Auth"
	case x509.ExtKeyUsageClientAuth:
		return "Client Auth"
	case x509.ExtKeyUsageCodeSigning:
		return "Code Signing"
	case x509.ExtKeyUsageEmailProtection:
		return "Email Protection"
	case x509.ExtKeyUsageIPSECEndSystem:
		return "IPSEC End System"
	case x509.ExtKeyUsageIPSECTunnel:
		return "IPSEC Tunnel"
	case x509.ExtKeyUsageIPSECUser:
		return "IPSEC User"
	case x509.ExtKeyUsageTimeStamping:
		return "Time Stamping"
	case x509.ExtKeyUsageOCSPSigning:
		return "OCSP Signing"
	case x509.ExtKeyUsageMicrosoftServerGatedCrypto:
		return "Microsoft Server Gated Crypto"
	case x509.ExtKeyUsageNetscapeServerGatedCrypto:
		return "Netscape Server Gated Crypto"
	case x509.ExtKeyUsageMicrosoftCommercialCodeSigning:
		return "Microsoft Commercial Code Signing"
	case x509.ExtKeyUsageMicrosoftKernelCodeSigning:
		return "Microsoft Kernel Code Signing"
	default:
		return "Unknown"
	}
}

func ExtKeyUsagesToStrings(keyUsages []x509.ExtKeyUsage) []string {
	var usages []string

	for _, keyUsage := range keyUsages {
		usages = append(usages, ExtKeyUsageToString(keyUsage))
	}

	return usages
}

func X509KeyPair(certPEM, keyPEM string) (tls.Certificate, error) {
	//nolint:wrapcheck //this is just a helper to avoid casting, no need to wrap
	return tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
}

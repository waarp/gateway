package common

import (
	"bytes"
	"crypto/x509"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type CertValidator struct {
	Host   string
	Usages []x509.ExtKeyUsage
}

func (c CertValidator) Validate(cert, issuer *x509.Certificate) error {
	validOpts := x509.VerifyOptions{
		DNSName:       c.Host,
		Roots:         utils.TLSCertPool(),
		Intermediates: x509.NewCertPool(),
		KeyUsages:     c.Usages,
	}

	if issuer != nil {
		validOpts.Intermediates.AddCert(issuer)
	}

	// If certificate is self-signed, add it to root CAs
	if bytes.Equal(cert.RawSubject, cert.RawIssuer) {
		validOpts.Roots.AddCert(cert)
	}

	_, err := cert.Verify(validOpts)

	return err //nolint:wrapcheck //wrapping adds nothing here
}

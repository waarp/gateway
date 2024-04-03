package auth

import (
	"crypto/x509"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
)

const (
	AuthorityTLS = "tls_authority"
)

//nolint:gochecknoinits //init is required here
func init() {
	authentication.AddAuthorityType(AuthorityTLS, &TLSAuthorityHandler{})
}

type TLSAuthorityHandler struct{}

func (*TLSAuthorityHandler) Validate(identity string) error {
	cert, err := ParseCertPEM(identity)
	if err != nil {
		return err
	}

	options := &x509.VerifyOptions{KeyUsages: []x509.ExtKeyUsage{
		x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth,
	}}

	return verifyCert(cert, nil, options)
}

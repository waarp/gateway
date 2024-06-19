package auth

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"golang.org/x/exp/slices"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
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

func AddTLSAuthorities(db database.ReadAccess, tlsConfig *tls.Config) error {
	var authorities model.Authorities
	if err := db.Select(&authorities).Where("type=?", AuthorityTLS).Run(); err != nil {
		return fmt.Errorf("failed to retrieve the TLS certification authorities: %w", err)
	}

	if tlsConfig.RootCAs == nil {
		tlsConfig.RootCAs = utils.TLSCertPool()
	}

	for _, authority := range authorities {
		// If the authority is not valid for the server name, skip it
		if len(authority.ValidHosts) != 0 && !slices.Contains(
			authority.ValidHosts, tlsConfig.ServerName) {
			continue
		}

		tlsConfig.RootCAs.AppendCertsFromPEM([]byte(authority.PublicIdentity))
	}

	return nil
}

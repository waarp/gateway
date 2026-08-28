package auth

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"slices"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	AuthorityTLS = "tls_authority"
)

var ErrNotCACertificate = errors.New("the provided certificate is not a CA certificate")

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

	if !cert.IsCA {
		return ErrNotCACertificate
	}

	store := x509.NewCertPool()
	store.AddCert(cert)

	return verifyCertChain([]*x509.Certificate{cert}, store, "", x509.ExtKeyUsageAny)
}

func AddTLSAuthorities(db database.ReadAccess, tlsConfig *tls.Config) error {
	return addTLSAuthorities(db, tlsConfig.RootCAs, tlsConfig.ServerName)
}

func addTLSAuthorities(db database.ReadAccess, store *x509.CertPool, host string) error {
	var authorities model.Authorities
	if err := db.Select(&authorities).Where("type=?", AuthorityTLS).Run(); err != nil {
		return fmt.Errorf("failed to retrieve the TLS certification authorities: %w", err)
	}

	if store == nil {
		store = utils.TLSCertPool()
	}

	for _, authority := range authorities {
		// If the authority is not valid for the server name, skip it
		if host != "" && len(authority.ValidHosts) != 0 &&
			!slices.Contains(authority.ValidHosts, host) {
			continue
		}

		store.AppendCertsFromPEM([]byte(authority.PublicIdentity))
	}

	return nil
}

func newTLSRootPool(db database.ReadAccess, host string) (*x509.CertPool, error) {
	pool := utils.TLSCertPool()
	if err := addTLSAuthorities(db, pool, host); err != nil {
		return nil, err
	}

	return pool, nil
}

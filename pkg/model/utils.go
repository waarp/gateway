package model

import (
	"crypto/x509"
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

type agent interface {
	database.Table
	database.Identifier
}

// GetCryptos fetch from the database then return the associated Cryptos if they exist
func GetCryptos(db database.ReadAccess, agent agent) ([]Crypto, database.Error) {
	var certs Cryptos
	query := db.Select(&certs).Where("owner_type=? AND owner_id=?",
		agent.TableName(), agent.GetID())
	if err := query.Run(); err != nil {
		return nil, err
	}

	// TODO: get only validate certificates
	return certs, nil
}

// CheckClientAuthent checks whether the given certificate chain is valid as a
// a client certificate for the given login, using the target cryptos as root CAs.
func (c *Cryptos) CheckClientAuthent(login string, certs []*x509.Certificate) error {
	if len(*c) == 0 {
		return fmt.Errorf("no certificates found for user '%s'", login)
	}

	roots, err := x509.SystemCertPool()
	if err != nil {
		roots = x509.NewCertPool()
	}
	for _, crypto := range *c {
		roots.AppendCertsFromPEM([]byte(crypto.Certificate))
	}
	intermediate := x509.NewCertPool()
	for _, cert := range certs {
		intermediate.AddCert(cert)
	}
	opt := x509.VerifyOptions{
		DNSName:       login,
		Roots:         roots,
		Intermediates: intermediate,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	if _, err := certs[0].Verify(opt); err != nil {
		return fmt.Errorf("certificate is not valid for user '%s'", login)
	}

	return nil
}

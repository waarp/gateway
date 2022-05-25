package model

import (
	"crypto/x509"
	"fmt"
	"math"
	"math/big"

	"github.com/bwmarrin/snowflake"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

type agent interface {
	database.Table
	database.Identifier
}

var errNoCertificates = fmt.Errorf("no certificates found for user")

// GetCryptos fetch from the database then return the associated Cryptos if they exist.
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
// client certificate for the given login, using the target cryptos as root CAs.
func (c *Cryptos) CheckClientAuthent(login string, certs []*x509.Certificate) error {
	if len(*c) == 0 {
		return errNoCertificates
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
		return fmt.Errorf("certificate is not valid for user '%s': %w", login, err)
	}

	return nil
}

func makeIDGenerator() (*snowflake.Node, error) {
	var nodeID, mod, machineID big.Int

	nodeID.SetBytes([]byte(conf.GlobalConfig.NodeID))
	mod.SetInt64(math.MaxInt64)

	machineID.Mod(&nodeID, &mod)

	generator, err := snowflake.NewNode(machineID.Int64())
	if err != nil {
		return nil, fmt.Errorf("failed to create the ID generator: %w", err)
	}

	return generator, nil
}

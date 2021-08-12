package model

import "code.waarp.fr/apps/gateway/gateway/pkg/database"

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

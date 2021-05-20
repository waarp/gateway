package model

import "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"

type agent interface {
	database.Table
	database.Identifier
}

// GetCerts fetch from the database then return the associated Certificates if they exist
func GetCerts(db database.ReadAccess, agent agent) ([]Cert, database.Error) {
	var certs Certificates
	query := db.Select(&certs).Where("owner_type=? AND owner_id=?",
		agent.TableName(), agent.GetID())
	if err := query.Run(); err != nil {
		return nil, err
	}

	// TODO: get only validate certificates
	return certs, nil
}

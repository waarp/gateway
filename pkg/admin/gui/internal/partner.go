package internal

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func GetPartner(db database.ReadAccess, name string) (*model.RemoteAgent, error) {
	var partner model.RemoteAgent

	return &partner, db.Get(&partner, "name=?", name).Owner().Run()
}

func GetPartnerByID(db database.ReadAccess, id int64) (*model.RemoteAgent, error) {
	var partner model.RemoteAgent

	return &partner, db.Get(&partner, "id=?", id).Run()
}

func GetPartnersLike(db *database.DB, prefix string) ([]*model.RemoteAgent, error) {
	const limit = 5
	var partners model.RemoteAgents

	return partners, db.Select(&partners).Owner().Where("name LIKE ?", prefix+"%").
		OrderBy("name", true).Limit(limit, 0).Run()
}

func ListPartners(db database.ReadAccess, orderByCol string, orderByAsc bool, limit, offset int,
	protocols ...string,
) ([]*model.RemoteAgent, error) {
	var partners model.RemoteAgents
	query := db.Select(&partners).Limit(limit, offset).OrderBy(orderByCol, orderByAsc)

	protoSlice := make([]any, len(protocols))
	for i, protocol := range protocols {
		protoSlice[i] = protocol
	}

	query.In("protocol", protoSlice...)

	return partners, query.Run()
}

func InsertPartner(db database.Access, partner *model.RemoteAgent) error {
	return db.Insert(partner).Run()
}

func UpdatePartner(db database.Access, partner *model.RemoteAgent) error {
	return db.Update(partner).Run()
}

func DeletePartner(db database.Access, partner *model.RemoteAgent) error {
	return db.Delete(partner).Run()
}

func GetPartnerCredential(db database.ReadAccess, partnerName, name string) (*model.Credential, error) {
	partner, pErr := GetPartner(db, partnerName)
	if pErr != nil {
		return nil, pErr
	}

	return getCredential(db, partner, name)
}

func GetPartnerCredentialByID(db database.ReadAccess, partnerName string, id int64) (*model.Credential, error) {
	partner, pErr := GetPartner(db, partnerName)
	if pErr != nil {
		return nil, pErr
	}

	return getCredentialByID(db, partner, id)
}

func ListPartnerCredentials(db database.ReadAccess, partnerName string,
	orderByCol string, orderByAsc bool, limit, offset int, types ...string,
) ([]*model.Credential, error) {
	partner, pErr := GetPartner(db, partnerName)
	if pErr != nil {
		return nil, pErr
	}

	return listCredentials(db, partner, orderByCol, orderByAsc, limit, offset, types...)
}

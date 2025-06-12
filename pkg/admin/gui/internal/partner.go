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

func ListPartners(db database.ReadAccess, orderByCol string, orderByAsc bool, limit, offset int,
	protocols ...string,
) ([]*model.RemoteAgent, error) {
	var partners model.RemoteAgents
	query := db.Select(&partners).Limit(limit, offset).OrderBy(orderByCol, orderByAsc)

	for _, protocol := range protocols {
		query = query.Where("protocol=?", protocol)
	}

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

func ListPartnerCredentials(db database.ReadAccess, partnerName string,
	orderByCol string, orderByAsc bool, limit, offset int, types ...string,
) ([]*model.Credential, error) {
	partner, pErr := GetPartner(db, partnerName)
	if pErr != nil {
		return nil, pErr
	}

	return listCredentials(db, partner, orderByCol, orderByAsc, limit, offset, types...)
}

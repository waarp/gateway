//nolint:dupl // another page
package internal

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func GetPartnerAccountByID(db database.ReadAccess, id int64) (*model.RemoteAccount, error) {
	var account model.RemoteAccount

	return &account, db.Get(&account, "id=?", id).Run()
}

func ListPartnerAccounts(db database.ReadAccess, partnerName,
	orderByCol string, orderByAsc bool, limit, offset int,
) ([]*model.RemoteAccount, error) {
	partner, pErr := GetPartner(db, partnerName)
	if pErr != nil {
		return nil, pErr
	}

	var accounts model.RemoteAccounts

	return accounts, db.Select(&accounts).Where("remote_agent_id=?", partner.ID).
		Limit(limit, offset).OrderBy(orderByCol, orderByAsc).Run()
}

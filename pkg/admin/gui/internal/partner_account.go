//nolint:dupl // another page
package internal

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func GetPartnerAccount(db database.ReadAccess, partnerName, login string) (*model.RemoteAccount, error) {
	partner, pErr := GetPartner(db, partnerName)
	if pErr != nil {
		return nil, pErr
	}

	var account model.RemoteAccount

	return &account, db.Get(&account, "remote_agent_id=? AND login=?", partner.ID, login).Run()
}

func GetPartnerAccountByID(db database.ReadAccess, partnerName string, id int64) (*model.RemoteAccount, error) {
	partner, pErr := GetPartner(db, partnerName)
	if pErr != nil {
		return nil, pErr
	}

	var account model.RemoteAccount

	return &account, db.Get(&account, "remote_agent_id=? AND id=?", partner.ID, id).Run()
}

func GetPartnerAccountsLike(db *database.DB, partnerName, prefix string) ([]*model.RemoteAccount, error) {
	partner, pErr := GetPartner(db, partnerName)
	if pErr != nil {
		return nil, pErr
	}
	const limit = 5
	var accounts model.RemoteAccounts

	return accounts, db.Select(&accounts).Where("remote_agent_id=? AND login LIKE ?", partner.ID, prefix+"%").
		OrderBy("login", true).Limit(limit, 0).Run()
}

func ListPartnerAccounts(db database.ReadAccess, partnerName string,
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

func InsertPartnerAccount(db database.Access, partnerAccount *model.RemoteAccount) error {
	return db.Insert(partnerAccount).Run()
}

func UpdatePartnerAccount(db database.Access, partnerAccount *model.RemoteAccount) error {
	return db.Update(partnerAccount).Run()
}

func DeletePartnerAccount(db database.Access, partnerAccount *model.RemoteAccount) error {
	return db.Delete(partnerAccount).Run()
}

func GetPartnerAccountCredential(db database.ReadAccess, partnerName, login, name string,
) (*model.Credential, error) {
	partnerAccount, pErr := GetPartnerAccount(db, partnerName, login)
	if pErr != nil {
		return nil, pErr
	}

	return getCredential(db, partnerAccount, name)
}

func ListPartnerAccountCredentials(db database.ReadAccess, partnerName, login string,
	orderByCol string, orderByAsc bool, limit, offset int, types ...string,
) ([]*model.Credential, error) {
	partnerAccount, pErr := GetPartnerAccount(db, partnerName, login)
	if pErr != nil {
		return nil, pErr
	}

	return listCredentials(db, partnerAccount, orderByCol, orderByAsc, limit, offset, types...)
}

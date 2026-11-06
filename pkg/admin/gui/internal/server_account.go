//nolint:dupl // another page
package internal

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func GetServerAccountByID(db database.ReadAccess, id int64) (*model.LocalAccount, error) {
	var account model.LocalAccount

	return &account, db.Get(&account, "id=?", id).Run()
}

func ListServerAccounts(db database.ReadAccess, serverName,
	orderByCol string, orderByAsc bool, limit, offset int,
) ([]*model.LocalAccount, error) {
	server, pErr := GetServer(db, serverName)
	if pErr != nil {
		return nil, pErr
	}

	var accounts model.LocalAccounts

	return accounts, db.Select(&accounts).Where("local_agent_id=?", server.ID).
		Limit(limit, offset).OrderBy(orderByCol, orderByAsc).Run()
}

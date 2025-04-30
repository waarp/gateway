package internal

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func GetServerAccount(db database.ReadAccess, serverName, login string) (*model.LocalAccount, error) {
	server, pErr := GetServer(db, serverName)
	if pErr != nil {
		return nil, pErr
	}

	var account model.LocalAccount

	return &account, db.Get(&account, "local_agent_id=? AND login=?", server.ID, login).Run()
}

func ListServerAccounts(db database.ReadAccess, serverName string,
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

func InsertServerAccount(db database.Access, serverAccount *model.LocalAccount) error {
	return db.Insert(serverAccount).Run()
}

func UpdateServerAccount(db database.Access, serverAccount *model.LocalAccount) error {
	return db.Update(serverAccount).Run()
}

func DeleteServerAccount(db database.Access, serverAccount *model.LocalAccount) error {
	return db.Delete(serverAccount).Run()
}

func GetServerAccountCredential(db database.ReadAccess, serverAccountName, login, name string,
) (*model.Credential, error) {
	serverAccount, pErr := GetServerAccount(db, serverAccountName, login)
	if pErr != nil {
		return nil, pErr
	}

	return getCredential(db, serverAccount, name)
}

func ListServerAccountCredentials(db database.ReadAccess, serverAccountName, login string,
	orderByCol string, orderByAsc bool, limit, offset int, types ...string,
) ([]*model.Credential, error) {
	serverAccount, pErr := GetServerAccount(db, serverAccountName, login)
	if pErr != nil {
		return nil, pErr
	}

	return listCredentials(db, serverAccount, orderByCol, orderByAsc, limit, offset, types...)
}

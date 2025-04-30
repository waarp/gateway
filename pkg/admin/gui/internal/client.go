package internal

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func GetClient(db database.ReadAccess, name string) (*model.Client, error) {
	var client model.Client

	return &client, db.Get(&client, "name=?", name).Owner().Run()
}

func ListClients(db database.ReadAccess, orderByCol string, orderByAsc bool, limit, offset int,
	protocols ...string,
) ([]*model.Client, error) {
	var clients model.Clients
	query := db.Select(&clients).Limit(limit, offset).OrderBy(orderByCol, orderByAsc)

	for _, protocol := range protocols {
		query = query.Where("protocol=?", protocol)
	}

	return clients, query.Run()
}

func InsertClient(db database.Access, client *model.Client) error {
	return db.Insert(client).Run()
}

func UpdateClient(db database.Access, client *model.Client) error {
	return db.Update(client).Run()
}

func DeleteClient(db database.Access, client *model.Client) error {
	return db.Delete(client).Run()
}

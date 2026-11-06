//nolint:dupl // the same helpers for the partners
package internal

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func GetServer(db database.ReadAccess, name string) (*model.LocalAgent, error) {
	var server model.LocalAgent

	return &server, db.Get(&server, "name=?", name).Run()
}

func GetServerByID(db database.ReadAccess, id int64) (*model.LocalAgent, error) {
	var server model.LocalAgent

	return &server, db.Get(&server, "id=?", id).Run()
}

func ListServers(db database.ReadAccess, orderByCol string, orderByAsc bool, limit, offset int,
	protocols ...string,
) ([]*model.LocalAgent, error) {
	var servers model.LocalAgents
	query := db.Select(&servers).Limit(limit, offset).OrderBy(orderByCol, orderByAsc)

	protoSlice := make([]any, len(protocols))
	for i, protocol := range protocols {
		protoSlice[i] = protocol
	}

	query.In("protocol", protoSlice...)

	return servers, query.Run()
}

func InsertServer(db database.Access, server *model.LocalAgent) error {
	return db.Insert(server).Run()
}

func UpdateServer(db database.Access, server *model.LocalAgent) error {
	return db.Update(server).Run()
}

func DeleteServer(db database.Access, server *model.LocalAgent) error {
	return db.Delete(server).Run()
}

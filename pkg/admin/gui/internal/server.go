package internal

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func GetServer(db database.ReadAccess, name string) (*model.LocalAgent, error) {
	var server model.LocalAgent

	return &server, db.Get(&server, "name=?", name).Owner().Run()
}

func GetServerByID(db database.ReadAccess, id int64) (*model.LocalAgent, error) {
	var server model.LocalAgent

	return &server, db.Get(&server, "id=?", id).Run()
}

func GetServersLike(db *database.DB, prefix string) ([]*model.LocalAgent, error) {
	var servers model.LocalAgents

	return servers, db.Select(&servers).Owner().Where("name LIKE ?", prefix+"%").
		OrderBy("name", true).Limit(LimitLike, 0).Run()
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

func GetServerCredential(db database.ReadAccess, serverName, name string) (*model.Credential, error) {
	server, pErr := GetServer(db, serverName)
	if pErr != nil {
		return nil, pErr
	}

	return getCredential(db, server, name)
}

func GetServerCredentialByID(db database.ReadAccess, serverName string, id int64) (*model.Credential, error) {
	server, pErr := GetServer(db, serverName)
	if pErr != nil {
		return nil, pErr
	}

	return GetCredentialByID(db, server, id)
}

func ListServerCredentials(db database.ReadAccess, partnerName,
	orderByCol string, orderByAsc bool, limit, offset int, types ...string,
) ([]*model.Credential, error) {
	server, pErr := GetServer(db, partnerName)
	if pErr != nil {
		return nil, pErr
	}

	return listCredentials(db, server, orderByCol, orderByAsc, limit, offset, types...)
}

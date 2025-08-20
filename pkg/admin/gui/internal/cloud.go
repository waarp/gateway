package internal

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func GetCloud(db database.ReadAccess, name string) (*model.CloudInstance, error) {
	var cloud model.CloudInstance

	return &cloud, db.Get(&cloud, "name=?", name).Run()
}

func GetCloudByID(db database.ReadAccess, id int64) (*model.CloudInstance, error) {
	var cloud model.CloudInstance

	return &cloud, db.Get(&cloud, "id=?", id).Run()
}

func GetCloudInstanceLike(db *database.DB, prefix string) ([]*model.CloudInstance, error) {
	var clouds model.CloudInstances

	return clouds, db.Select(&clouds).Owner().Where("name LIKE ?", prefix+"%").
		OrderBy("name", true).Limit(LimitLike, 0).Run()
}

func ListClouds(db database.ReadAccess, orderByCol string, orderByAsc bool, limit, offset int,
) ([]*model.CloudInstance, error) {
	var clouds model.CloudInstances

	return clouds, db.Select(&clouds).Limit(limit, offset).OrderBy(orderByCol, orderByAsc).Run()
}

func InsertCloud(db database.Access, cloud *model.CloudInstance) error {
	return db.Insert(cloud).Run()
}

func UpdateCloud(db database.Access, cloud *model.CloudInstance) error {
	return db.Update(cloud).Run()
}

func DeleteCloud(db database.Access, cloud *model.CloudInstance) error {
	return db.Delete(cloud).Run()
}

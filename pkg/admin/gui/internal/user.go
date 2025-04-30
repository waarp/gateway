// Package internal provides utility functions for the web GUI package.
package internal

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func GetUser(db database.ReadAccess, username string) (*model.User, error) {
	var user model.User

	return &user, db.Get(&user, "username=?", username).Owner().Run()
}

func ListUsers(db database.ReadAccess, orderByCol string, orderByAsc bool, limit, offset int,
) ([]*model.User, error) {
	var users model.Users

	return users, db.Select(&users).Owner().Limit(limit, offset).OrderBy(orderByCol, orderByAsc).Run()
}

func InsertUser(db database.Access, user *model.User) error {
	return db.Insert(user).Run()
}

func UpdateUser(db database.Access, user *model.User) error {
	return db.Update(user).Run()
}

func DeleteUser(db database.Access, user *model.User) error {
	return db.Delete(user).Run()
}

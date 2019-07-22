package database

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

const (
	defaultUser     = "admin"
	defaultPassword = "admin_password"
)

// initTables creates the database tables if they don't exist and fills them
// with the default entries.
func initTables(db *Db) error {

	trans := db.engine.NewSession()
	defer trans.Close()

	for _, table := range model.Tables {
		if ok, err := trans.IsTableExist(table); err != nil {
			return err
		} else if !ok {
			if err := trans.CreateTable(table); err != nil {
				return err
			}
		}
	}

	admin := &model.User{
		Login:    defaultUser,
		Password: []byte(defaultPassword),
	}

	if _, err := trans.Insert(admin); err != nil {
		return err
	}

	return trans.Commit()
}

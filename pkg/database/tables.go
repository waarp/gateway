package database

import (
	"golang.org/x/crypto/bcrypt"
)

// The Table enum lists all database tables
type Table byte

// Tables lists the schema of all database tables
var Tables = make([]interface{}, 0)

const (
	defaultUser     = "admin"
	defaultPassword = "admin_password"
)

// initTables creates the database tables if they don't exist and fills them
// with the default entries.
func initTables(db *Db) error {

	trans := db.engine.NewSession()
	defer trans.Close()

	for _, table := range Tables {
		if ok, err := trans.IsTableExist(table); err != nil {
			return err
		} else if !ok {
			if err := trans.CreateTable(table); err != nil {
				return err
			}
		}
	}

	password, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), 14)
	if err != nil {
		return err
	}
	admin := &User{
		Login:    defaultUser,
		Password: password,
	}

	if _, err = trans.Insert(admin); err != nil {
		return err
	}

	return trans.Commit()
}

// Package model contains all the definitions of the database models. Each
// model instance represents an entry in one of the database's tables.
package model

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

//nolint:gochecknoinits // init is used by design
func init() {
	database.AddTable(&User{})
}

// User represents a human account on the gateway. These accounts allow users
// to manage the gateway via its administration interface.
type User struct {
	// The user's database ID
	ID int64 `xorm:"BIGINT PK AUTOINCR <- 'id'"`

	// The user's owner (i.e. the name of the gateway instance to which the
	// agent belongs to).
	Owner string `xorm:"VARCHAR(100) UNIQUE(name) NOTNULL 'owner'"`

	// The user's login
	Username string `xorm:"VARCHAR(100) UNIQUE(name) NOTNULL 'username'"`

	// The user's password
	PasswordHash string `xorm:"TEXT NOTNULL 'password_hash'"`

	// The users permissions for reading and writing the database.
	Permissions PermsMask `xorm:"BINARY(4) NOTNULL 'permissions'"`
}

// TableName returns the users table name.
func (u *User) TableName() string {
	return TableUsers
}

// Appellation returns the name of 1 element of the users table.
func (*User) Appellation() string {
	return "user"
}

// GetID returns the user's ID.
func (u *User) GetID() int64 {
	return u.ID
}

// Init inserts the default user in the database when the table is created.
func (u *User) Init(db database.Access) database.Error {
	if n, err := u.countAdmins(db); err != nil {
		return err
	} else if n != 0 {
		return nil // there is already an admin
	}

	hash, err := utils.HashPassword(database.BcryptRounds, "admin_password")
	if err != nil {
		db.GetLogger().Error("Failed to hash the user password: %s", err)

		return database.NewInternalError(err)
	}

	user := &User{
		Username:     "admin",
		Owner:        conf.GlobalConfig.GatewayName,
		PasswordHash: hash,
		Permissions:  PermAll,
	}

	if err := db.Insert(user).Run(); err != nil {
		return err
	}

	return nil
}

// BeforeDelete is called before removing the user from the database. Its
// role is to check that at least one admin user remains.
func (u *User) BeforeDelete(db database.Access) database.Error {
	if u.Permissions&PermUsersWrite != 0 {
		if n, err := u.countAdmins(db); err != nil {
			return err
		} else if n <= 1 {
			return database.NewValidationError("cannot delete the last gateway admin")
		}
	}

	return nil
}

// BeforeWrite checks if the new `User` entry is valid and can be
// inserted in the database.
func (u *User) BeforeWrite(db database.ReadAccess) database.Error {
	u.Owner = conf.GlobalConfig.GatewayName
	if u.Username == "" {
		return database.NewValidationError("the username cannot be empty")
	}

	if u.PasswordHash == "" {
		return database.NewValidationError("the user password cannot be empty")
	}

	n, err := db.Count(&User{}).Where("id<>? AND owner=? AND username=?",
		u.ID, u.Owner, u.Username).Run()
	if err != nil {
		return err
	} else if n != 0 {
		return database.NewValidationError("a user named '%s' already exist", u.Username)
	}

	return nil
}

func (*User) countAdmins(db database.ReadAccess) (uint, database.Error) {
	var users Users
	if err := db.Select(&users).Where("owner=?", conf.GlobalConfig.GatewayName).
		Run(); err != nil {
		return 0, err
	}

	var n uint

	for i := range users {
		if users[i].Permissions&PermUsersWrite != 0 {
			n++
		}
	}

	return n, nil
}

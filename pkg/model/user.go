// Package model contains all the definitions of the database models. Each
// model instance represents an entry in one of the database's tables.
package model

import (
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
	ID uint64 `xorm:"pk autoincr <- 'id'"`

	// The user's owner (i.e. the name of the gateway instance to which the
	// agent belongs to.
	Owner string `xorm:"unique(name) notnull 'owner'"`

	// The user's login
	Username string `xorm:"unique(name) notnull 'username'"`

	// The user's password
	Password []byte `xorm:"notnull 'password'"`

	// The users permissions for reading and writing the database.
	Permissions PermsMask `xorm:"notnull binary(4) 'permissions'"`
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
func (u *User) GetID() uint64 {
	return u.ID
}

// Init inserts the default user in the database when the table is created.
func (u *User) Init(ses *database.Session) database.Error {
	user := &User{
		Username:    "admin",
		Owner:       database.Owner,
		Password:    []byte("admin_password"),
		Permissions: PermAll,
	}
	err := ses.Insert(user).Run()

	return err
}

// BeforeDelete is called before removing the user from the database. Its
// role is to check that at least one admin user remains.
//
// FIXME This function seems to just check that another user also exists, whether it is an admin or not.
func (u *User) BeforeDelete(db database.Access) database.Error {
	n, err := db.Count(&User{}).Where("owner=?", database.Owner).Run()
	if err != nil {
		return err
	}

	if n < 2 { //nolint:gomnd // This is not a reuseable constant
		return database.NewValidationError("cannot delete gateway last admin")
	}

	return nil
}

// BeforeWrite checks if the new `User` entry is valid and can be
// inserted in the database.
func (u *User) BeforeWrite(db database.ReadAccess) database.Error {
	u.Owner = database.Owner
	if u.Username == "" {
		return database.NewValidationError("the username cannot be empty")
	}

	if len(u.Password) == 0 {
		return database.NewValidationError("the user password cannot be empty")
	}

	n, err := db.Count(&User{}).Where("id<>? AND owner=? AND username=?",
		u.ID, u.Owner, u.Username).Run()
	if err != nil {
		return err
	} else if n != 0 {
		return database.NewValidationError("a user named '%s' already exist", u.Username)
	}

	var err1 error
	if u.Password, err1 = utils.HashPassword(database.BcryptRounds, u.Password); err1 != nil {
		db.GetLogger().Errorf("Failed to hash the user password: %s", err)

		return database.NewInternalError(err)
	}

	return nil
}

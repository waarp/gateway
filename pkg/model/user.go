// Package model contains all the definitions of the database models. Each
// model instance represents an entry in one of the database's tables.
package model

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"github.com/go-xorm/builder"
)

func init() {
	database.Tables = append(database.Tables, &User{})
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

	// The account's password
	Password []byte `xorm:"notnull 'password'"`
}

// TableName returns the users table name.
func (u *User) TableName() string {
	return "users"
}

// Init inserts the default user in the database when the table is created.
func (u *User) Init(acc database.Accessor) error {
	user := &User{
		Username: "admin",
		Owner:    database.Owner,
		Password: []byte("admin_password"),
	}
	return acc.Create(user)
}

// BeforeDelete is called before removing the user from the database. Its
// role is to check that at least one admin user remains
func (u *User) BeforeDelete(db database.Accessor) error {
	users := []User{}
	err := db.Select(&users, &database.Filters{
		// TODO update for admin user
		Conditions: builder.Eq{"owner": database.Owner},
	})
	if err != nil {
		return err
	}
	if len(users) < 2 {
		return fmt.Errorf("cannot delete gateway last admin")
	}
	return nil
}

// BeforeInsert checks if the new `User` entry is valid and can be
// inserted in the database.
func (u *User) BeforeInsert(db database.Accessor) (err error) {
	u.Owner = database.Owner
	if u.ID != 0 {
		return database.InvalidError("the user's ID cannot be entered manually")
	}
	if u.Username == "" {
		return database.InvalidError("the username cannot be empty")
	}
	if len(u.Password) == 0 {
		return database.InvalidError("the user password cannot be empty")
	}

	if res, err := db.Query("SELECT id FROM users WHERE owner=? AND username=?",
		database.Owner, u.Username); err != nil {
		return err
	} else if len(res) != 0 {
		return database.InvalidError("a user named '%s' already exist", u.Username)
	}

	u.Password, err = hashPassword(u.Password)
	return err
}

// BeforeUpdate checks if the updated `User` entry is valid and can be
// updated in the database.
func (u *User) BeforeUpdate(db database.Accessor, id uint64) (err error) {
	u.Owner = database.Owner

	if u.ID != 0 {
		return database.InvalidError("the user's ID cannot be entered manually")
	}

	if u.Username != "" {
		if res, err := db.Query("SELECT id FROM users WHERE owner=? AND username=? AND id<>?",
			database.Owner, u.Username, id); err != nil {
			return err
		} else if len(res) != 0 {
			return database.InvalidError("a user named '%s' already exist", u.Username)
		}
	}

	if u.Password != nil {
		u.Password, err = hashPassword(u.Password)
	}
	return err
}

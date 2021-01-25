// Package model contains all the definitions of the database models. Each
// model instance represents an entry in one of the database's tables.
package model

import (
	"fmt"
	"math"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"github.com/go-xorm/builder"
)

func init() {
	database.Tables = append(database.Tables, &User{})
}

// PermsMask is a bitmask specifying which actions the user is allowed to
// perform on the database.
type PermsMask uint32

// Masks for user permissions.
const (
	PermTransfersRead PermsMask = 1 << (32 - 1 - iota)
	PermTransfersWrite
	permTransferDelete // placeholder, transfers CANNOT be deleted by users
	PermServersRead
	PermServersWrite
	PermServersDelete
	PermPartnersRead
	PermPartnersWrite
	PermPartnersDelete
	PermRulesRead
	PermRulesWrite
	PermRulesDelete
	PermUsersRead
	PermUsersWrite
	PermUsersDelete

	PermAll PermsMask = math.MaxUint32 &^ permTransferDelete
)

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
	Permissions PermsMask `xorm:"notnull binary 'permissions'"`
}

// TableName returns the users table name.
func (u *User) TableName() string {
	return "users"
}

// GetID returns the user's ID.
func (u *User) GetID() uint64 {
	return u.ID
}

// Init inserts the default user in the database when the table is created.
func (u *User) Init(acc database.Accessor) error {
	user := &User{
		Username:    "admin",
		Owner:       database.Owner,
		Password:    []byte("admin_password"),
		Permissions: PermAll,
	}
	return acc.Create(user)
}

// BeforeDelete is called before removing the user from the database. Its
// role is to check that at least one admin user remains
func (u *User) BeforeDelete(db database.Accessor) error {
	var users []User
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

// Validate checks if the new `User` entry is valid and can be
// inserted in the database.
func (u *User) Validate(db database.Accessor) error {
	u.Owner = database.Owner
	if u.Username == "" {
		return database.InvalidError("the username cannot be empty")
	}
	if len(u.Password) == 0 {
		return database.InvalidError("the user password cannot be empty")
	}

	if res, err := db.Query("SELECT id FROM users WHERE id<>? AND owner=? AND username=?",
		u.ID, u.Owner, u.Username); err != nil {
		return err
	} else if len(res) != 0 {
		return database.InvalidError("a user named '%s' already exist", u.Username)
	}

	var err error
	u.Password, err = utils.HashPassword(u.Password)
	return err
}

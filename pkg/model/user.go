// Package model contains all the definitions of the database models. Each
// model instance represents an entry in one of the database's tables.
package model

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// User represents a human account on the gateway. These accounts allow users
// to manage the gateway via its administration interface.
type User struct {
	ID    int64  `xorm:"<- id AUTOINCR"` // The user's database ID.
	Owner string `xorm:"owner"`          // The agent's owner (the gateway which runs it)

	Username     string `xorm:"username"`      // The user's login
	PasswordHash string `xorm:"password_hash"` // A bcrypt hash of the user's password

	// The users permissions for reading and writing the database.
	Permissions PermsMask `xorm:"permissions"`
}

func (u *User) TableName() string { return TableUsers }
func (*User) Appellation() string { return "user" }
func (u *User) GetID() int64      { return u.ID }

// Init inserts the default user in the database when the table is created.
func (u *User) Init(db database.Access) error {
	if n, err := u.countAdmins(db); err != nil {
		return err
	} else if n != 0 {
		return nil // there is already an admin
	}

	hash, hashErr := utils.HashPassword(database.BcryptRounds, "admin_password")
	if hashErr != nil {
		db.GetLogger().Error("Failed to hash the user password: %s", hashErr)

		return database.NewInternalError(hashErr)
	}

	user := &User{
		Username:     "admin",
		Owner:        conf.GlobalConfig.GatewayName,
		PasswordHash: hash,
		Permissions:  PermAll,
	}

	if err := db.Insert(user).Run(); err != nil {
		return fmt.Errorf("failed to insert the default user: %w", err)
	}

	return nil
}

// BeforeDelete is called before removing the user from the database. Its
// role is to check that at least one admin user remains.
func (u *User) BeforeDelete(db database.Access) error {
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
func (u *User) BeforeWrite(db database.Access) error {
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
		return fmt.Errorf("failed to check usernames: %w", err)
	} else if n != 0 {
		return database.NewValidationError("a user named %q already exist", u.Username)
	}

	return nil
}

func (*User) countAdmins(db database.ReadAccess) (uint, error) {
	var users Users
	if err := db.Select(&users).Where("owner=?", conf.GlobalConfig.GatewayName).
		Run(); err != nil {
		return 0, fmt.Errorf("failed to count the number of admins: %w", err)
	}

	var n uint

	for i := range users {
		if users[i].Permissions&PermUsersWrite != 0 {
			n++
		}
	}

	return n, nil
}

// Package model contains all the definitions of the database models. Each
// model instance represents an entry in one of the database's tables.
package model

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// User represents a human account on the gateway. These accounts allow users
// to manage the gateway via its administration interface.
type User struct {
	Identifier
	Owner string `gorm:"column:owner"` // The agent's owner (the gateway which runs it)

	Username     string `gorm:"column:username"`      // The user's login
	PasswordHash string `gorm:"column:password_hash"` // A bcrypt hash of the user's password

	// The users permissions for reading and writing the database.
	Permissions PermsMask `gorm:"column:permissions"`
}

func (u *User) TableName() string { return TableUsers }
func (*User) Appellation() string { return "user" }

// BeforeDelete is called before removing the user from the database. Its
// role is to check that at least one admin user remains.
func (u *User) BeforeDelete(db database.Access) error {
	if u.Permissions&PermUsersWrite != 0 {
		if n, err := countAdmins(db); err != nil {
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
	if u.Username == "" {
		return database.NewValidationError("the username cannot be empty")
	}

	if u.PasswordHash == "" {
		return database.NewValidationError("the user password cannot be empty")
	}

	n, err := db.Count(u).Where("id<>? AND username=?", u.ID, u.Username).Run()
	if err != nil {
		return fmt.Errorf("failed to check usernames: %w", err)
	} else if n != 0 {
		return database.NewValidationErrorf("a user named %q already exist", u.Username)
	}

	return nil
}

// Init inserts the default user in the database when the table is created.
func userInit(db database.Access) error {
	if n, err := countAdmins(db); err != nil {
		return err
	} else if n != 0 {
		return nil // there is already an admin
	}

	hash, hashErr := utils.HashPassword(database.BcryptRounds, "admin_password")
	if hashErr != nil {
		return database.NewInternalError(hashErr)
	}

	user := &User{
		Username:     "admin",
		PasswordHash: hash,
		Permissions:  PermAll,
	}

	if err := db.Insert(user).Run(); err != nil {
		return fmt.Errorf("failed to insert the default user: %w", err)
	}

	return nil
}

func countAdmins(db database.ReadAccess) (uint64, error) {
	n, err := db.Count(&User{}).Where("permissions&? <> 0", PermUsersWrite).Run()
	if err != nil {
		return 0, fmt.Errorf("failed to count the number of admins: %w", err)
	}

	return n, nil
}

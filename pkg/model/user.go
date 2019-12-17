package model

import "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"

func init() {
	database.Tables = append(database.Tables, &User{})
}

// User represents a human account on the gateway. These accounts allow users
// to manage the gateway via its administration interface.
type User struct {
	// The user's database ID
	ID uint64 `xorm:"pk autoincr <- 'id'"`

	// The user's login
	Username string `xorm:"unique notnull 'username'"`

	// The account's password
	Password []byte `xorm:"notnull 'password'"`
}

// TableName returns the users table name.
func (u *User) TableName() string {
	return "users"
}

// Init inserts the default user in the database when the table is created.
func (u *User) Init(acc database.Accessor) error {
	return acc.Create(&User{Username: "admin", Password: []byte("admin_password")})
}

// BeforeInsert is called before inserting the user in the database. Its
// role is to hash the password.
func (u *User) BeforeInsert(database.Accessor) error {
	if u.Password != nil {
		var err error
		if u.Password, err = hashPassword(u.Password); err != nil {
			return err
		}
	}
	return nil
}

// BeforeUpdate is called before updating the user from the database. Its
// role is to hash the password.
func (u *User) BeforeUpdate(database.Accessor) error {
	return u.BeforeInsert(nil)
}

// ValidateInsert checks if the new `User` entry is valid and can be
// inserted in the database.
func (u *User) ValidateInsert(acc database.Accessor) error {
	if u.ID != 0 {
		return database.InvalidError("The user's ID cannot be entered manually")
	}
	if u.Username == "" {
		return database.InvalidError("The username cannot be empty")
	}
	if len(u.Password) == 0 {
		return database.InvalidError("The user password cannot be empty")
	}

	if ok, err := acc.Exists(&User{Username: u.Username}); err != nil {
		return err
	} else if ok {
		return database.InvalidError("A user named '%s' already exist", u.Username)
	}
	return nil
}

// ValidateUpdate checks if the updated `User` entry is valid and can be
// updated in the database.
func (u *User) ValidateUpdate(acc database.Accessor, id uint64) error {
	if u.ID != 0 {
		return database.InvalidError("The user's ID cannot be entered manually")
	}

	if u.Username != "" {
		if res, err := acc.Query("SELECT id FROM users WHERE username=? AND id<>?",
			u.Username, id); err != nil {
			return err
		} else if len(res) != 0 {
			return database.InvalidError("A user named '%s' already exist", u.Username)
		}
	}
	return nil
}

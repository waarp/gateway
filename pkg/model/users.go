package model

import (
	"encoding/json"
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultUser     = "admin"
	defaultPassword = "admin_password"
)

func init() {
	database.Tables = append(database.Tables, &User{})
}

func hashPassword(clear []byte) []byte {
	if _, isHashed := bcrypt.Cost(clear); isHashed != nil {
		hashed, err := bcrypt.GenerateFromPassword(clear, database.BcryptRounds)
		if err != nil {
			return nil
		}
		return hashed
	}
	return clear
}

// User represents one record of the 'users' table. The field tags define the
// table columns and the contraints on those columns.
type User struct {
	// The user's unique id
	ID uint64 `xorm:"pk autoincr 'id'"`
	// The user's login name
	Login string `xorm:"unique notnull 'login'"`
	// The user's display name
	Name string `xorm:"'name'"`
	// The user's password hash
	Password []byte `xorm:"notnull 'password'"`
}

// MarshalJSON removes the password from the user before marshalling it to JSON.
func (u *User) MarshalJSON() ([]byte, error) {
	u.Password = nil
	return json.Marshal(*u)
}

// Validate checks if the user entry can be inserted in the database
func (u *User) Validate(db *database.Db, isInsert bool) error {
	if u.Login == "" {
		return ErrInvalid{msg: "The user's login cannot be empty"}
	}
	if u.Password == nil || len(u.Password) == 0 {
		return ErrInvalid{msg: "The user's password cannot be empty"}
	}

	if isInsert {
		res, err := db.Query("SELECT id FROM users WHERE id=? OR login=?",
			u.ID, u.Login)
		if err != nil {
			return err
		}
		if len(res) > 0 {
			return ErrInvalid{msg: "A user with the same ID or login already exist"}
		}
	} else {
		res, err := db.Query("SELECT id FROM users WHERE id=?", u.ID)
		if err != nil {
			return err
		}
		if len(res) == 0 {
			return ErrInvalid{fmt.Sprintf("Unknown user id: '%v'", u.ID)}
		}

		loginCheck := &User{Login: u.Login}
		if err := db.Get(loginCheck); err != nil {
			if err != database.ErrNotFound {
				return err
			}
		} else if loginCheck.ID != u.ID {
			return ErrInvalid{msg: "A user with the same login already exist"}
		}
	}

	return nil
}

// BeforeUpdate hashes the user password before updating the record.
func (u *User) BeforeUpdate() {
	u.Password = hashPassword(u.Password)
}

// BeforeInsert hashes the user password before updating the record.
func (u *User) BeforeInsert() {
	u.Password = hashPassword(u.Password)
}

// TableName returns the name of the users SQL table
func (*User) TableName() string {
	return "users"
}

// Init initializes the 'users' table by inserting the default user into it.
func (*User) Init(acc database.Accessor) error {
	admin := &User{
		Login:    defaultUser,
		Password: []byte(defaultPassword),
	}

	if err := acc.Create(admin); err != nil {
		return err
	}
	return nil
}

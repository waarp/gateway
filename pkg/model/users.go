package model

import "golang.org/x/crypto/bcrypt"

func init() {
	Tables = append(Tables, &User{})
}

// BcryptRounds defines the number of rounds taken by bcrypt to hash passwords
// in the database
var BcryptRounds = 12

func hashPassword(clear []byte) []byte {
	if _, isHashed := bcrypt.Cost(clear); isHashed != nil {
		hashed, err := bcrypt.GenerateFromPassword(clear, BcryptRounds)
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

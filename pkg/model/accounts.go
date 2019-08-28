package model

import "fmt"

func init() {
	Tables = append(Tables, &Account{})
}

// Account represents one record of the 'accounts' table.
type Account struct {
	// The account's unique ID
	ID uint64 `xorm:"pk autoincr 'id'" json:"id"`
	// The account's username
	Username string `xorm:"unique(acc) notnull 'username'" json:"username"`
	// The account's password hash
	Password []byte `xorm:"notnull 'password'" json:"password,omitempty"`
	// The Id of the partner this account belongs to
	PartnerID uint64 `xorm:"unique(acc) notnull 'partner_id'" json:"partnerID"`
}

// Validate checks that the account entry can be inserted into the database
func (a *Account) Validate(exists func(interface{}) (bool, error)) error {
	if a.Username == "" {
		return ErrInvalid{msg: "The account's username cannot be empty."}
	}
	if len(a.Password) == 0 {
		return ErrInvalid{msg: "The account's password cannot be empty."}
	}
	if a.PartnerID == 0 {
		return ErrInvalid{msg: "The account's partnerID cannot be empty."}
	}

	parent := &Partner{ID: a.PartnerID}
	if ok, err := exists(parent); err != nil {
		return err
	} else if !ok {
		return ErrInvalid{msg: fmt.Sprintf("The partner n°%v does not exist",
			parent.ID)}
	}

	test := &Account{
		PartnerID: a.PartnerID,
		Username:  a.Username,
	}
	if ok, err := exists(test); err != nil {
		return err
	} else if ok {
		return ErrInvalid{msg: fmt.Sprintf("An account named '%s' already exists",
			a.Username)}
	}

	return nil
}

// BeforeUpdate hashes the account password before updating the record.
func (a *Account) BeforeUpdate() {
	a.Password = hashPassword(a.Password)
}

// BeforeInsert hashes the account password before updating the record.
func (a *Account) BeforeInsert() {
	a.Password = hashPassword(a.Password)
}

// TableName returns the name of the accounts SQL table
func (*Account) TableName() string {
	return "accounts"
}

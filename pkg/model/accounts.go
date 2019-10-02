package model

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

func init() {
	database.Tables = append(database.Tables, &Account{})
}

func encryptPassword(password []byte) []byte {
	// If password is already encrypted, don't encrypt it again.
	if strings.HasPrefix(string(password), "$AES$") {
		return password
	}

	nonce := make([]byte, database.GCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil
	}

	cipherText := database.GCM.Seal(nonce, nonce, password, nil)
	return append([]byte("$AES$"), cipherText...)
}

// Account represents one record of the 'accounts' table.
type Account struct {
	// The account's unique ID
	ID uint64 `xorm:"pk autoincr 'id'" json:"id"`
	// The account's username
	Username string `xorm:"unique(acc) notnull 'username'" json:"username"`
	// The account's password hash if the account is internal. If the account is
	// not internal, then this is the password cypher.
	Password []byte `xorm:"notnull 'password'" json:"password,omitempty"`
	// The Id of the partner this account belongs to
	PartnerID uint64 `xorm:"unique(acc) notnull 'partner_id'" json:"partnerID"`
	// If true then the account is used by the partner to connect to the gateway,
	// if false it is used by the gateway to connect to the partner
	IsInternal bool `xorm:"notnull 'is_internal'" json:"isInternal"`
}

// MarshalJSON removes the password before marshalling the account to JSON.
func (a *Account) MarshalJSON() ([]byte, error) {
	a.Password = nil
	return json.Marshal(*a)
}

// Validate checks that the account entry can be inserted into the database
func (a *Account) Validate(db *database.Db, isInsert bool) error {
	if a.Username == "" {
		return ErrInvalid{msg: "The account's username cannot be empty"}
	}
	if len(a.Password) == 0 {
		return ErrInvalid{msg: "The account's password cannot be empty"}
	}
	if a.PartnerID == 0 {
		return ErrInvalid{msg: "The account's partnerID cannot be empty"}
	}

	parts, err := db.Query("SELECT id FROM partners WHERE id=?", a.PartnerID)
	if err != nil {
		return err
	}
	if len(parts) == 0 {
		return ErrInvalid{msg: fmt.Sprintf("No partner found with ID '%v'", a.PartnerID)}
	}

	names, err := db.Query("SELECT id FROM accounts WHERE partner_id=? AND username=?",
		a.PartnerID, a.Username)
	if err != nil {
		return err
	}

	if isInsert {
		if len(names) > 0 {
			return ErrInvalid{msg: "An account with the same username already exist for this partner"}
		}

		ids, err := db.Query("SELECT id FROM accounts WHERE id=?", a.ID)
		if err != nil {
			return err
		}
		if len(ids) > 0 {
			return ErrInvalid{msg: "An account with the same ID already exist"}
		}
	} else {
		if len(names) > 0 && names[0]["id"] != a.ID {
			return ErrInvalid{msg: "An account with the same username already exist for this partner"}
		}

		res, err := db.Query("SELECT id FROM accounts WHERE id=?", a.ID)
		if err != nil {
			return err
		}
		if len(res) == 0 {
			return ErrInvalid{fmt.Sprintf("Unknown account ID: '%v'", a.ID)}
		}
	}

	return nil
}

// BeforeUpdate encrypts or hashes the account password before updating the record.
func (a *Account) BeforeUpdate() {
	if a.IsInternal {
		a.Password = hashPassword(a.Password)
	} else {
		a.Password = encryptPassword(a.Password)
	}
}

// BeforeInsert encrypts or hashes the account password before updating the record.
func (a *Account) BeforeInsert() {
	a.BeforeUpdate()
}

// TableName returns the name of the accounts SQL table
func (*Account) TableName() string {
	return "accounts"
}

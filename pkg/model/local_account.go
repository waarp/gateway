package model

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	database.AddTable(&LocalAccount{})
}

// LocalAccount represents an account on a local agent. It is used by remote
// partners to authenticate on the gateway for transfers.
type LocalAccount struct {
	// The account's database ID
	ID uint64 `xorm:"pk autoincr <- 'id'"`

	// The ID of the `LocalAgent` this account is attached to
	LocalAgentID uint64 `xorm:"unique(loc_ac) notnull 'local_agent_id'"`

	// The account's login
	Login string `xorm:"unique(loc_ac) notnull 'login'"`

	// A bcrypt hash of the account's password
	PasswordHash []byte `xorm:"text 'password_hash'"`
}

// TableName returns the local accounts table name.
func (*LocalAccount) TableName() string {
	return TableLocAccounts
}

// Appellation returns the name of 1 element of the local accounts table.
func (*LocalAccount) Appellation() string {
	return "local account"
}

// GetID returns the account's ID.
func (l *LocalAccount) GetID() uint64 {
	return l.ID
}

// GetCryptos fetch in the database then return the associated Cryptos if they exist
func (l *LocalAccount) GetCryptos(db *database.DB) ([]Crypto, database.Error) {
	return GetCryptos(db, l)
}

// BeforeWrite checks if the new `LocalAccount` entry is valid and can be
// inserted in the database.
//nolint:dupl
func (l *LocalAccount) BeforeWrite(db database.ReadAccess) database.Error {
	if l.LocalAgentID == 0 {
		return database.NewValidationError("the account's agentID cannot be empty")
	}
	if l.Login == "" {
		return database.NewValidationError("the account's login cannot be empty")
	}

	if len(l.PasswordHash) > 0 {
		if _, isHashed := bcrypt.Cost(l.PasswordHash); isHashed != nil {
			return database.NewValidationError("the password is not hashed")
		}
	}

	parent := &LocalAgent{}
	if err := db.Get(parent, "id=?", l.LocalAgentID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationError("no local agent found with the ID '%v'", l.LocalAgentID)
		}
		return err
	}

	n, err := db.Count(l).Where("id<>? AND local_agent_id=? AND login=?",
		l.ID, l.LocalAgentID, l.Login).Run()
	if err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError("a local account with the same login '%s' "+
			"already exist", l.Login)
	}

	return nil
}

// BeforeDelete is called before deleting the account from the database. Its
// role is to delete all the certificates tied to the account.
func (l *LocalAccount) BeforeDelete(db database.Access) database.Error {
	n, err := db.Count(&Transfer{}).Where("is_server=? AND account_id=?",
		true, l.ID).Run()
	if err != nil {
		return err
	}
	if n > 0 {
		return database.NewValidationError("this account is currently being used in one " +
			"or more running transfers and thus cannot be deleted, cancel " +
			"the transfers or wait for them to finish")
	}

	certQuery := db.DeleteAll(&Crypto{}).Where("owner_type=? AND owner_id=?",
		TableLocAccounts, l.ID)
	if err := certQuery.Run(); err != nil {
		return err
	}

	accessQuery := db.DeleteAll(&RuleAccess{}).Where(
		"object_type=? AND object_id=?", TableLocAccounts, l.ID)
	if err := accessQuery.Run(); err != nil {
		return err
	}

	return nil
}

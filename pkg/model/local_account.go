package model

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

//nolint:gochecknoinits // init is used by design
func init() {
	database.AddTable(&LocalAccount{})
}

// LocalAccount represents an account on a local agent. It is used by remote
// partners to authenticate on the gateway for transfers.
type LocalAccount struct {
	// The account's database ID.
	ID int64 `xorm:"BIGINT PK AUTOINCR <- 'id'"`

	// The ID of the `LocalAgent` this account is attached to.
	LocalAgentID int64 `xorm:"BIGINT UNIQUE(loc_ac) NOTNULL 'local_agent_id'"` // foreign key (local_agents.id)

	// The account's login.
	Login string `xorm:"VARCHAR(100) UNIQUE(loc_ac) NOTNULL 'login'"`

	// A bcrypt hash of the account's password.
	PasswordHash string `xorm:"TEXT NOTNULL DEFAULT('') 'password_hash'"`
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
func (l *LocalAccount) GetID() int64 {
	return l.ID
}

// GetCryptos fetch in the database then return the associated Cryptos if they exist.
func (l *LocalAccount) GetCryptos(db *database.DB) ([]*Crypto, error) {
	return getCryptos(db, l)
}

// BeforeWrite checks if the new `LocalAccount` entry is valid and can be
// inserted in the database.
//
//nolint:dupl // too many differences
func (l *LocalAccount) BeforeWrite(db database.ReadAccess) database.Error {
	if l.LocalAgentID == 0 {
		return database.NewValidationError("the account's agentID cannot be empty")
	}

	if l.Login == "" {
		return database.NewValidationError("the account's login cannot be empty")
	}

	if l.PasswordHash != "" {
		if _, isHashed := bcrypt.Cost([]byte(l.PasswordHash)); isHashed != nil {
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
// role is to check whether the account is still used in any ongoing transfer.
func (l *LocalAccount) BeforeDelete(db database.Access) database.Error {
	if n, err := db.Count(&Transfer{}).Where("local_account_id=?", l.ID).Run(); err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError("this account is currently being used " +
			"in one or more running transfers and thus cannot be deleted, cancel " +
			"these transfers or wait for them to finish")
	}

	return nil
}

func (*LocalAccount) MakeExtraConstraints(db *database.Executor) database.Error {
	// add a foreign key to 'local_agent_id'
	return redefineColumn(db, TableLocAccounts, "local_agent_id", fmt.Sprintf(
		`BIGINT NOT NULL REFERENCES %s ON UPDATE RESTRICT ON DELETE CASCADE`, TableLocAgents))
}

//nolint:goconst //different columns having the same name does not warrant making that name a constant
func (l *LocalAccount) GenCryptoSelectCond() (string, int64) { return "local_account_id=?", l.ID }
func (l *LocalAccount) SetCryptoOwner(c *Crypto)             { c.LocalAccountID = utils.NewNullInt64(l.ID) }
func (l *LocalAccount) GenAccessSelectCond() (string, int64) { return "local_account_id=?", l.ID }

func (l *LocalAccount) SetAccessTarget(a *RuleAccess) { a.LocalAccountID = utils.NewNullInt64(l.ID) }

func (l *LocalAccount) GetAuthorizedRules(db database.ReadAccess) ([]*Rule, error) {
	return getAuthorizedRules(db, "local_account_id", l.ID)
}

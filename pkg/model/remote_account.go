package model

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

// RemoteAccount represents an account on a remote agent. It is used by the
// gateway to authenticate on distant servers for transfers.
type RemoteAccount struct {
	// The account's database ID
	ID int64 `xorm:"BIGINT PK AUTOINCR <- 'id'"`

	// The ID of the `RemoteAgent` this account is attached to
	RemoteAgentID int64 `xorm:"BIGINT UNIQUE(rem_ac) NOTNULL 'remote_agent_id'"` // foreign key (remote_agents.id)

	// The account's login
	Login string `xorm:"VARCHAR(100) UNIQUE(rem_ac) NOTNULL 'login'"`

	// The account's password
	Password types.CypherText `xorm:"TEXT NOTNULL DEFAULT('') 'password'"`
}

// TableName returns the remote accounts table name.
func (*RemoteAccount) TableName() string {
	return TableRemAccounts
}

// Appellation returns the name of 1 element of the remote accounts table.
func (*RemoteAccount) Appellation() string {
	return "remote account"
}

// GetID returns the account's ID.
func (r *RemoteAccount) GetID() int64 {
	return r.ID
}

// BeforeWrite checks if the new `RemoteAccount` entry is valid and can be
// inserted in the database.
//
//nolint:dupl // too many differences to be factorized easily
func (r *RemoteAccount) BeforeWrite(db database.ReadAccess) database.Error {
	if r.RemoteAgentID == 0 {
		return database.NewValidationError("the account's agentID cannot be empty")
	}

	if r.Login == "" {
		return database.NewValidationError("the account's login cannot be empty")
	}

	n, err := db.Count(&RemoteAgent{}).Where("id=?", r.RemoteAgentID).Run()
	if err != nil {
		return err
	} else if n == 0 {
		return database.NewValidationError("no remote agent found with the ID '%v'",
			r.RemoteAgentID)
	}

	n, err = db.Count(&RemoteAccount{}).Where("id<>? AND remote_agent_id=? AND login=?",
		r.ID, r.RemoteAgentID, r.Login).Run()
	if err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError(
			"a remote account with the same login '%s' already exist", r.Login)
	}

	return nil
}

// BeforeDelete is called before deleting the account from the database. Its
// role is to check whether the account is still used in any ongoing transfer.
func (r *RemoteAccount) BeforeDelete(db database.Access) database.Error {
	if n, err := db.Count(&Transfer{}).Where("remote_account_id=?", r.ID).Run(); err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError("this account is currently being used " +
			"in one or more running transfers and thus cannot be deleted, cancel " +
			"these transfers or wait for them to finish")
	}

	return nil
}

// GetCryptos fetch in the database then return the associated Cryptos if they exist.
func (r *RemoteAccount) GetCryptos(db database.ReadAccess) ([]*Crypto, error) {
	return getCryptos(db, r)
}

func (*RemoteAccount) MakeExtraConstraints(db *database.Executor) database.Error {
	// add a foreign key to 'remote_agent_id'
	return redefineColumn(db, TableRemAccounts, "remote_agent_id", fmt.Sprintf(
		`BIGINT NOT NULL REFERENCES %s(id) ON UPDATE RESTRICT ON DELETE CASCADE `, TableRemAgents))
}

//nolint:goconst //different columns having the same name does not warrant making that name a constant
func (r *RemoteAccount) GenCryptoSelectCond() (string, int64) { return "remote_account_id=?", r.ID }
func (r *RemoteAccount) SetCryptoOwner(c *Crypto)             { c.RemoteAccountID = utils.NewNullInt64(r.ID) }
func (r *RemoteAccount) GenAccessSelectCond() (string, int64) { return "remote_account_id=?", r.ID }
func (r *RemoteAccount) SetAccessTarget(a *RuleAccess)        { a.RemoteAccountID = utils.NewNullInt64(r.ID) }

func (r *RemoteAccount) GetAuthorizedRules(db database.ReadAccess) ([]*Rule, error) {
	return getAuthorizedRules(db, "remote_account_id", r.ID)
}

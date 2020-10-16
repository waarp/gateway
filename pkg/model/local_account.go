package model

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-r66/r66"
	"github.com/go-xorm/builder"
)

func init() {
	database.Tables = append(database.Tables, &LocalAccount{})
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

	// The account's password
	Password []byte `xorm:"'password'"`
}

// GetID returns the account's ID.
func (l *LocalAccount) GetID() uint64 {
	return l.ID
}

// TableName returns the local accounts table name.
func (l *LocalAccount) TableName() string {
	return "local_accounts"
}

// GetCerts fetch in the database then return the associated Certificates if they exist
func (l *LocalAccount) GetCerts(db database.Accessor) ([]Cert, error) {
	filters := &database.Filters{
		Conditions: builder.And(builder.Eq{"owner_type": l.TableName()},
			builder.Eq{"owner_id": l.ID}),
	}

	results := []Cert{}
	if err := db.Select(&results, filters); err != nil {
		return nil, err
	}
	return results, nil
}

// Validate checks if the new `LocalAccount` entry is valid and can be
// inserted in the database.
//nolint:dupl
func (l *LocalAccount) Validate(db database.Accessor) (err error) {
	if l.LocalAgentID == 0 {
		return database.InvalidError("the account's agentID cannot be empty")
	}
	if l.Login == "" {
		return database.InvalidError("the account's login cannot be empty")
	}
	if len(l.Password) == 0 {
		return database.InvalidError("the account's password cannot be empty")
	}

	agent := &LocalAgent{ID: l.LocalAgentID}
	if err := db.Get(agent); err != nil {
		if err != database.ErrNotFound {
			return err
		}
		return database.InvalidError("no local agent found with the ID '%v'", l.LocalAgentID)
	}

	if res, err := db.Query("SELECT id FROM local_accounts WHERE id<>? AND "+
		"local_agent_id=? AND login=?", l.ID, l.LocalAgentID, l.Login); err != nil {
		return err
	} else if len(res) > 0 {
		return database.InvalidError("a local account with the same login '%s' "+
			"already exist", l.Login)
	}

	if agent.Protocol == "r66" {
		l.Password = r66.CryptPass(l.Password)
	}
	l.Password, err = hashPassword(l.Password)
	return err
}

// BeforeDelete is called before deleting the account from the database. Its
// role is to delete all the certificates tied to the account.
func (l *LocalAccount) BeforeDelete(db database.Accessor) error {
	trans, err := db.Query("SELECT id FROM transfers WHERE is_server=? AND account_id=?", true, l.ID)
	if err != nil {
		return err
	}
	if len(trans) > 0 {
		return database.InvalidError("this account is currently being used in a " +
			"running transfer and cannot be deleted, cancel the transfer or wait " +
			"for it to finish")
	}

	certQuery := "DELETE FROM certificates WHERE owner_type='local_accounts' AND owner_id=?"
	if err := db.Execute(certQuery, l.ID); err != nil {
		return err
	}

	accessQuery := "DELETE FROM rule_access WHERE object_type='local_accounts' AND object_id=?"
	return db.Execute(accessQuery, l.ID)
}

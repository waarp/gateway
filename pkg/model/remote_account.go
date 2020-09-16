package model

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"github.com/go-xorm/builder"
)

func init() {
	database.Tables = append(database.Tables, &RemoteAccount{})
}

// RemoteAccount represents an account on a remote agent. It is used by the
// gateway to authenticate on distant servers for transfers.
type RemoteAccount struct {

	// The account's database ID
	ID uint64 `xorm:"pk autoincr <- 'id'"`

	// The ID of the `RemoteAgent` this account is attached to
	RemoteAgentID uint64 `xorm:"unique(rem_ac) notnull 'remote_agent_id'"`

	// The account's login
	Login string `xorm:"unique(rem_ac) notnull 'login'"`

	// The account's password
	Password []byte `xorm:"'password'"`
}

// TableName returns the remote accounts table name.
func (*RemoteAccount) TableName() string {
	return "remote_accounts"
}

// ElemName returns the name of 1 element of the remote accounts table.
func (*RemoteAccount) ElemName() string {
	return "remote account"
}

// GetID returns the account's ID.
func (r *RemoteAccount) GetID() uint64 {
	return r.ID
}

// GetCerts fetch in the database then return the associated Certificates if they exist
func (r *RemoteAccount) GetCerts(db database.Accessor) ([]Cert, error) {
	filters := &database.Filters{
		Conditions: builder.And(builder.Eq{"owner_type": r.TableName()},
			builder.Eq{"owner_id": r.ID}),
	}

	results := []Cert{}
	if err := db.Select(&results, filters); err != nil {
		return nil, err
	}
	return results, nil
}

// Validate checks if the new `RemoteAccount` entry is valid and can be
// inserted in the database.
//nolint:dupl
func (r *RemoteAccount) Validate(db database.Accessor) error {
	if r.RemoteAgentID == 0 {
		return database.NewValidationError("the account's agentID cannot be empty")
	}
	if r.Login == "" {
		return database.NewValidationError("the account's login cannot be empty")
	}
	if len(r.Password) == 0 {
		return database.NewValidationError("The account's password cannot be empty")
	}

	if res, err := db.Query("SELECT id FROM remote_agents WHERE id=?",
		r.RemoteAgentID); err != nil {
		return database.NewInternalError(err, "failed to retrieve the list of partners")
	} else if len(res) == 0 {
		return database.NewValidationError("no remote agent found with the ID '%v'",
			r.RemoteAgentID)
	}

	if res, err := db.Query("SELECT id FROM remote_accounts WHERE id<>? AND "+
		"remote_agent_id=? AND login=?", r.ID, r.RemoteAgentID, r.Login); err != nil {
		return database.NewInternalError(err, "failed to retrieve the list of existing accounts")
	} else if len(res) > 0 {
		return database.NewValidationError(
			"a remote account with the same login '%s' already exist", r.Login)
	}

	var pErr error
	if r.Password, pErr = utils.CryptPassword(r.Password); pErr != nil {
		return database.NewInternalError(pErr, "failed to encrypt the account password")
	}
	return nil
}

// BeforeDelete is called before deleting the account from the database. Its
// role is to delete all the certificates tied to the account.
func (r *RemoteAccount) BeforeDelete(db database.Accessor) error {
	trans, err := db.Query("SELECT id FROM transfers WHERE is_server=? AND account_id=?",
		false, r.ID)
	if err != nil {
		return database.NewInternalError(err, "failed to retrieve the list of transfers")
	}
	if len(trans) > 0 {
		return database.NewValidationError("this account is currently being used in a " +
			"running transfer and cannot be deleted, cancel the transfer or wait " +
			"for it to finish")
	}

	certQuery := "DELETE FROM certificates WHERE owner_type='remote_accounts' AND owner_id=?"
	if err := db.Execute(certQuery, r.ID); err != nil {
		return database.NewInternalError(err, "failed to delete the account's certificates")
	}

	accessQuery := "DELETE FROM rule_access WHERE object_type='remote_accounts' AND object_id=?"
	if err := db.Execute(accessQuery, r.ID); err != nil {
		return database.NewInternalError(err, "failed to delete the account's rule permissions")
	}

	return nil
}

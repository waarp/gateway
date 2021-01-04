package model

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
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

// Appellation returns the name of 1 element of the remote accounts table.
func (*RemoteAccount) Appellation() string {
	return "remote account"
}

// GetID returns the account's ID.
func (r *RemoteAccount) GetID() uint64 {
	return r.ID
}

// BeforeWrite checks if the new `RemoteAccount` entry is valid and can be
// inserted in the database.
//nolint:dupl
func (r *RemoteAccount) BeforeWrite(db database.ReadAccess) database.Error {
	if r.RemoteAgentID == 0 {
		return database.NewValidationError("the account's agentID cannot be empty")
	}
	if r.Login == "" {
		return database.NewValidationError("the account's login cannot be empty")
	}
	if len(r.Password) == 0 {
		return database.NewValidationError("The account's password cannot be empty")
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

	var pErr error
	if r.Password, pErr = utils.CryptPassword(r.Password); pErr != nil {
		db.GetLogger().Errorf("Failed to encrypt the remote agent password: %s", pErr)
		return database.NewInternalError(pErr)
	}
	return nil
}

// BeforeDelete is called before deleting the account from the database. Its
// role is to delete all the certificates tied to the account.
func (r *RemoteAccount) BeforeDelete(db database.Access) database.Error {
	n, err := db.Count(&Transfer{}).Where("is_server=? AND account_id=?", false, r.ID).Run()
	if err != nil {
		return err
	}
	if n > 0 {
		return database.NewValidationError("this account is currently being used in a " +
			"running transfer and cannot be deleted, cancel the transfer or wait " +
			"for it to finish")
	}

	certQuery := db.DeleteAll(&Cert{}).Where(
		"owner_type='remote_accounts' AND owner_id=?", r.ID)
	if err := certQuery.Run(); err != nil {
		return err
	}

	accessQuery := db.DeleteAll(&RuleAccess{}).Where(
		"object_type='remote_accounts' AND object_id=?", r.ID)
	if err := accessQuery.Run(); err != nil {
		return err
	}

	return nil
}

// GetCerts fetch in the database then return the associated Certificates if they exist
func (r *RemoteAccount) GetCerts(db database.ReadAccess) ([]Cert, database.Error) {
	return GetCerts(db, r)
}

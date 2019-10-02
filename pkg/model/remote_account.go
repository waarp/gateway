package model

import (
	"encoding/json"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

func init() {
	database.Tables = append(database.Tables, &RemoteAccount{})
}

// RemoteAccount represents an account on a remote agent. It is used by the
// gateway to authenticate on distant servers for transfers.
type RemoteAccount struct {

	// The account's database ID
	ID uint64 `xorm:"pk autoincr <- 'id'" json:"id"`

	// The ID of the `RemoteAgent` this account is attached to
	RemoteAgentID uint64 `xorm:"unique(rem_ac) notnull 'remote_agent_id'" json:"remoteAgentID"`

	// The account's login
	Login string `xorm:"unique(rem_ac) notnull 'login'" json:"login"`

	// The account's password
	Password []byte `xorm:"'password'" json:"password,omitempty"`
}

// TableName returns the remote accounts table name.
func (r *RemoteAccount) TableName() string {
	return "remote_accounts"
}

// MarshalJSON removes the password and then returns the account in JSON format.
func (r *RemoteAccount) MarshalJSON() ([]byte, error) {
	acc := *r
	acc.Password = nil
	return json.Marshal(acc)
}

// BeforeInsert is called before inserting the account in the database. Its
// role is to encrypt the password, if one was entered.
func (r *RemoteAccount) BeforeInsert(ses *database.Session) error {
	if r.Password != nil {
		var err error
		if r.Password, err = encryptPassword(r.Password); err != nil {
			return err
		}
	}
	return nil
}

// BeforeUpdate is called before updating the account from the database. Its
// role is to encrypt the password, if a new one was entered.
func (r *RemoteAccount) BeforeUpdate(ses *database.Session) error {
	return r.BeforeInsert(ses)
}

// ValidateInsert checks if the new `RemoteAccount` entry is valid and can be
// inserted in the database.
func (r *RemoteAccount) ValidateInsert(ses *database.Session) error {
	if r.ID != 0 {
		return database.InvalidError("The account's ID cannot be entered manually")
	}
	if r.RemoteAgentID == 0 {
		return database.InvalidError("The account's agentID cannot be empty")
	}
	if r.Login == "" {
		return database.InvalidError("The account's login cannot be empty")
	}

	if res, err := ses.Query("SELECT id FROM remote_agents WHERE id=?", r.RemoteAgentID); err != nil {
		return err
	} else if len(res) == 0 {
		return database.InvalidError("No remote agent found with the ID '%v'", r.RemoteAgentID)
	}

	if res, err := ses.Query("SELECT id FROM remote_accounts WHERE remote_agent_id=? "+
		"AND login=?", r.RemoteAgentID, r.Login); err != nil {
		return err
	} else if len(res) > 0 {
		return database.InvalidError("A remote account with the same login '%s' "+
			"already exist", r.Login)
	}

	return nil
}

// ValidateUpdate checks if the updated `RemoteAccount` entry is valid and can be
// updated in the database.
func (r *RemoteAccount) ValidateUpdate(ses *database.Session, id uint64) error {
	if r.ID != 0 {
		return database.InvalidError("The account's ID cannot be entered manually")
	}

	if r.Login != "" {
		old := RemoteAccount{ID: id}
		if err := ses.Get(&old); err != nil {
			return err
		}
		if r.RemoteAgentID != 0 {
			old.RemoteAgentID = r.RemoteAgentID
		}

		if res, err := ses.Query("SELECT id FROM remote_accounts WHERE "+
			"remote_agent_id=? AND login=?", old.RemoteAgentID, r.Login); err != nil {
			return err
		} else if len(res) > 0 {
			return database.InvalidError("A remote account with the same login '%s' "+
				"already exist", r.Login)
		}
	}

	if r.RemoteAgentID != 0 {
		if res, err := ses.Query("SELECT id FROM remote_agents WHERE id=?",
			r.RemoteAgentID); err != nil {
			return err
		} else if len(res) == 0 {
			return database.InvalidError("No remote agent found with the ID '%v'",
				r.RemoteAgentID)
		}
	}

	return nil
}

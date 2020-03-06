package model

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
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
func (r *RemoteAccount) TableName() string {
	return "remote_accounts"
}

// BeforeInsert is called before inserting the account in the database. Its
// role is to encrypt the password, if one was entered.
func (r *RemoteAccount) BeforeInsert(database.Accessor) error {
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
func (r *RemoteAccount) BeforeUpdate(database.Accessor) error {
	return r.BeforeInsert(nil)
}

// BeforeDelete is called before deleting the account from the database. Its
// role is to delete all the certificates tied to the account.
func (r *RemoteAccount) BeforeDelete(acc database.Accessor) error {
	filter := builder.Eq{"owner_type": r.TableName(), "owner_id": r.ID}
	if err := acc.Execute(builder.Delete().From((&Cert{}).TableName()).
		Where(filter)); err != nil {
		return err
	}

	return nil
}

// ValidateInsert checks if the new `RemoteAccount` entry is valid and can be
// inserted in the database.
func (r *RemoteAccount) ValidateInsert(acc database.Accessor) error {
	if r.ID != 0 {
		return database.InvalidError("The account's ID cannot be entered manually")
	}
	if r.RemoteAgentID == 0 {
		return database.InvalidError("The account's agentID cannot be empty")
	}
	if r.Login == "" {
		return database.InvalidError("The account's login cannot be empty")
	}

	if res, err := acc.Query("SELECT id FROM remote_agents WHERE id=?", r.RemoteAgentID); err != nil {
		return err
	} else if len(res) == 0 {
		return database.InvalidError("No remote agent found with the ID '%v'", r.RemoteAgentID)
	}

	if res, err := acc.Query("SELECT id FROM remote_accounts WHERE remote_agent_id=? "+
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
func (r *RemoteAccount) ValidateUpdate(acc database.Accessor, id uint64) error {
	if r.ID != 0 {
		return database.InvalidError("The account's ID cannot be entered manually")
	}

	if r.Login != "" {
		old := RemoteAccount{ID: id}
		if err := acc.Get(&old); err != nil {
			return err
		}
		if r.RemoteAgentID != 0 {
			old.RemoteAgentID = r.RemoteAgentID
		}

		if res, err := acc.Query("SELECT id FROM remote_accounts WHERE "+
			"remote_agent_id=? AND login=?", old.RemoteAgentID, r.Login); err != nil {
			return err
		} else if len(res) > 0 {
			return database.InvalidError("A remote account with the same login '%s' "+
				"already exist", r.Login)
		}
	}

	if r.RemoteAgentID != 0 {
		if res, err := acc.Query("SELECT id FROM remote_agents WHERE id=?",
			r.RemoteAgentID); err != nil {
			return err
		} else if len(res) == 0 {
			return database.InvalidError("No remote agent found with the ID '%v'",
				r.RemoteAgentID)
		}
	}

	return nil
}

// GetCerts fetch in the database then return the associated Certificates if they exist
func (r *RemoteAccount) GetCerts(ses database.Accessor) ([]Cert, error) {
	filters := &database.Filters{
		Conditions: builder.And(builder.Eq{"owner_type": r.TableName()},
			builder.Eq{"owner_id": r.ID}),
	}

	results := []Cert{}
	if err := ses.Select(&results, filters); err != nil {
		return nil, err
	}
	return results, nil
}

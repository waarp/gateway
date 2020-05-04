package model

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
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

// BeforeInsert checks if the new `LocalAccount` entry is valid and can be
// inserted in the database.
func (l *LocalAccount) BeforeInsert(db database.Accessor) error {
	if l.Password != nil {
		var err error
		if l.Password, err = hashPassword(l.Password); err != nil {
			return err
		}
	}

	if l.ID != 0 {
		return database.InvalidError("the account's ID cannot be entered manually")
	}
	if l.LocalAgentID == 0 {
		return database.InvalidError("the account's agentID cannot be empty")
	}
	if l.Login == "" {
		return database.InvalidError("the account's login cannot be empty")
	}

	if res, err := db.Query("SELECT id FROM local_agents WHERE id=?", l.LocalAgentID); err != nil {
		return err
	} else if len(res) == 0 {
		return database.InvalidError("no local agent found with the ID '%v'", l.LocalAgentID)
	}

	if res, err := db.Query("SELECT id FROM local_accounts WHERE local_agent_id=? "+
		"AND login=?", l.LocalAgentID, l.Login); err != nil {
		return err
	} else if len(res) > 0 {
		return database.InvalidError("a local account with the same login '%s' "+
			"already exist", l.Login)
	}

	return nil
}

// BeforeUpdate checks if the updated `LocalAccount` entry is valid and can be
// updated in the database.
func (l *LocalAccount) BeforeUpdate(db database.Accessor, id uint64) error {
	if l.Password != nil {
		var err error
		if l.Password, err = hashPassword(l.Password); err != nil {
			return err
		}
	}

	if l.ID != 0 {
		return database.InvalidError("the account's ID cannot be entered manually")
	}

	if l.LocalAgentID != 0 {
		if res, err := db.Query("SELECT id FROM local_agents WHERE id=?", l.LocalAgentID); err != nil {
			return err
		} else if len(res) == 0 {
			return database.InvalidError("no local agent found with the ID '%v'", l.LocalAgentID)
		}
	}

	if l.Login != "" {
		old := LocalAccount{ID: id}
		if err := db.Get(&old); err != nil {
			return err
		}
		if l.LocalAgentID != 0 {
			old.LocalAgentID = l.LocalAgentID
		}

		if res, err := db.Query("SELECT id FROM local_accounts WHERE local_agent_id=? "+
			"AND login=?", old.LocalAgentID, l.Login); err != nil {
			return err
		} else if len(res) > 0 {
			return database.InvalidError("a local account with the same login '%s' "+
				"already exist", l.Login)
		}
	}

	return nil
}

// BeforeDelete is called before deleting the account from the database. Its
// role is to delete all the certificates tied to the account.
func (l *LocalAccount) BeforeDelete(db database.Accessor) error {
	filter := builder.Eq{"owner_type": l.TableName(), "owner_id": l.ID}
	if err := db.Execute(builder.Delete().From((&Cert{}).TableName()).
		Where(filter)); err != nil {
		return err
	}

	return nil
}

package model

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"github.com/go-xorm/builder"
)

func init() {
	database.Tables = append(database.Tables, &RemoteAgent{})
}

// RemoteAgent represents a distant server instance with which the gateway can
// communicate and make transfers. The struct contains the information needed by
// the gateway to connect to the server.
type RemoteAgent struct {

	// The agent's database ID.
	ID uint64 `xorm:"pk autoincr 'id'"`

	// The agent's display name.
	Name string `xorm:"unique notnull 'name'"`

	// The protocol used by the agent.
	Protocol string `xorm:"notnull 'protocol'"`

	// The agent's configuration in raw JSON format.
	ProtoConfig []byte `xorm:"notnull 'proto_config'"`
}

// TableName returns the local_agent table name.
func (r *RemoteAgent) TableName() string {
	return "remote_agents"
}

func (r *RemoteAgent) validateProtoConfig() error {
	conf, err := config.GetProtoConfig(r.Protocol, r.ProtoConfig)
	if err != nil {
		return err
	}
	return conf.ValidPartner()
}

// GetCerts fetch in the database then return the associated Certificates if they exist
func (r *RemoteAgent) GetCerts(db database.Accessor) ([]Cert, error) {
	conditions := make([]builder.Cond, 0)
	conditions = append(conditions, builder.Eq{"owner_type": "remote_agents"})
	conditions = append(conditions, builder.Eq{"owner_id": r.ID})

	filters := &database.Filters{
		Conditions: builder.And(conditions...),
	}

	// TODO get only validate certificates
	results := []Cert{}
	if err := db.Select(&results, filters); err != nil {
		return nil, err
	}
	return results, nil
}

// BeforeInsert is called before inserting a new `RemoteAgent` entry in the
// database. It checks whether the new entry is valid or not.
func (r *RemoteAgent) BeforeInsert(db database.Accessor) error {
	if r.ID != 0 {
		return database.InvalidError("the agent's ID cannot be entered manually")
	}
	if r.Name == "" {
		return database.InvalidError("the agent's name cannot be empty")
	}
	if r.ProtoConfig == nil {
		return database.InvalidError("the agent's configuration cannot be empty")
	}
	if err := r.validateProtoConfig(); err != nil {
		return database.InvalidError(err.Error())
	}

	if res, err := db.Query("SELECT id FROM remote_agents WHERE name=?", r.Name); err != nil {
		return err
	} else if len(res) > 0 {
		return database.InvalidError("a remote agent with the same name '%s' "+
			"already exist", r.Name)
	}

	return nil
}

// BeforeUpdate is called before updating an existing `RemoteAgent` entry from
// the database. It checks whether the updated entry is valid or not.
func (r *RemoteAgent) BeforeUpdate(db database.Accessor, id uint64) error {
	if r.ID != 0 {
		return database.InvalidError("the agent's ID cannot be entered manually")
	}

	if r.Protocol != "" || r.ProtoConfig != nil {
		if err := r.validateProtoConfig(); err != nil {
			return database.InvalidError(err.Error())
		}
	}

	if r.Name != "" {
		if res, err := db.Query("SELECT id FROM remote_agents WHERE name=? AND "+
			"id<>?", r.Name, id); err != nil {
			return err
		} else if len(res) > 0 {
			return database.InvalidError("a remote agent with the same name "+
				"'%s' already exist", r.Name)
		}
	}

	return nil
}

// BeforeDelete is called before deleting the account from the database. Its
// role is to delete all the certificates tied to the account.
func (r *RemoteAgent) BeforeDelete(db database.Accessor) error {
	filterCert := builder.Eq{"owner_type": r.TableName(), "owner_id": r.ID}
	if err := db.Execute(builder.Delete().From((&Cert{}).TableName()).
		Where(filterCert)); err != nil {
		return err
	}

	accounts := []*RemoteAccount{}
	filterAcc := builder.Eq{"remote_agent_id": r.ID}
	if err := db.Select(&accounts, &database.Filters{Conditions: filterAcc}); err != nil {
		return err
	}
	for _, account := range accounts {
		if err := account.BeforeDelete(db); err != nil {
			return err
		}
	}
	if err := db.Execute(builder.Delete().From((&RemoteAccount{}).TableName()).
		Where(filterAcc)); err != nil {
		return err
	}

	return nil
}

package model

import (
	"encoding/json"
	"net"

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
	ID uint64 `xorm:"pk autoincr <- 'id'"`

	// The agent's display name.
	Name string `xorm:"unique notnull 'name'"`

	// The protocol used by the agent.
	Protocol string `xorm:"notnull 'protocol'"`

	// The agent's configuration in raw JSON format.
	ProtoConfig json.RawMessage `xorm:"notnull 'proto_config'"`

	// The agent's address (including the port)
	Address string `xorm:"notnull 'address'"`
}

// TableName returns the remote_agent table name.
func (r *RemoteAgent) TableName() string {
	return "remote_agents"
}

// GetID returns the agent's ID.
func (r *RemoteAgent) GetID() uint64 {
	return r.ID
}

func (r *RemoteAgent) validateProtoConfig() error {
	conf, err := config.GetProtoConfig(r.Protocol, r.ProtoConfig)
	if err != nil {
		return err
	}
	if err := conf.ValidPartner(); err != nil {
		return err
	}
	r.ProtoConfig, err = json.Marshal(conf)

	return err
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

// Validate is called before inserting a new `RemoteAgent` entry in the
// database. It checks whether the new entry is valid or not.
func (r *RemoteAgent) Validate(db database.Accessor) error {
	if r.Name == "" {
		return database.InvalidError("the agent's name cannot be empty")
	}
	if r.Address == "" {
		return database.InvalidError("the partner's address cannot be empty")
	}
	if _, _, err := net.SplitHostPort(r.Address); err != nil {
		return database.InvalidError("'%s' is not a valid partner address", r.Address)
	}

	if r.ProtoConfig == nil {
		return database.InvalidError("the agent's configuration cannot be empty")
	}
	if err := r.validateProtoConfig(); err != nil {
		return database.InvalidError(err.Error())
	}

	if res, err := db.Query("SELECT id FROM remote_agents WHERE id<>? AND name=?",
		r.ID, r.Name); err != nil {
		return err
	} else if len(res) > 0 {
		return database.InvalidError("a remote agent with the same name '%s' "+
			"already exist", r.Name)
	}

	return nil
}

// BeforeDelete is called before deleting the account from the database. Its
// role is to delete all the certificates tied to the account.
func (r *RemoteAgent) BeforeDelete(db database.Accessor) error {
	trans, err := db.Query("SELECT id FROM transfers WHERE is_server=? AND agent_id=?", false, r.ID)
	if err != nil {
		return err
	}
	if len(trans) > 0 {
		return database.InvalidError("this partner is currently being used in a " +
			"running transfer and cannot be deleted, cancel the transfer or wait " +
			"for it to finish")
	}

	certQuery := "DELETE FROM certificates WHERE " +
		" (owner_type='remote_agents' AND owner_id=?) " +
		"OR" +
		" (owner_type='remote_accounts' AND owner_id IN " +
		"  (SELECT id FROM remote_accounts WHERE remote_agent_id=?))"
	if err := db.Execute(certQuery, r.ID, r.ID); err != nil {
		return err
	}

	accessQuery := "DELETE FROM rule_access WHERE " +
		" (object_type='remote_agents' AND object_id=?) " +
		"OR" +
		" (object_type='remote_accounts' AND object_id IN " +
		"  (SELECT id FROM remote_accounts WHERE remote_agent_id=?))"
	if err := db.Execute(accessQuery, r.ID, r.ID); err != nil {
		return err
	}

	accountQuery := "DELETE FROM remote_accounts WHERE remote_agent_id=?"
	if err := db.Execute(accountQuery, r.ID); err != nil {
		return err
	}

	return nil
}

package model

import (
	"encoding/json"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

func init() {
	database.Tables = append(database.Tables, &RemoteAgent{})
}

// RemoteAgent represents a distant server instance with which the gateway can
// communicate and make transfers. The struct contains the information needed by
// the gateway to connect to the server.
type RemoteAgent struct {

	// The agent's database ID.
	ID uint64 `xorm:"pk autoincr 'id'" json:"id"`

	// The agent's display name.
	Name string `xorm:"unique notnull 'name'" json:"name"`

	// The protocol used by the agent.
	Protocol string `xorm:"notnull 'protocol'" json:"protocol"`

	// The agent's configuration in raw JSON format.
	ProtoConfig []byte `xorm:"notnull 'proto_config'" json:"protoConfig"`
}

// TableName returns the local_agent table name.
func (r *RemoteAgent) TableName() string {
	return "remote_agents"
}

// ValidateInsert is called before inserting a new `RemoteAgent` entry in the
// database. It checks whether the new entry is valid or not.
func (r *RemoteAgent) ValidateInsert(ses *database.Session) error {
	if r.ID != 0 {
		return database.InvalidError("The agent's ID cannot be entered manually")
	}
	if r.Name == "" {
		return database.InvalidError("The agent's name cannot be empty")
	}
	if !IsValidProtocol(r.Protocol) {
		return database.InvalidError("The agent's protocol must be one of: %s",
			validProtocols)
	}
	if !json.Valid(r.ProtoConfig) {
		return database.InvalidError("The agent's configuration is not a valid " +
			"JSON configuration")
	}

	if res, err := ses.Query("SELECT id FROM remote_agents WHERE name=?", r.Name); err != nil {
		return err
	} else if len(res) > 0 {
		return database.InvalidError("A remote agent with the same name '%s' "+
			"already exist", r.Name)
	}

	return nil
}

// ValidateUpdate is called before updating an existing `RemoteAgent` entry from
// the database. It checks whether the updated entry is valid or not.
func (r *RemoteAgent) ValidateUpdate(ses *database.Session, id uint64) error {
	if r.ID != 0 {
		return database.InvalidError("The agent's ID cannot be entered manually")
	}

	if r.Protocol != "" {
		if !IsValidProtocol(r.Protocol) {
			return database.InvalidError("The agent's protocol must be one of: %s",
				validProtocols)
		}
	}
	if r.ProtoConfig != nil {
		if !json.Valid(r.ProtoConfig) {
			return database.InvalidError("The agent's configuration is not a valid " +
				"JSON configuration")
		}
	}

	if r.Name != "" {
		if res, err := ses.Query("SELECT id FROM remote_agents WHERE name=? AND "+
			"id<>?", r.Name, id); err != nil {
			return err
		} else if len(res) > 0 {
			return database.InvalidError("A remote agent with the same name "+
				"'%s' already exist", r.Name)
		}
	}

	return nil
}

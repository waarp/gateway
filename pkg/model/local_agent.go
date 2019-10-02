package model

import (
	"encoding/json"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

func init() {
	database.Tables = append(database.Tables, &LocalAgent{})
}

// LocalAgent represents a local server instance operated by the gateway itself.
// The struct contains the information needed by external agents to connect to
// the server.
type LocalAgent struct {

	// The agent's database ID.
	ID uint64 `xorm:"pk autoincr <- 'id'" json:"id"`

	// The agent's owner (i.e. the name of the gateway instance to which the
	// agent belongs to.
	Owner string `xorm:"unique(loc_ag) notnull 'owner'" json:"-"`

	// The agent's display name.
	Name string `xorm:"unique(loc_ag) notnull 'name'" json:"name"`

	// The protocol used by the agent.
	Protocol string `xorm:"notnull 'protocol'" json:"protocol"`

	// The agent's configuration in raw JSON format.
	ProtoConfig []byte `xorm:"notnull 'proto_config'" json:"protoConfig"`
}

// BeforeInsert is called before inserting the agent in the database. Its
// role is to set the agent's owner.
func (l *LocalAgent) BeforeInsert(ses *database.Session) error {
	l.Owner = database.Owner
	return nil
}

// TableName returns the local_agent table name.
func (l *LocalAgent) TableName() string {
	return "local_agents"
}

// ValidateInsert is called before inserting a new `LocalAgent` entry in the
// database. It checks whether the new entry is valid or not.
func (l *LocalAgent) ValidateInsert(ses *database.Session) error {
	if l.ID != 0 {
		return database.InvalidError("The agent's ID cannot be entered manually")
	}
	if l.Name == "" {
		return database.InvalidError("The agent's name cannot be empty")
	}
	if !IsValidProtocol(l.Protocol) {
		return database.InvalidError("The agent's protocol must be one of: %s",
			validProtocols)
	}
	if !json.Valid(l.ProtoConfig) {
		return database.InvalidError("The agent's configuration is not a " +
			"valid JSON configuration")
	}

	if res, err := ses.Query("SELECT id FROM local_agents WHERE owner=? AND name=?",
		l.Owner, l.Name); err != nil {
		return err
	} else if len(res) > 0 {
		return database.InvalidError("A local agent with the same name '%s' "+
			"already exist", l.Name)
	}

	return nil
}

// ValidateUpdate is called before updating an existing `LocalAgent` entry from
// the database. It checks whether the updated entry is valid or not.
func (l *LocalAgent) ValidateUpdate(ses *database.Session, id uint64) error {
	if l.ID != 0 {
		return database.InvalidError("The agent's ID cannot be entered manually")
	}
	if l.Owner != "" {
		return database.InvalidError("The agent's owner cannot be changed")
	}

	if l.Name != "" {
		if res, err := ses.Query("SELECT id FROM local_agents WHERE owner=? "+
			"AND name=? AND id<>?", database.Owner, l.Name, id); err != nil {
			return err
		} else if len(res) > 0 {
			return database.InvalidError("A local agent with the same name "+
				"'%s' already exist", l.Name)
		}
	}

	if l.Protocol != "" {
		if !IsValidProtocol(l.Protocol) {
			return database.InvalidError("The agent's protocol must be one of: %s",
				validProtocols)
		}
	}

	if l.ProtoConfig != nil {
		if !json.Valid(l.ProtoConfig) {
			return database.InvalidError("The agent's configuration is not a " +
				"valid JSON configuration")
		}
	}

	return nil
}

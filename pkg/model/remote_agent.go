package model

import (
	"encoding/json"
	"fmt"
	"net"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
)

//nolint:gochecknoinits // init is used by design
func init() {
	database.AddTable(&RemoteAgent{})
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

// TableName returns the remote agents table name.
func (*RemoteAgent) TableName() string {
	return TableRemAgents
}

// Appellation returns the name of 1 element of the remote agents table.
func (*RemoteAgent) Appellation() string {
	return "partner"
}

// GetID returns the agent's ID.
func (r *RemoteAgent) GetID() uint64 {
	return r.ID
}

func (r *RemoteAgent) validateProtoConfig() error {
	conf, err := config.GetProtoConfig(r.Protocol, r.ProtoConfig)
	if err != nil {
		return fmt.Errorf("cannot parse protocol configuration for server %q: %w", r.Name, err)
	}

	if err2 := conf.ValidPartner(); err2 != nil {
		return fmt.Errorf("protocol configuration for %q is invalid: %w", r.Name, err2)
	}

	r.ProtoConfig, err = json.Marshal(conf)
	if err != nil {
		return fmt.Errorf("cannot marshal protocol configuration for %q to JSON: %w", r.Name, err)
	}

	return nil
}

// BeforeWrite is called before inserting a new `RemoteAgent` entry in the
// database. It checks whether the new entry is valid or not.
func (r *RemoteAgent) BeforeWrite(db database.ReadAccess) database.Error {
	if r.Name == "" {
		return database.NewValidationError("the agent's name cannot be empty")
	}

	if r.Address == "" {
		return database.NewValidationError("the partner's address cannot be empty")
	}

	if _, _, err := net.SplitHostPort(r.Address); err != nil {
		return database.NewValidationError("'%s' is not a valid partner address", r.Address)
	}

	if r.ProtoConfig == nil {
		return database.NewValidationError("the agent's configuration cannot be empty")
	}

	if err := r.validateProtoConfig(); err != nil {
		return database.NewValidationError(err.Error())
	}

	n, err := db.Count(&RemoteAgent{}).Where("id<>? AND name=?", r.ID, r.Name).Run()
	if err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError("a remote agent with the same name '%s' "+
			"already exist", r.Name)
	}

	return nil
}

// BeforeDelete is called before deleting the account from the database. Its
// role is to delete all the certificates tied to the account.
//nolint:dupl // too many differences to factorize esily into a function
func (r *RemoteAgent) BeforeDelete(db database.Access) database.Error {
	n, err := db.Count(&Transfer{}).Where("is_server=? AND agent_id=?", false, r.ID).Run()
	if err != nil {
		return err
	}

	if n > 0 {
		return database.NewValidationError("this partner is currently being used in one " +
			"or more running transfers and thus cannot be deleted, cancel these " +
			"transfers or wait for them to finish")
	}

	certQuery := db.DeleteAll(&Crypto{}).Where(
		"(owner_type=? AND owner_id=?) OR (owner_type=? AND owner_id IN "+
			"(SELECT id FROM "+TableRemAccounts+" WHERE remote_agent_id=?))",
		TableRemAgents, r.ID, TableRemAccounts, r.ID)
	if err := certQuery.Run(); err != nil {
		return err
	}

	accessQuery := db.DeleteAll(&RuleAccess{}).Where(
		" (object_type=? AND object_id=?) OR (object_type=? AND object_id IN "+
			"(SELECT id FROM "+TableRemAccounts+" WHERE remote_agent_id=?))",
		TableRemAgents, r.ID, TableRemAccounts, r.ID)
	if err := accessQuery.Run(); err != nil {
		return err
	}

	accountQuery := db.DeleteAll(&RemoteAccount{}).Where("remote_agent_id=?", r.ID)

	return accountQuery.Run()
}

// GetCryptos returns a list of all the partner's certificates.
func (r *RemoteAgent) GetCryptos(db database.ReadAccess) ([]Crypto, database.Error) {
	return GetCryptos(db, r)
}

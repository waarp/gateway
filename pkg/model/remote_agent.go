package model

import (
	"encoding/json"
	"fmt"
	"net"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

// RemoteAgent represents a distant server instance with which the gateway can
// communicate and make transfers. The struct contains the information needed by
// the gateway to connect to the server.
type RemoteAgent struct {
	ID int64 `xorm:"<- id AUTOINCR"` // The partner's database ID.

	Name     string `xorm:"name"`     // The partner's display name.
	Protocol string `xorm:"protocol"` // The partner's protocol.
	Address  string `xorm:"address"`  // The partner's address (including the port)

	// The partner's protocol configuration in raw JSON format.
	ProtoConfig json.RawMessage `xorm:"proto_config"`
}

func (*RemoteAgent) TableName() string   { return TableRemAgents }
func (*RemoteAgent) Appellation() string { return "partner" }
func (r *RemoteAgent) GetID() int64      { return r.ID }

//nolint:dupl // factorizing would add complexity
func (r *RemoteAgent) validateProtoConfig() error {
	conf, err := config.GetProtoConfig(r.Protocol, r.ProtoConfig)
	if err != nil {
		return fmt.Errorf("cannot parse protocol configuration for server %q: %w", r.Name, err)
	}

	if r66Conf, ok := conf.(*config.R66ProtoConfig); ok && r66Conf.IsTLS != nil && *r66Conf.IsTLS {
		r.Protocol = config.ProtocolR66TLS
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
		r.ProtoConfig = json.RawMessage(`{}`)
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
// role is to check whether the partner is still used in any ongoing transfer.
func (r *RemoteAgent) BeforeDelete(db database.Access) database.Error {
	if n, err := db.Count(&Transfer{}).Where(`remote_account_id IN (SELECT id 
		FROM remote_accounts WHERE remote_agent_id=?)`, r.ID).Run(); err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError("this partner is currently being used " +
			"in one or more running transfers and thus cannot be deleted, cancel " +
			"these transfers or wait for them to finish")
	}

	return nil
}

// GetCryptos returns a list of all the partner's certificates.
func (r *RemoteAgent) GetCryptos(db database.ReadAccess) ([]*Crypto, error) {
	return getCryptos(db, r)
}

func (r *RemoteAgent) SetCryptoOwner(c *Crypto)             { c.RemoteAgentID = utils.NewNullInt64(r.ID) }
func (r *RemoteAgent) GenCryptoSelectCond() (string, int64) { return "remote_agent_id=?", r.ID }
func (r *RemoteAgent) SetAccessTarget(a *RuleAccess)        { a.RemoteAgentID = utils.NewNullInt64(r.ID) }
func (r *RemoteAgent) GenAccessSelectCond() (string, int64) { return "remote_agent_id=?", r.ID }

func (r *RemoteAgent) GetAuthorizedRules(db database.ReadAccess) ([]*Rule, error) {
	return getAuthorizedRules(db, "remote_agent_id", r.ID)
}

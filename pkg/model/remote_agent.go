package model

import (
	"fmt"
	"net"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// RemoteAgent represents a distant server instance with which the gateway can
// communicate and make transfers. The struct contains the information needed by
// the gateway to connect to the server.
type RemoteAgent struct {
	ID    int64  `xorm:"<- id AUTOINCR"` // The partner's database ID.
	Owner string `xorm:"owner"`          // The client's owner (the gateway to which it belongs)

	Name     string `xorm:"name"`     // The partner's display name.
	Protocol string `xorm:"protocol"` // The partner's protocol.
	Address  string `xorm:"address"`  // The partner's address (including the port)

	// The partner's protocol configuration as a map.
	ProtoConfig map[string]any `xorm:"proto_config"`
}

func (*RemoteAgent) TableName() string   { return TableRemAgents }
func (*RemoteAgent) Appellation() string { return "partner" }
func (r *RemoteAgent) GetID() int64      { return r.ID }

// BeforeWrite is called before inserting a new `RemoteAgent` entry in the
// database. It checks whether the new entry is valid or not.
func (r *RemoteAgent) BeforeWrite(db database.ReadAccess) error {
	r.Owner = conf.GlobalConfig.GatewayName

	if r.Name == "" {
		return database.NewValidationError("the agent's name cannot be empty")
	}

	if r.Address == "" {
		return database.NewValidationError("the partner's address cannot be empty")
	}

	if _, err := net.ResolveTCPAddr("tcp", r.Address); err != nil {
		return database.NewValidationError("'%s' is not a valid partner address", r.Address)
	}

	if r.ProtoConfig == nil {
		r.ProtoConfig = map[string]any{}
	}

	if err := ConfigChecker.CheckPartnerConfig(r.Protocol, r.ProtoConfig); err != nil {
		return database.NewValidationError("%v", err)
	}

	if n, err := db.Count(&RemoteAgent{}).Where("id<>? AND owner=? AND name=?",
		r.ID, r.Owner, r.Name).Run(); err != nil {
		return fmt.Errorf("failed to check for duplicate remote agents: %w", err)
	} else if n > 0 {
		return database.NewValidationError("a remote agent with the same name '%s' "+
			"already exist", r.Name)
	}

	return nil
}

// BeforeDelete is called before deleting the account from the database. Its
// role is to check whether the partner is still used in any ongoing transfer.
func (r *RemoteAgent) BeforeDelete(db database.Access) error {
	if n, err := db.Count(&Transfer{}).Where(`remote_account_id IN (SELECT id 
		FROM remote_accounts WHERE remote_agent_id=?)`, r.ID).Run(); err != nil {
		return fmt.Errorf("failed to check for ongoing transfers: %w", err)
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

//nolint:goconst //duplicate is for different columns, best keep separate
func (r *RemoteAgent) GenCryptoSelectCond() (string, int64) { return "remote_agent_id=?", r.ID }
func (r *RemoteAgent) SetCryptoOwner(c *Crypto)             { c.RemoteAgentID = utils.NewNullInt64(r.ID) }
func (r *RemoteAgent) SetAccessTarget(a *RuleAccess)        { a.RemoteAgentID = utils.NewNullInt64(r.ID) }
func (r *RemoteAgent) GenAccessSelectCond() (string, int64) { return "remote_agent_id=?", r.ID }

func (r *RemoteAgent) GetAuthorizedRules(db database.ReadAccess) ([]*Rule, error) {
	var rules Rules
	if err := db.Select(&rules).Where(fmt.Sprintf(
		`id IN (SELECT DISTINCT rule_id FROM %s WHERE remote_agent_id=?)
		  OR (SELECT COUNT(*) FROM %s WHERE rule_id = id) = 0`,
		TableRuleAccesses, TableRuleAccesses), r.ID).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve the authorized rules: %w", err)
	}

	return rules, nil
}

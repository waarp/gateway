package model

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

// RemoteAgent represents a distant server instance with which the gateway can
// communicate and make transfers. The struct contains the information needed by
// the gateway to connect to the server.
type RemoteAgent struct {
	ID    int64  `xorm:"<- id AUTOINCR"` // The partner's database ID.
	Owner string `xorm:"owner"`          // The client's owner (the gateway to which it belongs)

	Name     string        `xorm:"name"`     // The partner's display name.
	Protocol string        `xorm:"protocol"` // The partner's protocol.
	Address  types.Address `xorm:"address"`  // The partner's address (including the port)

	// The partner's protocol configuration as a map.
	ProtoConfig map[string]any `xorm:"proto_config"`
}

func (*RemoteAgent) TableName() string   { return TableRemAgents }
func (*RemoteAgent) Appellation() string { return NameRemoteAgent }
func (r *RemoteAgent) GetID() int64      { return r.ID }
func (*RemoteAgent) IsServer() bool      { return true }
func (r *RemoteAgent) Host() string      { return r.Address.Host }

func (r *RemoteAgent) validateProtoConfig() error {
	if err := ConfigChecker.CheckPartnerConfig(r.Protocol, r.ProtoConfig); err != nil {
		return database.WrapAsValidationError(err)
	}

	// For backwards compatibility, in the presence of the r66 "isTLS" property,
	// we change the protocol to r66-tls.
	if r.Protocol == protoR66 && compatibility.IsTLS(r.ProtoConfig) {
		r.Protocol = protoR66TLS
	}

	return nil
}

// BeforeWrite is called before inserting a new `RemoteAgent` entry in the
// database. It checks whether the new entry is valid or not.
func (r *RemoteAgent) BeforeWrite(db database.Access) error {
	r.Owner = conf.GlobalConfig.GatewayName

	if r.Name == "" {
		return database.NewValidationError("the agent's name cannot be empty")
	}

	if err := r.Address.Validate(); err != nil {
		return database.NewValidationError("address validation failed: %w", err)
	}

	if r.ProtoConfig == nil {
		r.ProtoConfig = map[string]any{}
	}

	if err := r.validateProtoConfig(); err != nil {
		return database.WrapAsValidationError(err)
	}

	if n, err := db.Count(&RemoteAgent{}).Where("id<>? AND owner=? AND name=?",
		r.ID, r.Owner, r.Name).Run(); err != nil {
		return fmt.Errorf("failed to check for duplicate remote agents: %w", err)
	} else if n > 0 {
		return database.NewValidationError("a remote agent with the same name %q "+
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

// GetCredentials returns a list of all the partner's auth methods.
func (r *RemoteAgent) GetCredentials(db database.ReadAccess, authTypes ...string,
) (Credentials, error) {
	return getCredentials(db, r, authTypes...)
}

func (r *RemoteAgent) SetCredOwner(a *Credential)           { a.RemoteAgentID = utils.NewNullInt64(r.ID) }
func (r *RemoteAgent) GetCredCond() (string, int64)         { return "remote_agent_id=?", r.ID }
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

func (r *RemoteAgent) GetProtocol(database.ReadAccess) (string, error) {
	return r.Protocol, nil
}

func (r *RemoteAgent) Authenticate(db database.ReadAccess, authType string, value any,
) (*authentication.Result, error) {
	return authenticate(db, r, authType, r.Protocol, value)
}

func (r *RemoteAgent) AfterInsert(db database.Access) error { return r.AfterUpdate(db) }

func (r *RemoteAgent) getR66ServerPswd() string {
	if !isR66(r.Protocol) {
		return ""
	}

	serverPwd, hasPwd := r.ProtoConfig["serverPassword"]
	if !hasPwd {
		return ""
	}

	serverPasswd, pwdIsStr := serverPwd.(string)
	if !pwdIsStr || serverPasswd == "" {
		return ""
	}

	return serverPasswd
}

func (r *RemoteAgent) AfterUpdate(db database.Access) error {
	serverPasswd := r.getR66ServerPswd()
	if serverPasswd == "" {
		return nil
	}

	var pswd Credential
	if getErr := db.Get(&pswd, "remote_agent_id=? AND type=?",
		r.ID, authPassword).Run(); database.IsNotFound(getErr) {
		pswd.RemoteAgentID = utils.NewNullInt64(r.ID)
		pswd.Type = authPassword
		pswd.Value = serverPasswd

		if insErr := db.Insert(&pswd).Run(); insErr != nil {
			return fmt.Errorf("failed to insert R66 partner password: %w", insErr)
		}
	} else if getErr != nil {
		return fmt.Errorf("failed to check for existing credentials: %w", getErr)
	} else {
		pswd.Value = serverPasswd
		if updErr := db.Update(&pswd).Run(); updErr != nil {
			return fmt.Errorf("failed to update R66 partner password: %w", updErr)
		}
	}

	return nil
}

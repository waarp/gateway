package model

import (
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

// LocalAgent represents a local server instance operated by the gateway itself.
// The struct contains the information needed by external agents to connect to
// the server.
type LocalAgent struct {
	ID    int64  `xorm:"<- id AUTOINCR"` // The agent's database ID.
	Owner string `xorm:"owner"`          // The agent's owner (the gateway to which it belongs).

	Name     string        `xorm:"name"`     // The server's name.
	Address  types.Address `xorm:"address"`  // The agent's address (including the port)
	Protocol string        `xorm:"protocol"` // The server's protocol.
	Disabled bool          `xorm:"disabled"` // Whether the server is enabled at startup or not.

	RootDir       string `xorm:"root_dir"`        // The root directory of the agent.
	ReceiveDir    string `xorm:"receive_dir"`     // The server's directory for received files.
	SendDir       string `xorm:"send_dir"`        // The server's directory for files to be sent.
	TmpReceiveDir string `xorm:"tmp_receive_dir"` // The server's temporary directory for partially received files.

	// The server's protocol configuration as a map.
	ProtoConfig map[string]any `xorm:"proto_config"`
}

func (*LocalAgent) TableName() string   { return TableLocAgents }
func (*LocalAgent) Appellation() string { return "server" }
func (l *LocalAgent) GetID() int64      { return l.ID }
func (*LocalAgent) IsServer() bool      { return true }
func (l *LocalAgent) Host() string      { return l.Address.Host }

func (l *LocalAgent) validateProtoConfig() error {
	if err := ConfigChecker.CheckServerConfig(l.Protocol, l.ProtoConfig); err != nil {
		return database.WrapAsValidationError(err)
	}

	// For backwards compatibility, in the presence of the r66 "isTLS" property,
	// we change the protocol to r66-tls.
	if l.Protocol == protoR66 && compatibility.IsTLS(l.ProtoConfig) {
		l.Protocol = protoR66TLS
	}

	return nil
}

func (l *LocalAgent) makePaths() {
	isEmpty := func(path string) bool {
		return path == "." || path == ""
	}

	if !isEmpty(l.RootDir) {
		if isEmpty(l.ReceiveDir) {
			l.ReceiveDir = "in"
		}

		if isEmpty(l.SendDir) {
			l.SendDir = "out"
		}

		if isEmpty(l.TmpReceiveDir) {
			l.TmpReceiveDir = "tmp"
		}
	}
}

// BeforeWrite is called before inserting a new `LocalAgent` entry in the
// database. It checks whether the new entry is valid or not.
func (l *LocalAgent) BeforeWrite(db database.Access) error {
	l.Owner = conf.GlobalConfig.GatewayName
	l.makePaths()

	if l.Name == "" {
		return database.NewValidationError("the agent's name cannot be empty")
	}

	if err := l.Address.Validate(); err != nil {
		return database.NewValidationError("address validation failed: %w", err)
	}

	if l.ProtoConfig == nil {
		l.ProtoConfig = map[string]any{}
	}

	if err := l.validateProtoConfig(); err != nil {
		return database.WrapAsValidationError(err)
	}

	if n, err := db.Count(l).Where("id<>? AND owner=? AND name=?", l.ID, l.Owner,
		l.Name).Run(); err != nil {
		return fmt.Errorf("failed to check for duplicate local agents: %w", err)
	} else if n > 0 {
		return database.NewValidationError(
			"a local agent with the same name %q already exist", l.Name)
	}

	if n, err := db.Count(l).Where("id<>? AND owner=? AND address=?",
		l.ID, l.Owner, l.Address.String()).Run(); err != nil {
		return fmt.Errorf("failed to check for duplicate local agent addresses: %w", err)
	} else if n > 0 {
		return database.NewValidationError(
			"a local agent with the same address %q already exist", &l.Address)
	}

	return nil
}

// BeforeDelete is called before deleting the account from the database. Its
// role is to check whether the server is still used in any ongoing transfer.
func (l *LocalAgent) BeforeDelete(db database.Access) error {
	if n, err := db.Count(&Transfer{}).Where(`local_account_id IN (SELECT id
		FROM local_accounts WHERE local_agent_id=?)`, l.ID).Run(); err != nil {
		return fmt.Errorf("failed to check for ongoing transfers: %w", err)
	} else if n > 0 {
		//nolint:goconst //too specific
		return database.NewValidationError("this server is currently being used " +
			"in one or more running transfers and thus cannot be deleted, cancel " +
			"these transfers or wait for them to finish")
	}

	return nil
}

// GetCredentials fetch in the database then return the associated Credentials if they exist.
func (l *LocalAgent) GetCredentials(db database.ReadAccess, authTypes ...string,
) (Credentials, error) {
	return getCredentials(db, l, authTypes...)
}

func (l *LocalAgent) SetCredOwner(a *Credential)           { a.LocalAgentID = utils.NewNullInt64(l.ID) }
func (l *LocalAgent) GetCredCond() (string, int64)         { return "local_agent_id=?", l.ID }
func (l *LocalAgent) SetAccessTarget(a *RuleAccess)        { a.LocalAgentID = utils.NewNullInt64(l.ID) }
func (l *LocalAgent) GenAccessSelectCond() (string, int64) { return "local_agent_id=?", l.ID }

func (l *LocalAgent) GetAuthorizedRules(db database.ReadAccess) ([]*Rule, error) {
	var rules Rules
	if err := db.Select(&rules).Where(fmt.Sprintf(
		`id IN (SELECT DISTINCT rule_id FROM %s WHERE local_agent_id=?)
		  OR (SELECT COUNT(*) FROM %s WHERE rule_id = id) = 0`,
		TableRuleAccesses, TableRuleAccesses), l.ID).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve the authorized rules: %w", err)
	}

	return rules, nil
}

func (l *LocalAgent) GetProtocol(database.ReadAccess) (string, error) {
	return l.Protocol, nil
}
func (l *LocalAgent) AfterInsert(db database.Access) error { return l.AfterUpdate(db) }

// AfterUpdate is called after any write operation on the local_agents table.
// If the agent uses R66, the function checks if is still uses the old credentials
// stored in the proto config. If it does, an equivalent Credential is inserted.
// Will be removed once server passwords are definitely removed from the proto config.
//
//nolint:dupl //duplicate is for RemoteAgent, best keep separate
func (l *LocalAgent) AfterUpdate(db database.Access) error {
	if !isR66(l.Protocol) {
		return nil
	}

	cipher, pwdErr := utils.GetAs[string](l.ProtoConfig, "serverPassword")
	if pwdErr != nil {
		return nil // no server password in proto config
	}

	serverPasswd, aesErr := utils.AESDecrypt(database.GCM, cipher)
	if aesErr != nil {
		return fmt.Errorf("failed to decrypt R66 server JSON password: %w", aesErr)
	}

	if n, err := db.Count(&Credential{}).Where("local_agent_id=? AND type=?",
		l.ID, authPassword).Run(); err != nil {
		return fmt.Errorf("failed to check for duplicate credentials: %w", err)
	} else if n != 0 {
		return nil // already has a password
	}

	pswd := &Credential{
		LocalAgentID: utils.NewNullInt64(l.ID),
		Type:         authPassword,
		Value:        serverPasswd,
	}

	if err := db.Insert(pswd).Run(); err != nil {
		return fmt.Errorf("failed to insert server password: %w", err)
	}

	return l.AfterRead(db)
}

func (l *LocalAgent) AfterRead(database.ReadAccess) error {
	if !isR66(l.Protocol) {
		return nil
	}

	servPwd, err := utils.GetAs[string](l.ProtoConfig, "serverPassword")
	if errors.Is(err, utils.ErrKeyNotFound) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to retrieve the server password: %w", err)
	}

	clear, err := utils.AESDecrypt(database.GCM, servPwd)
	if err != nil {
		return fmt.Errorf("failed to decrypt the server password: %w", err)
	}

	l.ProtoConfig["serverPassword"] = clear

	return nil
}

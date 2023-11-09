package model

import (
	"encoding/json"
	"fmt"
	"net"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/names"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

// LocalAgent represents a local server instance operated by the gateway itself.
// The struct contains the information needed by external agents to connect to
// the server.
type LocalAgent struct {
	ID    int64  `xorm:"<- id AUTOINCR"` // The agent's database ID.
	Owner string `xorm:"owner"`          // The agent's owner (the gateway to which it belongs)

	Name     string `xorm:"name"`     // The server's name.
	Address  string `xorm:"address"`  // The agent's address (including the port)
	Protocol string `xorm:"protocol"` // The server's protocol.
	Enabled  bool   `xorm:"enabled"`  // Whether the server is enabled at startup or not.

	RootDir       string `xorm:"root_dir"`        // The root directory of the agent.
	ReceiveDir    string `xorm:"receive_dir"`     // The server's directory for received files.
	SendDir       string `xorm:"send_dir"`        // The server's directory for files to be sent.
	TmpReceiveDir string `xorm:"tmp_receive_dir"` // The server's temporary directory for partially received files.

	// The server's protocol configuration in raw JSON format.
	ProtoConfig json.RawMessage `xorm:"proto_config"`
}

func (*LocalAgent) TableName() string   { return TableLocAgents }
func (*LocalAgent) Appellation() string { return "server" }
func (l *LocalAgent) GetID() int64      { return l.ID }

//nolint:dupl // factorizing would add complexity
func (l *LocalAgent) validateProtoConfig() error {
	protoConf, err := config.GetProtoConfig(l.Protocol, l.ProtoConfig)
	if err != nil {
		return fmt.Errorf("cannot parse protocol config for server %q: %w", l.Name, err)
	}

	if r66Conf, ok := protoConf.(*config.R66ProtoConfig); ok && r66Conf.IsTLS != nil && *r66Conf.IsTLS {
		l.Protocol = config.ProtocolR66TLS
	}

	if err2 := protoConf.ValidServer(); err2 != nil {
		return fmt.Errorf("the protocol configuration for server %q is not valid: %w",
			l.Name, err2)
	}

	l.ProtoConfig, err = json.Marshal(protoConf)
	if err != nil {
		return fmt.Errorf("cannot marshal the protocol config for server %q to JSON: %w",
			l.Name, err)
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
func (l *LocalAgent) BeforeWrite(db database.ReadAccess) database.Error {
	l.Owner = conf.GlobalConfig.GatewayName
	l.makePaths()

	if l.Name == "" {
		return database.NewValidationError("the agent's name cannot be empty")
	}

	if names.IsReservedServiceName(l.Name) {
		return database.NewValidationError("%s is a reserved server name", l.Name)
	}

	if l.Address == "" {
		return database.NewValidationError("the server's address cannot be empty")
	}

	if _, _, err := net.SplitHostPort(l.Address); err != nil {
		return database.NewValidationError("'%s' is not a valid server address", l.Address)
	}

	if l.ProtoConfig == nil {
		l.ProtoConfig = json.RawMessage(`{}`)
	}

	if err := l.validateProtoConfig(); err != nil {
		return database.NewValidationError(err.Error())
	}

	if n, err := db.Count(l).Where("id<>? AND owner=? AND name=?", l.ID, l.Owner,
		l.Name).Run(); err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError(
			"a local agent with the same name '%s' already exist", l.Name)
	}

	if n, err := db.Count(l).Where("id<>? AND owner=? AND address=?", l.ID,
		l.Owner, l.Address).Run(); err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError(
			"a local agent with the same address '%s' already exist", l.Address)
	}

	return nil
}

// BeforeDelete is called before deleting the account from the database. Its
// role is to check whether the server is still used in any ongoing transfer.
func (l *LocalAgent) BeforeDelete(db database.Access) database.Error {
	if n, err := db.Count(&Transfer{}).Where(`local_account_id IN (SELECT id
		FROM local_accounts WHERE local_agent_id=?)`, l.ID).Run(); err != nil {
		return err
	} else if n > 0 {
		//nolint:goconst //too specific
		return database.NewValidationError("this server is currently being used " +
			"in one or more running transfers and thus cannot be deleted, cancel " +
			"these transfers or wait for them to finish")
	}

	return nil
}

// GetCryptos fetch in the database then return the associated Cryptos if they exist.
func (l *LocalAgent) GetCryptos(db database.ReadAccess) ([]*Crypto, error) {
	return getCryptos(db, l)
}

func (l *LocalAgent) SetCryptoOwner(c *Crypto)             { c.LocalAgentID = utils.NewNullInt64(l.ID) }
func (l *LocalAgent) GenCryptoSelectCond() (string, int64) { return "local_agent_id=?", l.ID }
func (l *LocalAgent) SetAccessTarget(a *RuleAccess)        { a.LocalAgentID = utils.NewNullInt64(l.ID) }
func (l *LocalAgent) GenAccessSelectCond() (string, int64) { return "local_agent_id=?", l.ID }

func (l *LocalAgent) GetAuthorizedRules(db database.ReadAccess) ([]*Rule, error) {
	return getAuthorizedRules(db, "local_agent_id", l.ID)
}

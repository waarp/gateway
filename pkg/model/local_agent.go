package model

import (
	"encoding/json"
	"fmt"
	"net"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

//nolint:gochecknoinits // init is used by design
func init() {
	database.AddTable(&LocalAgent{})
}

// LocalAgent represents a local server instance operated by the gateway itself.
// The struct contains the information needed by external agents to connect to
// the server.
type LocalAgent struct {
	// The agent's database ID.
	ID uint64 `xorm:"pk autoincr <- 'id'"`

	// The agent's owner (i.e. the name of the gateway instance to which the
	// agent belongs to).
	Owner string `xorm:"unique(loc_ag) notnull 'owner'"`

	// The agent's display name.
	Name string `xorm:"unique(loc_ag) notnull 'name'"`

	// The protocol used by the agent.
	Protocol string `xorm:"notnull 'protocol'"`

	// Whether the server is enabled at startup or not.
	Enabled bool `xorm:"NOTNULL BOOL DEFAULT(true) 'enabled'"`

	// The root directory of the agent.
	RootDir string `xorm:"notnull 'root_dir'"`

	// The server's directory for received files.
	ReceiveDir string `xorm:"notnull 'receive_dir'"`

	// The server's directory for files to be sent.
	SendDir string `xorm:"notnull 'send_dir'"`

	// The server's temporary directory for partially received files.
	TmpReceiveDir string `xorm:"notnull 'tmp_receive_dir'"`

	// The agent's configuration in raw JSON format.
	ProtoConfig json.RawMessage `xorm:"notnull 'proto_config'"`

	// The agent's address (including the port)
	Address string `xorm:"notnull 'address'"`
}

// TableName returns the local agents table name.
func (*LocalAgent) TableName() string {
	return TableLocAgents
}

// Appellation returns the name of 1 element of the local agents table.
func (*LocalAgent) Appellation() string {
	return "server"
}

// GetID returns the agent's ID.
func (l *LocalAgent) GetID() uint64 {
	return l.ID
}

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
		l.RootDir = utils.ToOSPath(l.RootDir)

		if isEmpty(l.ReceiveDir) {
			l.ReceiveDir = "in"
		} else {
			l.ReceiveDir = utils.ToOSPath(l.ReceiveDir)
		}

		if isEmpty(l.SendDir) {
			l.SendDir = "out"
		} else {
			l.SendDir = utils.ToOSPath(l.SendDir)
		}

		if isEmpty(l.TmpReceiveDir) {
			l.TmpReceiveDir = "tmp"
		} else {
			l.TmpReceiveDir = utils.ToOSPath(l.TmpReceiveDir)
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

	if service.IsReservedServiceName(l.Name) {
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

	n, err := db.Count(l).Where("id<>? AND owner=? AND name=?", l.ID, l.Owner, l.Name).Run()
	if err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError(
			"a local agent with the same name '%s' already exist", l.Name)
	}

	return nil
}

// BeforeDelete is called before deleting the account from the database. Its
// role is to delete all the certificates tied to the account.
//
//nolint:dupl // to many differences
func (l *LocalAgent) BeforeDelete(db database.Access) database.Error {
	n, err := db.Count(&Transfer{}).Where("is_server=? AND agent_id=?", true, l.ID).Run()
	if err != nil {
		return err
	}

	if n > 0 {
		return database.NewValidationError("this server is currently being used in " +
			"one or more running transfers and thus cannot be deleted, cancel " +
			"these transfers or wait for them to finish")
	}

	certQuery := db.DeleteAll(&Crypto{}).Where(
		"(owner_type=? AND owner_id=?) OR (owner_type=? AND owner_id IN "+
			"(SELECT id FROM "+TableLocAccounts+" WHERE local_agent_id=?))",
		TableLocAgents, l.ID, TableLocAccounts, l.ID)
	if err := certQuery.Run(); err != nil {
		return err
	}

	accessQuery := db.DeleteAll(&RuleAccess{}).Where(
		"(object_type=? AND object_id=?) OR (object_type=? AND object_id IN "+
			"(SELECT id FROM "+TableLocAccounts+" WHERE local_agent_id=?))",
		TableLocAgents, l.ID, TableLocAccounts, l.ID)
	if err := accessQuery.Run(); err != nil {
		return err
	}

	accountQuery := db.DeleteAll(&LocalAccount{}).Where("local_agent_id=?", l.ID)

	return accountQuery.Run()
}

// GetCryptos fetch in the database then return the associated Cryptos if they exist.
func (l *LocalAgent) GetCryptos(db database.ReadAccess) ([]Crypto, database.Error) {
	return GetCryptos(db, l)
}

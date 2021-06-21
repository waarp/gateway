package model

import (
	"encoding/json"
	"net"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

func init() {
	database.Tables = append(database.Tables, &LocalAgent{})
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

	// The root directory of the agent.
	Root string `xorm:"notnull 'root'"`

	// The server's directory for received files.
	LocalInDir string `xorm:"notnull 'server_local_in_dir'"`

	// The server's directory for files to be sent.
	LocalOutDir string `xorm:"notnull 'server_local_out_dir'"`

	// The server's temporary directory for partially received files.
	LocalTmpDir string `xorm:"notnull 'server_local_tmp_dir'"`

	// The agent's configuration in raw JSON format.
	ProtoConfig json.RawMessage `xorm:"notnull 'proto_config'"`

	// The agent's address (including the port)
	Address string `xorm:"notnull 'address'"`
}

// TableName returns the local agents table name.
func (*LocalAgent) TableName() string {
	return "local_agents"
}

// Appellation returns the name of 1 element of the local agents table.
func (*LocalAgent) Appellation() string {
	return "server"
}

// GetID returns the agent's ID.
func (l *LocalAgent) GetID() uint64 {
	return l.ID
}

func (l *LocalAgent) validateProtoConfig() error {
	conf, err := config.GetProtoConfig(l.Protocol, l.ProtoConfig)
	if err != nil {
		return err
	}
	if err := conf.ValidServer(); err != nil {
		return err
	}
	l.ProtoConfig, err = json.Marshal(conf)
	return err
}

func (l *LocalAgent) makePaths() {
	isEmpty := func(path string) bool {
		return path == "." || path == ""
	}

	if !isEmpty(l.Root) {
		l.Root = utils.ToOSPath(l.Root)

		if isEmpty(l.LocalInDir) {
			l.LocalInDir = "in"
		} else {
			l.LocalInDir = utils.ToOSPath(l.LocalInDir)
		}
		if isEmpty(l.LocalOutDir) {
			l.LocalOutDir = "out"
		} else {
			l.LocalOutDir = utils.ToOSPath(l.LocalOutDir)
		}
		if isEmpty(l.LocalTmpDir) {
			l.LocalTmpDir = "tmp"
		} else {
			l.LocalTmpDir = utils.ToOSPath(l.LocalTmpDir)
		}
	}
}

// BeforeWrite is called before inserting a new `LocalAgent` entry in the
// database. It checks whether the new entry is valid or not.
func (l *LocalAgent) BeforeWrite(db database.ReadAccess) database.Error {
	l.Owner = database.Owner
	l.makePaths()

	if l.Name == "" {
		return database.NewValidationError("the agent's name cannot be empty")
	}
	if _, ok := service.CoreServiceNames[l.Name]; ok {
		return database.NewValidationError("%s is reserved server name", l.Name)
	}

	if l.Address == "" {
		return database.NewValidationError("the server's address cannot be empty")
	}
	if _, _, err := net.SplitHostPort(l.Address); err != nil {
		return database.NewValidationError("'%s' is not a valid server address", l.Address)
	}

	if l.ProtoConfig == nil {
		return database.NewValidationError("the agent's configuration cannot be empty")
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
//nolint:dupl
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
		"(owner_type='local_agents' AND owner_id=?) OR "+
			"(owner_type='local_accounts' AND owner_id IN "+
			"(SELECT id FROM local_accounts WHERE local_agent_id=?))",
		l.ID, l.ID)
	if err := certQuery.Run(); err != nil {
		return err
	}

	accessQuery := db.DeleteAll(&RuleAccess{}).Where(
		"(object_type='local_agents' AND object_id=?) OR "+
			"(object_type='local_accounts' AND object_id IN "+
			"(SELECT id FROM local_accounts WHERE local_agent_id=?))",
		l.ID, l.ID)
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

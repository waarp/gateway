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

	// The root directory of the agent.
	Root string `xorm:"notnull 'root'"`

	// The agent's directory for received files.
	InDir string `xorm:"notnull 'in_dir'"`

	// The agent's directory for files to be sent.
	OutDir string `xorm:"notnull 'out_dir'"`

	// The working directory of the agent.
	WorkDir string `xorm:"notnull 'work_dir'"`

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

func (l *LocalAgent) validateProtoConfig() error {
	conf, err := config.GetProtoConfig(l.Protocol, l.ProtoConfig)
	if err != nil {
		return fmt.Errorf("cannot parse protocol config for server %q: %w", l.Name, err)
	}

	if err2 := conf.ValidServer(); err2 != nil {
		return fmt.Errorf("the protocol configuration for server %q is not valid: %w",
			l.Name, err2)
	}

	l.ProtoConfig, err = json.Marshal(conf)

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

	if !isEmpty(l.Root) {
		if isEmpty(l.InDir) {
			l.InDir = "in"
		}

		if isEmpty(l.OutDir) {
			l.OutDir = "out"
		}

		if isEmpty(l.WorkDir) {
			l.WorkDir = "work"
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
	if err := accountQuery.Run(); err != nil {
		return err
	}

	return nil
}

// GetCryptos fetch in the database then return the associated Cryptos if they exist.
func (l *LocalAgent) GetCryptos(db database.ReadAccess) ([]Crypto, database.Error) {
	return GetCryptos(db, l)
}

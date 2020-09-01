package model

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
)

func init() {
	database.Tables = append(database.Tables, &LocalAgent{})
}

// ServerPaths regroups the different directories of a local server.
type ServerPaths struct {
	// The root directory of the agent.
	Root string `json:"root"`
	// The agent's directory for received files.
	InDir string `json:"inDir"`
	// The agent's directory for files to be sent.
	OutDir string `json:"outDir"`
	// The working directory of the agent.
	WorkDir string `json:"workDir"`
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

	// The agent's various directories
	Paths *ServerPaths `xorm:"extends 'paths'"`

	// The agent's configuration in raw JSON format.
	ProtoConfig []byte `xorm:"notnull 'proto_config'"`
}

// TableName returns the local_agent table name.
func (l *LocalAgent) TableName() string {
	return "local_agents"
}

// Id returns the agent's ID.
func (l *LocalAgent) Id() uint64 {
	return l.ID
}

func (l *LocalAgent) validateProtoConfig() error {
	conf, err := config.GetProtoConfig(l.Protocol, l.ProtoConfig)
	if err != nil {
		return err
	}
	return conf.ValidServer()
}

func (l *LocalAgent) makePaths(isInsert bool) {
	isEmpty := func(path string) bool {
		return path == "." || path == ""
	}

	if l.Paths == nil {
		if !isInsert {
			return
		}
		l.Paths = &ServerPaths{}
	}

	if !isEmpty(l.Paths.Root) {
		if isEmpty(l.Paths.InDir) {
			l.Paths.InDir = "in"
		}
		if isEmpty(l.Paths.OutDir) {
			l.Paths.OutDir = "out"
		}
		if isEmpty(l.Paths.WorkDir) {
			l.Paths.WorkDir = "work"
		}
	}
}

// BeforeInsert is called before inserting a new `LocalAgent` entry in the
// database. It checks whether the new entry is valid or not.
func (l *LocalAgent) Validate(db database.Accessor) error {
	l.Owner = database.Owner
	l.makePaths(true)

	if l.ID != 0 {
		return database.InvalidError("the agent's ID cannot be entered manually")
	}
	if l.Name == "" {
		return database.InvalidError("the agent's name cannot be empty")
	}
	if l.ProtoConfig == nil {
		return database.InvalidError("the agent's configuration cannot be empty")
	}
	if err := l.validateProtoConfig(); err != nil {
		return database.InvalidError(err.Error())
	}

	if res, err := db.Query("SELECT id FROM local_agents WHERE owner=? AND name=?",
		l.Owner, l.Name); err != nil {
		return err
	} else if len(res) > 0 {
		return database.InvalidError("a local agent with the same name '%s' "+
			"already exist", l.Name)
	}

	return nil
}

// BeforeDelete is called before deleting the account from the database. Its
// role is to delete all the certificates tied to the account.
func (l *LocalAgent) BeforeDelete(db database.Accessor) error {
	trans, err := db.Query("SELECT id FROM transfers WHERE is_server=? AND agent_id=?", true, l.ID)
	if err != nil {
		return err
	}
	if len(trans) > 0 {
		return database.InvalidError("this server is currently being used in a " +
			"running transfer and cannot be deleted, cancel the transfer or wait " +
			"for it to finish")
	}

	certQuery := "DELETE FROM certificates WHERE " +
		" (owner_type='local_agents' AND owner_id=?) " +
		"OR" +
		" (owner_type='local_accounts' AND owner_id IN " +
		"  (SELECT id FROM local_accounts WHERE local_agent_id=?))"
	if err := db.Execute(certQuery, l.ID, l.ID); err != nil {
		return err
	}

	accessQuery := "DELETE FROM rule_access WHERE " +
		" (object_type='local_agents' AND object_id=?) " +
		"OR" +
		" (object_type='local_accounts' AND object_id IN " +
		"  (SELECT id FROM local_accounts WHERE local_agent_id=?))"
	if err := db.Execute(accessQuery, l.ID, l.ID); err != nil {
		return err
	}

	accountQuery := "DELETE FROM local_accounts WHERE local_agent_id=?"
	if err := db.Execute(accountQuery, l.ID); err != nil {
		return err
	}

	return nil
}

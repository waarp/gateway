package model

import (
	"encoding/json"
	"net"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"github.com/go-xorm/builder"
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

// TableName returns the local_agent table name.
func (l *LocalAgent) TableName() string {
	return "local_agents"
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

// Validate is called before inserting a new `LocalAgent` entry in the
// database. It checks whether the new entry is valid or not.
func (l *LocalAgent) Validate(db database.Accessor) error {
	l.Owner = database.Owner
	l.makePaths()

	if l.Name == "" {
		return database.InvalidError("the agent's name cannot be empty")
	}

	if l.Address == "" {
		return database.InvalidError("the server's address cannot be empty")
	}
	if _, _, err := net.SplitHostPort(l.Address); err != nil {
		return database.InvalidError("'%s' is not a valid server address", l.Address)
	}

	if l.ProtoConfig == nil {
		return database.InvalidError("the agent's configuration cannot be empty")
	}
	if err := l.validateProtoConfig(); err != nil {
		return database.InvalidError(err.Error())
	}

	if res, err := db.Query("SELECT id FROM local_agents WHERE id<>? AND owner=? AND name=?",
		l.ID, l.Owner, l.Name); err != nil {
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

// GetCerts fetch in the database then return the associated Certificates if they exist
func (l *LocalAgent) GetCerts(db database.Accessor) ([]Cert, error) {
	filters := &database.Filters{
		Conditions: builder.And(builder.Eq{"owner_type": "local_agents"},
			builder.Eq{"owner_id": l.ID}),
	}

	// TODO filter only valid certificates
	var results []Cert
	if err := db.Select(&results, filters); err != nil {
		return nil, err
	}
	return results, nil
}

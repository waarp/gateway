package model

import (
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

//nolint:gochecknoinits // init is used by design
func init() {
	database.AddTable(&Rule{})
}

// Rule represents a transfer rule.
type Rule struct {

	// The Rule's ID
	ID uint64 `xorm:"pk autoincr <- 'id'"`

	// The rule's name
	Name string `xorm:"unique(dir) notnull 'name'"`

	// The rule's comment
	Comment string `xorm:"notnull 'comment'"`

	// The rule's direction (pull/push)
	IsSend bool `xorm:"unique(dir) unique(path) notnull 'send'"`

	// The path used to differentiate the rule when the protocol does not allow it.
	// This path is always an absolute path.
	Path string `xorm:"unique(path) notnull 'path'"`

	// The local directory for all file transfers using this rule.
	LocalDir string `xorm:"notnull 'local_dir'"`

	// The remote directory for all file transfers using this rule.
	RemoteDir string `xorm:"notnull 'remote_dir'"`

	// The temporary directory for running incoming transfer files.
	TmpLocalRcvDir string `xorm:"notnull 'tmp_local_receive_dir'"`
}

// TableName returns the remote accounts table name.
func (*Rule) TableName() string {
	return TableRules
}

// Appellation returns the name of 1 element of the rules table.
func (*Rule) Appellation() string {
	return "rule"
}

// GetID returns the rule's ID.
func (r *Rule) GetID() uint64 {
	return r.ID
}

func (r *Rule) normalizePaths() database.Error {
	if r.Path == "" {
		r.Path = "/" + r.Name
	} else {
		r.Path = utils.ToStandardPath(r.Path)
		if !path.IsAbs(r.Path) {
			r.Path = "/" + r.Path
		}
	}

	if r.LocalDir != "" {
		r.LocalDir = utils.ToOSPath(r.LocalDir)
	}

	if r.RemoteDir != "" {
		r.RemoteDir = utils.ToStandardPath(r.RemoteDir)
	}

	if r.TmpLocalRcvDir != "" {
		r.TmpLocalRcvDir = utils.ToOSPath(r.TmpLocalRcvDir)
	}

	return nil
}

// BeforeWrite is called before writing the `Rule` entry in the database. It
// checks whether the new entry is valid or not.
func (r *Rule) BeforeWrite(db database.ReadAccess) database.Error {
	if r.Name == "" {
		return database.NewValidationError("the rule's name cannot be empty")
	}

	if err := r.normalizePaths(); err != nil {
		return err
	}

	n, err := db.Count(r).Where("id<>? AND name=? AND send=?", r.ID,
		r.Name, r.IsSend).Run()
	if err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError("a %s rule named '%s' already exist",
			r.Direction(), r.Name)
	}

	n, err = db.Count(r).Where("id<>? AND path=? AND send=?", r.ID, r.Path, r.IsSend).Run()
	if err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError("a rule with path: %s already exist", r.Path)
	}

	return nil
}

// Direction returns the direction (send or receive) of the rule as a string.
func (r *Rule) Direction() string {
	if r.IsSend {
		return "send"
	}

	return "receive"
}

// BeforeDelete is called before deleting the rule from the database. Its
// role is to delete all the RuleAccess entries attached to this rule.
func (r *Rule) BeforeDelete(db database.Access) database.Error {
	n, err := db.Count(&Transfer{}).Where("rule_id=?", r.ID).Run()
	if err != nil {
		return err
	}

	if n > 0 {
		return database.NewValidationError("this rule is currently being used in a " +
			"running transfer and cannot be deleted, cancel the transfer or wait " +
			"for it to finish")
	}

	err = db.DeleteAll(&RuleAccess{}).Where("rule_id=?", r.ID).Run()
	if err != nil {
		return err
	}

	err = db.DeleteAll(&Task{}).Where("rule_id=?", r.ID).Run()
	if err != nil {
		return err
	}

	return nil
}

// IsAuthorized returns whether the given target is authorized to use the rule
// for transfers. It will return true either if the rule has no restrictions, or
// if the target has been given access to the rule.
//
// Valid target types are: LocalAgent, RemoteAgent, LocalAccount & RemoteAccount.
func (r *Rule) IsAuthorized(db database.Access, target database.IterateBean) (bool, database.Error) {
	var perms RuleAccess

	permCount, err := db.Count(&perms).Where("rule_id=?", r.ID).Run()
	if err != nil {
		return false, err
	}

	if permCount == 0 {
		return true, nil
	}

	var query *database.CountQuery
	switch object := target.(type) {
	case *LocalAgent:
		query = db.Count(&perms).Where("rule_id=? AND (object_type=? AND object_id=?)",
			r.ID, object.TableName(), object.ID)
	case *RemoteAgent:
		query = db.Count(&perms).Where("rule_id=? AND (object_type=? AND object_id=?)",
			r.ID, object.TableName(), object.ID)
	case *LocalAccount:
		query = db.Count(&perms).Where("rule_id=? AND ((object_type=? AND object_id=?) "+
			"OR (object_type=? AND object_id=?))", r.ID, object.TableName(), object.ID,
			"local_agents", object.LocalAgentID)
	case *RemoteAccount:
		query = db.Count(&perms).Where("rule_id=? AND ((object_type=? AND object_id=?) "+
			"OR (object_type=? AND object_id=?))", r.ID, object.TableName(), object.ID,
			"local_agents", object.RemoteAgentID)
	default:
		return false, database.NewValidationError("%T is not a valid target model for RuleAccess", target)
	}

	permCount, err = query.Run()
	if err != nil {
		return false, err
	}

	return permCount != 0, nil
}

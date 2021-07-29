package model

import (
	"path"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

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

	// The directory for all incoming files.
	InPath string `xorm:"notnull 'in_path'"`

	// The directory for all outgoing files.
	OutPath string `xorm:"notnull 'out_path'"`

	// The temp directory for all running transfer files.
	WorkPath string `xorm:"notnull 'work_path'"`
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

func (r *Rule) normalizePaths() {
	r.Path = path.Clean(r.Path)
	if r.Path == "/" || r.Path == "." {
		r.Path = r.Name
	} else {
		r.Path = utils.NormalizePath(r.Path)
		if path.IsAbs(r.Path) {
			r.Path = r.Path[1:]
		}
	}
	if r.InPath != "" {
		r.InPath = utils.NormalizePath(r.InPath)
	}
	if r.OutPath != "" {
		r.OutPath = utils.NormalizePath(r.OutPath)
	}
	if r.WorkPath != "" {
		r.WorkPath = utils.NormalizePath(r.WorkPath)
	}
}

func (r *Rule) checkAncestor(db database.ReadAccess, rulePath string) database.Error {
	if rulePath == "" || rulePath == "." {
		return nil
	}
	var rule Rule
	if err := db.Get(&rule, "path=?", rulePath).Run(); err != nil {
		if database.IsNotFound(err) {
			return r.checkAncestor(db, path.Dir(rulePath))
		}
		return err
	}
	return database.NewValidationError("the rule's path cannot be the descendant of "+
		"another rule's path (the path '%s' is already used by rule '%s')", rulePath, rule.Name)
}

func (r *Rule) checkPath(db database.ReadAccess) database.Error {
	if n, err := db.Count(r).Where("id<>? AND path=? AND send=?", r.ID, r.Path,
		r.IsSend).Run(); err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError("a rule with path: %s already exist", r.Path)
	}

	// check descendents
	if n, err := db.Count(r).Where("path LIKE ?", r.Path+"/%").Run(); err != nil {
		return err
	} else if n != 0 {
		return database.NewValidationError("the rule's path cannot be the ancestor " +
			"of another rule's path")
	}

	return r.checkAncestor(db, path.Dir(r.Path))
}

// BeforeWrite is called before writing the `Rule` entry in the database. It
// checks whether the new entry is valid or not.
func (r *Rule) BeforeWrite(db database.ReadAccess) database.Error {
	if r.Name == "" {
		return database.NewValidationError("the rule's name cannot be empty")
	}

	n, err := db.Count(r).Where("id<>? AND name=? AND send=?", r.ID,
		r.Name, r.IsSend).Run()
	if err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError("a %s rule named '%s' already exist",
			r.Direction(), r.Name)
	}

	r.normalizePaths()
	return r.checkPath(db)
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
	if n, err := db.Count(&perms).Where("rule_id=?", r.ID).Run(); err != nil {
		return false, err
	} else if n == 0 {
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

	n, err := query.Run()
	if err != nil {
		return false, err
	}
	return n != 0, nil
}

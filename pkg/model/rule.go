package model

import (
	"path"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

func init() {
	database.Tables = append(database.Tables, &Rule{})
}

// Rule represents a transfer rule.
type Rule struct {

	// The Rule's ID
	ID uint64 `xorm:"pk autoincr 'id'"`

	// The rule's name
	Name string `xorm:"unique(send) notnull 'name'"`

	// The rule's comment
	Comment string `xorm:"notnull 'comment'"`

	// The rule's direction (pull/push)
	IsSend bool `xorm:"unique(send) notnull 'send'"`

	// The path used to differentiate the rule when the protocol does not allow it.
	// This path is always an absolute path.
	Path string `xorm:"unique notnull 'path'"`

	// The directory for all incoming files.
	InPath string `xorm:"unique notnull 'in_path'"`

	// The directory for all outgoing files.
	OutPath string `xorm:"unique notnull 'out_path'"`

	// The temp directory for all running transfer files.
	WorkPath string `xorm:"unique notnull 'work_path'"`
}

// TableName returns the remote accounts table name.
func (*Rule) TableName() string {
	return "rules"
}

// Id returns the rule's ID.
func (r *Rule) Id() uint64 {
	return r.ID
}

func (r *Rule) normalizePaths() error {
	if r.Path == "" {
		r.Path = r.Name
	} else {
		r.Path = utils.NormalizePath(r.Path)
		if !path.IsAbs(r.Path) {
			r.Path = "/" + r.Path
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

	return nil
}

// Validate is called before inserting a new `Rule` entry in the
// database. It checks whether the new entry is valid or not.
func (r *Rule) Validate(db database.Accessor) error {
	if r.Path == "" {
		return database.InvalidError("the rule's path cannot be empty")
	}
	if r.ID != 0 {
		return database.InvalidError("the rule's ID cannot be entered manually")
	}
	if r.Name == "" {
		return database.InvalidError("the rule's name cannot be empty")
	}

	if err := r.normalizePaths(); err != nil {
		return err
	}

	if res, err := db.Query("SELECT id FROM rules WHERE name=? AND send=?", r.Name, r.IsSend); err != nil {
		return err
	} else if len(res) > 0 {
		return database.InvalidError("a rule named '%s' with send = %t already exist", r.Name, r.IsSend)
	}

	if res, err := db.Query("SELECT id FROM rules WHERE name=? AND path=?", r.Name, r.Path); err != nil {
		return err
	} else if len(res) > 0 {
		return database.InvalidError("a rule named '%s' with path: %s already exist", r.Name, r.Path)
	}

	return nil
}

// BeforeDelete is called before deleting the rule from the database. Its
// role is to delete all the RuleAccess entries attached to this rule.
func (r *Rule) BeforeDelete(db database.Accessor) error {
	trans, err := db.Query("SELECT id FROM transfers WHERE rule_id=?", r.ID)
	if err != nil {
		return err
	}
	if len(trans) > 0 {
		return database.InvalidError("this rule is currently being used in a " +
			"running transfer and cannot be deleted, cancel the transfer or wait " +
			"for it to finish")
	}

	if err := db.Execute("DELETE FROM rule_access WHERE rule_id=?", r.ID); err != nil {
		return err
	}
	return db.Execute("DELETE FROM tasks WHERE rule_id=?", r.ID)
}

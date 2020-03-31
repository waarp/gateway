package model

import (
	"path/filepath"

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

// BeforeInsert is called before inserting the rule in the database. Its
// role is to set the path to the default value if non was entered.
func (r *Rule) BeforeInsert(database.Accessor) error {
	if r.Path == "" {
		r.Path = "/" + r.Name
	}

	if !filepath.IsAbs(r.Path) {
		r.Path = "/" + r.Path
	}
	if r.InPath != "" {
		r.InPath = utils.CleanSlash(r.InPath)
		if !filepath.IsAbs(r.InPath) {
			r.InPath = "/" + r.InPath
		}
	}
	if r.OutPath != "" {
		r.OutPath = utils.CleanSlash(r.OutPath)
		if !filepath.IsAbs(r.OutPath) {
			r.OutPath = "/" + r.OutPath
		}
	}
	if r.WorkPath != "" {
		r.WorkPath = utils.CleanSlash(r.WorkPath)
		if !filepath.IsAbs(r.WorkPath) {
			r.WorkPath = "/" + r.WorkPath
		}
	}

	return nil
}

// BeforeDelete is called before deleting the rule from the database. Its
// role is to delete all the RuleAccess entries attached to this rule.
func (r *Rule) BeforeDelete(acc database.Accessor) error {
	return acc.Execute("DELETE FROM rule_access WHERE rule_id=?", r.ID)
}

// ValidateInsert is called before inserting a new `Rule` entry in the
// database. It checks whether the new entry is valid or not.
func (r *Rule) ValidateInsert(acc database.Accessor) error {
	if r.ID != 0 {
		return database.InvalidError("The rule's ID cannot be entered manually")
	}
	if r.Name == "" {
		return database.InvalidError("The rule's name cannot be empty")
	}

	if res, err := acc.Query("SELECT id FROM rules WHERE name=? and send=?", r.Name, r.IsSend); err != nil {
		return err
	} else if len(res) > 0 {
		return database.InvalidError("A rule named '%s' with send = %t already exist", r.Name, r.IsSend)
	}

	if res, err := acc.Query("SELECT id FROM rules WHERE name=? and path=?", r.Name, r.Path); err != nil {
		return err
	} else if len(res) > 0 {
		return database.InvalidError("A rule named '%s' with path: %s already exist", r.Name, r.Path)
	}

	if r.Path == "" {
		return database.InvalidError("The rule's path cannot be empty")
	}

	return nil
}

// ValidateUpdate is called before updating an existing `Rule` entry from
// the database. It checks whether the updated entry is valid or not.
func (r *Rule) ValidateUpdate(acc database.Accessor, id uint64) error {
	if r.ID != 0 {
		return database.InvalidError("The rule's ID cannot be entered manually")
	}

	old := &Rule{ID: id}
	if err := acc.Get(old); err != nil {
		return err
	}
	name := old.Name
	r.IsSend = old.IsSend

	if r.Name != "" && r.Name != old.Name {
		name = r.Name
		if res, err := acc.Query("SELECT id FROM rules WHERE name=? and send=?",
			r.Name, old.IsSend); err != nil {
			return err
		} else if len(res) > 0 {
			return database.InvalidError("A rule named '%s' with send = %t already exist",
				r.Name, old.IsSend)
		}
	}

	if r.Path != "" {
		if res, err := acc.Query("SELECT id FROM rules WHERE name=? and path=?", name, r.Path); err != nil {
			return err
		} else if len(res) > 0 {
			return database.InvalidError("A rule named '%s' with path: %s already exist", name, r.Path)
		}
	}
	return nil
}

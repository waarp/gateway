package model

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

func init() {
	database.Tables = append(database.Tables, &Rule{})
}

// Rule represents a transfer rule.
type Rule struct {

	// The Rule's ID
	ID uint64 `xorm:"pk autoincr 'id'"`

	// The rule's name
	Name string `xorm:"unique(send) unique(path) notnull 'name'"`

	// The rule's comment
	Comment string `xorm:"notnull 'comment'"`

	// The rule's direction (pull/push)
	IsSend bool `xorm:"unique(send) notnull 'send'"`

	// The rule's directory
	Path string `xorm:"unique(path) notnull 'path'"`
}

// TableName returns the remote accounts table name.
func (*Rule) TableName() string {
	return "rules"
}

// BeforeInsert is called before inserting the rule in the database. Its
// role is to set the IN and OUT path to the default value if non was
// entered.
func (r *Rule) BeforeInsert(database.Accessor) error {
	if r.Path == "" {
		if r.IsSend {
			r.Path = fmt.Sprintf("/%s/out", r.Name)
		} else {
			r.Path = fmt.Sprintf("/%s/in", r.Name)
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
	if res, err := acc.Query("SELECT id FROM rules WHERE name=? and send=? and id<>?", r.Name, r.IsSend, id); err != nil {
		return err
	} else if len(res) > 0 {
		return database.InvalidError("A rule Send: %t named '%s' already exist", r.IsSend, r.Name)
	}
	return nil
}

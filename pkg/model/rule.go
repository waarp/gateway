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
	ID uint64 `xorm:"pk autoincr 'id'" json:"id"`

	// The rule's name
	Name string `xorm:"unique notnull 'name'"`

	// The rule's comment
	Comment string `xorm:"notnull 'comment'" json:"comment"`

	// The rule's direction (pull/push)
	Send bool `xorm:"notnull 'send'" json:"send"`

	// The rule's directory
	Path string `xorm:"notnull 'path'" json:"path"`
}

// TableName returns the remote accounts table name.
func (*Rule) TableName() string {
	return "rules"
}

// Init initialises the database with 2 basic 'push' and 'pull' rules.
func (*Rule) Init(acc database.Accessor) error {
	push := Rule{Name: "push", Send: true}
	if err := acc.Create(&push); err != nil {
		return err
	}

	pull := Rule{Name: "pull", Send: false}
	if err := acc.Create(&pull); err != nil {
		return err
	}

	return nil
}

// BeforeInsert is called before inserting the rule in the database. Its
// role is to set the IN and OUT path to the default value if non was
// entered.
func (r *Rule) BeforeInsert(acc database.Accessor) error {
	if r.Path == "" {
		if r.Send {
			r.Path = fmt.Sprintf("/%s/out", r.Name)
		} else {
			r.Path = fmt.Sprintf("/%s/in", r.Name)
		}
	}
	return nil
}

// ValidateInsert is called before inserting a new `Rule` entry in the
// database. It checks whether the new entry is valid or not.
func (r *Rule) ValidateInsert(acc database.Accessor) error {
	if r.Name == "" {
		return database.InvalidError("The rule's name cannot be empty")
	}

	if res, err := acc.Query("SELECT id FROM rules WHERE name=? and send=?", r.Name, r.Send); err != nil {
		return err
	} else if len(res) > 0 {
		return database.InvalidError("A rule named '%s' with send: %t already exist", r.Name, r.Send)
	}

	if r.Path == "" {
		return database.InvalidError("The rule's path cannot be empty")
	}

	return nil
}

// ValidateUpdate is called before updating an existing `Rule` entry from
// the database. It checks whether the updated entry is valid or not.
func (r *Rule) ValidateUpdate(acc database.Accessor, id uint64) error {

	if res, err := acc.Query("SELECT id FROM rules WHERE name=? AND id<>?", r.Name, id); err != nil {
		return err
	} else if len(res) > 0 {
		return database.InvalidError("A rule named '%s' already exist", r.Name)
	}
	return nil
}

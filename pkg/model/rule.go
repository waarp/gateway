package model

import "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"

func init() {
	database.Tables = append(database.Tables, &Rule{})
}

// Rule represents a transfer rule.
type Rule struct {

	// The Rule's ID
	ID uint64 `xorm:"pk autoincr 'id'" json:"id"`

	// The rule's name
	Name string `xorm:"unique notnull 'name'"`

	// The rule's direction (pull/push)
	IsGet bool `xorm:"notnull 'is_get'"`
}

// TableName returns the remote accounts table name.
func (*Rule) TableName() string {
	return "rules"
}

// Init initialises the database with 2 basic 'push' and 'pull' rules.
func (*Rule) Init(acc database.Accessor) error {
	push := Rule{Name: "push", IsGet: false}
	if err := acc.Create(&push); err != nil {
		return err
	}

	pull := Rule{Name: "pull", IsGet: false}
	if err := acc.Create(&pull); err != nil {
		return err
	}

	return nil
}

// ValidateInsert is called before inserting a new `Rule` entry in the
// database. It checks whether the new entry is valid or not.
func (r *Rule) ValidateInsert(acc database.Accessor) error {
	if r.Name == "" {
		return database.InvalidError("The rule's name cannot be empty")
	}

	if res, err := acc.Query("SELECT id FROM rules WHERE name=?", r.Name); err != nil {
		return err
	} else if len(res) > 0 {
		return database.InvalidError("A rule named '%s' already exist", r.Name)
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

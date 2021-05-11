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
	if r.Path == "" {
		r.Path = "/" + r.Name
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
}

// BeforeWrite is called before writing the `Rule` entry in the database. It
// checks whether the new entry is valid or not.
func (r *Rule) BeforeWrite(db database.ReadAccess) database.Error {
	if r.Name == "" {
		return database.NewValidationError("the rule's name cannot be empty")
	}

	r.normalizePaths()

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

package model

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

func init() {
	database.Tables = append(database.Tables, &Task{})
}

// Chain represents the valid chains for a task entry
type Chain string

const (
	// ChainPre is the chain for pre transfer tasks
	ChainPre Chain = "PRE"

	// ChainPost is the chain for post transfer tasks
	ChainPost Chain = "POST"

	// ChainError is the chain for error transfer tasks
	ChainError Chain = "ERROR"
)

// Task represents one record of the 'tasks' table.
type Task struct {
	RuleID uint64 `xorm:"notnull 'rule_id'" json:"-"`
	Chain  Chain  `xorm:"notnull 'chain'" json:"-"`
	Rank   uint32 `xorm:"notnull 'rank'" json:"-"`
	Type   string `xorm:"notnull 'type'" json:"type"`
	Args   []byte `xorm:"notnull 'args'" json:"args"`
}

// TableName returns the name of the tasks table.
func (*Task) TableName() string {
	return "tasks"
}

// ValidateInsert checks if the new `Task` entry is valid and cand be
// inserted in the database.
func (t *Task) ValidateInsert(acc database.Accessor) error {
	if res, err := acc.Query("SELECT id FROM rules WHERE id=?", t.RuleID); err != nil {
		return err
	} else if len(res) < 1 {
		return database.InvalidError("No rule found with ID %d", t.RuleID)
	}

	if !validateChain(t.Chain) {
		return database.InvalidError("%s is not a valid task chain", t.Chain)
	}

	if res, err := acc.Query("SELECT rule_id FROM tasks WHERE rule_id=? AND chain=? AND rank=?",
		t.RuleID, t.Chain, t.Rank); err != nil {
		return err
	} else if len(res) > 0 {
		return database.InvalidError("Rule %d already has a task in %s at %d", t.RuleID, t.Chain, t.Rank)
	}

	return nil
}

// ValidateUpdate is called before updating and existing `Task` entry from
// the database. It rejects all update.
func (t *Task) ValidateUpdate(acc database.Accessor) error {
	return database.InvalidError("Unallowed operation")
}

func validateChain(c Chain) bool {
	return c == ChainPre ||
		c == ChainPost ||
		c == ChainError
}

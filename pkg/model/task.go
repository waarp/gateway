package model

import (
	"encoding/json"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

// ValidTasks is a list of all the tasks known by the gateway
var ValidTasks = map[string]Validator{}

// Validator permits to validate the arguments for a given task
type Validator interface {
	Validate(map[string]string) error
}

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
	RuleID uint64          `xorm:"notnull 'rule_id'"`
	Chain  Chain           `xorm:"notnull 'chain'"`
	Rank   uint32          `xorm:"notnull 'rank'"`
	Type   string          `xorm:"notnull 'type'"`
	Args   json.RawMessage `xorm:"notnull 'args'"`
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
		return database.InvalidError("no rule found with ID %d", t.RuleID)
	}

	if err := validateTasks(t); err != nil {
		return err
	}

	if res, err := acc.Query("SELECT rule_id FROM tasks WHERE rule_id=? AND chain=? AND rank=?",
		t.RuleID, t.Chain, t.Rank); err != nil {
		return err
	} else if len(res) > 0 {
		return database.InvalidError("rule %d already has a task in %s at %d", t.RuleID, t.Chain, t.Rank)
	}

	return nil
}

// ValidateUpdate is called before updating and existing `Task` entry from
// the database. It rejects all update.
func (t *Task) ValidateUpdate(database.Accessor) error {
	return database.InvalidError("operation not allowed")
}

func validateChain(c Chain) bool {
	return c == ChainPre ||
		c == ChainPost ||
		c == ChainError
}

func validateTasks(t *Task) error {

	if !validateChain(t.Chain) {
		return database.InvalidError("%s is not a valid task chain", t.Chain)
	}

	args := map[string]string{}
	if err := json.Unmarshal(t.Args, &args); err != nil {
		return err
	}

	v, ok := ValidTasks[t.Type]
	if !ok {
		return database.InvalidError("%s is not a valid task Type", t.Type)
	}
	return v.Validate(args)
}

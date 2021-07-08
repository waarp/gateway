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
	return TableTasks
}

// Appellation returns the name of 1 element of the tasks table.
func (*Task) Appellation() string {
	return "task"
}

func (t *Task) validateTasks() database.Error {
	if t.Chain != ChainPre && t.Chain != ChainPost && t.Chain != ChainError {
		return database.NewValidationError("%s is not a valid task chain", t.Chain)
	}

	if len(t.Args) == 0 {
		t.Args = json.RawMessage(`{}`)
	}
	args := map[string]string{}
	if err := json.Unmarshal(t.Args, &args); err != nil {
		return database.NewValidationError("incorrect task format: %s", err)
	}

	v, ok := ValidTasks[t.Type]
	if !ok {
		return database.NewValidationError("%s is not a valid task Type", t.Type)
	}

	if err := v.Validate(args); err != nil {
		return database.NewValidationError("invalid task: %s", err)
	}

	return nil
}

// BeforeWrite checks if the new `Task` entry is valid and can be
// inserted in the database.
func (t *Task) BeforeWrite(db database.ReadAccess) database.Error {
	n, err := db.Count(&Rule{}).Where("id=?", t.RuleID).Run()
	if err != nil {
		return err
	} else if n < 1 {
		return database.NewValidationError("no rule found with ID %d", t.RuleID)
	}

	if err := t.validateTasks(); err != nil {
		return err
	}

	n, err = db.Count(t).Where("rule_id=? AND chain=? AND rank=?", t.RuleID,
		t.Chain, t.Rank).Run()
	if err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError("rule %d already has a task in %s at %d",
			t.RuleID, t.Chain, t.Rank)
	}

	return nil
}

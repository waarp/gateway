package model

import (
	"context"
	"encoding/json"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

// ValidTasks is a list of all the tasks known by the gateway.
//
//nolint:gochecknoglobals // global var is used by design
var ValidTasks = map[string]TaskRunner{}

// TaskValidator is an optional interface which can be implemented by task
// executors (alongside TaskRunner). This interface can be implemented if the
// task's arguments need to be checked before executing the task.
type TaskValidator interface {
	Validate(args map[string]string) error
}

// TaskRunner is the interface which represents a task. All tasks executors must
// implement this interface in order for the tasks.Runner to be able to execute
// them.
type TaskRunner interface {
	Run(context.Context, map[string]string, *database.DB, *TransferContext) (string, error)
}

// Chain represents the valid chains for a task entry.
type Chain string

const (
	ChainPre   Chain = "PRE"   // ChainPre is the chain for pre transfer tasks.
	ChainPost  Chain = "POST"  // ChainPost is the chain for post transfer tasks.
	ChainError Chain = "ERROR" // ChainError is the chain for error transfer tasks.
)

// Task represents one record of the 'tasks' table.
type Task struct {
	RuleID int64           `xorm:"rule_id"` // The ID of the rule this tasks belongs to.
	Chain  Chain           `xorm:"chain"`   // The chain this task belongs to (ChainPre, ChainPost or ChainError)
	Rank   int8            `xorm:"rank"`    // The task's index in the chain.
	Type   string          `xorm:"type"`    // The type of task.
	Args   json.RawMessage `xorm:"args"`    // The task's arguments as a raw JSON object.
}

func (*Task) TableName() string   { return TableTasks }
func (*Task) Appellation() string { return "task" }

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

	runner, ok := ValidTasks[t.Type]
	if !ok {
		return database.NewValidationError("%s is not a valid task Type", t.Type)
	}

	if validator, ok := runner.(TaskValidator); ok {
		if err := validator.Validate(args); err != nil {
			return database.NewValidationError("invalid task: %s", err)
		}
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

	if err2 := t.validateTasks(); err2 != nil {
		return err2
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

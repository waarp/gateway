package model

import (
	"context"
	"fmt"
	"strings"

	"code.waarp.fr/lib/log"

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

type TaskDBConverter interface {
	ToDB(args map[string]string) error
	FromDB(args map[string]string) error
}

type TaskValidatorDB interface {
	ValidateDB(db database.ReadAccess, params map[string]string) error
}

// TaskRunner is the interface which represents a task. All tasks executors must
// implement this interface in order for the tasks.Runner to be able to execute
// them.
type TaskRunner interface {
	Run(ctx context.Context, args map[string]string, db *database.DB,
		logger *log.Logger, transCtx *TransferContext) error
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
	RuleID int64             `xorm:"rule_id"` // The ID of the rule this tasks belongs to.
	Chain  Chain             `xorm:"chain"`   // The chain this task belongs to (ChainPre, ChainPost or ChainError)
	Rank   int8              `xorm:"rank"`    // The task's index in the chain.
	Type   string            `xorm:"type"`    // The type of task.
	Args   map[string]string `xorm:"args"`    // The task's arguments as a map.
}

func (*Task) TableName() string   { return TableTasks }
func (*Task) Appellation() string { return NameTask }

func (t *Task) validateTasks(db database.ReadAccess) error {
	if t.Chain != ChainPre && t.Chain != ChainPost && t.Chain != ChainError {
		return database.NewValidationError("%s is not a valid task chain", t.Chain)
	}

	if len(t.Args) == 0 {
		t.Args = map[string]string{}
	}

	runner, ok := ValidTasks[t.Type]
	if !ok {
		return database.NewValidationError("%s is not a valid task Type", t.Type)
	}

	if validatorDB, ok := runner.(TaskValidatorDB); ok {
		if err := validatorDB.ValidateDB(db, t.Args); err != nil {
			return database.NewValidationError("invalid task: %s", err)
		}
	} else if validator, ok := runner.(TaskValidator); ok {
		if err := validator.Validate(t.Args); err != nil {
			return database.NewValidationError("invalid task: %s", err)
		}
	}

	return nil
}

// BeforeWrite checks if the new `Task` entry is valid and can be
// inserted in the database.
func (t *Task) BeforeWrite(db database.Access) error {
	if n, err := db.Count(&Rule{}).Where("id=?", t.RuleID).Run(); err != nil {
		return fmt.Errorf("failed to check for parrent rule: %w", err)
	} else if n < 1 {
		return database.NewValidationError("no rule found with ID %d", t.RuleID)
	}

	if err := t.validateTasks(db); err != nil {
		return err
	}

	if n, err := db.Count(t).Where("rule_id=? AND chain=? AND rank=?", t.RuleID,
		t.Chain, t.Rank).Run(); err != nil {
		return fmt.Errorf("failed to check for duplicate tasks: %w", err)
	} else if n > 0 {
		return database.NewValidationError("rule %d already has a %s-task at rank %d",
			t.RuleID, strings.ToLower(string(t.Chain)), t.Rank)
	}

	return nil
}

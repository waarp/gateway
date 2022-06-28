// Package taskstest defines a dummy transfer task which can be used for test
// purposes.
package taskstest

import (
	"context"
	"errors"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	// TaskOK is a dummy task type which can be used during transfer tests.
	// The task always succeeds.
	TaskOK = "TASKOK"

	// TaskErr is a dummy task type which can be used during transfer tests.
	// The task always fails.
	TaskErr = "TASKERR"
)

//nolint:gochecknoinits //init is required here
func init() {
	model.ValidTasks[TaskOK] = &TestTask{}
	model.ValidTasks[TaskErr] = &TestTaskError{}
}

var ErrTaskFailed = errors.New("task failed")

// TestTask is a dummy task made for testing. It always succeeds.
type TestTask struct{}

// Run executes the dummy task, which will always succeed.
func (t *TestTask) Run(context.Context, map[string]string, *database.DB,
	*model.TransferContext,
) (string, error) {
	return "", nil
}

// TestTaskError is a dummy task made for testing. It always fails.
type TestTaskError struct{}

// Run executes the dummy task, which will always return an error.
func (t *TestTaskError) Run(context.Context, map[string]string, *database.DB,
	*model.TransferContext,
) (string, error) {
	return "task failed", ErrTaskFailed
}

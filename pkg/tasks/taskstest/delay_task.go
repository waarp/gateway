package taskstest

import (
	"context"
	"fmt"
	"time"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	// TaskDelay is a dummy task type which can be used during transfer tests.
	// The task wait the specified amount of time and then always succeeds.
	TaskDelay = "TASKDELAY"
)

//nolint:gochecknoinits //init is required here
func init() {
	model.ValidTasks[TaskDelay] = &TestTaskDelay{}
}

// TestTaskDelay is a dummy task made for testing. It waits a specified amount
// of time and succeeds.
type TestTaskDelay struct{}

//nolint:goerr113 //errors are too specific
func (*TestTaskDelay) getDelay(args map[string]string) (time.Duration, error) {
	d, hasDelay := args["delay"]
	if !hasDelay {
		return -1, fmt.Errorf(`missing "delay" argument for task %q`, TaskDelay)
	}

	delay, err := time.ParseDuration(d)
	if err != nil {
		return -1, fmt.Errorf(`invalid "delay" argument for task %q: %w`, TaskDelay, err)
	}

	return delay, nil
}

func (t *TestTaskDelay) Validate(args map[string]string) error {
	_, err := t.getDelay(args)

	return err
}

// Run executes the dummy task, which will always succeed.
func (t *TestTaskDelay) Run(_ context.Context, args map[string]string,
	_ *database.DB, _ *log.Logger, _ *model.TransferContext,
) error {
	delay, err := t.getDelay(args)
	if err != nil {
		return err
	}

	<-time.After(delay)

	return nil
}

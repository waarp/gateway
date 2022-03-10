// Package taskstest defines a dummy transfer task which can be used for test
// purposes.
package taskstest

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

const (
	// TaskOK is a a dummy task type which can be used during transfer tests.
	// The task always succeeds.
	TaskOK = "TASKOK"

	// TaskErr is a a dummy task type which can be used during transfer tests.
	// The task always fails.
	TaskErr = "TASKERR"
)

const taskTimeout = 10 * time.Second

// TaskChecker is a struct used in tests to check if a transfer's tasks have been
// executed properly.
type TaskChecker struct {
	Cond chan bool
	cPre, sPre,
	cPost, sPost,
	cErr, sErr uint32
	cDone, sDone chan struct{}
}

// InitTaskChecker initializes and returns a new TaskChecker.
func InitTaskChecker() *TaskChecker {
	cond := make(chan bool)
	defer close(cond)

	return &TaskChecker{
		Cond:  cond,
		cDone: make(chan struct{}),
		sDone: make(chan struct{}),
	}
}

// Retry resets the TaskChecker so that a the transfer can be retried. Be aware
// that the tasks counters are not reset, only the end transfer channels.
func (t *TaskChecker) Retry() {
	t.cDone = make(chan struct{})
	t.sDone = make(chan struct{})
}

// ClientPreTaskNB returns the number of pre-tasks executed by the client.
func (t *TaskChecker) ClientPreTaskNB() uint32 { return atomic.LoadUint32(&t.cPre) }

// ClientPostTaskNB returns the number of post-tasks executed by the client.
func (t *TaskChecker) ClientPostTaskNB() uint32 { return atomic.LoadUint32(&t.cPost) }

// ServerPreTaskNB returns the number of pre-tasks executed by the server.
func (t *TaskChecker) ServerPreTaskNB() uint32 { return atomic.LoadUint32(&t.sPre) }

// ServerPostTaskNB returns the number of post-tasks executed by the server.
func (t *TaskChecker) ServerPostTaskNB() uint32 { return atomic.LoadUint32(&t.sPost) }

// ClientErrTaskNB returns the number of error-tasks executed by the client.
func (t *TaskChecker) ClientErrTaskNB() uint32 { return atomic.LoadUint32(&t.cErr) }

// ServerErrTaskNB returns the number of error-tasks executed by the server.
func (t *TaskChecker) ServerErrTaskNB() uint32 { return atomic.LoadUint32(&t.sErr) }

// ClientDone signals the TaskChecker that the client has finished its side of the transfer.
func (t *TaskChecker) ClientDone() { close(t.cDone) }

// ServerDone signals the TaskChecker that the server has finished its side of the transfer.
func (t *TaskChecker) ServerDone() { close(t.sDone) }

// WaitClientPreTasks waits until the client has executed at least 1 pre-task.
func (t *TaskChecker) WaitClientPreTasks() { waitTask(t.ClientPreTaskNB) }

// WaitServerPreTasks waits until the server has executed at least 1 pre-task.
func (t *TaskChecker) WaitServerPreTasks() { waitTask(t.ServerPreTaskNB) }

// WaitClientPostTasks waits until the client has executed at least 1 post-task.
func (t *TaskChecker) WaitClientPostTasks() { waitTask(t.ClientPreTaskNB) }

// WaitServerPostTasks waits until the server has executed at least 1 post-task.
func (t *TaskChecker) WaitServerPostTasks() { waitTask(t.ServerPreTaskNB) }

// waitTask waits until the given function returns any number greater than 0.
func waitTask(f func() uint32) {
	timer := time.NewTimer(taskTimeout)

	for {
		select {
		case <-timer.C:
			panic("timeout waiting for task")
		default:
			if f() > 0 {
				return
			}
		}
	}
}

// WaitClientDone waits for the client to have finished its side of the transfer.
func (t *TaskChecker) WaitClientDone() {
	timer := time.NewTimer(taskTimeout)
	select {
	case <-timer.C:
		panic("timeout waiting for client transfer end")
	case <-t.cDone:
	}
}

// WaitServerDone waits for the server to have finished its side of the transfer.
func (t *TaskChecker) WaitServerDone() {
	timer := time.NewTimer(taskTimeout)
	select {
	case <-timer.C:
		panic("timeout waiting for server transfer end")
	case <-t.sDone:
	}
}

func (t *TaskChecker) execTask(ctx context.Context, c *model.TransferContext) error {
	select {
	case <-t.Cond:
	case <-ctx.Done():
		return fmt.Errorf("task failed: %w", ctx.Err())
	}

	if c.Transfer.IsServer {
		switch c.Transfer.Step {
		case types.StepPreTasks:
			atomic.AddUint32(&t.sPre, 1)
		case types.StepPostTasks:
			atomic.AddUint32(&t.sPost, 1)
		case types.StepErrorTasks:
			atomic.AddUint32(&t.sErr, 1)
		default:
			panic(fmt.Sprintf("invalid transfer state '%s' for tasks", c.Transfer.Step))
		}
	} else {
		switch c.Transfer.Step {
		case types.StepPreTasks:
			atomic.AddUint32(&t.cPre, 1)
		case types.StepPostTasks:
			atomic.AddUint32(&t.cPost, 1)
		case types.StepErrorTasks:
			atomic.AddUint32(&t.cErr, 1)
		default:
			panic(fmt.Sprintf("invalid transfer state '%s' for tasks", c.Transfer.Step))
		}
	}

	return nil
}

// TestTask is a dummy task made for testing. It always succeeds.
type TestTask struct{ *TaskChecker }

// Run executes the dummy task, which will always succeed.
func (t *TestTask) Run(ctx context.Context, _ map[string]string, _ *database.DB,
	c *model.TransferContext) (string, error) {
	return "", t.execTask(ctx, c)
}

// TestTaskError is a dummy task made for testing. It always fails.
type TestTaskError struct{ *TaskChecker }

// Run executes the dummy task, which will always return an error.
func (t *TestTaskError) Run(ctx context.Context, _ map[string]string, _ *database.DB,
	c *model.TransferContext) (string, error) {
	if err := t.execTask(ctx, c); err != nil {
		return "", err
	}

	//nolint:goerr113 // this is a test error, its value is irrelevant
	return "task failed", errors.New("task failed")
}

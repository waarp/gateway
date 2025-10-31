// Package testtasks provides 3 types of tasks designed to be used in tests.
// This package should NOT be imported in production code.
package tasks

import (
	"context"
	"errors"
	"time"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

//nolint:gochecknoinits // designed this way
func init() {
	model.ValidTasks[taskSuccess] = newRunner[*testTaskSuccess]
	model.ValidTasks[taskWarning] = newRunner[*testTaskWarning]
	model.ValidTasks[taskFail] = newRunner[*testTaskFail]
	model.ValidTasks[taskLong] = newRunner[*testTaskLong]
}

var (
	dummyTaskCheck = make(chan string) //nolint:gochecknoglobals // cannot be a constant
	errTaskFailed  = errors.New("task failed")
)

const (
	taskSuccess = "TESTSUCCESS"
	taskWarning = "TESTWARNING"
	taskFail    = "TESTFAIL"
	taskLong    = "TESTLONG"
)

type testTaskSuccess struct{}

func (t *testTaskSuccess) Validate(map[string]string) error {
	return nil
}

func (t *testTaskSuccess) Run(context.Context, map[string]string, *database.DB,
	*log.Logger, *model.TransferContext, any,
) error {
	dummyTaskCheck <- "SUCCESS"

	return nil
}

type testTaskWarning struct{}

func (t *testTaskWarning) Validate(map[string]string) error {
	return nil
}

func (t *testTaskWarning) Run(context.Context, map[string]string, *database.DB,
	*log.Logger, *model.TransferContext, any,
) error {
	dummyTaskCheck <- "WARNING"

	return &WarningError{"warning message"}
}

type testTaskFail struct{}

func (t *testTaskFail) Validate(map[string]string) error {
	return nil
}

func (t *testTaskFail) Run(context.Context, map[string]string, *database.DB,
	*log.Logger, *model.TransferContext, any,
) error {
	dummyTaskCheck <- "FAILURE"

	return errTaskFailed
}

type testTaskLong struct{}

func (t *testTaskLong) Validate(map[string]string) error {
	return nil
}

func (t *testTaskLong) Run(context.Context, map[string]string, *database.DB,
	*log.Logger, *model.TransferContext, any,
) error {
	dummyTaskCheck <- "LONG"

	time.Sleep(time.Minute)

	return errTaskFailed
}

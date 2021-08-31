// Package testtasks provides 3 types of tasks designed to be used in tests.
// This package should NOT be imported in production code.
package tasks

import (
	"context"
	"fmt"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func init() {
	model.ValidTasks[taskSuccess] = &testTaskSuccess{}
	model.ValidTasks[taskWarning] = &testTaskWarning{}
	model.ValidTasks[taskFail] = &testTaskFail{}
	model.ValidTasks[taskLong] = &testTaskLong{}
}

var dummyTaskCheck = make(chan string)

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

func (t *testTaskSuccess) Run(context.Context, map[string]string, *database.DB, *model.TransferContext) (string, error) {
	dummyTaskCheck <- "SUCCESS"
	return "", nil
}

type testTaskWarning struct{}

func (t *testTaskWarning) Validate(map[string]string) error {
	return nil
}

func (t *testTaskWarning) Run(context.Context, map[string]string, *database.DB, *model.TransferContext) (string, error) {
	dummyTaskCheck <- "WARNING"
	return "warning message", &errWarning{"warning message"}
}

type testTaskFail struct{}

func (t *testTaskFail) Validate(map[string]string) error {
	return nil
}

func (t *testTaskFail) Run(context.Context, map[string]string, *database.DB, *model.TransferContext) (string, error) {
	dummyTaskCheck <- "FAILURE"
	return "task failed", fmt.Errorf("task failed")
}

type testTaskLong struct{}

func (t *testTaskLong) Validate(map[string]string) error {
	return nil
}

func (t *testTaskLong) Run(context.Context, map[string]string, *database.DB, *model.TransferContext) (string, error) {
	dummyTaskCheck <- "LONG"
	time.Sleep(time.Minute)
	return "task failed", fmt.Errorf("task failed")
}

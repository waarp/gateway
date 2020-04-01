// Package testtasks provides 3 types of tasks designed to be used in tests.
// This package should NOT be imported in production code.
package tasks

import (
	"fmt"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func init() {
	RunnableTasks[taskSuccess] = &testTaskSuccess{}
	RunnableTasks[taskWarning] = &testTaskWarning{}
	RunnableTasks[taskFail] = &testTaskFail{}
	RunnableTasks[taskLong] = &testTaskLong{}
}

var dummyTaskCheck = make(chan string)

const (
	taskSuccess = "TESTSUCCESS"
	taskWarning = "TESTWARNING"
	taskFail    = "TESTFAIL"
	taskLong    = "TESTLONG"
)

type testTaskSuccess struct{}

func (t *testTaskSuccess) Validate(*model.Task) error {
	return nil
}

func (t *testTaskSuccess) Run(map[string]string, *Processor) (string, error) {
	dummyTaskCheck <- "SUCCESS"
	return "", nil
}

type testTaskWarning struct{}

func (t *testTaskWarning) Validate(*model.Task) error {
	return nil
}

func (t *testTaskWarning) Run(map[string]string, *Processor) (string, error) {
	dummyTaskCheck <- "WARNING"
	return "warning message", errWarning
}

type testTaskFail struct{}

func (t *testTaskFail) Validate(*model.Task) error {
	return nil
}

func (t *testTaskFail) Run(map[string]string, *Processor) (string, error) {
	dummyTaskCheck <- "FAILURE"
	return "task failed", fmt.Errorf("task failed")
}

type testTaskLong struct{}

func (t *testTaskLong) Validate(*model.Task) error {
	return nil
}

func (t *testTaskLong) Run(map[string]string, *Processor) (string, error) {
	dummyTaskCheck <- "LONG"
	time.Sleep(time.Minute)
	return "task failed", fmt.Errorf("task failed")
}

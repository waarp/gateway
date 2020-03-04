package executor

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
)

func init() {
	tasks.RunnableTasks["TESTSUCCESS"] = &testTaskSuccess{}
	tasks.RunnableTasks["TESTFAIL"] = &testTaskFail{}
}

type testTaskSuccess struct{}

func (t *testTaskSuccess) Validate(*model.Task) error {
	return nil
}

func (t *testTaskSuccess) Run(map[string]interface{}, *tasks.Processor) (string, error) {
	return "", nil
}

type testTaskFail struct{}

func (t *testTaskFail) Validate(*model.Task) error {
	return nil
}

func (t *testTaskFail) Run(map[string]interface{}, *tasks.Processor) (string, error) {
	return "task failed", fmt.Errorf("task failed")
}

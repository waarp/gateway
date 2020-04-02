package executor

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
)

func init() {
	tasks.RunnableTasks["TESTSUCCESS"] = &testTaskSuccess{}
	tasks.RunnableTasks["TESTFAIL"] = &testTaskFail{}
	tasks.RunnableTasks["TESTINFINITE"] = &testTaskInfinite{}
	model.ValidTasks["TESTSUCCESS"] = &testTaskSuccess{}
	model.ValidTasks["TESTFAIL"] = &testTaskFail{}
	model.ValidTasks["TESTINFINITE"] = &testTaskInfinite{}
}

type testTaskSuccess struct{}

func (t *testTaskSuccess) Validate(map[string]string) error {
	return nil
}

func (t *testTaskSuccess) Run(map[string]string, *tasks.Processor) (string, error) {
	return "", nil
}

type testTaskFail struct{}

func (t *testTaskFail) Validate(map[string]string) error {
	return nil
}

func (t *testTaskFail) Run(map[string]string, *tasks.Processor) (string, error) {
	return "task failed", fmt.Errorf("task failed")
}

type testTaskInfinite struct{}

func (t *testTaskInfinite) Validate(map[string]string) error {
	return nil
}

func (t *testTaskInfinite) Run(map[string]string, *tasks.Processor) (string, error) {
	for {
		if false {
			break
		}
	}
	return "task failed", fmt.Errorf("task failed")
}

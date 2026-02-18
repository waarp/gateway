package modeltest

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks/taskstest"
)

func AddDummyTask(task string) {
	model.ValidTasks[task] = func() model.TaskRunner { return &taskstest.TestTask{} }
}

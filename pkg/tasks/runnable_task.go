package tasks

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// RunnableTasks is a list of all the tasks known by the gateway
var RunnableTasks = map[string]Runner{}

// Validator permits to validate the arguments for a given task
type Validator interface {
	Validate(*model.Task) error
}

// Runner permits to execute a given task
type Runner interface {
	Run(map[string]interface{}, *TasksRunner) error
}

package tasks

// RunnableTasks is a list of all the tasks known by the gateway
var RunnableTasks = map[string]Runner{}

// Runner permits to execute a given task
type Runner interface {
	Run(map[string]string, *Processor) (string, error)
}

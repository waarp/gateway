package r66

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
	"github.com/smartystreets/goconvey/convey"
)

const (
	clientOK  = "CLIENTOK"
	clientErr = "CLIENTERR"
	serverOK  = "SERVEROK"
	serverErr = "SERVERERR"
)

var (
	clientCheckChannel chan string
	serverCheckChannel chan string
)

func clientMsgShouldBe(c convey.C, msg string) {
	c.So(<-clientCheckChannel, convey.ShouldEqual, msg)
}

func serverMsgShouldBe(c convey.C, msg string) {
	c.So(<-serverCheckChannel, convey.ShouldEqual, msg)
}

func init() {
	tasks.RunnableTasks[clientOK] = &clientTask{}
	tasks.RunnableTasks[clientErr] = &clientTaskError{}
	tasks.RunnableTasks[serverOK] = &serverTask{}
	tasks.RunnableTasks[serverErr] = &serverTaskError{}

	model.ValidTasks[clientOK] = &clientTask{}
	model.ValidTasks[clientErr] = &clientTaskError{}
	model.ValidTasks[serverOK] = &serverTask{}
	model.ValidTasks[serverErr] = &serverTaskError{}
}

// ##### CLIENT #####

type clientTask struct{}

func (*clientTask) Validate(map[string]string) error { return nil }
func (*clientTask) Run(args map[string]string, _ *tasks.Processor) (string, error) {
	clientCheckChannel <- "CLIENT | " + args["msg"] + " | OK"
	return "", nil
}

type clientTaskError struct{}

func (*clientTaskError) Validate(map[string]string) error { return nil }
func (*clientTaskError) Run(args map[string]string, _ *tasks.Processor) (string, error) {
	clientCheckChannel <- "CLIENT | " + args["msg"] + " | ERROR"
	return "task failed", fmt.Errorf("task failed")
}

// ##### SERVER #####

type serverTask struct{}

func (*serverTask) Validate(map[string]string) error { return nil }
func (*serverTask) Run(args map[string]string, _ *tasks.Processor) (string, error) {
	serverCheckChannel <- "SERVER | " + args["msg"] + " | OK"
	return "", nil
}

type serverTaskError struct{}

func (*serverTaskError) Validate(map[string]string) error { return nil }
func (*serverTaskError) Run(args map[string]string, _ *tasks.Processor) (string, error) {
	serverCheckChannel <- "SERVER | " + args["msg"] + " | ERROR"
	return "task failed", fmt.Errorf("task failed")
}

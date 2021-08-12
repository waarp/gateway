package r66

import (
	"fmt"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
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

func clientMsgShouldBe(c convey.C, exp string) {
	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()

	select {
	case msg := <-clientCheckChannel:
		c.So(msg, convey.ShouldEqual, exp)
	case <-ticker.C:
		panic(fmt.Sprintf("Test timed out waiting for client message '%s'", exp))
	}
}

func serverMsgShouldBe(c convey.C, exp string) {
	timer := time.NewTimer(time.Second * 3)
	defer timer.Stop()

	select {
	case msg := <-serverCheckChannel:
		c.So(msg, convey.ShouldEqual, exp)
	case <-timer.C:
		panic(fmt.Sprintf("Test timed out waiting for server message '%s'", exp))
	}
	return
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
	timer := time.NewTimer(time.Second * 1)
	defer timer.Stop()

	select {
	case clientCheckChannel <- "CLIENT | " + args["msg"] + " | OK":
		return "", nil
	case <-timer.C:
		panic(fmt.Sprintf("task timed out sending client message '%s'", args["msg"]))
	}

}

type clientTaskError struct{}

func (*clientTaskError) Validate(map[string]string) error { return nil }
func (*clientTaskError) Run(args map[string]string, _ *tasks.Processor) (string, error) {
	timer := time.NewTimer(time.Second * 1)
	defer timer.Stop()

	select {
	case clientCheckChannel <- "CLIENT | " + args["msg"] + " | ERROR":
		return "task failed", fmt.Errorf("task failed")
	case <-timer.C:
		panic(fmt.Sprintf("task timed out sending client message '%s'", args["msg"]))
	}
}

// ##### SERVER #####

type serverTask struct{}

func (*serverTask) Validate(map[string]string) error { return nil }
func (*serverTask) Run(args map[string]string, _ *tasks.Processor) (string, error) {
	timer := time.NewTimer(time.Second * 1)
	defer timer.Stop()

	select {
	case serverCheckChannel <- "SERVER | " + args["msg"] + " | OK":
		return "", nil
	case <-timer.C:
		panic(fmt.Sprintf("task timed out sending server message '%s'", args["msg"]))
	}
}

type serverTaskError struct{}

func (*serverTaskError) Validate(map[string]string) error { return nil }
func (*serverTaskError) Run(args map[string]string, _ *tasks.Processor) (string, error) {
	timer := time.NewTimer(time.Second * 1)
	defer timer.Stop()

	select {
	case serverCheckChannel <- "SERVER | " + args["msg"] + " | ERROR":
		return "task failed", fmt.Errorf("task failed")
	case <-timer.C:
		panic(fmt.Sprintf("task timed out sending server message '%s'", args["msg"]))
	}
}

package pipelinetest

import (
	"fmt"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks/taskstest"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"github.com/smartystreets/goconvey/convey"
)

func setTestVar() {
	pipeline.TestPipelineEnd = func(isServer bool) {
		if taskstest.ServerCheckChannel == nil || taskstest.ClientCheckChannel == nil {
			panic("nil test task channels")
		}
		if isServer {
			taskstest.ServerCheckChannel <- "SERVER TRANSFER END"
		} else {
			taskstest.ClientCheckChannel <- "CLIENT TRANSFER END"
		}
	}
}

// ServerPreTasksShouldBeOK asserts that the server's pre-tasks should have
// been executed without errors.
func (s *SelfContext) ServerPreTasksShouldBeOK(c convey.C) {
	serverMsgShouldBe(c, fmt.Sprintf("SERVER | %s | PRE-TASKS[0] | OK", s.ServerRule.Name))
}

// ClientPreTasksShouldBeOK asserts that the client's pre-tasks should have
// been executed without errors.
func (s *SelfContext) ClientPreTasksShouldBeOK(c convey.C) {
	clientMsgShouldBe(c, fmt.Sprintf("CLIENT | %s | PRE-TASKS[0] | OK", s.ClientRule.Name))
}

// ServerPosTasksShouldBeOK asserts that the server's post-tasks should have
// been executed without errors.
func (s *SelfContext) ServerPosTasksShouldBeOK(c convey.C) {
	serverMsgShouldBe(c, fmt.Sprintf("SERVER | %s | POST-TASKS[0] | OK", s.ServerRule.Name))
}

// ClientPosTasksShouldBeOK asserts that the client's post-tasks should have
// been executed without errors.
func (s *SelfContext) ClientPosTasksShouldBeOK(c convey.C) {
	clientMsgShouldBe(c, fmt.Sprintf("CLIENT | %s | POST-TASKS[0] | OK", s.ClientRule.Name))
}

// ServerPreTasksShouldBeError asserts that the server's pre-tasks should have
// been executed, and that they should have produced an error.
func (s *SelfContext) ServerPreTasksShouldBeError(c convey.C) {
	s.ServerPreTasksShouldBeOK(c)
	serverMsgShouldBe(c, fmt.Sprintf("SERVER | %s | PRE-TASKS[1] | ERROR", s.ServerRule.Name))
}

// ClientPreTasksShouldBeError asserts that the client's pre-tasks should have
// been executed, and that they should have produced an error.
func (s *SelfContext) ClientPreTasksShouldBeError(c convey.C) {
	s.ClientPreTasksShouldBeOK(c)
	clientMsgShouldBe(c, fmt.Sprintf("CLIENT | %s | PRE-TASKS[1] | ERROR", s.ClientRule.Name))
}

// ServerPosTasksShouldBeError asserts that the server's post-tasks should have
// been executed, and that they should have produced an error.
func (s *SelfContext) ServerPosTasksShouldBeError(c convey.C) {
	s.ServerPosTasksShouldBeOK(c)
	serverMsgShouldBe(c, fmt.Sprintf("SERVER | %s | POST-TASKS[1] | ERROR", s.ServerRule.Name))
}

// ClientPosTasksShouldBeError asserts that the client's post-tasks should have
// been executed, and that they should have produced an error.
func (s *SelfContext) ClientPosTasksShouldBeError(c convey.C) {
	s.ClientPosTasksShouldBeOK(c)
	clientMsgShouldBe(c, fmt.Sprintf("CLIENT | %s | POST-TASKS[1] | ERROR", s.ClientRule.Name))
}

func (s *SelfContext) shouldBeErrorTasks(c convey.C) {
	clientMsgShouldBe(c, fmt.Sprintf("CLIENT | %s | ERROR-TASKS[0] | OK", s.ClientRule.Name))
	serverMsgShouldBe(c, fmt.Sprintf("SERVER | %s | ERROR-TASKS[0] | OK", s.ServerRule.Name))
}

func (s *SelfContext) shouldBeEndTransfer(c convey.C) {
	serverMsgShouldBe(c, "SERVER TRANSFER END")
	clientMsgShouldBe(c, "CLIENT TRANSFER END")
}

func clientMsgShouldBe(c convey.C, exp string) {
	timer := time.NewTimer(time.Second * 10)
	defer timer.Stop()
	select {
	case <-timer.C:
		panic(fmt.Sprintf("timeout waiting for client message '%s'", exp))
	case msg := <-taskstest.ClientCheckChannel:
		c.So(msg, convey.ShouldEqual, exp)
	}
}

func serverMsgShouldBe(c convey.C, exp string) {
	timer := time.NewTimer(time.Second * 10)
	defer timer.Stop()
	select {
	case <-timer.C:
		panic(fmt.Sprintf("timeout waiting for server message '%s'", exp))
	case msg := <-taskstest.ServerCheckChannel:
		c.So(msg, convey.ShouldEqual, exp)
	}
}

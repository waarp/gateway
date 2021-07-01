package pipelinetest

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks/taskstest"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"github.com/smartystreets/goconvey/convey"
)

func setTestVar() {
	pipeline.TestPipelineEnd = func(isServer bool) {
		if isServer {
			if taskstest.ServerCheckChannel == nil {
				panic("nil server task channels")
			}
			close(taskstest.ServerCheckChannel)
		} else {
			if taskstest.ClientCheckChannel == nil {
				panic("nil client task channels")
			}
			close(taskstest.ClientCheckChannel)
		}
	}
}

// ServerPreTasksShouldBeOK asserts that the server's pre-tasks should have
// been executed without errors.
func (s *SelfContext) ServerPreTasksShouldBeOK(c convey.C) {
	taskstest.ServerMsgShouldBe(c, fmt.Sprintf("SERVER | %s | PRE-TASKS[0] | OK", s.ServerRule.Name))
}

// ClientPreTasksShouldBeOK asserts that the client's pre-tasks should have
// been executed without errors.
func (s *SelfContext) ClientPreTasksShouldBeOK(c convey.C) {
	taskstest.ClientMsgShouldBe(c, fmt.Sprintf("CLIENT | %s | PRE-TASKS[0] | OK", s.ClientRule.Name))
}

// ServerPosTasksShouldBeOK asserts that the server's post-tasks should have
// been executed without errors.
func (s *SelfContext) ServerPosTasksShouldBeOK(c convey.C) {
	taskstest.ServerMsgShouldBe(c, fmt.Sprintf("SERVER | %s | POST-TASKS[0] | OK", s.ServerRule.Name))
}

// ClientPosTasksShouldBeOK asserts that the client's post-tasks should have
// been executed without errors.
func (s *SelfContext) ClientPosTasksShouldBeOK(c convey.C) {
	taskstest.ClientMsgShouldBe(c, fmt.Sprintf("CLIENT | %s | POST-TASKS[0] | OK", s.ClientRule.Name))
}

// ServerPreTasksShouldBeError asserts that the server's pre-tasks should have
// been executed, and that they should have produced an error.
func (s *SelfContext) ServerPreTasksShouldBeError(c convey.C) {
	s.ServerPreTasksShouldBeOK(c)
	taskstest.ServerMsgShouldBe(c, fmt.Sprintf("SERVER | %s | PRE-TASKS[1] | ERROR", s.ServerRule.Name))
}

// ClientPreTasksShouldBeError asserts that the client's pre-tasks should have
// been executed, and that they should have produced an error.
func (s *SelfContext) ClientPreTasksShouldBeError(c convey.C) {
	s.ClientPreTasksShouldBeOK(c)
	taskstest.ClientMsgShouldBe(c, fmt.Sprintf("CLIENT | %s | PRE-TASKS[1] | ERROR", s.ClientRule.Name))
}

// ServerPosTasksShouldBeError asserts that the server's post-tasks should have
// been executed, and that they should have produced an error.
func (s *SelfContext) ServerPosTasksShouldBeError(c convey.C) {
	s.ServerPosTasksShouldBeOK(c)
	taskstest.ServerMsgShouldBe(c, fmt.Sprintf("SERVER | %s | POST-TASKS[1] | ERROR", s.ServerRule.Name))
}

// ClientPosTasksShouldBeError asserts that the client's post-tasks should have
// been executed, and that they should have produced an error.
func (s *SelfContext) ClientPosTasksShouldBeError(c convey.C) {
	s.ClientPosTasksShouldBeOK(c)
	taskstest.ClientMsgShouldBe(c, fmt.Sprintf("CLIENT | %s | POST-TASKS[1] | ERROR", s.ClientRule.Name))
}

// ShouldBeClientErrorTasks asserts that the client's error-tasks should have
// been executed.
func (s *SelfContext) ShouldBeClientErrorTasks(c convey.C) {
	taskstest.ClientMsgShouldBe(c, fmt.Sprintf("CLIENT | %s | ERROR-TASKS[0] | OK", s.ClientRule.Name))
}

// ShouldBeServerErrorTasks asserts that the server's error-tasks should have
// been executed.
func (s *SelfContext) ShouldBeServerErrorTasks(c convey.C) {
	taskstest.ServerMsgShouldBe(c, fmt.Sprintf("SERVER | %s | ERROR-TASKS[0] | OK", s.ServerRule.Name))
}

// ShouldBeEndTransfer asserts that both the client & server transfers should
// have finished.
func (s *SelfContext) ShouldBeEndTransfer(c convey.C) {
	taskstest.ServerShouldBeEnd(c)
	taskstest.ClientShouldBeEnd(c)
	s.shouldNotBeInLists()
}

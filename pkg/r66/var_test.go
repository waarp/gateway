package r66

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
	"github.com/smartystreets/goconvey/convey"
)

func init() {
	tasks.RunnableTasks["TESTCHECK"] = &testTaskSuccess{}
	tasks.RunnableTasks["TESTFAIL"] = &testTaskFail{}
	model.ValidTasks["TESTCHECK"] = &testTaskSuccess{}
	model.ValidTasks["TESTFAIL"] = &testTaskFail{}

	logConf := conf.LogConfig{
		Level: "DEBUG",
		LogTo: "stdout",
	}
	_ = log.InitBackend(logConf)
}

var checkChannel chan string

func shouldFinishOK(cEnd, sEnd string) {
	msg := <-checkChannel
	convey.So(msg, convey.ShouldBeIn, cEnd, sEnd)
	if msg == cEnd {
		convey.So(<-checkChannel, convey.ShouldEqual, sEnd)
	} else {
		convey.So(<-checkChannel, convey.ShouldEqual, cEnd)
	}
	convey.So(checkChannel, convey.ShouldBeEmpty)
}

func shouldFinishError(cErr, cEnd, sErr, sEnd string) {
	msg1 := <-checkChannel
	convey.So(msg1, convey.ShouldBeIn, cErr, sErr)
	if msg1 == cErr {
		msg2 := <-checkChannel
		convey.So(msg2, convey.ShouldBeIn, cEnd, sErr)
		if msg2 == cEnd {
			convey.So(<-checkChannel, convey.ShouldEqual, sErr)
			convey.So(<-checkChannel, convey.ShouldEqual, sEnd)
		} else {
			shouldFinishOK(cEnd, sEnd)
		}
	} else {
		msg2 := <-checkChannel
		convey.So(msg2, convey.ShouldBeIn, cErr, sEnd)
		if msg2 == sEnd {
			convey.So(<-checkChannel, convey.ShouldEqual, cErr)
			convey.So(<-checkChannel, convey.ShouldEqual, cEnd)
		} else {
			shouldFinishOK(cEnd, sEnd)
		}
	}
	convey.So(checkChannel, convey.ShouldBeEmpty)
}

type testTaskSuccess struct{}

func (t *testTaskSuccess) Validate(map[string]string) error {
	return nil
}
func (t *testTaskSuccess) Run(args map[string]string, _ *tasks.Processor) (string, error) {
	checkChannel <- args["msg"]
	return "", nil
}

type testTaskFail struct {
	msg string
}

func (t *testTaskFail) Validate(map[string]string) error {
	return nil
}

func (t *testTaskFail) Run(args map[string]string, _ *tasks.Processor) (string, error) {
	checkChannel <- args["msg"]
	return "task failed", fmt.Errorf("task failed")
}

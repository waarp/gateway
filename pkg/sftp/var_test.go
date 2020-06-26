package sftp

import (
	"fmt"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
)

const (
	testLogin    = "test_user"
	testPassword = "test_password"
)

var (
	clientTestPort uint16
	testPK         = []byte(rsaPK)
	testPBK        = []byte(rsaPBK)
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

var checkChannel = make(chan string)

func getNextTask() string {
	timer := time.NewTimer(time.Second)
	select {
	case msg := <-checkChannel:
		return msg
	case <-timer.C:
		return "new task timeout expired"
	}
}

func waitChannel(ch chan struct{}) error {
	timer := time.NewTimer(time.Second)
	select {
	case <-ch:
		return nil
	case <-timer.C:
		return fmt.Errorf("channel close timeout expired")
	}
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

const rsaPK = `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`

const rsaPBK = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDhbxVecyg3NbOuGgIbzUuB3" +
	"GyVIKBRWhYUOaEJtqMR8ckb3WM6cy0yplbZ6is4y8gqGhE9pQ8g3JrbYYlrb8/HnjuCnSzA9" +
	"BVhMNUxp/9Ar7GSvBO2bPIcPYBePe19AJ6MsjoT2jcZhUwlsiacHAnRWaOfYeJQP0Fw9zqhh" +
	"PcjOnWIewNQaghwBXyyzQB/BbiYAMvPo0uveYY+Yr18ExIv3ybtqgAgSVHji4Jg4JFwVd9VP" +
	"fAz3y4ucEYiOr/4bkOBTuAMxbvE+S8mvbOTQ+itsFQxuJgWTrx/53Yth3QYDwgjTaT7TLSSR" +
	"pi1+s9QQg6XTanJyjtEmmYbnaB+EhAQfI0mfOripP/1cTq9StZfYTKl58ObrYWmc5CDH338u" +
	"CdK5GxIP9eNz4RcLqPLvcVBrm62qsYReoD62InykggeOSgkOo4UGbC7JSEdW3afMBGdh797e" +
	"ht6qX3ywKbs7GNVwOt2M7xrpmCehU1uegN7GtIRvCZR0JH4+KSGitWFY3E= test@waarp.org"

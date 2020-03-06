package sftp

import (
	"fmt"

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
	testPK         = []byte(pk)
	testPBK        = []byte(pbk)
)

func init() {
	tasks.RunnableTasks["TESTCHECK"] = &testTaskCheck{}
	tasks.RunnableTasks["TESTSUCCESS"] = &testTaskSuccess{}
	tasks.RunnableTasks["TESTFAIL"] = &testTaskFail{}
	model.ValidTasks["TESTCHECK"] = &testTaskSuccess{}
	model.ValidTasks["TESTSUCCESS"] = &testTaskSuccess{}
	model.ValidTasks["TESTFAIL"] = &testTaskFail{}

	logConf := conf.LogConfig{
		Level: "DEBUG",
		LogTo: "stdout",
	}
	_ = log.InitBackend(logConf)
}

var checkChannel = make(chan string)

type testTaskCheck struct {
	msg string
}

func (t *testTaskCheck) Validate(args map[string]string) error {
	t.msg = args["msg"]
	return nil
}
func (t *testTaskCheck) Run(map[string]string, *tasks.Processor) (string, error) {
	checkChannel <- t.msg
	return "", nil
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

const pk = `-----BEGIN OPENSSH PRIVATE KEY-----
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
Ku9aYVMWJrluF5AAAACnRlc3RAd2FhcnA=
-----END OPENSSH PRIVATE KEY-----`

const pbk = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDFDfzw3Rg6yNL7/ioFBlPjnwhy" +
	"TDk4pRsxN3ocdQ0WTaRfMj8TOgJVy0pug2LlsX+gElwjN8/l10bYR6WvFxT4yxCd/G/PG8a7" +
	"o3CEkjKWusDGkG9I/bNW2foAdHep10dJmdYIdJy6664Rzd7q/VRfUueRcqOk726+W51XenoK" +
	"c4YD2cF24QM6+NL/fbNFi34e5fD6rVPZuUwDIQz033KB58tIzD0cNW5EUVIuCPCTqJHzuDMk" +
	"Z/F3gbLExkcCmU2K5m2NiBlvsfSfvnWimueDoURklyyhClDuTzmpjZxhKLSzSyE6Xis/7z2G" +
	"7san4aUjSVMi68kRi3bnH2pUECZda7hQvbcfTK0xgpEikNIcOyLq99u4x4Jn/0ckoJnAJCYh" +
	"uxuHg2o2255TIC1Z/5BoCp0NG6jMYlAaVjAwGQ+VJQuZAG5AwjB/68hKvalNWu4cWFqWgl3u" +
	"OHrH1dPn01LfTeSzcP1E913Yg7zYkmL0P6eOBLO8SK48hLG4bBkhJmk= test@waarp"

const rsaPK = `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`

const rsaPBK = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDhbxVecyg3NbOuGgIbzUuB3GyVIKBRWhYUOaEJtqMR8ckb3WM6cy0yplbZ6is4y8gqGhE9pQ8g3JrbYYlrb8/HnjuCnSzA9BVhMNUxp/9Ar7GSvBO2bPIcPYBePe19AJ6MsjoT2jcZhUwlsiacHAnRWaOfYeJQP0Fw9zqhhPcjOnWIewNQaghwBXyyzQB/BbiYAMvPo0uveYY+Yr18ExIv3ybtqgAgSVHji4Jg4JFwVd9VPfAz3y4ucEYiOr/4bkOBTuAMxbvE+S8mvbOTQ+itsFQxuJgWTrx/53Yth3QYDwgjTaT7TLSSRpi1+s9QQg6XTanJyjtEmmYbnaB+EhAQfI0mfOripP/1cTq9StZfYTKl58ObrYWmc5CDH338uCdK5GxIP9eNz4RcLqPLvcVBrm62qsYReoD62InykggeOSgkOo4UGbC7JSEdW3afMBGdh797eht6qX3ywKbs7GNVwOt2M7xrpmCehU1uegN7GtIRvCZR0JH4+KSGitWFY3E= test@waarp.org"

package sftp

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
)

const (
	testLogin    = "test_user"
	testPassword = "test_password"
)

var (
	port    int
	testPK  = []byte(pk)
	testPBK = []byte(pbk)

	testLogConf = conf.LogConfig{
		Level: "DEBUG",
		LogTo: "stdout",
	}
)

func init() {
	tasks.RunnableTasks["TESTSUCCESS"] = &testTaskSuccess{}
	tasks.RunnableTasks["TESTFAIL"] = &testTaskFail{}
	model.ValidTasks["TESTSUCCESS"] = &testTaskSuccess{}
	model.ValidTasks["TESTFAIL"] = &testTaskFail{}
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

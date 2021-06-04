package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	. "github.com/smartystreets/goconvey/convey"
)

var discard *log.Logger

func init() {
	_ = log.InitBackend("CRITICAL", "stdout", "")
	discard = log.NewLogger("test_client")
	discard.SetBackend(&logging.NoopBackend{})

	config.ProtoConfigs["test"] = func() config.ProtoConfig { return new(TestProtoConfig) }
	config.ProtoConfigs["test2"] = func() config.ProtoConfig { return new(TestProtoConfig) }
	config.ProtoConfigs["fail"] = func() config.ProtoConfig { return new(TestProtoConfigFail) }
}

func hash(pwd string) []byte {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	So(err, ShouldBeNil)
	return h
}

func writeFile(content string) *os.File {
	file, err := ioutil.TempFile("", "*")
	So(err, ShouldBeNil)
	Reset(func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	})
	_, err = file.WriteString(content)
	So(err, ShouldBeNil)
	return file
}

type TestProtoConfig struct{}

func (*TestProtoConfig) ValidServer() error  { return nil }
func (*TestProtoConfig) ValidPartner() error { return nil }

type TestProtoConfigFail struct{}

func (*TestProtoConfigFail) ValidServer() error {
	return fmt.Errorf("server config validation failed")
}
func (*TestProtoConfigFail) ValidPartner() error {
	return fmt.Errorf("partner config validation failed")
}

func testFile() io.Writer {
	return &strings.Builder{}
}

func getOutput() string {
	str, ok := out.(*strings.Builder)
	So(ok, ShouldBeTrue)
	return str.String()
}

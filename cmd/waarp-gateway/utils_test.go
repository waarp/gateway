package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	. "github.com/smartystreets/goconvey/convey"
)

var discard *log.Logger

func init() {
	logConf := conf.LogConfig{
		Level: "CRITICAL",
		LogTo: "stdout",
	}
	_ = log.InitBackend(logConf)
	discard = log.NewLogger("test_client")
	discard.SetBackend(&logging.NoopBackend{})

	config.ProtoConfigs["test"] = func() config.ProtoConfig { return new(TestProtoConfig) }
	config.ProtoConfigs["test2"] = func() config.ProtoConfig { return new(TestProtoConfig) }
	config.ProtoConfigs["fail"] = func() config.ProtoConfig { return new(TestProtoConfigFail) }
}

func writeFile(content string) *os.File {
	file := testFile()
	_, err := file.WriteString(content)
	So(err, ShouldBeNil)
	return file
}

type TestProtoConfig struct{}

func (*TestProtoConfig) ValidServer() error  { return nil }
func (*TestProtoConfig) ValidPartner() error { return nil }

type TestProtoConfigFail struct{}

func (*TestProtoConfigFail) ValidServer() error  { return fmt.Errorf("test fail") }
func (*TestProtoConfigFail) ValidPartner() error { return fmt.Errorf("test fail") }

func testFile() *os.File {
	tmp, err := ioutil.TempFile("", "*")
	So(err, ShouldBeNil)
	Reset(func() { _ = os.Remove(tmp.Name()) })
	return tmp
}

func getOutput() string {
	_, err := out.Seek(0, 0)
	So(err, ShouldBeNil)
	cont, err := ioutil.ReadAll(out)
	So(err, ShouldBeNil)
	return string(cont)
}

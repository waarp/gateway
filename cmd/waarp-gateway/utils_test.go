package main

import (
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
		Level: "DEBUG",
		LogTo: "stdout",
	}
	discard = log.NewLogger("test_client", logConf)
	discard.SetBackend(&logging.NoopBackend{})

	config.ProtoConfigs["test"] = func() config.ProtoConfig { return new(TestProtoConfig) }
}

type TestProtoConfig struct{}

func (*TestProtoConfig) ValidServer() error { return nil }
func (*TestProtoConfig) ValidClient() error { return nil }

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

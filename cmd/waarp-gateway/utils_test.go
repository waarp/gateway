package main

import (
	"io/ioutil"
	"os"

	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
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
}

func testFile() *os.File {
	tmp, err := ioutil.TempFile("", "*")
	So(err, ShouldBeNil)
	Reset(func() { _ = os.Remove(tmp.Name()) })
	return tmp
}

package main

import (
	"io/ioutil"
	"os"

	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	. "github.com/smartystreets/goconvey/convey"
)

var discard *log.Logger

func init() {
	discard = log.NewLogger("test_client")
	discard.SetBackend(&logging.NoopBackend{})
}

func testFile() *os.File {
	tmp, err := ioutil.TempFile("", "*")
	So(err, ShouldBeNil)
	return tmp
}

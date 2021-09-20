package main

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"code.bcarlin.xyz/go/logging"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
)

//nolint:gochecknoglobals // global var is used by design
var discard *log.Logger

//nolint:gochecknoinits // init is used by design
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
func (*TestProtoConfig) CertRequired() bool  { return false }

type TestProtoConfigFail struct{}

func (*TestProtoConfigFail) ValidServer() error {
	//nolint:goerr113 // base case for a test
	return errors.New("server config validation failed")
}

func (*TestProtoConfigFail) ValidPartner() error {
	//nolint:goerr113 // base case for a test
	return errors.New("partner config validation failed")
}
func (*TestProtoConfigFail) CertRequired() bool { return false }

func testFile() io.Writer {
	return &strings.Builder{}
}

func getOutput() string {
	str, ok := out.(*strings.Builder)
	So(ok, ShouldBeTrue)

	return str.String()
}

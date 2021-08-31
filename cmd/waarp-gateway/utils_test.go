package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"

	"golang.org/x/crypto/bcrypt"

	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	. "github.com/smartystreets/goconvey/convey"
)

var discard *log.Logger

const (
	testProto1   = "cli_proto_1"
	testProto2   = "cli_proto_2"
	testProtoErr = "cli_proto_err"
)

func init() {
	_ = log.InitBackend("CRITICAL", "stdout", "")
	discard = log.NewLogger("test_client")
	discard.SetBackend(&logging.NoopBackend{})

	config.ProtoConfigs[testProto1] = func() config.ProtoConfig { return new(TestProtoConfig) }
	config.ProtoConfigs[testProto2] = func() config.ProtoConfig { return new(TestProtoConfig) }
	config.ProtoConfigs[testProtoErr] = func() config.ProtoConfig { return new(TestProtoConfigFail) }
}

func testHandler(db *database.DB) http.Handler {
	return admin.MakeHandler(discard, db, nil, nil)
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

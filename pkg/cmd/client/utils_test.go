package wg

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"code.bcarlin.xyz/go/logging"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
)

//nolint:gochecknoglobals // global var is used by design
var discard *log.Logger

const (
	testProto1   = "cli_proto_1"
	testProto2   = "cli_proto_2"
	testProtoErr = "cli_proto_err"
)

//nolint:gochecknoinits // init is used by design
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

func hash(pwd string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	So(err, ShouldBeNil)

	return string(h)
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

type TestProtoConfig map[string]interface{}

func (*TestProtoConfig) ValidServer() error  { return nil }
func (*TestProtoConfig) ValidPartner() error { return nil }

type TestProtoConfigFail struct{}

func (*TestProtoConfigFail) ValidServer() error {
	//nolint:goerr113 // base case for a test
	return errors.New("server config validation failed")
}

func (*TestProtoConfigFail) ValidPartner() error {
	//nolint:goerr113 // base case for a test
	return errors.New("partner config validation failed")
}

func testFile() io.Writer {
	return &strings.Builder{}
}

func getOutput() string {
	str, ok := out.(*strings.Builder)
	So(ok, ShouldBeTrue)

	return str.String()
}

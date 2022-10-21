package wg

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"code.waarp.fr/lib/log"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/constructors"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
)

const (
	testProto1   = "cli_proto_1"
	testProto2   = "cli_proto_2"
	testProtoErr = "cli_proto_err"
)

//nolint:gochecknoinits // init is used by design
func init() {
	config.ProtoConfigs[testProto1] = func() config.ProtoConfig { return new(TestProtoConfig) }
	config.ProtoConfigs[testProto2] = func() config.ProtoConfig { return new(TestProtoConfig) }
	config.ProtoConfigs[testProtoErr] = func() config.ProtoConfig { return new(TestProtoConfigFail) }
	constructors.ServiceConstructors[testProto1] = newTestServer
}

func discard() *log.Logger {
	back, err := log.NewBackend(log.LevelTrace, log.Discard, "", "")
	So(err, ShouldBeNil)

	return back.NewLogger("discard")
}

func testHandler(db *database.DB) http.Handler {
	return admin.MakeHandler(discard(), db, nil, nil)
}

func hash(pwd string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	So(err, ShouldBeNil)

	return string(h)
}

func writeFile(content string) *os.File {
	file, err := os.CreateTemp("", "*")
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

func executeCommand(command flags.Commander, args ...string) error {
	params, err := flags.ParseArgs(command, args)
	So(err, ShouldBeNil)

	return command.Execute(params) //nolint:wrapcheck //no need to wrap here
}

type testLocalServer struct {
	name  string
	state state.State
}

func newTestServer(*database.DB, *log.Logger) proto.Service    { return &testLocalServer{} }
func (t *testLocalServer) State() *state.State                 { return &t.state }
func (*testLocalServer) ManageTransfers() *service.TransferMap { return service.NewTransferMap() }

func (t *testLocalServer) Start(a *model.LocalAgent) error {
	t.name = a.Name
	t.state.Set(state.Running, "")

	return nil
}

func (t *testLocalServer) Stop(context.Context) error {
	t.state.Set(state.Offline, "")

	return nil
}

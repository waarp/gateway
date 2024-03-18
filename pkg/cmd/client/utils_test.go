package wg

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"code.waarp.fr/lib/log"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/constructors"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

const (
	testProto1   = "cli_proto_1"
	testProto2   = "cli_proto_2"
	testProtoErr = "cli_proto_err"
)

//nolint:gochecknoglobals //global vars are necessary here
var (
	testCoreServices  = map[string]service.Service{}
	testProtoServices = map[int64]proto.Service{}
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
	return admin.MakeHandler(discard(), db, testCoreServices, testProtoServices)
}

func testHandlerProto(db *database.DB, s map[int64]proto.Service) http.Handler {
	return admin.MakeHandler(discard(), db, nil, s)
}

func hash(pwd string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	So(err, ShouldBeNil)

	return string(h)
}

func mkURL(elem ...string) *types.URL {
	full := path.Join(elem...)

	url, err := types.ParseURL(full)
	So(err, ShouldBeNil)

	return url
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

func fromTransfer(db *database.DB, trans *model.Transfer) *api.OutTransfer {
	var t model.NormalizedTransferView

	So(db.Get(&t, "id=?", trans.ID).Run(), ShouldBeNil)

	jTrans, err := rest.DBTransferToREST(db, &t)
	So(err, ShouldBeNil)

	return jTrans
}

func timePtr(t time.Time) *time.Time { return &t }

type testInterrupter int

const (
	none testInterrupter = iota
	paused
	interrupted
	canceled
)

func (t *testInterrupter) Pause(context.Context) error {
	*t = paused

	return nil
}

func (t *testInterrupter) Interrupt(context.Context) error {
	*t = interrupted

	return nil
}

func (t *testInterrupter) Cancel(context.Context) error {
	*t = canceled

	return nil
}

type testProtoService struct {
	m  *service.TransferMap
	st *state.State
}

func newTestProtoService(trans ...*testInterrupter) *testProtoService {
	serv := &testProtoService{
		m:  service.NewTransferMap(),
		st: &state.State{},
	}

	for i, t := range trans {
		serv.m.Add(int64(i), t)
	}

	serv.st.Set(state.Running, "")

	return serv
}

func (t *testProtoService) Start(*model.LocalAgent) error         { return nil }
func (t *testProtoService) Stop(context.Context) error            { return nil }
func (t *testProtoService) State() *state.State                   { return t.st }
func (t *testProtoService) ManageTransfers() *service.TransferMap { return t.m }

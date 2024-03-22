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
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

const (
	testProto1   = "cli_proto_1"
	testProto2   = "cli_proto_2"
	testProtoErr = "cli_proto_err"
)

//nolint:gochecknoglobals //global vars are necessary here
var (
	testCoreServices  = map[string]service.Service{}
	testProtoServices = map[string]proto.Service{}
)

//nolint:gochecknoinits // init is used by design
func init() {
	config.ProtoConfigs[testProto1] = &config.Constructor{
		Server:  func() config.ServerProtoConfig { return new(TestProtoConfig) },
		Partner: func() config.PartnerProtoConfig { return new(TestProtoConfig) },
		Client:  func() config.ClientProtoConfig { return new(TestProtoConfig) },
	}
	config.ProtoConfigs[testProto2] = &config.Constructor{
		Server:  func() config.ServerProtoConfig { return new(TestProtoConfig) },
		Partner: func() config.PartnerProtoConfig { return new(TestProtoConfig) },
		Client:  func() config.ClientProtoConfig { return new(TestProtoConfig) },
	}
	config.ProtoConfigs[testProtoErr] = &config.Constructor{
		Server:  func() config.ServerProtoConfig { return new(TestProtoConfigFail) },
		Partner: func() config.PartnerProtoConfig { return new(TestProtoConfigFail) },
		Client:  func() config.ClientProtoConfig { return new(TestProtoConfigFail) },
	}

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

func testHandlerProto(db *database.DB, s map[string]proto.Service) http.Handler {
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
func (*TestProtoConfig) ValidClient() error  { return nil }

var errConfFail = errors.New("proto config validation failed")

type TestProtoConfigFail struct{}

func (*TestProtoConfigFail) ValidServer() error  { return errConfFail }
func (*TestProtoConfigFail) ValidPartner() error { return errConfFail }
func (*TestProtoConfigFail) ValidClient() error  { return errConfFail }

// TODO: use executeCommand instead.
func testFile() io.Writer {
	return &strings.Builder{}
}

// TODO: use executeCommand instead.
func getOutput() string {
	str, ok := out.(*strings.Builder)
	So(ok, ShouldBeTrue)

	return str.String()
}

type commander interface{ execute(w io.Writer) error }

func executeCommand(w *strings.Builder, command commander, args ...string) error {
	_, err := flags.ParseArgs(command, args)
	So(err, ShouldBeNil)

	return command.execute(w) //nolint:wrapcheck //no need to wrap here
}

type testLocalServer struct {
	stopped bool
	state   state.State
}

func newTestServer(*database.DB, *log.Logger) proto.Service    { return &testLocalServer{} }
func (t *testLocalServer) State() *state.State                 { return &t.state }
func (*testLocalServer) ManageTransfers() *service.TransferMap { return service.NewTransferMap() }

func (t *testLocalServer) Start(a *model.LocalAgent) error {
	t.state.Set(state.Running, "")

	return nil
}

func (t *testLocalServer) Stop(context.Context) error {
	t.state.Set(state.Offline, "")
	t.stopped = true

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

type testClientService struct {
	m  *service.TransferMap
	st *state.State
}

func newTestClientService(trans ...*testInterrupter) *testClientService {
	serv := &testClientService{
		m:  service.NewTransferMap(),
		st: &state.State{},
	}

	for i, t := range trans {
		serv.m.Add(int64(i), t)
	}

	serv.st.Set(state.Running, "")

	return serv
}

func (t *testClientService) Start() error                          { return nil }
func (t *testClientService) Stop(context.Context) error            { return nil }
func (t *testClientService) State() *state.State                   { return t.st }
func (t *testClientService) ManageTransfers() *service.TransferMap { return t.m }
func (t *testClientService) InitTransfer(*pipeline.Pipeline) (pipeline.TransferClient, *types.TransferError) {
	panic("test-only, should never be called")
}

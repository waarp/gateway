package rest

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/constructors"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
)

const (
	testProto1 = "rest_test_1"
	testProto2 = "rest_test_2"
)

//nolint:gochecknoinits // init is used by design
func init() {
	config.ProtoConfigs[testProto1] = func() config.ProtoConfig { return new(testProtoConfig) }
	config.ProtoConfigs[testProto2] = func() config.ProtoConfig { return new(testProtoConfig) }
	constructors.ServiceConstructors[testProto1] = newTestServer
}

type testProtoConfig struct {
	Key string `json:"key,omitempty"`
}

func (*testProtoConfig) ValidServer() error  { return nil }
func (*testProtoConfig) ValidPartner() error { return nil }

func hash(pwd string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	convey.So(err, convey.ShouldBeNil)

	return string(h)
}

func testAdminServer(logger *log.Logger, db *database.DB) string {
	return testAdminServerWithServices(logger, db, nil, nil)
}

func testAdminServerWithServices(logger *log.Logger, db *database.DB,
	testCoreServices map[string]service.Service,
	testProtoServices map[int64]proto.Service,
) string {
	router := mux.NewRouter()
	MakeRESTHandler(logger, db, router, testCoreServices, testProtoServices)

	serv := httptest.NewServer(router)

	return serv.URL
}

func methodTestRequest(host, path string) *http.Response {
	resp := makeTestRequest(http.MethodPut, host, path, nil)

	convey.Reset(func() { convey.So(resp.Body.Close(), convey.ShouldBeNil) })

	return resp
}

func makeTestRequest(method, host, path string, body io.Reader) *http.Response {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	convey.Reset(cancel)

	r, err := http.NewRequestWithContext(ctx, method, host+path, body)
	convey.So(err, convey.ShouldBeNil)
	r.SetBasicAuth("admin", "admin_password")

	resp, err := http.DefaultClient.Do(r)
	convey.So(err, convey.ShouldBeNil)

	return resp
}

func fromTransfer(db *database.DB, trans *model.Transfer) *api.OutTransfer {
	var t model.NormalizedTransferView

	convey.So(db.Get(&t, "id=?", trans.ID).Run(), convey.ShouldBeNil)

	jTrans, err := DBTransferToREST(db, &t)
	convey.So(err, convey.ShouldBeNil)

	return jTrans
}

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

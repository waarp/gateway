package rest

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"time"

	"github.com/gorilla/mux"
	"github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	testProto1 = "rest_test_1"
	testProto2 = "rest_test_2"
)

//nolint:gochecknoinits // init is used by design
func init() {
	protocols.Register(testProto1, &testModule{})
	protocols.Register(testProto2, &testModule{})
}

func stateCode(service services.Service) utils.StateCode {
	code, _ := service.State()

	return code
}

func mustAddr(s string) types.Address {
	addr, err := types.NewAddress(s)
	convey.So(err, convey.ShouldBeNil)

	return *addr
}

type testModule struct{}

func (testModule) CanMakeTransfer(*model.TransferContext) error { return nil }
func (testModule) CheckServerConfig(map[string]any) error       { return nil }
func (testModule) CheckClientConfig(map[string]any) error       { return nil }
func (testModule) CheckPartnerConfig(map[string]any) error      { return nil }
func (testModule) NewServer(_ *database.DB, s *model.LocalAgent) protocol.Server {
	return &testService{name: s.Name}
}

func (testModule) NewClient(_ *database.DB, c *model.Client) protocol.Client {
	return &testService{name: c.Name}
}

func hash(pwd string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	convey.So(err, convey.ShouldBeNil)

	return string(h)
}

func localPath(fPath string) string {
	if runtime.GOOS == "windows" {
		fPath = "C:" + fPath
	}

	return fPath
}

func testAdminServer(logger *log.Logger, db *database.DB) string {
	return testAdminServerWithServices(logger, db)
}

func testAdminServerWithServices(logger *log.Logger, db *database.DB) string {
	router := mux.NewRouter()
	MakeRESTHandler(logger, db, router)

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

	return DBTransferToREST(&t)
}

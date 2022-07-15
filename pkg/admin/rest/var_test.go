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

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
)

const (
	testProto1 = "rest_test_1"
	testProto2 = "rest_test_2"
)

//nolint:gochecknoinits // init is used by design
func init() {
	config.ProtoConfigs[testProto1] = func() config.ProtoConfig { return new(TestProtoConfig) }
	config.ProtoConfigs[testProto2] = func() config.ProtoConfig { return new(TestProtoConfig) }
}

type TestProtoConfig struct {
	Key string `json:"key,omitempty"`
}

func (*TestProtoConfig) ValidServer() error  { return nil }
func (*TestProtoConfig) ValidPartner() error { return nil }

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
	testProtoServices map[string]service.ProtoService,
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

package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	. "github.com/smartystreets/goconvey/convey"
)

type testService struct {
	state service.State
}

func (*testService) Start() error                          { return nil }
func (*testService) Stop(context.Context) error            { return nil }
func (t *testService) State() *service.State               { return &t.state }
func (*testService) ManageTransfers() *service.TransferMap { return service.NewTransferMap() }

func TestStatus(t *testing.T) {
	statusLogger := log.NewLogger("rest_status_test")

	core := map[string]service.Service{
		"Core Running Service": &testService{state: service.State{}},
		"Core Offline Service": &testService{state: service.State{}},
		"Core Error Service":   &testService{state: service.State{}},
	}
	core["Core Running Service"].State().Set(service.Running, "")
	core["Core Offline Service"].State().Set(service.Offline, "")
	core["Core Error Service"].State().Set(service.Error, "Test Reason")

	proto := map[string]service.ProtoService{
		"Proto Running Service": &testService{state: service.State{}},
		"Proto Offline Service": &testService{state: service.State{}},
		"Proto Error Service":   &testService{state: service.State{}},
	}
	proto["Proto Running Service"].State().Set(service.Running, "")
	proto["Proto Offline Service"].State().Set(service.Offline, "")
	proto["Proto Error Service"].State().Set(service.Error, "Test Reason")

	statuses := map[string]api.Status{
		"Core Error Service":    {State: service.Error.Name(), Reason: "Test Reason"},
		"Core Offline Service":  {State: service.Offline.Name()},
		"Core Running Service":  {State: service.Running.Name()},
		"Proto Error Service":   {State: service.Error.Name(), Reason: "Test Reason"},
		"Proto Offline Service": {State: service.Offline.Name()},
		"Proto Running Service": {State: service.Running.Name()},
	}

	Convey("Given the REST status handler", t, func() {
		handler := getStatus(statusLogger, core, proto)

		Convey("Given a status request", func() {
			w := httptest.NewRecorder()
			r, err := http.NewRequest(http.MethodGet, "/api/status", nil)

			So(err, ShouldBeNil)

			Convey("When the request is sent to the handler", func() {
				handler.ServeHTTP(w, r)

				Convey("Then the handler should reply 'OK'", func() {
					So(w.Code, ShouldEqual, http.StatusOK)
				})

				Convey("Then the 'Content-Type' header should contain 'application/json", func() {
					contentType := w.Header().Get("Content-Type")

					So(contentType, ShouldEqual, "application/json")
				})

				Convey("Then the response body should contain the services in JSON format", func() {
					response := map[string]api.Status{}
					err := json.Unmarshal(w.Body.Bytes(), &response)

					So(err, ShouldBeNil)
					So(response, ShouldResemble, statuses)
				})
			})
		})
	})
}

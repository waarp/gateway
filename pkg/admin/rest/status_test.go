package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service/state"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

type testService struct{ state state.State }

func (*testService) Start() error               { return nil }
func (*testService) Stop(context.Context) error { return nil }
func (t *testService) State() *state.State      { return &t.state }

type testServer struct{ state state.State }

func (*testServer) Start(*model.LocalAgent) error         { return nil }
func (*testServer) Stop(context.Context) error            { return nil }
func (t *testServer) State() *state.State                 { return &t.state }
func (*testServer) ManageTransfers() *service.TransferMap { return service.NewTransferMap() }

func TestStatus(t *testing.T) {
	core := map[string]service.Service{
		"Core Running Service": &testService{state: state.State{}},
		"Core Offline Service": &testService{state: state.State{}},
		"Core Error Service":   &testService{state: state.State{}},
	}
	core["Core Running Service"].State().Set(state.Running, "")
	core["Core Offline Service"].State().Set(state.Offline, "")
	core["Core Error Service"].State().Set(state.Error, "Test Reason")

	proto := map[string]service.ProtoService{
		"Proto Running Service": &testServer{state: state.State{}},
		"Proto Offline Service": &testServer{state: state.State{}},
		"Proto Error Service":   &testServer{state: state.State{}},
	}
	proto["Proto Running Service"].State().Set(state.Running, "")
	proto["Proto Offline Service"].State().Set(state.Offline, "")
	proto["Proto Error Service"].State().Set(state.Error, "Test Reason")

	statuses := map[string]api.Status{
		"Core Error Service":    {State: state.Error.Name(), Reason: "Test Reason"},
		"Core Offline Service":  {State: state.Offline.Name()},
		"Core Running Service":  {State: state.Running.Name()},
		"Proto Error Service":   {State: state.Error.Name(), Reason: "Test Reason"},
		"Proto Offline Service": {State: state.Offline.Name()},
		"Proto Running Service": {State: state.Running.Name()},
	}

	Convey("Given the REST status handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_status_test")
		handler := getStatus(logger, core, proto)

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

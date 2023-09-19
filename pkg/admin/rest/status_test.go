package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

type testService struct {
	state   utils.State
	stopped bool
}

func makeAndStartTestService() *testService {
	return &testService{state: utils.NewState(utils.StateRunning, "")}
}

func (t *testService) State() (utils.StateCode, string) { return t.state.Get() }
func (t *testService) InitTransfer(*pipeline.Pipeline) (protocol.TransferClient, error) {
	panic("should not be called")
}

func (t *testService) Start() error {
	t.state.Set(utils.StateRunning, "")

	return nil
}

func (t *testService) Stop(context.Context) error {
	t.state.Set(utils.StateOffline, "")
	t.stopped = true

	return nil
}

func TestStatus(t *testing.T) {
	Convey("Given a gateway with some services", t, func(c C) {
		services.Core = map[string]services.Service{
			"Running Core Service": &testService{state: utils.NewState(utils.StateRunning, "")},
			"Offline Core Service": &testService{state: utils.NewState(utils.StateOffline, "")},
			"Error Core Service":   &testService{state: utils.NewState(utils.StateError, "Test Reason")},
		}

		services.Servers = map[string]services.Server{
			"Running Server": &testService{state: utils.NewState(utils.StateRunning, "")},
			"Offline Server": &testService{state: utils.NewState(utils.StateOffline, "")},
			"Error Server":   &testService{state: utils.NewState(utils.StateError, "Test Reason")},
		}

		statuses := map[string]api.Status{
			"Error Core Service":   {State: utils.StateError.String(), Reason: "Test Reason"},
			"Offline Core Service": {State: utils.StateOffline.String()},
			"Running Core Service": {State: utils.StateRunning.String()},
			"Error Server":         {State: utils.StateError.String(), Reason: "Test Reason"},
			"Offline Server":       {State: utils.StateOffline.String()},
			"Running Server":       {State: utils.StateRunning.String()},
		}

		Convey("Given the REST status handler", func() {
			logger := testhelpers.TestLogger(c, "rest_status_test")
			handler := getStatus(logger)

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
	})
}

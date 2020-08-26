package main

import (
	"context"
	"net/http/httptest"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	. "github.com/smartystreets/goconvey/convey"
)

type emptyService struct {
	state *service.State
}

func (emptyService) Start() error {
	return nil
}

func (emptyService) Stop(context.Context) error {
	return nil
}

func (e emptyService) State() *service.State {
	return e.state
}

func TestRequestStatus(t *testing.T) {
	runningState := service.State{}
	runningState.Set(service.Running, "")
	offlineState := service.State{}
	offlineState.Set(service.Offline, "")
	errorState := service.State{}
	errorState.Set(service.Error, "Error message")

	Convey("Testing the 'status' command", t, func() {
		out = testFile()
		s := statusCommand{}

		Convey("Given a running gateway", func() {
			db := database.GetTestDatabase()
			services := map[string]service.Service{
				"Service 1": &emptyService{state: &offlineState},
				"Service 2": &emptyService{state: &runningState},
				"Service 3": &emptyService{state: &offlineState},
				"Service 4": &emptyService{state: &errorState},
				"Service 5": &emptyService{state: &errorState},
				"Service 6": &emptyService{state: &runningState},
			}
			gw := httptest.NewServer(admin.MakeHandler(discard, db, services))

			Convey("When executing the command", func() {
				commandLine.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

				So(s.Execute(nil), ShouldBeNil)

				Convey("Then it should display the services' status", func() {
					So(getOutput(), ShouldEqual, "Waarp-Gateway services:\n"+
						"[Error]   Service 4 (Error message)\n"+
						"[Error]   Service 5 (Error message)\n"+
						"[Active]  Service 2\n"+
						"[Active]  Service 6\n"+
						"[Offline] Service 1\n"+
						"[Offline] Service 3\n",
					)
				})
			})
		})
	})
}

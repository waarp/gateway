package main

import (
	"context"
	"io/ioutil"
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

func (emptyService) Stop(_ context.Context) error {
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
				"Offline Service 1": &emptyService{state: &offlineState},
				"Running Service 1": &emptyService{state: &runningState},
				"Offline Service 2": &emptyService{state: &offlineState},
				"Error Service 1":   &emptyService{state: &errorState},
				"Error Service 2":   &emptyService{state: &errorState},
				"Running Service 2": &emptyService{state: &runningState},
			}
			gw := httptest.NewServer(admin.MakeHandler(discard, db, services))

			Convey("When executing the command", func() {
				dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
				auth.DSN = dsn

				err := s.Execute(nil)

				Convey("Then it should return no error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then it should display the services' status", func() {

					_, err = out.Seek(0, 0)
					So(err, ShouldBeNil)
					cont, err := ioutil.ReadAll(out)
					So(err, ShouldBeNil)
					So(string(cont), ShouldEqual, "Waarp-Gateway services:\n"+
						"[Error]   Error Service 1 (Error message)\n"+
						"[Error]   Error Service 2 (Error message)\n"+
						"[Active]  Running Service 1\n"+
						"[Active]  Running Service 2\n"+
						"[Offline] Offline Service 1\n"+
						"[Offline] Offline Service 2\n",
					)
				})
			})
		})
	})
}

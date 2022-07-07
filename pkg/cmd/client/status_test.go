package wg

import (
	"context"
	"net/http/httptest"
	"net/url"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
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

func (e emptyService) ManageTransfers() *service.TransferMap {
	return nil
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
		s := Status{}

		Convey("Given a running gateway", func(c C) {
			db := database.TestDatabase(c)
			core := map[string]service.Service{
				"Core Service 1": &emptyService{state: &offlineState},
				"Core Service 2": &emptyService{state: &runningState},
				"Core Service 3": &emptyService{state: &offlineState},
				"Core Service 4": &emptyService{state: &errorState},
				"Core Service 5": &emptyService{state: &errorState},
				"Core Service 6": &emptyService{state: &runningState},
			}
			proto := map[string]service.ProtoService{
				"Proto Service 1": &emptyService{state: &offlineState},
				"Proto Service 2": &emptyService{state: &runningState},
				"Proto Service 3": &emptyService{state: &offlineState},
				"Proto Service 4": &emptyService{state: &errorState},
				"Proto Service 5": &emptyService{state: &errorState},
				"Proto Service 6": &emptyService{state: &runningState},
			}
			gw := httptest.NewServer(admin.MakeHandler(discard(), db, core, proto))

			Convey("When executing the command", func() {
				var err error
				addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
				So(err, ShouldBeNil)

				So(s.Execute(nil), ShouldBeNil)

				Convey("Then it should display the services' status", func() {
					So(getOutput(), ShouldEqual, "Waarp-Gateway services:\n"+
						"[Error]   Core Service 4 (Error message)\n"+
						"[Error]   Core Service 5 (Error message)\n"+
						"[Error]   Proto Service 4 (Error message)\n"+
						"[Error]   Proto Service 5 (Error message)\n"+
						"[Active]  Core Service 2\n"+
						"[Active]  Core Service 6\n"+
						"[Active]  Proto Service 2\n"+
						"[Active]  Proto Service 6\n"+
						"[Offline] Core Service 1\n"+
						"[Offline] Core Service 3\n"+
						"[Offline] Proto Service 1\n"+
						"[Offline] Proto Service 3\n",
					)
				})
			})
		})
	})
}

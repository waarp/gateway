package wg

import (
	"context"
	"net/http/httptest"
	"net/url"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service/state"
)

type dummyService struct{ state *state.State }

func (*dummyService) Start() error               { return nil }
func (*dummyService) Stop(context.Context) error { return nil }
func (e *dummyService) State() *state.State      { return e.state }

type dummyServer struct{ *dummyService }

func (*dummyServer) Start(*model.LocalAgent) error          { return nil }
func (e dummyServer) ManageTransfers() *service.TransferMap { return nil }

func TestRequestStatus(t *testing.T) {
	runningState := state.State{}
	runningState.Set(state.Running, "")

	offlineState := state.State{}
	offlineState.Set(state.Offline, "")

	errorState := state.State{}
	errorState.Set(state.Error, "Error message")

	Convey("Testing the 'status' command", t, func() {
		out = testFile()
		s := Status{}

		Convey("Given a running gateway", func(c C) {
			db := database.TestDatabase(c)
			core := map[string]service.Service{
				"Core Service 1": &dummyService{&offlineState},
				"Core Service 2": &dummyService{&runningState},
				"Core Service 3": &dummyService{&offlineState},
				"Core Service 4": &dummyService{&errorState},
				"Core Service 5": &dummyService{&errorState},
				"Core Service 6": &dummyService{&runningState},
			}
			proto := map[string]service.ProtoService{
				"Proto Service 1": &dummyServer{&dummyService{&offlineState}},
				"Proto Service 2": &dummyServer{&dummyService{&runningState}},
				"Proto Service 3": &dummyServer{&dummyService{&offlineState}},
				"Proto Service 4": &dummyServer{&dummyService{&errorState}},
				"Proto Service 5": &dummyServer{&dummyService{&errorState}},
				"Proto Service 6": &dummyServer{&dummyService{&runningState}},
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

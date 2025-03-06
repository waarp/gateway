package gatewayd

import (
	"path"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func testSetup(c C, root string) (*WG, *model.LocalAgent, *model.LocalAgent) {
	db := database.TestDatabase(c)
	addServ := func(name string) *model.LocalAgent {
		s := &model.LocalAgent{
			Name: name, Protocol: testProtocol,
			Address: types.Addr("localhost", testhelpers.GetFreePort(c)),
		}
		So(db.Insert(s).Run(), ShouldBeNil)

		return s
	}

	s1 := addServ("serv1")
	s2 := addServ("serv2")

	conf.GlobalConfig.Paths = conf.PathsConfig{
		GatewayHome:   root,
		DefaultInDir:  path.Join(root, "in"),
		DefaultOutDir: path.Join(root, "out"),
		DefaultTmpDir: path.Join(root, "tmp"),
	}
	conf.GlobalConfig.Log = conf.LogConfig{
		Level: "WARNING",
		LogTo: "stdout",
	}
	conf.GlobalConfig.Admin = conf.AdminConfig{
		Host: "localhost",
		Port: testhelpers.GetFreePort(c),
	}
	conf.GlobalConfig.Controller = conf.ControllerConfig{
		Delay: time.Minute,
	}

	wg := NewWG()
	wg.Logger = testhelpers.TestLogger(c, "test_wg")

	return wg, s1, s2
}

func checkState(wg *WG, code utils.StateCode, s1, s2 *model.LocalAgent) {
	So(wg.DBService, ShouldNotBeNil)
	So(wg.AdminService, ShouldNotBeNil)
	So(wg.Controller, ShouldNotBeNil)

	dbState, _ := wg.DBService.State()
	adState, _ := wg.AdminService.State()
	contState, _ := wg.Controller.State()

	So(dbState, ShouldEqual, code)
	So(adState, ShouldEqual, code)
	So(contState, ShouldEqual, code)

	serv1, ok := services.Servers[s1.Name]
	So(ok, ShouldBeTrue)

	serv2, ok := services.Servers[s2.Name]
	So(ok, ShouldBeTrue)

	s1State, _ := serv1.State()
	s2State, _ := serv2.State()

	So(s1State, ShouldEqual, code)
	So(s2State, ShouldEqual, code)
}

func TestStartServices(t *testing.T) {
	root := t.TempDir()

	Convey("Given a gateway service", t, func(c C) {
		wg, s1, s2 := testSetup(c, root)

		Convey("When starting the gateway services", func() {
			So(wg.startServices(), ShouldBeNil)
			defer wg.stopServices()

			Convey("Then it should have started all the gateway services", func() {
				checkState(wg, utils.StateRunning, s1, s2)
			})
		})
	})
}

func TestStopServices(t *testing.T) {
	root := t.TempDir()

	Convey("Given a gateway service", t, func(c C) {
		wg, s1, s2 := testSetup(c, root)

		Convey("After starting the gateway services", func() {
			So(wg.startServices(), ShouldBeNil)

			Convey("When stopping the services", func() {
				wg.stopServices()

				Convey("Then it should have stopped all the services", func() {
					checkState(wg, utils.StateOffline, s1, s2)
				})
			})
		})
	})
}

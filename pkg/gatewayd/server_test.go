package gatewayd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func testSetup(c C) (*WG, *model.LocalAgent, *model.LocalAgent) {
	So(os.Setenv("GATEWAY_TEST_DB", database.SQLite), ShouldBeNil)

	db := database.TestDatabase(c)
	addServ := func(name string) *model.LocalAgent {
		s := &model.LocalAgent{
			Name:        name,
			Protocol:    testProtocol,
			ProtoConfig: json.RawMessage("{}"),
			Address:     fmt.Sprintf("localhost:%d", testhelpers.GetFreePort(c)),
		}
		So(db.Insert(s).Run(), ShouldBeNil)

		return s
	}

	s1 := addServ("serv1")
	s2 := addServ("serv2")

	root := testhelpers.TempDir(c, "start_services")
	conf.GlobalConfig.Paths = conf.PathsConfig{
		GatewayHome:   root,
		DefaultInDir:  filepath.Join(root, "in"),
		DefaultOutDir: filepath.Join(root, "out"),
		DefaultTmpDir: filepath.Join(root, "tmp"),
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

	return &WG{Logger: testhelpers.TestLogger(c, "test_wg")}, s1, s2
}

func checkState(wg *WG, code service.StateCode, s1, s2 *model.LocalAgent) {
	So(wg.dbService, ShouldNotBeNil)
	So(wg.adminService, ShouldNotBeNil)
	So(wg.controller, ShouldNotBeNil)

	dbState, _ := wg.dbService.State().Get()
	adState, _ := wg.adminService.State().Get()
	contState, _ := wg.controller.State().Get()

	So(dbState, ShouldEqual, code)
	So(adState, ShouldEqual, code)
	So(contState, ShouldEqual, code)

	serv1, ok := wg.ProtoServices[s1.Name]
	So(ok, ShouldBeTrue)

	serv2, ok := wg.ProtoServices[s2.Name]
	So(ok, ShouldBeTrue)

	s1State, _ := serv1.State().Get()
	s2State, _ := serv2.State().Get()

	So(s1State, ShouldEqual, code)
	So(s2State, ShouldEqual, code)
}

func TestStartServices(t *testing.T) {
	Convey("Given a gateway service", t, func(c C) {
		wg, s1, s2 := testSetup(c)

		Convey("When starting the gateway services", func() {
			So(wg.startServices(), ShouldBeNil)
			defer wg.stopServices()

			Convey("Then it should have started all the gateway services", func() {
				checkState(wg, service.Running, s1, s2)
			})
		})
	})
}

func TestStopServices(t *testing.T) {
	Convey("Given a gateway service", t, func(c C) {
		wg, s1, s2 := testSetup(c)

		Convey("After starting the gateway services", func() {
			So(wg.startServices(), ShouldBeNil)

			Convey("When stopping the services", func() {
				wg.stopServices()

				Convey("Then it should have stopped all the services", func() {
					checkState(wg, service.Offline, s1, s2)
				})
			})
		})
	})
}

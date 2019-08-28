package main

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"testing"

	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	. "github.com/smartystreets/goconvey/convey"
)

var testDb = database.GetTestDatabase()
var testServices map[string]service.Service
var testServer *httptest.Server

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

func init() {
	envPassword = "admin_password"

	testLogger := log.NewLogger("test")
	discard, err := logging.NewNoopBackend()
	if err != nil {
		panic(err)
	}
	testLogger.SetBackend(discard)

	runningState := service.State{}
	runningState.Set(service.Running, "")
	offlineState := service.State{}
	offlineState.Set(service.Offline, "")
	errorState := service.State{}
	errorState.Set(service.Error, "Error message")
	testServices = map[string]service.Service{
		"Running Service": emptyService{state: &runningState},
		"Offline Service": emptyService{state: &offlineState},
		"Error Service":   emptyService{state: &errorState},
	}

	testServer = httptest.NewServer(admin.MakeHandler(testLogger, testDb, testServices))
}

func TestRequestStatus(t *testing.T) {

	Convey("Given a server replying correctly", t, func() {
		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		s := statusCommand{}

		Convey("When calling requestStatus", func() {
			err := s.Execute(nil)

			Convey("Then it should return no error", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given an incorrect address", t, func() {
		auth = ConnectionOptions{
			Address:  "incorrect",
			Username: "admin",
		}
		s := statusCommand{}

		Convey("When requestStatus is called", func() {
			err := s.Execute(nil)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})

	Convey("Given an incorrect user", t, func() {
		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "unknown",
		}
		s := statusCommand{}

		Convey("When requestStatus is called", func() {
			err := s.Execute(nil)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestShowStatus(t *testing.T) {

	Convey("Given a list of status", t, func() {
		statuses := make(map[string]admin.Status)
		statuses["Test running 1"] = admin.Status{State: "Running", Reason: ""}
		statuses["Test running 2"] = admin.Status{State: "Running", Reason: ""}
		statuses["Test offline 1"] = admin.Status{State: "Offline", Reason: ""}
		statuses["Test offline 2"] = admin.Status{State: "Offline", Reason: ""}
		statuses["Test error 1"] = admin.Status{State: "Error", Reason: "Reason 1"}
		statuses["Test error 2"] = admin.Status{State: "Error", Reason: "Reason 2"}

		Convey("When calling 'showStatus'", func() {
			out, err := ioutil.TempFile(".", "waarp_gateway")
			So(err, ShouldBeNil)
			showStatus(out, statuses)
			_ = out.Close()
			in, err := os.Open(out.Name())
			So(err, ShouldBeNil)
			result, err := ioutil.ReadAll(in)
			So(err, ShouldBeNil)

			Reset(func() {
				_ = os.Remove(out.Name())
			})

			Convey("Then it should display the statuses correctly", func() {
				expected := "\n" +
					"Waarp-Gateway services :\n" +
					"[Error]   Test error 1 : Reason 1\n" +
					"[Error]   Test error 2 : Reason 2\n" +
					"[Active]  Test running 1\n" +
					"[Active]  Test running 2\n" +
					"[Offline] Test offline 1\n" +
					"[Offline] Test offline 2\n" +
					"\n"

				So(result, ShouldNotBeNil)
				So(string(result), ShouldEqual, expected)
			})
		})
	})
}

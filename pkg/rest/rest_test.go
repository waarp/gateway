package rest

import (
	"net/http"
	"net/url"
	"testing"

	"code.waarp.fr/waarp/gateway-ng/pkg/conf"
	"code.waarp.fr/waarp/gateway-ng/pkg/log"
	. "github.com/smartystreets/goconvey/convey"
)

func TestStatus(t *testing.T) {
	config := conf.ServerConfig{}
	config.Rest.Port = "9000"
	rest := Service{
		Logger: log.NewLogger(),
	}
	rest.StartRestService(config)

	Convey("Given a REST service", t, func() {

		Convey("When a status request is made", func() {
			client := new(http.Client)
			response, err := client.Get("http://localhost:9000/status")

			Convey("Then the service should respond OK", func() {
				So(response, ShouldNotBeNil)
				So(err, ShouldBeNil)
				So(response.StatusCode, ShouldEqual, http.StatusOK)
			})
		})

		Convey("When the service is stopped", func() {
			rest.StopRestService()

			Convey("Then the service should no longer respond", func() {
				client := new(http.Client)
				response, err := client.Get("http://localhost:9000/status")

				So(response, ShouldBeNil)
				urlError := new(url.Error)
				So(err, ShouldHaveSameTypeAs, urlError)
			})
		})
	})
}

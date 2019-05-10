package rest

import (
	"net/http"
	"testing"

	"code.waarp.fr/waarp/gateway-ng/pkg/log"
	. "github.com/smartystreets/goconvey/convey"
)

func TestStart(t *testing.T) {

	Convey("Given a REST service", t, func() {
		rest := Service{
			port:   ":9999",
			logger: log.NewLogger(),
		}
		rest.StartRestService()

		Convey("When a request is made", func() {
			client := new(http.Client)
			response, err := client.Get("http://localhost:9999/status")

			Convey("Then the service should respond OK", func() {
				So(response, ShouldNotBeNil)
				So(err, ShouldBeNil)
				So(response.StatusCode, ShouldEqual, http.StatusOK)
			})
		})
	})
}

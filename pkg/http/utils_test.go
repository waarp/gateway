package http

import (
	"net/http"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
)

func init() {
	pipelinetest.Protocols["http"] = pipelinetest.ProtoFeatures{
		ClientConstr: NewHTTPClient, ServiceConstr: NewService,
		TransID: true, RuleName: true, Size: true,
	}
	pipelinetest.Protocols["https"] = pipelinetest.ProtoFeatures{
		ClientConstr: NewHTTPSClient, ServiceConstr: NewService,
		TransID: true, RuleName: true, Size: true,
	}
}

func TestGetContentRange(t *testing.T) {
	Convey("Given the Content-Range parsing function", t, func() {
		Convey("Given a Content-Range with both range & size", func() {
			headers := make(http.Header)
			headers.Add("Content-Range", "bytes 10-100/100")

			Convey("When calling the function", func() {
				offset, size, err := getContentRange(headers)
				So(err, ShouldBeNil)

				Convey("Then it should return the correct offset", func() {
					So(offset, ShouldEqual, 10)
				})

				Convey("Then it should return the correct size", func() {
					So(size, ShouldEqual, 100)
				})
			})
		})

		Convey("Given a Content-Range with only range", func() {
			headers := make(http.Header)
			headers.Add("Content-Range", "bytes 10-100/*")

			Convey("When calling the function", func() {
				offset, size, err := getContentRange(headers)
				So(err, ShouldBeNil)

				Convey("Then it should return the correct offset", func() {
					So(offset, ShouldEqual, 10)
				})

				Convey("Then it should return the default size", func() {
					So(size, ShouldEqual, model.UnknownSize)
				})
			})
		})

		Convey("Given a Content-Range with only size", func() {
			headers := make(http.Header)
			headers.Add("Content-Range", "bytes */100")

			Convey("When calling the function", func() {
				offset, size, err := getContentRange(headers)
				So(err, ShouldBeNil)

				Convey("Then it should return the default offset", func() {
					So(offset, ShouldEqual, 0)
				})

				Convey("Then it should return the correct size", func() {
					So(size, ShouldEqual, 100)
				})
			})
		})
	})
}

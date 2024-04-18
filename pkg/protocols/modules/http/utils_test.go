package http

import (
	"net/http"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
)

func init() {
	pipelinetest.Protocols[HTTP] = pipelinetest.ProtoFeatures{
		MakeClient:        Module{}.NewClient,
		MakeServer:        Module{}.NewServer,
		MakeServerConfig:  Module{}.MakeServerConfig,
		MakeClientConfig:  Module{}.MakeClientConfig,
		MakePartnerConfig: Module{}.MakePartnerConfig,
		TransID:           true, RuleName: true,
	}
	pipelinetest.Protocols[HTTPS] = pipelinetest.ProtoFeatures{
		MakeClient:        ModuleHTTPS{}.NewClient,
		MakeServer:        ModuleHTTPS{}.NewServer,
		MakeServerConfig:  ModuleHTTPS{}.MakeServerConfig,
		MakeClientConfig:  ModuleHTTPS{}.MakeClientConfig,
		MakePartnerConfig: ModuleHTTPS{}.MakePartnerConfig,
		TransID:           true, RuleName: true,
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

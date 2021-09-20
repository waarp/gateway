package r66

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func TestServiceStart(t *testing.T) {
	logger := log.NewLogger("test_r66_start")

	Convey("Given an R66 service", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		server := &model.LocalAgent{
			Name:        "r66_server",
			Protocol:    "r66",
			ProtoConfig: json.RawMessage(`{"blockSize":512,"serverPassword":"c2VzYW1l"}`),
			Address:     "localhost:8066",
		}
		So(db.Insert(server).Run(), ShouldBeNil)

		service := NewService(db, server, logger)

		Convey("When calling the 'Start' function", func() {
			err := service.Start()

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestServiceStop(t *testing.T) {
	logger := log.NewLogger("test_r66_stop")

	Convey("Given a running R66 service", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		server := &model.LocalAgent{
			Name:        "r66_server",
			Protocol:    "r66",
			ProtoConfig: json.RawMessage(`{"blockSize":512,"serverPassword":"c2VzYW1l"}`),
			Address:     "localhost:8067",
		}
		So(db.Insert(server).Run(), ShouldBeNil)

		service := NewService(db, server, logger)
		So(service.Start(), ShouldBeNil)

		Convey("When calling the 'Stop' function", func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := service.Stop(ctx)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

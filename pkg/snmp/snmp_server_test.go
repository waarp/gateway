package snmp

import (
	"testing"
	"time"

	"code.waarp.fr/lib/log"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestServer(t *testing.T) {
	require.NoError(t, logging.AddLogBackend("TRACE", "stdout", "", ""))

	Convey("Testing the SNMP server", t, func(c C) {
		db := database.TestDatabase(c)

		analytics.GlobalService = &analytics.Service{DB: db}
		So(analytics.GlobalService.Start(), ShouldBeNil)

		service := &Service{DB: db}

		dbConfig := ServerConfig{
			LocalUDPAddress: ":1610",
			Community:       "public",
		}
		So(db.Insert(&dbConfig).Run(), ShouldBeNil)

		So(service.Start(), ShouldBeNil)
		Reset(func() {
			ctx, cancel := testhelpers.ContextWithTimeout(5 * time.Second)
			defer cancel()
			So(service.Stop(ctx), ShouldBeNil)
		})

		service.Logger = testhelpers.TestLoggerWithLevel(c, ServiceName, log.LevelTrace)
		service.startTime = time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)

		<-make(chan bool)
	})
}

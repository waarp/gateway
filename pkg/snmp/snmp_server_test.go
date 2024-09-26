//go:build manual_test

package snmp

import (
	"testing"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"

	"code.waarp.fr/lib/log"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestServer(t *testing.T) {
	require.NoError(t, logging.AddLogBackend("TRACE", "stdout", "", ""))

	db := dbtest.TestDatabase(t)

	analytics.GlobalService = &analytics.Service{DB: db}
	require.NoError(t, analytics.GlobalService.Start())

	service := &Service{DB: db}

	dbConfig := ServerConfig{
		LocalUDPAddress: ":1610",
		Community:       "public",
	}
	require.NoError(t, db.Insert(&dbConfig).Run())

	require.NoError(t, service.Start())
	t.Cleanup(func() {
		ctx, cancel := testhelpers.ContextWithTimeout(5 * time.Second)
		defer cancel()
		require.NoError(t, service.Stop(ctx))
	})

	service.Logger = testhelpers.GetTestLoggerWithLevel(t, log.LevelTrace)
	service.startTime = time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)

	<-make(chan bool)
}

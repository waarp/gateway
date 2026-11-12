package as2_test

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/as2"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/as2/internal/as2test"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func TestServerCountsConnections(t *testing.T) {
	const (
		timeout = time.Second
		tick    = 10 * time.Millisecond
	)

	ctx := gwtesting.NewTestServerCtx(t, as2.AS2, map[string]any{
		"maxFileSize": as2test.MaxBodySize,
	})

	analytics.GlobalService = &analytics.Service{DB: ctx.DB}
	t.Cleanup(func() { analytics.GlobalService = nil })

	open := analytics.GlobalService.OpenIncomingConnections.Load
	require.Zero(t, open())

	conn, err := net.Dial("tcp", ctx.Server.Address.String())
	require.NoError(t, err)
	require.Eventually(t, func() bool { return open() == 1 }, timeout, tick,
		"the connection should be counted once accepted")

	require.NoError(t, conn.Close())
	require.Eventually(t, func() bool { return open() == 0 }, timeout, tick,
		"the connection should be counted off once closed")
}

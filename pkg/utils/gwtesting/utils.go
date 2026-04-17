package gwtesting

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

// GetLocalPort returns a local address with an unused port which can be
// used for testing.
func GetLocalPort(tb testing.TB) uint16 {
	tb.Helper()

	addr := GetLocalAddr(tb)

	//nolint:forcetypeassert //no need, the type assertion will always succeed
	return uint16(addr.(*net.TCPAddr).Port)
}

func GetLocalAddr(tb testing.TB) net.Addr {
	tb.Helper()

	//nolint:gosec //this is just for testing
	list, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(tb, err, "Failed to start listener")

	defer require.NoError(tb, list.Close(), "Failed to stop listener")

	return list.Addr()
}

func Addr(tb testing.TB, addr string) types.Address {
	tb.Helper()

	parsed, err := types.NewAddress(addr)
	require.NoError(tb, err)

	return *parsed
}

func requireNoError(tb testing.TB, err *pipeline.Error) {
	tb.Helper()

	if err != nil {
		require.NoError(tb, err)
	}
}

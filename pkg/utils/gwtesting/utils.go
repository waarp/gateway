package gwtesting

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

// GetLocalPort returns a local address with an unused port which can be
// used for testing.
func GetLocalPort(tb testing.TB) uint16 {
	tb.Helper()

	//nolint:gosec //this is just for testing
	list, err := net.Listen("tcp", ":0")
	require.NoError(tb, err, "Failed to start listener")

	defer require.NoError(tb, list.Close(), "Failed to stop listener")

	addr, err := types.NewAddress(list.Addr().String())
	require.NoError(tb, err, "Failed to parse listener address")

	return addr.Port
}

// HashPassword takes the given password and hashes it using the Bcrypt
// algorithm with the minimum allowed cost (for better performance).
func HashPassword(tb testing.TB, password string) string {
	tb.Helper()

	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoErrorf(tb, err, "Failed to hash password %q", password)

	return string(h)
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

package pesit

import (
	"testing"

	"code.waarp.fr/lib/pesit"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

// Ensures Service implements the test server interface.
var _ gwtesting.TestService = &Service{}

func TestNoPassword(t *testing.T) {
	ctx := gwtesting.NewTestServerCtx(t, Pesit, nil)
	ctx.AddPassword(t, "")

	client := pesit.NewClient(ctx.Account.Login, "", ctx.Server.Name)
	client.SetPreConnectionUsage(true)
	client.SetPreConnectPassword("")

	require.NoError(t, client.Dial(ctx.Server.Address.String(), nil))
	defer client.Close(nil)
}

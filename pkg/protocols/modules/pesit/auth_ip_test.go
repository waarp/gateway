package pesit

import (
	"testing"

	"code.waarp.fr/lib/pesit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func TestAuthIPRestriction(t *testing.T) {
	const password = "sesame"

	ctx := gwtesting.NewTestServerCtx(t, Pesit, nil)
	ctx.AddPassword(t, password)

	// The test client connects from the loopback address, which is not allowed.
	ctx.Account.IPAddresses = []string{"1.2.3.4"}
	require.NoError(t, ctx.DB.Update(ctx.Account).Run())

	connect := func(pswd string) error {
		client := pesit.NewClient(ctx.Account.Login, pswd, ctx.Server.Name)
		defer client.ForceClose()

		return client.Dial(ctx.Server.Address.String(), nil)
	}

	errBadPassword := connect("wrongpw") // PeSIT passwords are 8 characters at most
	errBadAddress := connect(password)

	require.Error(t, errBadPassword)
	require.Error(t, errBadAddress)
	assert.Equal(t, errBadPassword.Error(), errBadAddress.Error(),
		"a disallowed address must be refused like invalid credentials")
}

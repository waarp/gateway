package pesit

import (
	"testing"
	"time"

	"code.waarp.fr/lib/pesit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
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

func TestServerFileTS(t *testing.T) {
	ctx := gwtesting.NewTestServerCtx(t, Pesit, map[string]any{
		"compatibilityMode": CompatibilityModeNonStandard,
	})
	ctx.AddPassword(t, "sesame")

	const tz = "America/New_York"
	loc, err := time.LoadLocation(tz)
	require.NoError(t, err)

	aTime := time.Date(2023, 1, 1, 1, 0, 0, 0, loc)
	bTime := time.Date(2024, 2, 2, 2, 0, 0, 0, loc)

	require.NoError(t, ctx.DB.Insert(&model.Task{
		RuleID: ctx.RulePush.ID,
		Chain:  model.ChainPre,
		Rank:   0,
		Type:   tasks.SetInfo,
		Args:   map[string]string{fileTimeZone: tz},
	}).Run())

	client := pesit.NewClient(ctx.Account.Login, "sesame", ctx.Server.Name)
	require.NoError(t, client.Dial(ctx.Server.Address.String(), nil))
	defer client.Close(nil)

	transfer := pesit.NewTransfer(pesit.MethodSend, "test.txt")
	transfer.SetFileLabelPI37("test.txt")
	transfer.SetFilenamePI12(ctx.RulePush.Name)
	transfer.SetTransferID(1)
	transfer.SetLastAccess(aTime)
	transfer.SetCreationDate(bTime)

	require.NoError(t, client.SelectFile(transfer))
	require.NoError(t, transfer.OpenFile())
	require.NoError(t, transfer.CloseFile(nil))
	require.NoError(t, transfer.DeselectFile(nil))

	var trans model.NormalizedTransferView
	require.NoError(t, ctx.DB.Get(&trans, "dest_filename=?", "test.txt").Eager().Run())

	assert.Equal(t, aTime.Format(time.RFC3339), trans.TransferInfo[fileLastAccess])
	assert.Equal(t, bTime.Format(time.RFC3339), trans.TransferInfo[fileCreation])
}

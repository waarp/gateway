package pesit

import (
	"strings"
	"testing"

	"code.waarp.fr/lib/pesit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func TestGetRuleByNameNotFound(t *testing.T) {
	db := gwtesting.Database(t)
	handler := &transferHandler{db: db, logger: gwtesting.Logger(t)}

	_, err := handler.getRuleByName("does-not-exist", true)
	require.Error(t, err)

	var diag pesit.Diagnostic

	require.ErrorAs(t, err, &diag)

	// An unknown rule is a mistake of the caller, not a failure of the
	// server: the diagnostic must say so, rather than "database error".
	assert.Equal(t, pesit.CodeParameterError, diag.GetCode())
}

func TestMultiTransfer(t *testing.T) {
	// ########## SETUP ##########
	const (
		login    = "pesit-test-account"
		password = "sesame"
	)
	ctx := gwtesting.NewTestServerCtx(t, Pesit, nil)
	ctx.AddPassword(t, password)

	client := pesit.NewClient(login, password, ctx.Server.Name)
	require.NoError(t, client.Dial(ctx.Server.Address.String(), nil))
	defer client.ForceClose()

	// ########## TRANSFERS SETUP ##########
	trans1 := pesit.NewTransfer(pesit.MethodSend, ctx.RulePush.Path+"/trans1.txt")
	trans2 := pesit.NewTransfer(pesit.MethodSend, ctx.RulePush.Path+"/trans2.txt")
	trans1.SetTransferID(1)
	trans2.SetTransferID(2)
	file1 := strings.NewReader("file1 content")
	file2 := strings.NewReader("file2 content")

	// ########## TRANSFER 1 ##########
	require.NoError(t, client.SelectFile(trans1))
	require.NoError(t, trans1.OpenFile())

	_, err1 := file1.WriteTo(trans1)
	require.NoError(t, err1)

	require.NoError(t, trans1.CloseFile(nil))
	require.NoError(t, trans1.DeselectFile(nil))

	// ########## TRANSFER 2 ##########
	require.NoError(t, client.SelectFile(trans2))
	require.NoError(t, trans2.OpenFile())

	_, err2 := file2.WriteTo(trans2)
	require.NoError(t, err2)

	require.NoError(t, trans2.CloseFile(nil))
	require.NoError(t, trans2.DeselectFile(nil))
}

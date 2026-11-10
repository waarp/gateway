package tasks

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// The dates of the file announced by a partner (#CREATIONTIMESTAMP#,
// #LASTACCESSTIMESTAMP#) keep their wall clock whatever the zone of the
// gateway: run these tests with several TZ values to check it.
func TestFileDateVariables(t *testing.T) {
	t.Parallel()

	ctx := getExecTransCtx(t)
	ctx.Transfer.TransferInfo[model.FileCreationDate] = "2026-11-03T23:13:11"
	ctx.Transfer.TransferInfo[model.FileLastAccessDate] = "2026-11-04T01:02:03"

	for _, tc := range []struct{ in, out string }{
		{"#CREATIONTIMESTAMP(YYYYMMDD)#_#CREATIONTIMESTAMP(HHmmss)#00", "20261103_23131100"},
		{"#LASTACCESSTIMESTAMP(YYYY-MM-DD HH:mm:ss)#", "2026-11-04 01:02:03"},
		{"#CREATIONTIMESTAMP#", "2026-11-03_231311"},
	} {
		got, err := replaceVars(tc.in, ctx)
		require.NoError(t, err, tc.in)
		assert.Equal(t, tc.out, got, tc.in)
	}
}

func TestFileDateVariablesWithoutDate(t *testing.T) {
	t.Parallel()

	got, err := replaceVars("a_#CREATIONTIMESTAMP(YYYYMMDD)#_#LASTACCESSTIMESTAMP#_b", getExecTransCtx(t))
	require.NoError(t, err)
	assert.Equal(t, "a___b", got, "a date the partner did not send is empty")
}

func TestFileDateVariablesMalformed(t *testing.T) {
	t.Parallel()

	for _, val := range []any{"03/11/2026 23:13:11", 1234} {
		ctx := getExecTransCtx(t)
		ctx.Transfer.TransferInfo[model.FileCreationDate] = val

		_, err := replaceVars("#CREATIONTIMESTAMP(YYYYMMDD)#", ctx)
		require.ErrorIs(t, err, errBadFileDate, "value %v", val)
		assert.ErrorContains(t, err, model.FileCreationDate)
	}
}

func TestFileDateEnvVars(t *testing.T) {
	t.Parallel()

	ctx := getExecTransCtx(t)
	ctx.Transfer.TransferInfo[model.FileCreationDate] = "2026-11-03T23:13:11"
	ctx.Transfer.TransferInfo[model.FileLastAccessDate] = "malformed"

	cmd, err := makeCmd(ctx, shellCmd(), shellCmdArgs(`'echo'`))
	require.NoError(t, err, "a malformed date does not prevent the script from running")

	assert.Contains(t, cmd.Env, "WAARP_CREATIONTIMESTAMP=2026-11-03T23:13:11",
		"ISO 8601 without an offset: the date has no zone")
	assert.False(t, slices.ContainsFunc(cmd.Env, func(e string) bool {
		return strings.HasPrefix(e, "WAARP_LASTACCESSTIMESTAMP=")
	}), "a malformed date is left out of the environment")
}

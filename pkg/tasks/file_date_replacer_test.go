package tasks

import (
	"slices"
	"strings"
	"testing"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// The dates of the file announced by a partner (#CREATIONTIMESTAMP#,
// #LASTACCESSTIMESTAMP#) keep their wall clock whatever the zone of the
// gateway: run these tests with several TZ values to check it.
func TestFileDateVariables(t *testing.T) {
	t.Parallel()

	creationTS := time.Date(2026, time.November, 3, 23, 13, 11, 0, time.Local)
	lastModTS := time.Date(2026, time.November, 4, 1, 2, 3, 0, time.Local)
	lastAccTS := time.Date(2026, time.November, 4, 10, 45, 6, 0, time.Local)

	ctx := getExecTransCtx(t)
	ctx.Transfer.TransferInfo[model.InfoFileBTime] = creationTS.Format(time.RFC3339)
	ctx.Transfer.TransferInfo[model.InfoFileMTime] = lastModTS.Format(time.RFC3339)
	ctx.Transfer.TransferInfo[model.InfoFileATime] = lastAccTS.Format(time.RFC3339)

	for _, tc := range []struct{ in, out string }{{
		"#ORIGINALFILEBTIME(YYYYMMDD)#_#ORIGINALFILEBTIME(HHmmss)#00",
		creationTS.Format("20060102_15040500"),
	}, {
		"#ORIGINALFILEMTIME(YYYY-MM-DD HH:mm:ss)#",
		lastModTS.Format("2006-01-02 15:04:05"),
	}, {
		"#ORIGINALFILEATIME(YYYY-MM-DD HH:mm:ss)#",
		lastAccTS.Format("2006-01-02 15:04:05"),
	}, {
		"#ORIGINALFILEBTIME#",
		creationTS.Format("2006-01-02_150405"),
	}} {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			got, err := replaceVars(tc.in, ctx)
			require.NoError(t, err, tc.in)
			assert.Equal(t, tc.out, got, tc.in)
		})
	}
}

func TestFileDateVariablesWithoutDate(t *testing.T) {
	t.Parallel()

	got, err := replaceVars(
		"a_#ORIGINALFILEBTIME(YYYYMMDD)#_#ORIGINALFILEMTIME#_b",
		getExecTransCtx(t))
	require.NoError(t, err)
	assert.Equal(t, "a___b", got, "a date the partner did not send is empty")
}

func TestFileDateVariablesMalformed(t *testing.T) {
	t.Parallel()

	for _, val := range []any{"03/11/2026 23:13:11", 1234} {
		ctx := getExecTransCtx(t)
		ctx.Transfer.TransferInfo[model.InfoFileBTime] = val

		_, err := replaceVars("#ORIGINALFILEBTIME(YYYYMMDD)#", ctx)
		require.ErrorIs(t, err, utils.ErrIncorrectValueType, "value %v", val)
		assert.ErrorContains(t, err, model.InfoFileBTime)
	}
}

func TestFileDateEnvVars(t *testing.T) {
	t.Parallel()

	creationTS := time.Date(2026, 11, 3, 23, 13, 11, 0, time.Local)

	ctx := getExecTransCtx(t)
	ctx.Transfer.TransferInfo[model.InfoFileBTime] = creationTS.Format(time.RFC3339)
	ctx.Transfer.TransferInfo[model.InfoFileMTime] = "malformed"

	cmd, err := makeCmd(ctx, shellCmd(), shellCmdArgs(`'echo'`))
	require.NoError(t, err, "a malformed date does not prevent the script from running")

	assert.Contains(t, cmd.Env,
		"WAARP_ORIGINALFILEBTIME="+creationTS.Format(time.RFC3339),
		"ISO 8601")
	assert.False(t, slices.ContainsFunc(cmd.Env, func(e string) bool {
		return strings.HasPrefix(e, "WAARP_ORIGINALFILELASTMODTIMESTAMP=")
	}), "a malformed date is left out of the environment")
}

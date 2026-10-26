package transfers

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// keys is a translator that gives the key back, for the tests of what the
// other screens share.
func keys(key string, _ ...any) string { return key }

// TestLiveActions checks the menu the dashboard shows on a live failed
// transfer: the rules of the list, every action coming back to the dashboard.
func TestLiveActions(t *testing.T) {
	t.Parallel()

	client := &model.Transfer{
		Identifier: model.ID(42), Status: types.StatusError,
		ClientID: sql.NullInt64{Int64: 1, Valid: true}, RemoteAccountID: sql.NullInt64{Int64: 1, Valid: true},
	}
	server := &model.Transfer{
		Identifier: model.ID(43), Status: types.StatusError, LocalAccountID: sql.NullInt64{Int64: 1, Valid: true},
	}

	acts := LiveActions(keys, client, true)
	require.Len(t, acts, 4, "the sheet, resume, retry, cancel")
	assert.Equal(t, "transfers.action.detail", acts[0].Label)
	assert.Equal(t, "/webui/transfers/42", acts[0].URL)
	assert.Equal(t, "/webui/transfers/42/resume?from=dashboard", acts[1].Action, "a POST, back to the dashboard")
	assert.Equal(t, "/webui/transfers/42/retry?from=dashboard", acts[2].URL)
	assert.Equal(t, "/webui/transfers/42/cancel?from=dashboard", acts[3].URL)
	assert.True(t, acts[3].Danger)

	acts = LiveActions(keys, server, true)
	require.Len(t, acts, 2, "a server transfer neither resumes nor retries from the gateway")
	assert.Equal(t, "/webui/transfers/43/cancel?from=dashboard", acts[1].URL)

	acts = LiveActions(keys, client, false)
	require.Len(t, acts, 1, "the sheet alone without the right to write")

	child := &model.Transfer{
		Identifier: model.ID(44), Status: types.StatusError,
		ClientID: sql.NullInt64{Int64: 1, Valid: true}, RemoteAccountID: sql.NullInt64{Int64: 1, Valid: true},
		TransferInfo: map[string]any{model.SyncTransferID: int64(40)},
	}
	for _, act := range LiveActions(keys, child, true) {
		assert.NotContains(t, act.Action, "/resume", "the child of a synchronous transfer resumes through its parent")
	}
}

// TestFilteredListURL checks the lists another screen leads to: a state, a
// period from a moment, both said explicitly.
func TestFilteredListURL(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "/webui/transfers?state=error", FilteredListURL(StateError, time.Time{}))
	assert.Equal(t, "/webui/transfers?from=2026-10-20T17%3A00&state=done",
		FilteredListURL(StateDone, time.Date(2026, 10, 20, 17, 0, 0, 0, time.Local)))
	assert.Equal(t, "/webui/transfers?reset=1", FilteredListURL("", time.Time{}), "no criterion: the list reset")
}

// TestFileNameOf checks the name of a file: the file on the gateway when it is
// known, else the name at the source, else at the destination.
func TestFileNameOf(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "releve.pdf", FileNameOf(`C:\data\in\releve.pdf`, "RELEVE", "releve-dest.pdf"))
	assert.Equal(t, "paie.xml", FileNameOf("", "/out/paie.xml", "paie-dest.xml"))
	assert.Equal(t, "dest.csv", FileNameOf("", "", "dest.csv"))
	assert.Equal(t, "", FileNameOf("", "", ""))
}

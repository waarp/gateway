package transfers

import (
	"database/sql"
	"net/url"
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
// period from a moment, a role of the gateway, all said explicitly.
func TestFilteredListURL(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "/webui/transfers?state=error", FilteredListURL(StateError, time.Time{}))
	assert.Equal(t, "/webui/transfers?from=2026-10-20T17%3A00&state=done",
		FilteredListURL(StateDone, time.Date(2026, 10, 20, 17, 0, 0, 0, time.Local)))
	assert.Equal(t, "/webui/transfers?reset=1", FilteredListURL("", time.Time{}), "no criterion: the list reset")
	assert.Equal(t, "/webui/transfers?role=client&state=error", RoleListURL(StateError, RoleClient), "a state and a role (#837)")
	assert.Equal(t, "/webui/transfers?role=server", RoleListURL("", RoleServer))
}

// TestInfoNames checks the names of the rows of transfer information: a key
// is kept as typed, trimmed, its case counting (a reserved key is its name,
// __key__, as the documentation writes it); the rows of a task come by key.
func TestInfoNames(t *testing.T) {
	t.Parallel()

	pairs := ReadInfoPairs(url.Values{
		"info_name": {model.FollowID, " lot ", "FollowID"}, "info_value": {"1", "A", "x"},
	})
	assert.Equal(t, []InfoPair{
		{Name: model.FollowID, Value: "1"}, {Name: "lot", Value: "A"}, {Name: "FollowID", Value: "x"},
	}, pairs)

	assert.Equal(t, []InfoPair{{Name: model.FollowID, Value: "1"}, {Name: "lot", Value: "A"}},
		PairRows(map[string]string{"lot": "A", model.FollowID: "1"}), "by key")
}

// TestSelfReference checks the value that reads an information the same rows
// set (#821): itself, another row, a reserved key; a key the rows do not set
// is fine, and so is a spelling the tasks would not read.
func TestSelfReference(t *testing.T) {
	t.Parallel()

	row, ref, found := SelfReference([]InfoPair{{Name: "a", Value: "#TI_b# et #TRUEFILENAME#"}, {Name: "b", Value: "1"}})
	assert.True(t, found)
	assert.Equal(t, 0, row)
	assert.Equal(t, "b", ref)

	row, ref, found = SelfReference([]InfoPair{{Name: "lot", Value: "x"}, {Name: "a", Value: "#TI_a#"}})
	assert.True(t, found)
	assert.Equal(t, 1, row)
	assert.Equal(t, "a", ref, "itself")

	row, ref, found = SelfReference([]InfoPair{{Name: model.FollowID, Value: "1"}, {Name: "a", Value: "#TI___followID__#"}})
	assert.True(t, found)
	assert.Equal(t, 1, row)
	assert.Equal(t, model.FollowID, ref, "a reserved key, as the marker and the row write it")

	_, _, found = SelfReference([]InfoPair{{Name: "a", Value: "#TI_lot# #TI___ackExpected__#"}, {Name: "b", Value: "#DATE#"}})
	assert.False(t, found, "keys the rows do not set")

	_, _, found = SelfReference([]InfoPair{{Name: model.FollowID, Value: "1"}, {Name: "a", Value: "#TI_followID#"}})
	assert.False(t, found, "a spelling the tasks do not read as the key")
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

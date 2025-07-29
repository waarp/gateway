package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func TestTransferListFilePattern(t *testing.T) {
	db := dbtest.TestDatabase(t)

	hist1 := &model.HistoryEntry{
		ID:               123,
		RemoteTransferID: "abc",
		IsServer:         false,
		IsSend:           false,
		Rule:             "send",
		Account:          "foo",
		Agent:            "bar",
		Client:           "baz",
		Protocol:         "http",
		SrcFilename:      "/test/source_f%le.txt",
		DestFilename:     "/test/dest_file.txt",
		LocalPath:        "/full/local/path",
		RemotePath:       "/full/remote/path",
		Filesize:         1000,
		Start:            time.Date(2025, 1, 1, 1, 0, 0, 0, time.UTC),
		Stop:             time.Date(2025, 1, 1, 2, 0, 0, 0, time.UTC),
		Status:           types.StatusDone,
		Progress:         1000,
	}
	require.NoError(t, db.Insert(hist1).Run())

	hist2 := &model.HistoryEntry{
		ID:               456,
		RemoteTransferID: "def",
		IsServer:         false,
		IsSend:           false,
		Rule:             "send",
		Account:          "foo",
		Agent:            "bar",
		Client:           "baz",
		Protocol:         "http",
		SrcFilename:      "/test/source_file.txt",
		DestFilename:     "/test/dest_file.txt",
		LocalPath:        "/full/local/path",
		RemotePath:       "/full/remote/path",
		Filesize:         1000,
		Start:            time.Date(2025, 1, 1, 1, 0, 0, 0, time.UTC),
		Stop:             time.Date(2025, 1, 1, 2, 0, 0, 0, time.UTC),
		Status:           types.StatusDone,
		Progress:         1000,
	}
	require.NoError(t, db.Insert(hist2).Run())

	query := StartTransferQuery(db, "id", true)
	query.FilePattern("/test/*_f%le.t?t")

	result, err := query.Run()
	require.NoError(t, err)

	require.Len(t, result, 1)
	require.Equal(t, result[0].ID, hist1.ID)
}

package pesit

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"code.waarp.fr/lib/pesit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func TestAnnouncedInfos(t *testing.T) {
	t.Parallel()

	t.Run("every attribute", func(t *testing.T) {
		t.Parallel()

		a := announced{
			callerID:      "        USER0001",
			serverID:      "APPLI2  USER2",
			priority:      pesit.PriorityUrgent,
			keyLength:     8,
			keyOffset:     4,
			reservedSpace: 196,
			reservedUnit:  pesit.UnitKB,
			// The library labels the dates UTC: their wall clock is kept.
			creation:   time.Date(2026, 11, 3, 23, 13, 11, 0, time.UTC),
			lastAccess: time.Date(2026, 11, 4, 1, 2, 3, 0, time.UTC),
		}

		assert.Equal(t, map[string]any{
			fileCallerIDKey:          "        USER0001",
			fileServerIDKey:          "APPLI2  USER2",
			priorityKey:              "Urgent",
			keyLengthKey:             uint16(8),
			keyOffsetKey:             uint16(4),
			reservedSpaceKey:         uint32(196),
			reservedSpaceUnitKey:     "KB",
			model.FileCreationDate:   "2026-11-03T23:13:11",
			model.FileLastAccessDate: "2026-11-04T01:02:03",
		}, a.infos())
	})

	t.Run("nothing sent", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, (&announced{}).infos())
		assert.Empty(t, (&announced{priority: pesit.PriorityNormal}).infos(),
			"the normal priority is the default of the protocol")
	})

	t.Run("space reserved in articles", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, map[string]any{reservedSpaceKey: uint32(12), reservedSpaceUnitKey: "Article"},
			(&announced{reservedSpace: 12, reservedUnit: pesit.UnitArticle}).infos())
	})
}

func TestUnlessSame(t *testing.T) {
	t.Parallel()

	assert.Empty(t, unlessSame("LOGIN", "LOGIN"))
	assert.Equal(t, "APPLI", unlessSame("APPLI", "LOGIN"))
	assert.Empty(t, unlessSame("", "LOGIN"))
}

// A gateway that receives a file keeps the file type and the creation date the
// sender announced, and the gateway that retrieves a file keeps those of the
// server's answer.
func TestAnnouncedAttributesKept(t *testing.T) {
	db := gwtesting.Database(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit,
		&ServerConfig{CompatibilityMode: CompatibilityModeNonStandard},
		nil,
		&PartnerConfig{CompatibilityMode: CompatibilityModeNonStandard},
	)

	t.Run("Given a PESIT push", func(t *testing.T) {
		serverPushTrans := &model.Transfer{
			RuleID:         ctx.ServerRulePush.ID,
			LocalAccountID: ctx.LocalAccount.NullableID(),
			DestFilename:   ctx.TransferPush.DestFilename,
			Start:          time.Date(9999, 1, 1, 1, 0, 0, 0, time.UTC),
			Status:         types.StatusAvailable,
			Filesize:       model.UnknownSize,
		}
		require.NoError(t, db.Insert(serverPushTrans).Run())

		ctx.TransferPush.TransferInfo[fileTypeKey] = uint16(13)
		pip := ctx.PushPipeline(t)

		require.NoError(t, pip.Run())
		ctx.CheckPushTransferOK(t)

		var client, server model.HistoryEntry
		require.NoError(t, db.Get(&client, "id=?", ctx.TransferPush.ID).Eager().Run())
		require.NoError(t, db.Get(&server, "id=?", serverPushTrans.ID).Eager().Run())

		stat, err := os.Stat(client.LocalPath)
		require.NoError(t, err)

		assert.Equal(t, json.Number("13"), server.TransferInfo[fileTypeKey],
			"the file type of the CREATE reaches the server")
		assert.Equal(t, stat.ModTime().Format(model.FileDateLayout), server.TransferInfo[model.FileCreationDate],
			"the creation date is the modification time of the sent file, as its wall clock")
		assert.Equal(t, json.Number("1"), server.TransferInfo[reservedSpaceKey])
		assert.Equal(t, "KB", server.TransferInfo[reservedSpaceUnitKey])
		assert.NotContains(t, server.TransferInfo, fileCallerIDKey, "no file identifier sent")
		assert.NotContains(t, server.TransferInfo, priorityKey, "normal priority")
	})

	t.Run("Given a PESIT pull", func(t *testing.T) {
		serverPullTrans := &model.Transfer{
			RuleID:         ctx.ServerRulePull.ID,
			LocalAccountID: ctx.LocalAccount.NullableID(),
			SrcFilename:    ctx.TransferPull.SrcFilename,
			Start:          time.Date(9999, 1, 1, 1, 0, 0, 0, time.UTC),
			Status:         types.StatusAvailable,
		}
		require.NoError(t, db.Insert(serverPullTrans).Run())

		pip := ctx.PullPipeline(t)

		require.NoError(t, pip.Run())
		ctx.CheckPullTransferOK(t)

		var client, server model.HistoryEntry
		require.NoError(t, db.Get(&client, "id=?", ctx.TransferPull.ID).Eager().Run())
		require.NoError(t, db.Get(&server, "id=?", serverPullTrans.ID).Eager().Run())

		stat, err := os.Stat(server.LocalPath)
		require.NoError(t, err)

		assert.Equal(t, stat.ModTime().Format(model.FileDateLayout), client.TransferInfo[model.FileCreationDate],
			"the creation date of the answer reaches the client")
		assert.Equal(t, "KB", client.TransferInfo[reservedSpaceUnitKey])
		assert.NotContains(t, client.TransferInfo, fileCallerIDKey,
			"the server echoes the identifiers of the connection: nothing new to keep")
	})
}

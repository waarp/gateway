package model

import (
	"testing"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestNormalizedTransferCreateView(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		push := &Rule{Name: "push", IsSend: true, Path: "/push"}
		pull := &Rule{Name: "pull", IsSend: false, Path: "/pull"}
		serv := &LocalAgent{
			Name: "serv", Protocol: testProtocol,
			Address: types.Addr("localhost", 1234),
		}
		clie := &Client{Name: "cli", Protocol: testProtocol}
		part := &RemoteAgent{
			Name: "part", Protocol: testProtocol,
			Address: types.Addr("localhost", 5678),
		}

		So(db.Insert(push).Run(), ShouldBeNil)
		So(db.Insert(pull).Run(), ShouldBeNil)
		So(db.Insert(serv).Run(), ShouldBeNil)
		So(db.Insert(clie).Run(), ShouldBeNil)
		So(db.Insert(part).Run(), ShouldBeNil)

		locAcc := &LocalAccount{LocalAgentID: serv.ID, Login: "toto"}
		remAcc := &RemoteAccount{RemoteAgentID: serv.ID, Login: "tata"}

		So(db.Insert(locAcc).Run(), ShouldBeNil)
		So(db.Insert(remAcc).Run(), ShouldBeNil)

		trans1 := &Transfer{
			RuleID:         push.ID,
			LocalAccountID: utils.NewNullInt64(locAcc.ID),
			SrcFilename:    "file1",
		}
		trans2 := &Transfer{
			RuleID:          pull.ID,
			ClientID:        utils.NewNullInt64(clie.ID),
			RemoteAccountID: utils.NewNullInt64(remAcc.ID),
			SrcFilename:     "file2",
		}
		hist := &HistoryEntry{
			ID:               3,
			RemoteTransferID: "123456",
			IsServer:         true,
			IsSend:           false,
			Rule:             "default",
			Account:          "tutu",
			Agent:            "server",
			Protocol:         testProtocol,
			DestFilename:     "file3",
			LocalPath:        localPath(testLocalPath),
			Filesize:         1234,
			Start:            time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC),
			Stop:             time.Date(2021, 1, 1, 2, 0, 0, 0, time.UTC),
			Status:           types.StatusDone,
			Step:             types.StepNone,
			Progress:         1234,
		}

		So(db.Insert(trans1).Run(), ShouldBeNil)
		So(db.Insert(trans2).Run(), ShouldBeNil)
		So(db.Insert(hist).Run(), ShouldBeNil)

		Convey("When trying to retrieve entries from the normalized_transfers view", func() {
			var norm NormalizedTransfers

			So(db.Select(&norm).OrderBy("id", true).Run(), ShouldBeNil)
			So(norm, ShouldHaveLength, 3)

			Convey("Then it should have returned both the transfer & history entries", func() {
				So(norm[0].ID, ShouldEqual, 1)
				So(norm[0].IsServer, ShouldBeTrue)
				So(norm[0].IsSend, ShouldEqual, push.IsSend)
				So(norm[0].Rule, ShouldEqual, push.Name)
				So(norm[0].Account, ShouldEqual, locAcc.Login)
				So(norm[0].Agent, ShouldEqual, serv.Name)
				So(norm[0].IsTransfer, ShouldBeTrue)

				So(norm[1].ID, ShouldEqual, 2)
				So(norm[1].IsServer, ShouldBeFalse)
				So(norm[1].IsSend, ShouldEqual, pull.IsSend)
				So(norm[1].Rule, ShouldEqual, pull.Name)
				So(norm[1].Client, ShouldEqual, clie.Name)
				So(norm[1].Account, ShouldEqual, remAcc.Login)
				So(norm[1].Agent, ShouldEqual, part.Name)
				So(norm[1].IsTransfer, ShouldBeTrue)

				So(norm[2].ID, ShouldEqual, 3)
				So(norm[2].IsServer, ShouldBeTrue)
				So(norm[2].IsSend, ShouldBeFalse)
				So(norm[2].Rule, ShouldEqual, hist.Rule)
				So(norm[2].Account, ShouldEqual, hist.Account)
				So(norm[2].Agent, ShouldEqual, hist.Agent)
				So(norm[2].IsTransfer, ShouldBeFalse)
			})
		})
	})
}

func TestNormalizedTransferResume(t *testing.T) {
	db := dbtest.TestDatabase(t)

	rule := Rule{Name: "rule", IsSend: true}
	require.NoError(t, db.Insert(&rule).Run())

	client := Client{Name: "client", Protocol: testProtocol}
	require.NoError(t, db.Insert(&client).Run())

	partner := RemoteAgent{Name: "partner", Protocol: testProtocol, Address: types.Addr("localhost", 0)}
	require.NoError(t, db.Insert(&partner).Run())

	remAccount := RemoteAccount{RemoteAgentID: partner.ID, Login: "toto"}
	require.NoError(t, db.Insert(&remAccount).Run())

	server := LocalAgent{Name: "server", Protocol: testProtocol, Address: types.Addr("localhost", 0)}
	require.NoError(t, db.Insert(&server).Run())

	locAccount := LocalAccount{LocalAgentID: server.ID, Login: "toto"}
	require.NoError(t, db.Insert(&locAccount).Run())

	t.Run("Nominal case", func(t *testing.T) {
		t.Parallel()

		trans := &Transfer{
			RuleID:          rule.ID,
			ClientID:        utils.NewNullInt64(client.ID),
			RemoteAccountID: utils.NewNullInt64(remAccount.ID),
			SrcFilename:     "file.txt",
			Status:          types.StatusError,
			ErrCode:         types.TeUnknown,
			ErrDetails:      "test error",
		}
		require.NoError(t, db.Insert(trans).Run())

		var original NormalizedTransferView
		require.NoError(t, db.Get(&original, "id=?", trans.ID).Run())

		actual := utils.Clone(&original)
		when := time.Now()
		require.NoError(t, actual.Resume(db, when))

		assert.Equal(t, types.StatusPlanned, actual.Status)
		assert.Equal(t, when, actual.NextRetry)
		assert.Equal(t, types.TeOk, actual.ErrCode)
		assert.Empty(t, actual.ErrDetails)
	})

	t.Run("Done transfer", func(t *testing.T) {
		t.Parallel()

		hist := &HistoryEntry{
			ID:               1000,
			RemoteTransferID: "123456",
			IsServer:         true,
			IsSend:           false,
			Rule:             "default",
			Account:          "tutu",
			Agent:            "server",
			Protocol:         testProtocol,
			DestFilename:     "file.txt",
			Status:           types.StatusDone,
			Start:            time.Date(2025, 1, 1, 1, 0, 0, 0, time.UTC),
			Stop:             time.Date(2025, 1, 1, 2, 0, 0, 0, time.UTC),
		}
		require.NoError(t, db.Insert(hist).Run())

		var original NormalizedTransferView
		require.NoError(t, db.Get(&original, "id=?", hist.ID).Run())

		actual := utils.Clone(&original)
		when := time.Now()
		require.ErrorIs(t, actual.Resume(db, when), ErrResumeDone)
		assert.Equal(t, &original, actual)
	})
}

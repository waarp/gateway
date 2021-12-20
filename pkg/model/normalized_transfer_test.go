package model

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

func TestNormalizedTransferCreateView(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		push := &Rule{Name: "push", IsSend: true, Path: "/push"}
		pull := &Rule{Name: "pull", IsSend: false, Path: "/pull"}
		serv := &LocalAgent{Name: "serv", Protocol: testProtocol, Address: "localhost:1234"}
		part := &RemoteAgent{Name: "part", Protocol: testProtocol, Address: "localhost:5678"}

		So(db.Insert(push).Run(), ShouldBeNil)
		So(db.Insert(pull).Run(), ShouldBeNil)
		So(db.Insert(serv).Run(), ShouldBeNil)
		So(db.Insert(part).Run(), ShouldBeNil)

		locAcc := &LocalAccount{LocalAgentID: serv.ID, Login: "toto"}
		remAcc := &RemoteAccount{RemoteAgentID: serv.ID, Login: "tata"}

		So(db.Insert(locAcc).Run(), ShouldBeNil)
		So(db.Insert(remAcc).Run(), ShouldBeNil)

		trans1 := &Transfer{
			RuleID:         push.ID,
			LocalAccountID: utils.NewNullInt64(locAcc.ID),
			LocalPath:      "lFile1",
			RemotePath:     "rFile1",
		}
		trans2 := &Transfer{
			RuleID:          pull.ID,
			RemoteAccountID: utils.NewNullInt64(remAcc.ID),
			LocalPath:       "lFile2",
			RemotePath:      "rFile2",
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
			LocalPath:        "lFile3",
			RemotePath:       "rFile3",
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
				So(norm[0].IsSend, ShouldBeTrue)
				So(norm[0].Rule, ShouldEqual, "push")
				So(norm[0].Account, ShouldEqual, "toto")
				So(norm[0].Agent, ShouldEqual, "serv")
				So(norm[0].IsTransfer, ShouldBeTrue)

				So(norm[1].ID, ShouldEqual, 2)
				So(norm[1].IsServer, ShouldBeFalse)
				So(norm[1].IsSend, ShouldBeFalse)
				So(norm[1].Rule, ShouldEqual, "pull")
				So(norm[1].Account, ShouldEqual, "tata")
				So(norm[1].Agent, ShouldEqual, "part")
				So(norm[1].IsTransfer, ShouldBeTrue)

				So(norm[2].ID, ShouldEqual, 3)
				So(norm[2].IsServer, ShouldBeTrue)
				So(norm[2].IsSend, ShouldBeFalse)
				So(norm[2].Rule, ShouldEqual, "default")
				So(norm[2].Account, ShouldEqual, "tutu")
				So(norm[2].Agent, ShouldEqual, "server")
				So(norm[2].IsTransfer, ShouldBeFalse)
			})
		})
	})
}

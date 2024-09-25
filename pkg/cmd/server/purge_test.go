package wgd

import (
	"runtime"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocolstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used to ease the tests
func init() {
	protocols.Register(testProtocol, protocolstest.TestModule{})
}

func TestPurgeCommand(t *testing.T) {
	Convey("Given a database with some transfers", t, func(c C) {
		db := database.TestDatabase(c)

		t0 := &model.HistoryEntry{
			ID:               1,
			RemoteTransferID: "000",
			Rule:             "foobar",
			Client:           "cli",
			Account:          "foo",
			Agent:            "bar",
			Protocol:         testProtocol,
			LocalPath:        mkPath("/loc_path"),
			RemotePath:       "/rem_path",
			Start:            time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC),
			Status:           types.StatusCancelled,
		}
		So(db.Insert(t0).Run(), ShouldBeNil)

		t2 := &model.HistoryEntry{
			ID:               2,
			RemoteTransferID: "123",
			Rule:             "foobar",
			Client:           "cli",
			Account:          "foo",
			Agent:            "bar",
			Protocol:         testProtocol,
			LocalPath:        mkPath("/loc_path"),
			RemotePath:       "/rem_path",
			Start:            time.Date(2022, 1, 1, 1, 0, 0, 0, time.UTC),
			Stop:             time.Date(2022, 1, 1, 2, 0, 0, 0, time.UTC),
			Status:           types.StatusDone,
		}
		So(db.Insert(t2).Run(), ShouldBeNil)

		t3 := &model.HistoryEntry{
			ID:               3,
			RemoteTransferID: "456",
			Rule:             "foobar",
			Client:           "cli",
			Account:          "foo",
			Agent:            "bar",
			Protocol:         testProtocol,
			LocalPath:        mkPath("/loc_path"),
			RemotePath:       "/rem_path",
			Start:            time.Date(2022, 1, 1, 3, 0, 0, 0, time.UTC),
			Stop:             time.Date(2022, 1, 1, 4, 0, 0, 0, time.UTC),
			Status:           types.StatusDone,
		}
		So(db.Insert(t3).Run(), ShouldBeNil)

		command := &PurgeCommand{}
		out := &strings.Builder{}

		Convey("When purging the history", func() {
			Convey("When confirming the purge", func() {
				in := strings.NewReader("YES")
				So(command.run(db, time.Now(), in, out), ShouldBeNil)

				Convey("Then it should say that the history was purged", func() {
					So(out.String(), ShouldEqual, "You are about to purge 3 history entries.\n"+
						"This operation cannot be undone. Do you wish to proceed anyway ?\n"+
						"\n"+
						"(Type 'YES' in all caps to proceed): \n"+
						"\n"+
						"The transfer history has been purged successfully.\n")
				})

				Convey("Then it should have purged the history", func() {
					var history model.HistoryEntries

					So(db.Select(&history).Run(), ShouldBeNil)
					So(history, ShouldBeEmpty)
				})
			})

			Convey("When aborting the purge", func() {
				in := strings.NewReader("NO")
				So(command.run(db, time.Now(), in, out), ShouldBeNil)

				Convey("Then it should say that the history was NOT purged", func() {
					So(out.String(), ShouldEqual, "You are about to purge 3 history entries.\n"+
						"This operation cannot be undone. Do you wish to proceed anyway ?\n"+
						"\n"+
						"(Type 'YES' in all caps to proceed): \n"+
						"\n"+
						"Purge aborted.\n")
				})

				Convey("Then it should NOT have purged the history", func() {
					var history model.HistoryEntries

					So(db.Select(&history).Run(), ShouldBeNil)
					So(history, ShouldHaveLength, 3)
				})
			})
		})

		Convey("When purging the history with a time limit", func() {
			in := strings.NewReader("YES")

			Convey("When purging the history with a date limit", func() {
				command.OlderThan = t2.Stop.Add(time.Second).Local().Format(untilFormat)
				So(command.run(db, time.Now(), in, out), ShouldBeNil)

				Convey("Then it should say that the history was purged", func() {
					So(out.String(), ShouldEqual, "You are about to purge 2 history entries.\n"+
						"This operation cannot be undone. Do you wish to proceed anyway ?\n"+
						"\n"+
						"(Type 'YES' in all caps to proceed): \n"+
						"\n"+
						"The transfer history has been purged successfully.\n")
				})

				Convey("Then it should have purged the selected history entries", func() {
					var history model.HistoryEntries

					So(db.Select(&history).Run(), ShouldBeNil)
					So(history, ShouldHaveLength, 1)
					So(history[0].ID, ShouldEqual, t3.ID)
				})
			})

			Convey("When purging the history with a duration limit", func() {
				now := t2.Stop.Local().AddDate(1, 4, 17).Add(time.Second)
				command.OlderThan = "1year4months2weeks3days"
				So(command.run(db, now, in, out), ShouldBeNil)

				Convey("Then it should say that the history was purged", func() {
					So(out.String(), ShouldEqual, "You are about to purge 2 history entries.\n"+
						"This operation cannot be undone. Do you wish to proceed anyway ?\n"+
						"\n"+
						"(Type 'YES' in all caps to proceed): \n"+
						"\n"+
						"The transfer history has been purged successfully.\n")
				})

				Convey("Then it should have purged the selected history entries", func() {
					var history model.HistoryEntries

					So(db.Select(&history).Run(), ShouldBeNil)
					So(history, ShouldHaveLength, 1)
					So(history[0].ID, ShouldEqual, t3.ID)
				})
			})
		})

		Convey("When purging the history AND resetting the increment", func() {
			command.Reset = true
			in := strings.NewReader("YES")

			server := &model.LocalAgent{
				Name: "test_server", Protocol: testProtocol,
				Address: types.Addr("1.2.3.4", 5),
			}
			So(db.Insert(server).Run(), ShouldBeNil)

			account := &model.LocalAccount{
				LocalAgentID: server.ID,
				Login:        "test_account",
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			rule := &model.Rule{Name: "test_rule", IsSend: false}
			So(db.Insert(rule).Run(), ShouldBeNil)

			So(db.Insert(&model.Transfer{
				RemoteTransferID: "789",
				RuleID:           rule.ID,
				LocalAccountID:   utils.NewNullInt64(account.ID),
				SrcFilename:      "/src/path",
				DestFilename:     "/dst/path",
				Start:            time.Date(2022, 1, 1, 1, 0, 0, 0, time.UTC),
			}).Run(), ShouldBeNil)

			So(db.Insert(&model.Transfer{
				RemoteTransferID: "147",
				RuleID:           rule.ID,
				LocalAccountID:   utils.NewNullInt64(account.ID),
				SrcFilename:      "/src/path",
				DestFilename:     "/dst/path",
				Start:            time.Date(2022, 1, 1, 1, 0, 0, 0, time.UTC),
			}).Run(), ShouldBeNil)

			Convey("Given that the transfers table is empty", func() {
				So(db.DeleteAll(&model.Transfer{}).Run(), ShouldBeNil)
				So(command.run(db, time.Now(), in, out), ShouldBeNil)

				Convey("Then it should say that the database was purged and the"+
					"increment was reset", func() {
					So(out.String(), ShouldEqual, "You are about to purge 3 history entries.\n"+
						"This operation cannot be undone. Do you wish to proceed anyway ?\n"+
						"\n"+
						"(Type 'YES' in all caps to proceed): \n"+
						"\n"+
						"The transfer history has been purged successfully,\n"+
						"and the transfer ID increment has been reset.\n")
				})

				Convey("Then it should have purged the history", func() {
					var history model.HistoryEntries

					So(db.Select(&history).Run(), ShouldBeNil)
					So(history, ShouldBeEmpty)
				})

				Convey("Then it should have reset the transfer increment", func() {
					newTrans := &model.Transfer{
						RemoteTransferID: "258",
						RuleID:           rule.ID,
						LocalAccountID:   utils.NewNullInt64(account.ID),
						SrcFilename:      "/src/path",
						DestFilename:     "/dst/path",
						Start:            time.Date(2022, 1, 1, 1, 0, 0, 0, time.UTC),
					}
					So(db.Insert(newTrans).Run(), ShouldBeNil)
					So(newTrans.ID, ShouldEqual, 1)
				})
			})

			Convey("Given that the transfers table is NOT empty", func() {
				So(command.run(db, time.Now(), in, out), ShouldBeError, ErrResetTransfersNotEmpty)

				Convey("Then it should NOT have purged the history", func() {
					var history model.HistoryEntries

					So(db.Select(&history).Run(), ShouldBeNil)
					So(history, ShouldHaveLength, 3)
				})

				Convey("Then it should NOT have reset the transfer increment", func() {
					newTrans := &model.Transfer{
						RemoteTransferID: "258",
						RuleID:           rule.ID,
						LocalAccountID:   utils.NewNullInt64(account.ID),
						SrcFilename:      "/src/path",
						DestFilename:     "/dst/path",
						Start:            time.Date(2022, 1, 1, 1, 0, 0, 0, time.UTC),
					}
					So(db.Insert(newTrans).Run(), ShouldBeNil)
					So(newTrans.ID, ShouldEqual, 3)
				})
			})
		})
	})
}

func mkPath(raw string) types.FSPath {
	if runtime.GOOS == "windows" {
		raw = "C:" + raw
	}

	path, err := types.ParsePath(raw)
	So(err, ShouldBeNil)

	return *path
}

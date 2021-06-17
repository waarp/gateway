package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
)

func historyInfoString(h *api.OutHistory) string {
	role := "client"
	if h.IsServer {
		role = "server"
	}
	way := "receive"
	if h.IsSend {
		way = "send"
	}
	size := "unknown"
	if h.Filesize >= 0 {
		size = fmt.Sprint(h.Filesize)
	}
	rv := "● Transfer " + fmt.Sprint(h.ID) + " (as " + role + ") [" + string(h.Status) + "]\n"
	if h.RemoteID != "" {
		rv += "    Remote ID:        " + h.RemoteID + "\n"
	}
	rv += "    Way:             " + way + "\n" +
		"    Protocol:        " + h.Protocol + "\n" +
		"    Rule:            " + h.Rule + "\n" +
		"    Requester:       " + h.Requester + "\n" +
		"    Requested:       " + h.Requested + "\n" +
		"    Local filepath:  " + h.LocalPath + "\n" +
		"    Remote filepath: " + h.RemotePath + "\n" +
		"    File size:       " + size + "\n" +
		"    Start date:      " + h.Start.Local().Format(time.RFC3339Nano) + "\n" +
		"    End date:        " + h.Stop.Local().Format(time.RFC3339Nano) + "\n"
	if h.ErrorCode != types.TeOk {
		rv += "    Error code:      " + h.ErrorCode.String() + "\n"
		if h.ErrorMsg != "" {
			rv += "    Error message:   " + h.ErrorMsg + "\n"
		}
	}
	if h.Step != types.StepNone {
		rv += "    Failed step:     " + h.Step.String() + "\n"
		if h.Step == types.StepData {
			rv += "    Progress:        " + fmt.Sprint(h.Progress) + "\n"
		} else if h.Step == types.StepPreTasks || h.Step == types.StepPostTasks {
			rv += "    Failed task:     " + fmt.Sprint(h.TaskNumber) + "\n"
		}
	}

	return rv
}

func TestDisplayHistory(t *testing.T) {

	Convey("Given a history entry", t, func() {
		out = testFile()

		hist := &api.OutHistory{
			ID:         1,
			IsServer:   true,
			IsSend:     false,
			Rule:       "Rule",
			Requester:  "Account",
			Requested:  "Server",
			Protocol:   "sftp",
			LocalPath:  "/local/path",
			RemotePath: "/remote/path",
			Start:      time.Now(),
			Stop:       time.Now().Add(time.Hour),
			Status:     types.StatusPlanned,
			Step:       types.StepSetup,
			Progress:   1,
			TaskNumber: 2,
			ErrorMsg:   "error message",
			ErrorCode:  types.TeUnknown,
		}
		Convey("When calling the `displayHistory` function", func() {
			w := getColorable()
			displayHistory(w, hist)

			Convey("Then it should display the entry's info correctly", func() {
				So(getOutput(), ShouldEqual, historyInfoString(hist))
			})
		})
	})

	Convey("Given a history entry with error", t, func() {
		out = testFile()

		hist := api.OutHistory{
			ID:         1,
			IsServer:   true,
			IsSend:     false,
			Rule:       "rule",
			Requester:  "source",
			Requested:  "destination",
			Protocol:   "sftp",
			LocalPath:  "/local/path",
			RemotePath: "/remote/path",
			Start:      time.Now(),
			Stop:       time.Now().Add(time.Hour),
			Status:     types.StatusPlanned,
			ErrorCode:  types.TeConnectionReset,
			ErrorMsg:   "connexion reset by peer",
		}
		Convey("When calling the `displayHistory` function", func() {
			w := getColorable()
			displayHistory(w, &hist)

			Convey("Then it should display the entry's info correctly", func() {
				So(getOutput(), ShouldEqual, historyInfoString(&hist))
			})
		})
	})
}

func TestGetHistory(t *testing.T) {

	Convey("Testing the partner 'get' command", t, func() {
		out = testFile()
		command := &historyGet{}

		Convey("Given a database", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given a valid history entry", func() {
				h := &model.HistoryEntry{
					ID:         1,
					IsServer:   true,
					IsSend:     false,
					Rule:       "rule",
					Account:    "source",
					Agent:      "destination",
					Protocol:   "sftp",
					LocalPath:  "/local/path",
					RemotePath: "/remote/path",
					Start:      time.Date(2021, 1, 1, 1, 0, 0, 0, time.Local),
					Stop:       time.Date(2021, 1, 1, 2, 0, 0, 0, time.Local),
					Status:     types.StatusDone,
					Owner:      database.Owner,
				}
				So(db.Insert(h).Run(), ShouldBeNil)
				id := fmt.Sprint(h.ID)

				Convey("Given a valid history entry ID", func() {
					args := []string{id}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display the entry's info", func() {
							hist := rest.FromHistory(h)
							So(getOutput(), ShouldEqual, historyInfoString(hist))
						})
					})
				})

				Convey("Given an invalid entry ID", func() {
					args := []string{"1000"}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						err = command.Execute(params)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError, "transfer 1000 not found")
						})
					})
				})
			})
		})
	})
}

func TestListHistory(t *testing.T) {

	Convey("Testing the history 'list' command", t, func() {
		out = testFile()
		command := &historyList{}

		Convey("Given a database", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given 4 valid history entries", func() {
				h1 := &model.HistoryEntry{
					ID:         1,
					IsServer:   true,
					IsSend:     false,
					Account:    "src1",
					Agent:      "dst1",
					Protocol:   "sftp",
					LocalPath:  "/local/path1",
					RemotePath: "/remote/path1",
					Rule:       "rule1",
					Start:      time.Date(2019, 1, 1, 1, 1, 0, 1000, time.Local),
					Stop:       time.Date(2019, 1, 1, 1, 2, 0, 1000, time.Local),
					Status:     types.StatusDone,
				}
				h2 := &model.HistoryEntry{
					ID:         2,
					IsServer:   true,
					IsSend:     false,
					Account:    "src2",
					Agent:      "dst2",
					Protocol:   "sftp",
					LocalPath:  "/local/path2",
					RemotePath: "/remote/path2",
					Rule:       "rule2",
					Start:      time.Date(2019, 1, 1, 2, 0, 0, 2000, time.Local),
					Stop:       time.Date(2019, 1, 1, 2, 1, 0, 2000, time.Local),
					Status:     types.StatusCancelled,
				}
				h3 := &model.HistoryEntry{
					ID:         3,
					IsServer:   true,
					IsSend:     false,
					Account:    "src3",
					Agent:      "dst3",
					Protocol:   "sftp",
					LocalPath:  "/local/path3",
					RemotePath: "/remote/path3",
					Rule:       "rule3",
					Start:      time.Date(2019, 1, 1, 3, 0, 0, 3000, time.Local),
					Stop:       time.Date(2019, 1, 1, 3, 1, 0, 3000, time.Local),
					Status:     types.StatusDone,
				}
				h4 := &model.HistoryEntry{
					ID:         4,
					IsServer:   true,
					IsSend:     false,
					Account:    "src4",
					Agent:      "dst4",
					Protocol:   "sftp",
					LocalPath:  "/local/path4",
					RemotePath: "/remote/path4",
					Rule:       "rule4",
					Start:      time.Date(2019, 1, 1, 4, 0, 0, 4000, time.Local),
					Stop:       time.Date(2019, 1, 1, 4, 1, 0, 4000, time.Local),
					Status:     types.StatusCancelled,
				}
				So(db.Insert(h1).Run(), ShouldBeNil)
				So(db.Insert(h2).Run(), ShouldBeNil)
				So(db.Insert(h3).Run(), ShouldBeNil)
				So(db.Insert(h4).Run(), ShouldBeNil)

				hist1 := rest.FromHistory(h1)
				hist2 := rest.FromHistory(h2)
				hist3 := rest.FromHistory(h3)
				hist4 := rest.FromHistory(h4)

				Convey("Given a no filters", func() {
					args := []string{}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display all the entries", func() {
							So(getOutput(), ShouldEqual, "History:\n"+
								historyInfoString(hist1)+historyInfoString(hist2)+
								historyInfoString(hist3)+historyInfoString(hist4))
						})
					})
				})

				Convey("Given a limit parameter of '2'", func() {
					args := []string{"-l", "2"}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display the first 2 entries", func() {
							So(getOutput(), ShouldEqual, "History:\n"+
								historyInfoString(hist1)+historyInfoString(hist2))
						})
					})
				})

				Convey("Given an offset parameter of '2'", func() {
					args := []string{"-o", "2"}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display all but the first 2 entries", func() {
							So(getOutput(), ShouldEqual, "History:\n"+
								historyInfoString(hist3)+historyInfoString(hist4))
						})
					})
				})

				Convey("Given a different sorting parameter", func() {
					args := []string{"-s", "id-"}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display all the entries "+
							"sorted & in reverse order", func() {
							So(getOutput(), ShouldEqual, "History:\n"+
								historyInfoString(hist4)+historyInfoString(hist3)+
								historyInfoString(hist2)+historyInfoString(hist1))
						})
					})
				})

				Convey("Given a start parameter", func() {
					args := []string{"--start=" + h3.Start.Format(time.RFC3339Nano)}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display all the entries that "+
							"started after that date", func() {
							So(getOutput(), ShouldEqual, "History:\n"+
								historyInfoString(hist3)+historyInfoString(hist4))
						})
					})
				})

				Convey("Given a stop parameter", func() {
					args := []string{"--stop=" + h2.Stop.Format(time.RFC3339Nano)}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display all the entries that "+
							"ended before that date", func() {
							So(getOutput(), ShouldEqual, "History:\n"+
								historyInfoString(hist1)+historyInfoString(hist2))
						})
					})
				})

				Convey("Given a requester parameter", func() {
					args := []string{"--requester=" + h1.Account, "--requester=" + h2.Account}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display all the entries from "+
							"one of these sources", func() {
							So(getOutput(), ShouldEqual, "History:\n"+
								historyInfoString(hist1)+historyInfoString(hist2))
						})
					})
				})

				Convey("Given a requested parameter", func() {
					args := []string{"--requested=" + h2.Agent, "--requested=" + h3.Agent}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display all the entries to "+
							"one of these destinations", func() {
							So(getOutput(), ShouldEqual, "History:\n"+
								historyInfoString(hist2)+historyInfoString(hist3))
						})
					})
				})

				Convey("Given a rule parameter", func() {
					args := []string{"--rule=" + h3.Rule, "--rule=" + h4.Rule}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display all the entries using "+
							"one of these rules", func() {
							So(getOutput(), ShouldEqual, "History:\n"+
								historyInfoString(hist3)+historyInfoString(hist4))
						})
					})
				})

				Convey("Given a status parameter", func() {
					args := []string{"--status=DONE"}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display all the entries in "+
							"one of these statuses", func() {
							So(getOutput(), ShouldEqual, "History:\n"+
								historyInfoString(hist1)+historyInfoString(hist3))
						})
					})
				})

				Convey("Given a protocol parameter", func() {
					args := []string{"--protocol=sftp"}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display all the entries using "+
							"one of these protocoles", func() {
							So(getOutput(), ShouldEqual, "History:\n"+
								historyInfoString(hist1)+historyInfoString(hist2)+
								historyInfoString(hist3)+historyInfoString(hist4))
						})
					})
				})

				Convey("Given a combination of multiple parameters", func() {
					args := []string{
						"--start=" + time.Date(2019, 1, 1, 0, 0, 0, 0, time.Local).Format(time.RFC3339Nano),
						"--stop=" + time.Date(2019, 1, 3, 0, 0, 0, 0, time.Local).Format(time.RFC3339Nano),
						"--requester=" + h1.Account, "--requester=" + h2.Account,
						"--requested=" + h4.Agent, "--requested=" + h1.Agent,
						"--rule=" + h3.Rule, "--rule=" + h1.Rule, "--rule=" + h2.Rule,
						"--status=DONE", "--status=CANCELLED",
						"--protocol=sftp",
					}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display all the entries that "+
							"fill all of these parameters", func() {
							So(getOutput(), ShouldEqual, "History:\n"+
								historyInfoString(hist1))
						})
					})
				})
			})
		})
	})
}

func TestRetryHistory(t *testing.T) {

	Convey("Testing the history 'retry' command", t, func() {
		out = testFile()
		command := &historyRetry{}

		Convey("Given a database", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given a failed history entry", func() {
				part := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    testProto1,
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:1",
				}
				So(db.Insert(part).Run(), ShouldBeNil)

				acc := &model.RemoteAccount{
					Login:         "login",
					Password:      "password",
					RemoteAgentID: part.ID,
				}
				So(db.Insert(acc).Run(), ShouldBeNil)

				r := &model.Rule{
					Name:   "rule",
					IsSend: true,
					Path:   "path",
				}
				So(db.Insert(r).Run(), ShouldBeNil)

				hist := &model.HistoryEntry{
					ID:         1,
					IsServer:   false,
					IsSend:     r.IsSend,
					Rule:       r.Name,
					Account:    acc.Login,
					Agent:      part.Name,
					Protocol:   part.Protocol,
					LocalPath:  "/local/path.loc",
					RemotePath: "/remote/path.rem",
					Start:      time.Date(2021, 1, 1, 1, 0, 0, 0, time.Local),
					Stop:       time.Date(2021, 1, 1, 2, 0, 0, 0, time.Local),
					Status:     types.StatusCancelled,
					Owner:      database.Owner,
				}
				So(db.Insert(hist).Run(), ShouldBeNil)
				id := fmt.Sprint(hist.ID)

				Convey("Given a valid history entry ID and date", func() {
					args := []string{id, "-d", time.Date(2030, 1, 1, 1, 0, 0, 123000,
						time.Local).Format(time.RFC3339Nano)}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then is should display a message saying the transfer was restarted", func() {
							So(getOutput(), ShouldEqual, "The transfer will be "+
								"retried under the ID: 1\n")
						})

						Convey("Then the transfer should have been added", func() {
							expected := model.Transfer{
								ID:         1,
								RuleID:     r.ID,
								IsServer:   false,
								AgentID:    part.ID,
								AccountID:  acc.ID,
								LocalPath:  "path.loc",
								RemotePath: "path.rem",
								Start:      time.Date(2030, 1, 1, 1, 0, 0, 123000, time.Local),
								Status:     types.StatusPlanned,
								Owner:      hist.Owner,
							}

							var trans model.Transfers
							So(db.Select(&trans).Run(), ShouldBeNil)
							So(trans[0], ShouldResemble, expected)
						})
					})
				})

				Convey("Given an invalid entry ID", func() {
					args := []string{"1000"}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						err = command.Execute(params)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError, "transfer 1000 not found")
						})
					})
				})
			})
		})
	})
}

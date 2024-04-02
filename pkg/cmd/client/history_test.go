//go:build todo

package wg

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jessevdk/go-flags"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func historyInfoString(h *api.OutHistory) string {
	role := roleClient
	if h.IsServer {
		role = roleServer
	}

	way := directionRecv
	if h.IsSend {
		way = directionSend
	}

	size := sizeUnknown
	if h.Filesize >= 0 {
		size = utils.FormatInt(h.Filesize)
	}

	rv := "● Transfer " + utils.FormatInt(h.ID) + " (as " + role + ") [" + string(h.Status) + "]\n"
	if h.RemoteID != "" {
		rv += "    Remote ID:       " + h.RemoteID + "\n"
	}

	stop := NotApplicable
	if h.Stop != nil {
		stop = h.Stop.Local().Format(time.RFC3339Nano)
	}

	rv += "    Way:             " + way + "\n" +
		"    Protocol:        " + h.Protocol + "\n" +
		"    Rule:            " + h.Rule + "\n" +
		"    Requester:       " + h.Requester + "\n" +
		"    Requested:       " + h.Requested + "\n" +
		"    Local filepath:  " + h.LocalFilepath + "\n" +
		"    Remote filepath: " + h.RemoteFilepath + "\n" +
		"    File size:       " + size + "\n" +
		"    Start date:      " + h.Start.Local().Format(time.RFC3339Nano) + "\n" +
		"    End date:        " + stop + "\n"

	if h.ErrorCode != types.TeOk {
		rv += "    Error code:      " + h.ErrorCode.String() + "\n"

		if h.ErrorMsg != "" {
			rv += "    Error message:   " + h.ErrorMsg + "\n"
		}
	}

	if h.Step != types.StepNone {
		rv += "    Failed step:     " + h.Step.String() + "\n"
		if h.Step == types.StepData {
			rv += "    Progress:        " + utils.FormatInt(h.Progress) + "\n"
		} else if h.Step == types.StepPreTasks || h.Step == types.StepPostTasks {
			rv += "    Failed task:     " + utils.FormatInt(h.TaskNumber) + "\n"
		}
	}

	if len(h.TransferInfo) != 0 {
		rv += "    Transfer info:\n"

		info := make([]string, 0, len(h.TransferInfo))

		for key, val := range h.TransferInfo {
			info = append(info, fmt.Sprint("      - ", key, ": ", val))
		}

		sort.Strings(info)

		rv += strings.Join(info, "\n")
		rv += "\n"
	}

	return rv
}

func TestDisplayHistory(t *testing.T) {
	Convey("Given a history entry", t, func() {
		out = testFile()

		hist := &api.OutHistory{
			ID:             1,
			IsServer:       true,
			IsSend:         false,
			Rule:           "Rule",
			Requester:      "Account",
			Requested:      "Server",
			Protocol:       testProto1,
			LocalFilepath:  "/local/path",
			RemoteFilepath: "/remote/path",
			Start:          time.Now(),
			Stop:           nil,
			Status:         types.StatusCancelled,
			Step:           types.StepSetup,
			Progress:       1,
			TaskNumber:     2,
			ErrorMsg:       "error message",
			ErrorCode:      types.TeUnknown,
			TransferInfo:   map[string]any{"key1": "val1", "key2": 2, "key3": true},
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

		stopTime := time.Now().Add(time.Hour)
		hist := api.OutHistory{
			ID:             1,
			IsServer:       true,
			IsSend:         false,
			Rule:           "rule",
			Requester:      "source",
			Requested:      "destination",
			Protocol:       testProto1,
			LocalFilepath:  "/local/path",
			RemoteFilepath: "/remote/path",
			Start:          time.Now(),
			Stop:           &stopTime,
			Status:         types.StatusPlanned,
			ErrorCode:      types.TeConnectionReset,
			ErrorMsg:       "connection reset by peer",
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
		command := &HistoryGet{}

		Convey("Given a database", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))

			var err error

			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given a valid history entry", func() {
				h := &model.HistoryEntry{
					ID:               1,
					RemoteTransferID: "1234",
					IsServer:         true,
					IsSend:           false,
					Rule:             "rule",
					Account:          "source",
					Agent:            "destination",
					Protocol:         testProto1,
					SrcFilename:      "/source/file",
					DestFilename:     "/dest/file",
					LocalPath:        *mkURL("file:/local/path"),
					RemotePath:       "/remote/path",
					Start:            time.Date(2021, 1, 1, 1, 0, 0, 0, time.Local),
					Stop:             time.Date(2021, 1, 1, 2, 0, 0, 0, time.Local),
					Status:           types.StatusDone,
					Owner:            conf.GlobalConfig.GatewayName,
				}
				So(db.Insert(h).Run(), ShouldBeNil)
				id := utils.FormatInt(h.ID)

				Convey("Given a valid history entry ID", func() {
					args := []string{id}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display the entry's info", func() {
							hist, _ := rest.FromHistory(db, h)
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

//nolint:maintidx //FIXME factorize the function if possible to improve maintainability
func TestListHistory(t *testing.T) {
	Convey("Testing the history 'list' command", t, func() {
		out = testFile()
		command := &HistoryList{}

		Convey("Given a database", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))

			var err error

			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given 4 valid history entries", func() {
				h1 := &model.HistoryEntry{
					ID:               1,
					RemoteTransferID: "111",
					IsServer:         true,
					IsSend:           false,
					Account:          "src1",
					Agent:            "dst1",
					Protocol:         testProto1,
					SrcFilename:      "/source/file1",
					DestFilename:     "/dest/file1",
					LocalPath:        *mkURL("file:/local/path1"),
					RemotePath:       "/remote/path1",
					Rule:             "rule1",
					Start:            time.Date(2019, 1, 1, 1, 1, 0, 1000, time.Local),
					Stop:             time.Date(2019, 1, 1, 1, 2, 0, 1000, time.Local),
					Status:           types.StatusDone,
				}
				h2 := &model.HistoryEntry{
					ID:               2,
					RemoteTransferID: "222",
					IsServer:         true,
					IsSend:           false,
					Account:          "src2",
					Agent:            "dst2",
					Protocol:         testProto1,
					SrcFilename:      "/source/file2",
					DestFilename:     "/dest/file2",
					LocalPath:        *mkURL("file:/local/path2"),
					RemotePath:       "/remote/path2",
					Rule:             "rule2",
					Start:            time.Date(2019, 1, 1, 2, 0, 0, 2000, time.Local),
					Stop:             time.Date(2019, 1, 1, 2, 1, 0, 2000, time.Local),
					Status:           types.StatusCancelled,
				}
				h3 := &model.HistoryEntry{
					ID:               3,
					RemoteTransferID: "333",
					IsServer:         true,
					IsSend:           false,
					Account:          "src3",
					Agent:            "dst3",
					Protocol:         testProto2,
					SrcFilename:      "/source/file3",
					DestFilename:     "/dest/file3",
					LocalPath:        *mkURL("file:/local/path3"),
					RemotePath:       "/remote/path3",
					Rule:             "rule3",
					Start:            time.Date(2019, 1, 1, 3, 0, 0, 3000, time.Local),
					Stop:             time.Date(2019, 1, 1, 3, 1, 0, 3000, time.Local),
					Status:           types.StatusDone,
				}
				h4 := &model.HistoryEntry{
					ID:               4,
					RemoteTransferID: "444",
					IsServer:         true,
					IsSend:           false,
					Account:          "src4",
					Agent:            "dst4",
					Protocol:         testProto2,
					SrcFilename:      "/source/file4",
					DestFilename:     "/dest/file4",
					LocalPath:        *mkURL("file:/local/path4"),
					RemotePath:       "/remote/path4",
					Rule:             "rule4",
					Start:            time.Date(2019, 1, 1, 4, 0, 0, 4000, time.Local),
					Stop:             time.Date(2019, 1, 1, 4, 1, 0, 4000, time.Local),
					Status:           types.StatusCancelled,
				}

				So(db.Insert(h1).Run(), ShouldBeNil)
				So(db.Insert(h2).Run(), ShouldBeNil)
				So(db.Insert(h3).Run(), ShouldBeNil)
				So(db.Insert(h4).Run(), ShouldBeNil)

				hist1, err := rest.FromHistory(db, h1)
				So(err, ShouldBeNil)
				hist2, err := rest.FromHistory(db, h2)
				So(err, ShouldBeNil)
				hist3, err := rest.FromHistory(db, h3)
				So(err, ShouldBeNil)
				hist4, err := rest.FromHistory(db, h4)
				So(err, ShouldBeNil)

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
					start := time.Date(2019, 1, 1, 2, 30, 0, 0, time.Local)
					args := []string{"--start", start.Format(time.RFC3339)}

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
					stop := time.Date(2019, 1, 1, 2, 30, 0, 0, time.Local)
					args := []string{"--stop", stop.Format(time.RFC3339)}

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
					args := []string{"--protocol=" + testProto1}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display all the entries using "+
							"one of these protocols", func() {
							So(getOutput(), ShouldEqual, "History:\n"+
								historyInfoString(hist1)+historyInfoString(hist2))
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
						"--status=DONE", "--status=CANCELED",
						"--protocol=" + testProto1,
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
		command := &HistoryRetry{}

		Convey("Given a database", func(c C) {
			db := database.TestDatabase(c)
			gw := httptest.NewServer(testHandler(db))

			var err error

			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given a failed history entry", func() {
				client := &model.Client{Name: "client", Protocol: testProto1}
				So(db.Insert(client).Run(), ShouldBeNil)

				part := &model.RemoteAgent{
					Name:     "partner",
					Protocol: client.Protocol,
					Address:  "localhost:1",
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
					ID:               1,
					RemoteTransferID: "1234",
					IsServer:         false,
					IsSend:           r.IsSend,
					Rule:             r.Name,
					Client:           client.Name,
					Account:          acc.Login,
					Agent:            part.Name,
					Protocol:         client.Protocol,
					SrcFilename:      "/source/file",
					DestFilename:     "/dest/file",
					LocalPath:        *mkURL("file:/local/path.loc"),
					RemotePath:       "/remote/path.rem",
					Start:            time.Date(2021, 1, 1, 1, 0, 0, 0, time.Local),
					Stop:             time.Date(2021, 1, 1, 2, 0, 0, 0, time.Local),
					Status:           types.StatusCancelled,
					Owner:            conf.GlobalConfig.GatewayName,
				}

				So(db.Insert(hist).Run(), ShouldBeNil)
				id := utils.FormatInt(hist.ID)

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
							var trans model.Transfers

							So(db.Select(&trans).Run(), ShouldBeNil)
							So(trans[0], ShouldResemble, &model.Transfer{
								ID:               1,
								RemoteTransferID: trans[0].RemoteTransferID,
								RuleID:           r.ID,
								ClientID:         utils.NewNullInt64(client.ID),
								RemoteAccountID:  utils.NewNullInt64(acc.ID),
								SrcFilename:      hist.SrcFilename,
								DestFilename:     hist.DestFilename,
								Start:            time.Date(2030, 1, 1, 1, 0, 0, 123000, time.Local),
								Status:           types.StatusPlanned,
								Owner:            hist.Owner,
							})
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

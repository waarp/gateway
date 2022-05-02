package wg

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func transferInfoString(t *api.OutTransfer) string {
	role := roleClient
	if t.IsServer {
		role = roleServer
	}

	size := sizeUnknown
	if t.Filesize >= 0 {
		size = fmt.Sprint(t.Filesize)
	}

	dir := directionRecv

	if t.IsSend {
		dir = directionSend
	}

	rv := "● Transfer " + fmt.Sprint(t.ID) + " (" + dir + " as " + role + ") [" +
		string(t.Status) + "]\n"

	if t.RemoteID != "" {
		rv += "    Remote ID:       " + t.RemoteID + "\n"
	}

	rv += "    Rule:            " + t.Rule + "\n" +
		"    Protocol:        " + t.Protocol + "\n" +
		"    Requester:       " + t.Requester + "\n" +
		"    Requested:       " + t.Requested + "\n" +
		"    Local filepath:  " + t.LocalFilepath + "\n" +
		"    Remote filepath: " + t.RemoteFilepath + "\n" +
		"    File size:       " + size + "\n" +
		"    Start date:      " + t.Start.Local().Format(time.RFC3339Nano) + "\n" +
		"    Step:            " + t.Step + "\n" +
		"    Progress:        " + fmt.Sprint(t.Progress) + "\n" +
		"    Task number:     " + fmt.Sprint(t.TaskNumber) + "\n"

	if t.ErrorCode != types.TeOk.String() {
		rv += "    Error code:      " + t.ErrorCode + "\n"
	}

	if t.ErrorMsg != "" {
		rv += "    Error message:   " + t.ErrorMsg + "\n"
	}

	return rv
}

func TestDisplayTransfer(t *testing.T) {
	Convey("Given a transfer entry", t, func() {
		out = testFile()

		trans := &api.OutTransfer{
			ID:             1,
			RemoteID:       "1234",
			Rule:           "rule",
			Requester:      "requester",
			Requested:      "requested",
			LocalFilepath:  "/local/path",
			RemoteFilepath: "/remote/path",
			Start:          time.Now(),
			Status:         types.StatusPlanned,
			Step:           types.StepData.String(),
			Filesize:       98765,
			Progress:       1,
			TaskNumber:     2,
			ErrorCode:      types.TeForbidden.String(),
			ErrorMsg:       "custom error message",
		}
		Convey("When calling the `displayTransfer` function", func() {
			w := getColorable()
			displayTransfer(w, trans)

			Convey("Then it should display the transfer's info correctly", func() {
				So(getOutput(), ShouldEqual, transferInfoString(trans))
			})
		})
	})
}

func TestAddTransfer(t *testing.T) {
	Convey("Testing the partner 'add' command", t, func() {
		out = testFile()
		command := &transferAdd{}

		Convey("Given a database", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule",
				IsSend: false,
				Path:   "path",
			}
			So(db.Insert(rule).Run(), ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:     "partner",
				Protocol: testProto1,
				Address:  "localhost:1",
			}
			So(db.Insert(partner).Run(), ShouldBeNil)

			account := &model.RemoteAccount{
				Login:         "toto",
				Password:      "sesame",
				RemoteAgentID: partner.ID,
			}
			So(db.Insert(account).Run(), ShouldBeNil)

			Convey("Given all valid flags", func() {
				args := []string{
					"--partner", partner.Name, "--login", account.Login,
					"--way", rule.Direction(), "--rule", rule.Name,
					"--file", "src_dir/test_file", "--out", "dst_dir/test_file",
					"--date", "2020-01-01T01:00:00+00:00",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the transfer was added", func() {
						So(getOutput(), ShouldEqual, "The transfer of file "+
							command.File+" was successfully added.\n")
					})

					Convey("Then the new transfer should have been added", func() {
						var transfers model.Transfers
						So(db.Select(&transfers).Run(), ShouldBeNil)
						So(transfers, ShouldHaveLength, 1)

						So(transfers[0].ID, ShouldEqual, 1)
						So(transfers[0].RemoteTransferID, ShouldNotBeBlank)
						So(transfers[0].IsServer, ShouldBeFalse)
						So(transfers[0].RuleID, ShouldEqual, rule.ID)
						So(transfers[0].AgentID, ShouldEqual, partner.ID)
						So(transfers[0].AccountID, ShouldEqual, account.ID)
						So(transfers[0].LocalPath, ShouldEqual, filepath.Join("dst_dir", "test_file"))
						So(transfers[0].RemotePath, ShouldEqual, "src_dir/test_file")
						So(transfers[0].Filesize, ShouldEqual, model.UnknownSize)
						So(transfers[0].Start, ShouldEqual, time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC))
						So(transfers[0].Step, ShouldEqual, types.StepNone)
						So(transfers[0].Status, ShouldEqual, types.StatusPlanned)
						So(transfers[0].Owner, ShouldEqual, conf.GlobalConfig.GatewayName)
						So(transfers[0].Progress, ShouldEqual, 0)
						So(transfers[0].TaskNumber, ShouldEqual, 0)
						So(transfers[0].Error, ShouldBeZeroValue)
					})
				})
			})

			Convey("Given an invalid rule name", func() {
				args := []string{
					"--partner", partner.Name, "--login", account.Login,
					"--way", rule.Direction(), "--rule", "toto", "--file", "file",
					"--date", "2020-01-01T01:00:00+01:00",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "no rule 'toto' found")
					})
				})
			})

			Convey("Given an invalid account name", func() {
				args := []string{
					"--partner", partner.Name, "--login", "tata",
					"--way", rule.Direction(), "--rule", rule.Name, "--file", "file",
					"--date", "2020-01-01T01:00:00+01:00",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "no account 'tata' found for partner "+
							partner.Name)
					})
				})
			})

			Convey("Given an invalid partner name", func() {
				args := []string{
					"--partner", "tata", "--login", account.Login,
					"--way", rule.Direction(), "--rule", rule.Name, "--file", "file",
					"--date", "2020-01-01T01:00:00+01:00",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "no partner 'tata' found")
					})
				})
			})

			Convey("Given an invalid start date", func() {
				args := []string{
					"--partner", partner.Name, "--login", account.Login,
					"--way", rule.Direction(), "--rule", rule.Name, "--file", "file",
					"--date", "toto",
				}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldContainSubstring, "'toto' is not a valid date")
					})
				})
			})
		})
	})
}

func TestGetTransfer(t *testing.T) {
	Convey("Testing the partner 'get' command", t, func() {
		out = testFile()
		command := &transferGet{}

		Convey("Given a database", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given a valid transfer", func() {
				p := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    testProto1,
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:1",
				}
				So(db.Insert(p).Run(), ShouldBeNil)

				a := &model.RemoteAccount{
					Login:         "toto",
					Password:      "sesame",
					RemoteAgentID: p.ID,
				}
				So(db.Insert(a).Run(), ShouldBeNil)

				r := &model.Rule{
					Name:   "rule",
					IsSend: false,
					Path:   "path",
				}
				So(db.Insert(r).Run(), ShouldBeNil)

				trans := &model.Transfer{
					RuleID:     r.ID,
					AgentID:    p.ID,
					AccountID:  a.ID,
					LocalPath:  "/local/path",
					RemotePath: "/remote/path",
					Start:      time.Date(2021, 1, 1, 1, 0, 0, 123000, time.Local),
					Status:     types.StatusPlanned,
				}
				So(db.Insert(trans).Run(), ShouldBeNil)
				id := fmt.Sprint(trans.ID)

				Convey("Given a valid transfer ID", func() {
					args := []string{id}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display the transfer's info", func() {
							jsonObj, err := rest.FromTransfer(db, trans)
							So(err, ShouldBeNil)
							So(getOutput(), ShouldEqual, transferInfoString(jsonObj))
						})
					})
				})

				Convey("Given an invalid transfer ID", func() {
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
func TestListTransfer(t *testing.T) {
	Convey("Testing the transfer 'list' command", t, func() {
		out = testFile()
		command := &transferList{}

		Convey("Given a database", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			p1 := &model.RemoteAgent{
				Name:     "remote1",
				Protocol: testProto1,
				Address:  "localhost:1",
			}
			p2 := &model.RemoteAgent{
				Name:     "remote2",
				Protocol: testProto1,
				Address:  "localhost:2",
			}
			p3 := &model.RemoteAgent{
				Name:     "remote3",
				Protocol: testProto1,
				Address:  "localhost:3",
			}
			p4 := &model.RemoteAgent{
				Name:     "remote4",
				Protocol: testProto1,
				Address:  "localhost:4",
			}
			So(db.Insert(p1).Run(), ShouldBeNil)
			So(db.Insert(p2).Run(), ShouldBeNil)
			So(db.Insert(p3).Run(), ShouldBeNil)
			So(db.Insert(p4).Run(), ShouldBeNil)

			a1 := &model.RemoteAccount{
				RemoteAgentID: p1.ID,
				Login:         "toto",
				Password:      "sesame1",
			}
			a2 := &model.RemoteAccount{
				RemoteAgentID: p2.ID,
				Login:         "tata",
				Password:      "sesame2",
			}
			a3 := &model.RemoteAccount{
				RemoteAgentID: p3.ID,
				Login:         "titi",
				Password:      "sesame3",
			}
			a4 := &model.RemoteAccount{
				RemoteAgentID: p4.ID,
				Login:         "tutu",
				Password:      "sesame4",
			}
			So(db.Insert(a1).Run(), ShouldBeNil)
			So(db.Insert(a2).Run(), ShouldBeNil)
			So(db.Insert(a3).Run(), ShouldBeNil)
			So(db.Insert(a4).Run(), ShouldBeNil)

			r1 := &model.Rule{Name: "rule1", IsSend: false, Path: "path1"}
			r2 := &model.Rule{Name: "rule2", IsSend: false, Path: "path2"}
			r3 := &model.Rule{Name: "rule3", IsSend: false, Path: "path3"}
			r4 := &model.Rule{Name: "rule4", IsSend: false, Path: "path4"}
			So(db.Insert(r1).Run(), ShouldBeNil)
			So(db.Insert(r2).Run(), ShouldBeNil)
			So(db.Insert(r3).Run(), ShouldBeNil)
			So(db.Insert(r4).Run(), ShouldBeNil)

			Convey("Given 4 valid transfers", func() {
				trans1 := &model.Transfer{
					RuleID:     r1.ID,
					AgentID:    p1.ID,
					AccountID:  a1.ID,
					LocalPath:  "/local/path1",
					RemotePath: "/remote/path1",
					Step:       types.StepNone,
					Status:     types.StatusPlanned,
					Start:      time.Date(2019, 1, 1, 1, 0, 0, 0, time.Local),
				}
				trans2 := &model.Transfer{
					RuleID:     r2.ID,
					AgentID:    p2.ID,
					AccountID:  a2.ID,
					LocalPath:  "/local/path2",
					RemotePath: "/remote/path2",
					Step:       types.StepSetup,
					Status:     types.StatusRunning,
					Start:      time.Date(2019, 1, 1, 2, 0, 0, 0, time.Local),
				}
				trans3 := &model.Transfer{
					RuleID:     r3.ID,
					AgentID:    p3.ID,
					AccountID:  a3.ID,
					LocalPath:  "/local/path3",
					RemotePath: "/remote/path3",
					Step:       types.StepData,
					Status:     types.StatusError,
					Start:      time.Date(2019, 1, 1, 3, 0, 0, 0, time.Local),
				}
				trans4 := &model.Transfer{
					RuleID:     r4.ID,
					AgentID:    p4.ID,
					AccountID:  a4.ID,
					LocalPath:  "/local/path4",
					RemotePath: "/remote/path4",
					Step:       types.StepFinalization,
					Status:     types.StatusPlanned,
					Start:      time.Date(2019, 1, 1, 4, 0, 0, 0, time.Local),
				}
				So(db.Insert(trans1).Run(), ShouldBeNil)
				So(db.Insert(trans2).Run(), ShouldBeNil)
				So(db.Insert(trans3).Run(), ShouldBeNil)
				So(db.Insert(trans4).Run(), ShouldBeNil)

				t1, err := rest.FromTransfer(db, trans1)
				So(err, ShouldBeNil)
				t2, err := rest.FromTransfer(db, trans2)
				So(err, ShouldBeNil)
				t3, err := rest.FromTransfer(db, trans3)
				So(err, ShouldBeNil)
				t4, err := rest.FromTransfer(db, trans4)
				So(err, ShouldBeNil)

				Convey("Given a no filters", func() {
					args := []string{}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display all the transfer's info", func() {
							So(getOutput(), ShouldEqual, "Transfers:\n"+
								transferInfoString(t1)+transferInfoString(t2)+
								transferInfoString(t3)+transferInfoString(t4))
						})
					})
				})

				Convey("Given a limit parameter of '2'", func() {
					args := []string{"-l", "2"}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display the first 2 transfers", func() {
							So(getOutput(), ShouldEqual, "Transfers:\n"+
								transferInfoString(t1)+transferInfoString(t2))
						})
					})
				})

				Convey("Given an offset parameter of '2'", func() {
					args := []string{"-o", "2"}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display all but the 2 first transfers", func() {
							So(getOutput(), ShouldEqual, "Transfers:\n"+
								transferInfoString(t3)+transferInfoString(t4))
						})
					})
				})

				Convey("Given a different sorting parameter", func() {
					args := []string{"-s", "id-"}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display all the transfer's "+
							"info sorted & in reverse", func() {
							So(getOutput(), ShouldEqual, "Transfers:\n"+
								transferInfoString(t4)+transferInfoString(t3)+
								transferInfoString(t2)+transferInfoString(t1))
						})
					})
				})

				Convey("Given a start parameter", func() {
					args := []string{"--date", time.Date(2019, 1, 1, 2, 30, 0, 0, time.Local).
						Format(time.RFC3339Nano)}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display all transfers "+
							"starting after that date", func() {
							So(getOutput(), ShouldEqual, "Transfers:\n"+
								transferInfoString(t3)+transferInfoString(t4))
						})
					})
				})

				Convey("Given a rule parameter", func() {
					args := []string{"--rule", r1.Name, "--rule", r4.Name}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display all transfers using one "+
							"of these rules", func() {
							So(getOutput(), ShouldEqual, "Transfers:\n"+
								transferInfoString(t1)+transferInfoString(t4))
						})
					})
				})

				Convey("Given a status parameter", func() {
					args := []string{"-t", "PLANNED"}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display all transfers "+
							"currently in one of these statuses", func() {
							So(getOutput(), ShouldEqual, "Transfers:\n"+
								transferInfoString(t1)+transferInfoString(t4))
						})
					})
				})

				Convey("Given multiple parameters", func() {
					args := []string{
						"--date", t2.Start.Add(-time.Minute).Format(time.RFC3339Nano),
						"--rule", r1.Name, "--rule", r2.Name, "--rule", r4.Name,
						"-t", "RUNNING",
					}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then it should display all transfers that "+
							"fill all these parameters", func() {
							So(getOutput(), ShouldEqual, "Transfers:\n"+
								transferInfoString(t2))
						})
					})
				})
			})
		})
	})
}

func TestPauseTransfer(t *testing.T) {
	Convey("Testing the transfer 'pause' command", t, func() {
		out = testFile()
		command := &transferPause{}

		Convey("Given a database", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given a paused transfer entry", func() {
				part := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    testProto1,
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:1",
				}
				So(db.Insert(part).Run(), ShouldBeNil)

				account := &model.RemoteAccount{
					Login:         "toto",
					Password:      "sesame",
					RemoteAgentID: part.ID,
				}
				So(db.Insert(account).Run(), ShouldBeNil)

				rule := &model.Rule{
					Name:   "rule",
					IsSend: true,
					Path:   "path",
				}
				So(db.Insert(rule).Run(), ShouldBeNil)

				trans := &model.Transfer{
					IsServer:   false,
					RuleID:     rule.ID,
					AccountID:  account.ID,
					AgentID:    part.ID,
					LocalPath:  "/local/path",
					RemotePath: "/remote/path",
					Start:      time.Date(2021, 1, 1, 1, 0, 0, 123000, time.Local),
					Status:     types.StatusPlanned,
					Owner:      conf.GlobalConfig.GatewayName,
				}
				So(db.Insert(trans).Run(), ShouldBeNil)
				id := fmt.Sprint(trans.ID)

				Convey("Givens a valid transfer ID", func() {
					args := []string{id}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then is should display a message saying the transfer was restarted", func() {
							So(getOutput(), ShouldEqual, "The transfer "+id+
								" was successfully paused. It can be resumed"+
								" using the 'resume' command.\n")
						})

						Convey("Then the transfer should have been updated", func() {
							trans.Status = types.StatusPaused

							var transfers model.Transfers
							So(db.Select(&transfers).Run(), ShouldBeNil)
							So(transfers, ShouldContain, *trans)
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

func TestResumeTransfer(t *testing.T) {
	Convey("Testing the transfer 'resume' command", t, func() {
		out = testFile()
		command := &transferResume{}

		Convey("Given a database", func(cx C) {
			db := database.TestDatabase(cx, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given a paused transfer entry", func() {
				partner := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    testProto1,
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:1",
				}
				So(db.Insert(partner).Run(), ShouldBeNil)

				account := &model.RemoteAccount{
					Login:         "toto",
					Password:      "sesame",
					RemoteAgentID: partner.ID,
				}
				So(db.Insert(account).Run(), ShouldBeNil)

				r := &model.Rule{
					Name:   "rule",
					IsSend: true,
					Path:   "path",
				}
				So(db.Insert(r).Run(), ShouldBeNil)

				trans := &model.Transfer{
					IsServer:   false,
					RuleID:     r.ID,
					AccountID:  account.ID,
					AgentID:    partner.ID,
					LocalPath:  "/local/path",
					RemotePath: "/remote/path",
					Start:      time.Date(2021, 1, 1, 1, 0, 0, 123000, time.Local),
					Status:     types.StatusPaused,
					Owner:      conf.GlobalConfig.GatewayName,
				}
				So(db.Insert(trans).Run(), ShouldBeNil)
				id := fmt.Sprint(trans.ID)

				Convey("Given a valid transfer ID", func() {
					args := []string{id}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then is should display a message saying the transfer was restarted", func() {
							So(getOutput(), ShouldEqual, "The transfer "+id+
								" was successfully resumed.\n")
						})

						Convey("Then the transfer should have been updated", func() {
							trans.Status = types.StatusPlanned

							var transfers model.Transfers
							So(db.Select(&transfers).Run(), ShouldBeNil)
							So(transfers, ShouldContain, *trans)
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

func TestCancelTransfer(t *testing.T) {
	Convey("Testing the transfer 'cancel' command", t, func() {
		out = testFile()
		command := &transferCancel{}

		Convey("Given a database", func(c C) {
			db := database.TestDatabase(c, "ERROR")
			gw := httptest.NewServer(testHandler(db))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given a paused transfer entry", func() {
				partner := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    testProto1,
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:1",
				}
				So(db.Insert(partner).Run(), ShouldBeNil)

				account := &model.RemoteAccount{
					Login:         "toto",
					Password:      "sesame",
					RemoteAgentID: partner.ID,
				}
				So(db.Insert(account).Run(), ShouldBeNil)

				rule := &model.Rule{
					Name:   "rule",
					IsSend: true,
					Path:   "path",
				}
				So(db.Insert(rule).Run(), ShouldBeNil)

				trans := &model.Transfer{
					IsServer:   false,
					RuleID:     rule.ID,
					AccountID:  account.ID,
					AgentID:    partner.ID,
					LocalPath:  "/local/path",
					RemotePath: "/remote/path",
					Start:      time.Date(2021, 1, 1, 1, 0, 0, 123000, time.Local),
					Status:     types.StatusPlanned,
					Owner:      conf.GlobalConfig.GatewayName,
				}
				So(db.Insert(trans).Run(), ShouldBeNil)
				id := fmt.Sprint(trans.ID)

				Convey("Given a valid transfer ID", func() {
					args := []string{id}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then is should display a message saying the transfer was restarted", func() {
							So(getOutput(), ShouldEqual, "The transfer "+id+
								" was successfully canceled.\n")
						})

						Convey("Then the transfer should have been updated", func() {
							var history model.HistoryEntries
							So(db.Select(&history).Run(), ShouldBeNil)
							So(history, ShouldNotBeEmpty)

							hist := model.HistoryEntry{
								ID:               trans.ID,
								RemoteTransferID: trans.RemoteTransferID,
								Owner:            trans.Owner,
								IsServer:         trans.IsServer,
								IsSend:           rule.IsSend,
								Account:          account.Login,
								Agent:            partner.Name,
								Protocol:         partner.Protocol,
								LocalPath:        trans.LocalPath,
								RemotePath:       trans.RemotePath,
								Rule:             rule.Name,
								Start:            trans.Start,
								Stop:             history[0].Stop,
								Status:           types.StatusCancelled,
								Error:            types.TransferError{},
								Step:             trans.Step,
								Progress:         0,
								TaskNumber:       0,
							}
							So(history, ShouldContain, hist)
						})
					})
				})

				Convey("Given an invalid entry ID", func() {
					id := "1000"

					Convey("When executing the command", func() {
						err := command.Execute([]string{id})

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})
					})
				})
			})
		})
	})
}

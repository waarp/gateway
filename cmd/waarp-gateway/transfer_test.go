package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
)

func transferInfoString(t *api.OutTransfer) string {
	role := "client"
	if t.IsServer {
		role = "server"
	}

	rv := "● Transfer " + fmt.Sprint(t.ID) + " (as " + role + ") [" + string(t.Status) + "]\n"
	if t.RemoteID != "" {
		rv += "    Remote ID:        " + t.RemoteID + "\n"
	}
	rv += "    Rule:             " + t.Rule + "\n" +
		"    Requester:        " + t.Requester + "\n" +
		"    Requested:        " + t.Requested + "\n" +
		"    True filepath:    " + t.TrueFilepath + "\n" +
		"    Source file:      " + t.SourcePath + "\n" +
		"    Destination file: " + t.DestPath + "\n" +
		"    Start time:       " + t.Start.Local().Format(time.RFC3339) + "\n" +
		"    Step:             " + string(t.Step) + "\n" +
		"    Progress:         " + fmt.Sprint(t.Progress) + "\n" +
		"    Task number:      " + fmt.Sprint(t.TaskNumber) + "\n"
	if t.ErrorCode != types.TeOk {
		rv += "    Error code:       " + t.ErrorCode.String() + "\n"
	}
	if t.ErrorMsg != "" {
		rv += "    Error message:    " + t.ErrorMsg + "\n"
	}
	return rv
}

func TestDisplayTransfer(t *testing.T) {

	Convey("Given a transfer entry", t, func() {
		out = testFile()

		trans := &api.OutTransfer{
			ID:           1,
			RemoteID:     "1234",
			Rule:         "rule",
			Requester:    "requester",
			Requested:    "requested",
			SourcePath:   "source/path",
			DestPath:     "dest/path",
			TrueFilepath: "/true/filepath",
			Start:        time.Now(),
			Status:       types.StatusPlanned,
			Step:         types.StepData,
			Progress:     1,
			TaskNumber:   2,
			ErrorCode:    types.TeForbidden,
			ErrorMsg:     "custom error message",
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

		Convey("Given a database", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			rule := &model.Rule{
				Name:   "rule",
				IsSend: true,
				Path:   "path",
			}
			So(db.Create(rule), ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:        "partner",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			So(db.Create(partner), ShouldBeNil)

			account := &model.RemoteAccount{
				Login:         "login",
				Password:      []byte("password"),
				RemoteAgentID: partner.ID,
			}
			So(db.Create(account), ShouldBeNil)

			Convey("Given all valid flags", func() {
				args := []string{"-p", partner.Name, "-l", account.Login, "-w",
					"push", "-r", rule.Name, "-f", "file.src", "-n", "file.dst",
					"-d", "2020-01-01T01:00:00+01:00"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					So(command.Execute(params), ShouldBeNil)

					Convey("Then is should display a message saying the transfer was added", func() {
						So(getOutput(), ShouldEqual, "The transfer of file file.src"+
							" was successfully added.\n")
					})

					Convey("Then the new transfer should have been added", func() {
						trans := &model.Transfer{
							RuleID:     rule.ID,
							IsServer:   false,
							AgentID:    partner.ID,
							AccountID:  account.ID,
							SourceFile: "file.src",
							DestFile:   "file.dst",
							Status:     types.StatusPlanned,
							Owner:      database.Owner,
						}
						So(db.Get(trans), ShouldBeNil)
					})
				})
			})

			Convey("Given an invalid rule name", func() {
				args := []string{"-p", partner.Name, "-l", account.Login, "-w",
					"push", "-r", "toto", "-f", "file.src", "-n", "file.dst",
					"-d", "2020-01-01T01:00:00+01:00"}

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
				args := []string{"-p", partner.Name, "-l", "toto", "-w",
					"push", "-r", rule.Name, "-f", "file.src", "-n", "file.dst",
					"-d", "2020-01-01T01:00:00+01:00"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "no account 'toto' found for partner "+
							partner.Name)
					})
				})
			})

			Convey("Given an invalid partner name", func() {
				args := []string{"-p", "toto", "-l", account.Login, "-w",
					"push", "-r", rule.Name, "-f", "file.src", "-n", "file.dst",
					"-d", "2020-01-01T01:00:00+01:00"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "no partner 'toto' found")
					})
				})
			})

			Convey("Given an invalid start date", func() {
				args := []string{"-p", partner.Name, "-l", account.Login, "-w",
					"push", "-r", rule.Name, "-f", "file.src", "-n", "file.dst",
					"-d", "toto"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "'toto' is not a valid date")
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

		Convey("Given a database", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given a valid transfer", func() {
				p := &model.RemoteAgent{
					Name:        "test",
					Protocol:    "sftp",
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:1",
				}
				So(db.Create(p), ShouldBeNil)

				c := &model.Cert{
					Name:        "test",
					PublicKey:   []byte("test"),
					Certificate: []byte("test"),
					OwnerType:   "remote_agents",
					OwnerID:     p.ID,
				}
				So(db.Create(c), ShouldBeNil)

				a := &model.RemoteAccount{
					Login:         "login",
					Password:      []byte("password"),
					RemoteAgentID: p.ID,
				}
				So(db.Create(a), ShouldBeNil)

				r := &model.Rule{
					Name:   "rule",
					IsSend: false,
					Path:   "path",
				}
				So(db.Create(r), ShouldBeNil)

				trans := &model.Transfer{
					RuleID:     r.ID,
					AgentID:    p.ID,
					AccountID:  a.ID,
					SourceFile: "source",
					DestFile:   "dest",
					Start:      time.Now(),
					Status:     types.StatusPlanned,
				}
				So(db.Create(trans), ShouldBeNil)
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

func TestListTransfer(t *testing.T) {

	Convey("Testing the transfer 'list' command", t, func() {
		out = testFile()
		command := &transferList{}

		Convey("Given a database", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			p1 := &model.RemoteAgent{
				Name:        "remote1",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1",
			}
			p2 := &model.RemoteAgent{
				Name:        "remote2",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2",
			}
			p3 := &model.RemoteAgent{
				Name:        "remote3",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:3",
			}
			p4 := &model.RemoteAgent{
				Name:        "remote4",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:4",
			}
			So(db.Create(p1), ShouldBeNil)
			So(db.Create(p2), ShouldBeNil)
			So(db.Create(p3), ShouldBeNil)
			So(db.Create(p4), ShouldBeNil)

			a1 := &model.RemoteAccount{
				RemoteAgentID: p1.ID,
				Login:         "login",
				Password:      []byte("password"),
			}
			a2 := &model.RemoteAccount{
				RemoteAgentID: p2.ID,
				Login:         "login",
				Password:      []byte("password"),
			}
			a3 := &model.RemoteAccount{
				RemoteAgentID: p3.ID,
				Login:         "login",
				Password:      []byte("password"),
			}
			a4 := &model.RemoteAccount{
				RemoteAgentID: p4.ID,
				Login:         "login",
				Password:      []byte("password"),
			}
			So(db.Create(a1), ShouldBeNil)
			So(db.Create(a2), ShouldBeNil)
			So(db.Create(a3), ShouldBeNil)
			So(db.Create(a4), ShouldBeNil)

			r1 := &model.Rule{Name: "rule1", IsSend: false, Path: "path1"}
			r2 := &model.Rule{Name: "rule2", IsSend: false, Path: "path2"}
			r3 := &model.Rule{Name: "rule3", IsSend: false, Path: "path3"}
			r4 := &model.Rule{Name: "rule4", IsSend: false, Path: "path4"}
			So(db.Create(r1), ShouldBeNil)
			So(db.Create(r2), ShouldBeNil)
			So(db.Create(r3), ShouldBeNil)
			So(db.Create(r4), ShouldBeNil)

			c := &model.Cert{
				Name:        "cert",
				PublicKey:   []byte("test"),
				Certificate: []byte("test"),
				OwnerType:   "remote_agents",
			}
			c.OwnerID = p1.ID
			So(db.Create(c), ShouldBeNil)
			c.OwnerID = p2.ID
			c.ID = 0
			So(db.Create(c), ShouldBeNil)
			c.OwnerID = p3.ID
			c.ID = 0
			So(db.Create(c), ShouldBeNil)
			c.OwnerID = p4.ID
			c.ID = 0
			So(db.Create(c), ShouldBeNil)

			Convey("Given 4 valid transfers", func() {

				trans1 := &model.Transfer{
					RuleID:     r1.ID,
					AgentID:    p1.ID,
					AccountID:  a1.ID,
					SourceFile: "source1",
					DestFile:   "dest1",
					Start:      time.Date(2019, 1, 1, 1, 0, 0, 0, time.UTC),
				}
				trans2 := &model.Transfer{
					RuleID:     r2.ID,
					AgentID:    p2.ID,
					AccountID:  a2.ID,
					SourceFile: "source2",
					DestFile:   "dest2",
					Start:      time.Date(2019, 1, 1, 2, 0, 0, 0, time.UTC),
				}
				trans3 := &model.Transfer{
					RuleID:     r3.ID,
					AgentID:    p3.ID,
					AccountID:  a3.ID,
					SourceFile: "source3",
					DestFile:   "dest3",
					Start:      time.Date(2019, 1, 1, 3, 0, 0, 0, time.UTC),
				}
				trans4 := &model.Transfer{
					RuleID:     r4.ID,
					AgentID:    p4.ID,
					AccountID:  a4.ID,
					SourceFile: "source3",
					DestFile:   "dest3",
					Start:      time.Date(2019, 1, 1, 4, 0, 0, 0, time.UTC),
				}
				So(db.Create(trans1), ShouldBeNil)
				So(db.Create(trans2), ShouldBeNil)
				So(db.Create(trans3), ShouldBeNil)
				So(db.Create(trans4), ShouldBeNil)

				trans2.Status = types.StatusRunning
				trans3.Status = types.StatusRunning
				So(db.Update(trans2), ShouldBeNil)
				So(db.Update(trans3), ShouldBeNil)

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
					args := []string{"-d", time.Date(2019, 1, 1, 2, 30, 0, 0, time.UTC).
						Format(time.RFC3339)}

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
					args := []string{"-r", r1.Name, "-r", r4.Name}

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
					args := []string{"-d", t2.Start.Add(-time.Minute).Format(time.RFC3339),
						"-r", r1.Name, "-r", r2.Name, "-r", r4.Name,
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

		Convey("Given a database", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given a paused transfer entry", func() {
				p := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    "test",
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:1",
				}
				So(db.Create(p), ShouldBeNil)

				c := &model.Cert{
					Name:        "test",
					PublicKey:   []byte("test"),
					Certificate: []byte("test"),
					OwnerType:   "remote_agents",
					OwnerID:     p.ID,
				}
				So(db.Create(c), ShouldBeNil)

				a := &model.RemoteAccount{
					Login:         "login",
					Password:      []byte("password"),
					RemoteAgentID: p.ID,
				}
				So(db.Create(a), ShouldBeNil)

				r := &model.Rule{
					Name:   "rule",
					IsSend: true,
					Path:   "path",
				}
				So(db.Create(r), ShouldBeNil)

				trans := &model.Transfer{
					IsServer:   false,
					RuleID:     r.ID,
					AccountID:  a.ID,
					AgentID:    p.ID,
					SourceFile: "source",
					DestFile:   "destination",
					Start:      time.Now().Truncate(time.Second),
					Status:     types.StatusPlanned,
					Owner:      database.Owner,
				}
				So(db.Create(trans), ShouldBeNil)
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

							var t []model.Transfer
							So(db.Select(&t, nil), ShouldBeNil)
							So(t, ShouldNotBeEmpty)
							So(t[0], ShouldResemble, *trans)
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

		Convey("Given a database", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given a paused transfer entry", func() {
				p := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    "test",
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:1",
				}
				So(db.Create(p), ShouldBeNil)

				c := &model.Cert{
					Name:        "test",
					PublicKey:   []byte("test"),
					Certificate: []byte("test"),
					OwnerType:   "remote_agents",
					OwnerID:     p.ID,
				}
				So(db.Create(c), ShouldBeNil)

				a := &model.RemoteAccount{
					Login:         "login",
					Password:      []byte("password"),
					RemoteAgentID: p.ID,
				}
				So(db.Create(a), ShouldBeNil)

				r := &model.Rule{
					Name:   "rule",
					IsSend: true,
					Path:   "path",
				}
				So(db.Create(r), ShouldBeNil)

				trans := &model.Transfer{
					IsServer:   false,
					RuleID:     r.ID,
					AccountID:  a.ID,
					AgentID:    p.ID,
					SourceFile: "source",
					DestFile:   "destination",
					Start:      time.Now().Truncate(time.Second),
					Status:     types.StatusPaused,
					Owner:      database.Owner,
				}
				So(db.Create(trans), ShouldBeNil)
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

							var t []model.Transfer
							So(db.Select(&t, nil), ShouldBeNil)
							So(t, ShouldNotBeEmpty)
							So(t[0], ShouldResemble, *trans)
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

		Convey("Given a database", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			var err error
			addr, err = url.Parse("http://admin:admin_password@" + gw.Listener.Addr().String())
			So(err, ShouldBeNil)

			Convey("Given a paused transfer entry", func() {
				p := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    "test",
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:1",
				}
				So(db.Create(p), ShouldBeNil)

				c := &model.Cert{
					Name:        "test",
					PublicKey:   []byte("test"),
					Certificate: []byte("test"),
					OwnerType:   "remote_agents",
					OwnerID:     p.ID,
				}
				So(db.Create(c), ShouldBeNil)

				a := &model.RemoteAccount{
					Login:         "login",
					Password:      []byte("password"),
					RemoteAgentID: p.ID,
				}
				So(db.Create(a), ShouldBeNil)

				r := &model.Rule{
					Name:   "rule",
					IsSend: true,
					Path:   "path",
				}
				So(db.Create(r), ShouldBeNil)

				trans := &model.Transfer{
					IsServer:   false,
					RuleID:     r.ID,
					AccountID:  a.ID,
					AgentID:    p.ID,
					SourceFile: "source",
					DestFile:   "destination",
					Start:      time.Now().Truncate(time.Second),
					Status:     types.StatusPlanned,
					Owner:      database.Owner,
				}
				So(db.Create(trans), ShouldBeNil)
				id := fmt.Sprint(trans.ID)

				Convey("Given a valid transfer ID", func() {
					args := []string{id}

					Convey("When executing the command", func() {
						params, err := flags.ParseArgs(command, args)
						So(err, ShouldBeNil)
						So(command.Execute(params), ShouldBeNil)

						Convey("Then is should display a message saying the transfer was restarted", func() {
							So(getOutput(), ShouldEqual, "The transfer "+id+
								" was successfully cancelled.\n")
						})

						Convey("Then the transfer should have been updated", func() {
							var h []model.TransferHistory
							So(db.Select(&h, nil), ShouldBeNil)
							So(h, ShouldNotBeEmpty)

							hist := model.TransferHistory{
								ID:             trans.ID,
								Owner:          trans.Owner,
								IsServer:       trans.IsServer,
								IsSend:         r.IsSend,
								Account:        a.Login,
								Agent:          p.Name,
								Protocol:       p.Protocol,
								SourceFilename: trans.SourceFile,
								DestFilename:   trans.DestFile,
								Rule:           r.Name,
								Start:          trans.Start,
								Stop:           h[0].Stop,
								Status:         types.StatusCancelled,
								Error:          types.TransferError{},
								Step:           trans.Step,
								Progress:       0,
								TaskNumber:     0,
							}
							So(h[0], ShouldResemble, hist)
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

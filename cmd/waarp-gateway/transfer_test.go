package main

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
)

func transferInfoString(t *rest.OutTransfer) string {
	role := "client"
	if t.IsServer {
		role = "server"
	}

	rv := "● Transfer " + fmt.Sprint(t.ID) + " (as " + role + ") [" + string(t.Status) + "]\n" +
		"    Rule:             " + t.Rule + "\n" +
		"    Requester:        " + t.Requester + "\n" +
		"    Requested:        " + t.Requested + "\n" +
		"    True filepath:    " + t.TrueFilepath + "\n" +
		"    Source file:      " + t.SourcePath + "\n" +
		"    Destination file: " + t.DestPath + "\n" +
		"    Start time:       " + t.Start.Local().Format(time.RFC3339) + "\n" +
		"    Step:             " + string(t.Step) + "\n" +
		"    Progress:         " + fmt.Sprint(t.Progress) + "\n" +
		"    Task number:      " + fmt.Sprint(t.TaskNumber) + "\n"
	if t.ErrorCode != model.TeOk {
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

		trans := &rest.OutTransfer{
			ID:           1,
			Rule:         "rule",
			Requester:    "requester",
			Requested:    "requested",
			SourcePath:   "source/path",
			DestPath:     "dest/path",
			TrueFilepath: "/true/filepath",
			Start:        time.Now(),
			Status:       model.StatusPlanned,
			Step:         model.StepData,
			Progress:     1,
			TaskNumber:   2,
			ErrorCode:    model.TeForbidden,
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
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			rule := &model.Rule{
				Name:   "rule",
				IsSend: true,
			}
			So(db.Create(rule), ShouldBeNil)

			partner := &model.RemoteAgent{
				Name:        "partner",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(partner), ShouldBeNil)

			account := &model.RemoteAccount{
				Login:         "login",
				Password:      []byte("password"),
				RemoteAgentID: partner.ID,
			}
			So(db.Create(account), ShouldBeNil)

			Convey("Given all valid flags", func() {
				args := []string{"-p", partner.Name, "-a", account.Login, "-w",
					"push", "-r", rule.Name, "-f", "file.src", "-d", "file.dst"}

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
							Status:     model.StatusPlanned,
							Owner:      database.Owner,
						}
						So(db.Get(trans), ShouldBeNil)
					})
				})
			})

			Convey("Given an invalid rule name", func() {
				args := []string{"-p", partner.Name, "-a", account.Login, "-w",
					"pull", "-r", "toto", "-f", "file.src", "-d", "file.dst"}

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
				args := []string{"-p", partner.Name, "-a", "toto", "-w",
					"pull", "-r", rule.Name, "-f", "file.src", "-d", "file.dst"}

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
				args := []string{"-p", "toto", "-a", account.Login, "-w",
					"pull", "-r", rule.Name, "-f", "file.src", "-d", "file.dst"}

				Convey("When executing the command", func() {
					params, err := flags.ParseArgs(command, args)
					So(err, ShouldBeNil)
					err = command.Execute(params)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "no partner 'toto' found")
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
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			Convey("Given a valid transfer", func() {
				p := &model.RemoteAgent{
					Name:        "test",
					Protocol:    "sftp",
					ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
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
				}
				So(db.Create(r), ShouldBeNil)

				trans := &model.Transfer{
					RuleID:     r.ID,
					AgentID:    p.ID,
					AccountID:  a.ID,
					SourceFile: "source",
					DestFile:   "dest",
					Start:      time.Now(),
					Status:     model.StatusPlanned,
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
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			p1 := &model.RemoteAgent{
				Name:        "remote1",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
			}
			p2 := &model.RemoteAgent{
				Name:        "remote2",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2023}`),
			}
			p3 := &model.RemoteAgent{
				Name:        "remote3",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2024}`),
			}
			p4 := &model.RemoteAgent{
				Name:        "remote4",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2025}`),
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

			r1 := &model.Rule{Name: "rule1", IsSend: false}
			r2 := &model.Rule{Name: "rule2", IsSend: false}
			r3 := &model.Rule{Name: "rule3", IsSend: false}
			r4 := &model.Rule{Name: "rule4", IsSend: false}
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

				trans2.Status = model.StatusRunning
				trans3.Status = model.StatusRunning
				So(db.Update(&model.Transfer{Status: model.StatusRunning}, trans2.ID, false), ShouldBeNil)
				So(db.Update(&model.Transfer{Status: model.StatusRunning}, trans3.ID, false), ShouldBeNil)

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
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			Convey("Given a paused transfer entry", func() {
				p := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    "test",
					ProtoConfig: []byte(`{}`),
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
					Status:     model.StatusPlanned,
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
							trans.Status = model.StatusPaused

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
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			Convey("Given a paused transfer entry", func() {
				p := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    "test",
					ProtoConfig: []byte(`{}`),
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
					Status:     model.StatusPaused,
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
							trans.Status = model.StatusPlanned

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
			commandLine.Args.Address = "http://admin:admin_password@" + gw.Listener.Addr().String()

			Convey("Given a paused transfer entry", func() {
				p := &model.RemoteAgent{
					Name:        "partner",
					Protocol:    "test",
					ProtoConfig: []byte(`{}`),
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
					Status:     model.StatusPlanned,
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
								Status:         model.StatusCancelled,
								Error:          model.TransferError{},
								Step:           "",
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

package main

import (
	"fmt"
	"io/ioutil"
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
	rv := "Transfer n°" + fmt.Sprint(t.ID) + ":\n" +
		"          Rule ID: " + fmt.Sprint(t.RuleID) + "\n" +
		"       Partner ID: " + fmt.Sprint(t.AgentID) + "\n" +
		"       Account ID: " + fmt.Sprint(t.AccountID) + "\n" +
		"      Source file: " + t.SourcePath + "\n" +
		" Destination file: " + t.DestPath + "\n" +
		"       Start time: " + t.Start.Local().Format(time.RFC3339) + "\n" +
		"           Status: " + string(t.Status) + "\n" +
		"             Step: " + string(t.Step) + "\n" +
		"         Progress: " + fmt.Sprint(t.Progress) + "\n" +
		"      Task number: " + fmt.Sprint(t.TaskNumber) + "\n"
	if t.ErrorCode != model.TeOk {
		rv += "       Error code: " + t.ErrorCode.String() + "\n"
	}
	if t.ErrorMsg != "" {
		rv += "    Error message: " + t.ErrorMsg + "\n"
	}
	return rv
}

func TestDisplayTransfer(t *testing.T) {

	Convey("Given a transfer entry", t, func() {
		out = testFile()

		trans := &rest.OutTransfer{
			ID:         1,
			RuleID:     2,
			AgentID:    3,
			AccountID:  4,
			SourcePath: "source/path",
			DestPath:   "dest/path",
			Start:      time.Now(),
			Status:     model.StatusPlanned,
		}
		Convey("When calling the `displayTransfer` function", func() {
			displayTransfer(*trans)

			Convey("Then it should display the transfer's info correctly", func() {
				_, err := out.Seek(0, 0)
				So(err, ShouldBeNil)
				cont, err := ioutil.ReadAll(out)
				So(err, ShouldBeNil)
				So(string(cont), ShouldEqual, transferInfoString(trans))
			})
		})
	})

	Convey("Given a transfer entry with an error", t, func() {
		out = testFile()

		trans := &rest.OutTransfer{
			ID:         1,
			RuleID:     2,
			AgentID:    3,
			AccountID:  4,
			SourcePath: "source/path",
			DestPath:   "dest/path",
			Start:      time.Now(),
			Status:     model.StatusPlanned,
			ErrorCode:  model.TeForbidden,
			ErrorMsg:   "custom error message",
		}
		Convey("When calling the `displayTransfer` function", func() {
			displayTransfer(*trans)

			Convey("Then it should display the transfer's info correctly", func() {
				_, err := out.Seek(0, 0)
				So(err, ShouldBeNil)
				cont, err := ioutil.ReadAll(out)
				So(err, ShouldBeNil)
				So(string(cont), ShouldEqual, transferInfoString(trans))
			})
		})
	})
}

func TestAddTransfer(t *testing.T) {

	Convey("Testing the partner 'add' command", t, func() {
		command := &transferAddCommand{}

		Convey("Given a database", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			Convey("Given a valid remote agents", func() {
				p := model.RemoteAgent{
					Name:        "test",
					Protocol:    "sftp",
					ProtoConfig: []byte(`{"address":"localhost","port":1,"root":"toto"}`),
				}
				err := db.Create(&p)
				So(err, ShouldBeNil)

				Convey("Given a valid certificate", func() {
					c := model.Cert{
						Name:        "test",
						PublicKey:   []byte("test"),
						Certificate: []byte("test"),
						OwnerType:   "remote_agents",
						OwnerID:     p.ID,
					}
					err := db.Create(&c)
					So(err, ShouldBeNil)

					Convey("Given a valid account", func() {
						a := model.RemoteAccount{
							Login:         "login",
							Password:      []byte("password"),
							RemoteAgentID: p.ID,
						}
						err := db.Create(&a)
						So(err, ShouldBeNil)

						Convey("Given a valid rule", func() {
							r := model.Rule{
								Name:   "rule",
								IsSend: false,
							}
							err := db.Create(&r)
							So(err, ShouldBeNil)

							Convey("Given valid flags", func() {
								command.ServerID = p.ID
								command.AccountID = a.ID
								command.RuleID = r.ID
								command.File = "test"

								Convey("When executing the command", func() {
									addr := gw.Listener.Addr().String()
									dsn := "http://admin:admin_password@" + addr
									auth.DSN = dsn

									err := command.Execute(nil)

									Convey("Then it should NOT return an error", func() {
										So(err, ShouldBeNil)
									})
								})
							})

							Convey("Given no rule", func() {
								command.ServerID = p.ID
								command.AccountID = a.ID
								command.File = "test"

								Convey("When executing the command", func() {
									addr := gw.Listener.Addr().String()
									dsn := "http://admin:admin_password@" + addr
									auth.DSN = dsn

									err := command.Execute(nil)

									Convey("Then it should return an error", func() {
										So(err, ShouldNotBeNil)
									})
								})
							})

							Convey("Given no account", func() {
								command.ServerID = p.ID
								command.RuleID = r.ID
								command.File = "test"

								Convey("When executing the command", func() {
									addr := gw.Listener.Addr().String()
									dsn := "http://admin:admin_password@" + addr
									auth.DSN = dsn

									err := command.Execute(nil)

									Convey("Then it should return an error", func() {
										So(err, ShouldNotBeNil)
									})
								})
							})

							Convey("Given no remote", func() {
								command.AccountID = a.ID
								command.RuleID = r.ID
								command.File = "test"

								Convey("When executing the command", func() {
									addr := gw.Listener.Addr().String()
									dsn := "http://admin:admin_password@" + addr
									auth.DSN = dsn

									err := command.Execute(nil)

									Convey("Then it should return an error", func() {
										So(err, ShouldNotBeNil)
									})
								})
							})

							Convey("Given no File", func() {
								command.ServerID = p.ID
								command.AccountID = a.ID
								command.RuleID = r.ID

								Convey("When executing the command", func() {
									addr := gw.Listener.Addr().String()
									dsn := "http://admin:admin_password@" + addr
									auth.DSN = dsn

									err := command.Execute(nil)

									Convey("Then it should return an error", func() {
										So(err, ShouldNotBeNil)
									})
								})
							})

							Convey("Given another remote agent", func() {
								p2 := model.RemoteAgent{
									Name:        "dummy",
									Protocol:    "sftp",
									ProtoConfig: []byte(`{"address":"localhost","port":2,"root":"titi"}`),
								}
								err := db.Create(&p2)
								So(err, ShouldBeNil)

								Convey("Given an account link to another remote agent", func() {
									a2 := model.RemoteAccount{
										Login:         "login",
										Password:      []byte("password"),
										RemoteAgentID: p2.ID,
									}
									err := db.Create(&a2)
									So(err, ShouldBeNil)

									Convey("Given an incorrect account", func() {
										command.ServerID = p.ID
										command.AccountID = a2.ID
										command.RuleID = r.ID
										command.File = "test"

										Convey("When executing the command", func() {
											addr := gw.Listener.Addr().String()
											dsn := "http://admin:admin_password@" + addr
											auth.DSN = dsn

											err := command.Execute(nil)

											Convey("Then it should return an error", func() {
												So(err, ShouldNotBeNil)
											})
										})
									})
								})
							})
						})
					})
				})
			})
		})
	})
}

func TestGetTransfer(t *testing.T) {

	Convey("Testing the partner 'get' command", t, func() {
		out = testFile()
		command := &transferGetCommand{}

		Convey("Given a database", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			Convey("Given a valid transfer", func() {
				p := &model.RemoteAgent{
					Name:        "test",
					Protocol:    "sftp",
					ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
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
					SourcePath: "test/source/path",
					DestPath:   "test/dest/path",
					Start:      time.Now(),
					Status:     model.StatusPlanned,
				}
				So(db.Create(trans), ShouldBeNil)

				Convey("Given a valid transfer ID", func() {
					id := fmt.Sprint(trans.ID)

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

						err := command.Execute([]string{id})

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then it should display the transfer's info", func() {
							_, err = out.Seek(0, 0)
							So(err, ShouldBeNil)
							cont, err := ioutil.ReadAll(out)
							So(err, ShouldBeNil)
							So(string(cont), ShouldEqual, transferInfoString(rest.FromTransfer(trans)))
						})
					})
				})

				Convey("Given an invalid transfer ID", func() {
					id := "1000"

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

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

func TestListTransfer(t *testing.T) {

	Convey("Testing the transfer 'list' command", t, func() {
		out = testFile()
		command := &transferListCommand{}
		_, err := flags.ParseArgs(command, nil)
		So(err, ShouldBeNil)

		Convey("Given a database", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			auth.DSN = "http://admin:admin_password@" + gw.Listener.Addr().String()

			p1 := &model.RemoteAgent{
				Name:        "remote1",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			p2 := &model.RemoteAgent{
				Name:        "remote2",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2023,"root":"titi"}`),
			}
			p3 := &model.RemoteAgent{
				Name:        "remote3",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2024,"root":"tata"}`),
			}
			p4 := &model.RemoteAgent{
				Name:        "remote4",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2025,"root":"tutu"}`),
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

				t1 := &model.Transfer{
					RuleID:     r1.ID,
					AgentID:    p1.ID,
					AccountID:  a1.ID,
					SourcePath: "test/source/path",
					DestPath:   "test/dest/path",
					Start:      time.Date(2019, 1, 1, 1, 0, 0, 0, time.UTC),
				}
				t2 := &model.Transfer{
					RuleID:     r2.ID,
					AgentID:    p2.ID,
					AccountID:  a2.ID,
					SourcePath: "test/source/path",
					DestPath:   "test/dest/path",
					Start:      time.Date(2019, 1, 1, 2, 0, 0, 0, time.UTC),
				}
				t3 := &model.Transfer{
					RuleID:     r3.ID,
					AgentID:    p3.ID,
					AccountID:  a3.ID,
					SourcePath: "test/source/path",
					DestPath:   "test/dest/path",
					Start:      time.Date(2019, 1, 1, 3, 0, 0, 0, time.UTC),
				}
				t4 := &model.Transfer{
					RuleID:     r4.ID,
					AgentID:    p4.ID,
					AccountID:  a4.ID,
					SourcePath: "test/source/path",
					DestPath:   "test/dest/path",
					Start:      time.Date(2019, 1, 1, 4, 0, 0, 0, time.UTC),
				}
				So(db.Create(t1), ShouldBeNil)
				So(db.Create(t2), ShouldBeNil)
				So(db.Create(t3), ShouldBeNil)
				So(db.Create(t4), ShouldBeNil)

				t2.Status = model.StatusRunning
				t3.Status = model.StatusRunning
				So(db.Update(&model.Transfer{Status: model.StatusRunning}, t2.ID, false), ShouldBeNil)
				So(db.Update(&model.Transfer{Status: model.StatusRunning}, t3.ID, false), ShouldBeNil)

				Convey("Given a no filters", func() {

					Convey("When executing the command", func() {
						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)

							Convey("Then it should display all the transfer's "+
								"info", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "Transfers:\n"+
									transferInfoString(rest.FromTransfer(t1))+
									transferInfoString(rest.FromTransfer(t2))+
									transferInfoString(rest.FromTransfer(t3))+
									transferInfoString(rest.FromTransfer(t4)))
							})
						})
					})
				})

				Convey("Given a limit parameter of '2'", func() {
					command.Limit = 2

					Convey("When executing the command", func() {
						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)

							Convey("Then it should display the first 2 transfers",
								func() {
									_, err = out.Seek(0, 0)
									So(err, ShouldBeNil)
									cont, err := ioutil.ReadAll(out)
									So(err, ShouldBeNil)
									So(string(cont), ShouldEqual, "Transfers:\n"+
										transferInfoString(rest.FromTransfer(t1))+
										transferInfoString(rest.FromTransfer(t2)))
								})
						})
					})
				})

				Convey("Given an offset parameter of '2'", func() {
					command.Limit = 20
					command.Offset = 2

					Convey("When executing the command", func() {
						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)

							Convey("Then it should display all but the 2 first "+
								"transfers", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "Transfers:\n"+
									transferInfoString(rest.FromTransfer(t3))+
									transferInfoString(rest.FromTransfer(t4)))
							})
						})
					})
				})

				Convey("Given a different sorting parameter", func() {
					command.SortBy = "id"
					command.DescOrder = true

					Convey("When executing the command", func() {
						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)

							Convey("Then it should display all the transfer's "+
								"info sorted & in reverse", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "Transfers:\n"+
									transferInfoString(rest.FromTransfer(t4))+
									transferInfoString(rest.FromTransfer(t3))+
									transferInfoString(rest.FromTransfer(t2))+
									transferInfoString(rest.FromTransfer(t1)))
							})
						})
					})
				})

				Convey("Given a start parameter", func() {
					command.Start = time.Date(2019, 1, 1, 2, 30, 0, 0, time.UTC).
						Format(time.RFC3339)

					Convey("When executing the command", func() {
						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)

							Convey("Then it should display all transfers "+
								"starting after that date", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "Transfers:\n"+
									transferInfoString(rest.FromTransfer(t3))+
									transferInfoString(rest.FromTransfer(t4)))
							})
						})
					})
				})

				Convey("Given a remote parameter", func() {
					command.Remotes = []uint64{t1.AgentID, t3.AgentID}

					Convey("When executing the command", func() {
						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)

							Convey("Then it should display all transfers with one "+
								"of these partners", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "Transfers:\n"+
									transferInfoString(rest.FromTransfer(t1))+
									transferInfoString(rest.FromTransfer(t3)))
							})
						})
					})
				})

				Convey("Given an account parameter", func() {
					command.Accounts = []uint64{t2.AccountID, t3.AccountID}

					Convey("When executing the command", func() {
						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)

							Convey("Then it should display all transfers using one "+
								"of these accounts", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "Transfers:\n"+
									transferInfoString(rest.FromTransfer(t2))+
									transferInfoString(rest.FromTransfer(t3)))
							})
						})
					})
				})

				Convey("Given a rule parameter", func() {
					command.Rules = []uint64{t1.RuleID, t4.RuleID}

					Convey("When executing the command", func() {
						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)

							Convey("Then it should display all transfers using one "+
								"of these rules", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "Transfers:\n"+
									transferInfoString(rest.FromTransfer(t1))+
									transferInfoString(rest.FromTransfer(t4)))
							})
						})
					})
				})

				Convey("Given a status parameter", func() {
					command.Statuses = []string{"PLANNED"}

					Convey("When executing the command", func() {
						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)

							Convey("Then it should display all transfers "+
								"currently in one of these statuses", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "Transfers:\n"+
									transferInfoString(rest.FromTransfer(t1))+
									transferInfoString(rest.FromTransfer(t4)))
							})
						})
					})
				})

				Convey("Given multiple parameters", func() {
					command.Start = time.Date(2019, 1, 1, 1, 30, 0, 0, time.UTC).
						Format(time.RFC3339)
					command.Remotes = []uint64{t1.AgentID, t2.AgentID}
					command.Accounts = []uint64{t2.AccountID, t4.AccountID}
					command.Rules = []uint64{t1.RuleID, t2.RuleID, t3.RuleID}
					command.Statuses = []string{"RUNNING", "PLANNED"}

					Convey("When executing the command", func() {
						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)

							Convey("Then it should display all transfers that "+
								"fill all these parameters", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "Transfers:\n"+
									transferInfoString(rest.FromTransfer(t2)))
							})
						})
					})
				})
			})
		})
	})
}

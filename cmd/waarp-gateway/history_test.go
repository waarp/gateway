package main

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func historyInfoString(t *model.TransferHistory) string {
	return "Transfer " + fmt.Sprint(t.ID) + "=>\n" +
		"    IsServer: " + fmt.Sprint(t.IsServer) + "\n" +
		"        Send: " + fmt.Sprint(t.Send) + "\n" +
		"    Protocol: " + fmt.Sprint(t.Protocol) + "\n" +
		"        Rule: " + fmt.Sprint(t.Rule) + "\n" +
		"     Account: " + fmt.Sprint(t.Account) + "\n" +
		"      Remote: " + fmt.Sprint(t.Remote) + "\n" +
		"        File: " + t.Filename + "\n" +
		"  Start date: " + t.Start.Local().Format(time.RFC3339) + "\n" +
		"    End date: " + t.Stop.Local().Format(time.RFC3339) + "\n" +
		"      Status: " + string(t.Status) + "\n"
}

func TestDisplayHistory(t *testing.T) {

	Convey("Given a history entry", t, func() {
		out = testFile()

		hist := model.TransferHistory{
			ID:       1,
			IsServer: true,
			Send:     false,
			Rule:     "rule",
			Account:  "source",
			Remote:   "destination",
			Protocol: "sftp",
			Filename: "file/path",
			Start:    time.Now(),
			Stop:     time.Now().Add(time.Hour),
			Status:   model.StatusPlanned,
			Owner:    database.Owner,
		}
		Convey("When calling the `displayHistory` function", func() {
			displayHistory(hist)

			Convey("Then it should display the entry's info correctly", func() {
				_, err := out.Seek(0, 0)
				So(err, ShouldBeNil)
				cont, err := ioutil.ReadAll(out)
				So(err, ShouldBeNil)
				So(string(cont), ShouldEqual, historyInfoString(&hist))
			})
		})
	})
}

func TestHistoryGetCommand(t *testing.T) {

	Convey("Testing the partner 'get' command", t, func() {
		out = testFile()
		command := &historyGetCommand{}

		Convey("Given a database", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			auth.DSN = "http://admin:admin_password@" + gw.Listener.Addr().String()

			Convey("Given a valid history entry", func() {
				hist := model.TransferHistory{
					ID:       1,
					IsServer: true,
					Send:     false,
					Rule:     "rule",
					Account:  "source",
					Remote:   "destination",
					Protocol: "sftp",
					Filename: "file/path",
					Start:    time.Now(),
					Stop:     time.Now().Add(time.Hour),
					Status:   model.StatusDone,
					Owner:    database.Owner,
				}
				So(db.Create(&hist), ShouldBeNil)

				Convey("Given a valid history entry ID", func() {
					id := fmt.Sprint(hist.ID)

					Convey("When executing the command", func() {

						err := command.Execute([]string{id})

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)

							Convey("Then it should display the entry's info", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, historyInfoString(&hist))
							})
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

func TestHistoryListCommand(t *testing.T) {

	Convey("Testing the history 'list' command", t, func() {
		out = testFile()
		command := &historyListCommand{}

		Convey("Given a database", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))
			auth.DSN = "http://admin:admin_password@" + gw.Listener.Addr().String()

			Convey("Given 4 valid history entries", func() {
				h1 := model.TransferHistory{
					ID:       1,
					IsServer: true,
					Send:     false,
					Account:  "src1",
					Remote:   "dst1",
					Protocol: "sftp",
					Filename: "file1",
					Rule:     "rule1",
					Start:    time.Date(2019, 1, 1, 1, 0, 0, 0, time.UTC),
					Stop:     time.Date(2019, 1, 1, 1, 1, 0, 0, time.UTC),
					Status:   model.StatusDone,
				}
				h2 := model.TransferHistory{
					ID:       2,
					IsServer: true,
					Send:     false,
					Account:  "src2",
					Remote:   "dst2",
					Protocol: "sftp",
					Filename: "file2",
					Rule:     "rule2",
					Start:    time.Date(2019, 1, 1, 2, 0, 0, 0, time.UTC),
					Stop:     time.Date(2019, 1, 1, 2, 1, 0, 0, time.UTC),
					Status:   model.StatusError,
				}
				h3 := model.TransferHistory{
					ID:       3,
					IsServer: true,
					Send:     false,
					Account:  "src3",
					Remote:   "dst3",
					Protocol: "sftp",
					Filename: "file3",
					Rule:     "rule3",
					Start:    time.Date(2019, 1, 1, 3, 0, 0, 0, time.UTC),
					Stop:     time.Date(2019, 1, 1, 3, 1, 0, 0, time.UTC),
					Status:   model.StatusDone,
				}
				h4 := model.TransferHistory{
					ID:       4,
					IsServer: true,
					Send:     false,
					Account:  "src4",
					Remote:   "dst4",
					Protocol: "sftp",
					Filename: "file4",
					Rule:     "rule4",
					Start:    time.Date(2019, 1, 1, 4, 0, 0, 0, time.UTC),
					Stop:     time.Date(2019, 1, 1, 4, 1, 0, 0, time.UTC),
					Status:   model.StatusError,
				}
				So(db.Create(&h1), ShouldBeNil)
				So(db.Create(&h2), ShouldBeNil)
				So(db.Create(&h3), ShouldBeNil)
				So(db.Create(&h4), ShouldBeNil)

				Convey("Given a no filters", func() {

					Convey("When executing the command", func() {
						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)

							Convey("Then it should display all the entries", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "History:\n"+
									historyInfoString(&h1)+
									historyInfoString(&h2)+
									historyInfoString(&h3)+
									historyInfoString(&h4))
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

							Convey("Then it should display the first 2 entries", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "History:\n"+
									historyInfoString(&h1)+
									historyInfoString(&h2))
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

							Convey("Then it should display all but the first 2 "+
								"entries", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "History:\n"+
									historyInfoString(&h3)+
									historyInfoString(&h4))
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

							Convey("Then it should display all the entries "+
								"sorted & in reverse order", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "History:\n"+
									historyInfoString(&h4)+
									historyInfoString(&h3)+
									historyInfoString(&h2)+
									historyInfoString(&h1))
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

							Convey("Then it should display all the entries that "+
								"started after that date", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "History:\n"+
									historyInfoString(&h3)+
									historyInfoString(&h4))
							})
						})
					})
				})

				Convey("Given a stop parameter", func() {
					command.Stop = time.Date(2019, 1, 1, 2, 30, 0, 0, time.UTC).
						Format(time.RFC3339)

					Convey("When executing the command", func() {
						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)

							Convey("Then it should display all the entries that "+
								"ended before that date", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "History:\n"+
									historyInfoString(&h1)+
									historyInfoString(&h2))
							})
						})
					})
				})

				Convey("Given a source parameter", func() {
					command.Account = []string{h1.Account, h2.Account}

					Convey("When executing the command", func() {
						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)

							Convey("Then it should display all the entries from "+
								"one of these sources", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "History:\n"+
									historyInfoString(&h1)+
									historyInfoString(&h2))
							})
						})
					})
				})

				Convey("Given a destination parameter", func() {
					command.Remote = []string{h2.Remote, h3.Remote}

					Convey("When executing the command", func() {
						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)

							Convey("Then it should display all the entries to "+
								"one of these destinations", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "History:\n"+
									historyInfoString(&h2)+
									historyInfoString(&h3))
							})
						})
					})
				})

				Convey("Given a rule parameter", func() {
					command.Rules = []string{h3.Rule, h4.Rule}

					Convey("When executing the command", func() {
						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)

							Convey("Then it should display all the entries using "+
								"one of these rules", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "History:\n"+
									historyInfoString(&h3)+
									historyInfoString(&h4))
							})
						})
					})
				})

				Convey("Given a status parameter", func() {
					command.Statuses = []string{"DONE"}

					Convey("When executing the command", func() {
						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)

							Convey("Then it should display all the entries in "+
								"one of these statuses", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "History:\n"+
									historyInfoString(&h1)+
									historyInfoString(&h3))
							})
						})
					})
				})

				Convey("Given a protocol parameter", func() {
					command.Protocol = []string{"sftp"}

					Convey("When executing the command", func() {
						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)

							Convey("Then it should display all the entries using "+
								"one of these protocoles", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "History:\n"+
									historyInfoString(&h1)+
									historyInfoString(&h2)+
									historyInfoString(&h3)+
									historyInfoString(&h4))
							})
						})
					})
				})

				Convey("Given a combination of multiple parameters", func() {
					command.Start = time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
					command.Stop = time.Date(2019, 1, 3, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
					command.Account = []string{h1.Account, h2.Account}
					command.Remote = []string{h4.Remote, h1.Remote}
					command.Rules = []string{h3.Rule, h1.Rule, h2.Rule}
					command.Statuses = []string{"DONE", "TRANSFER"}
					command.Protocol = []string{"sftp"}

					Convey("When executing the command", func() {
						err := command.Execute(nil)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)

							Convey("Then it should display all the entries that "+
								"fill all of these parameters", func() {
								_, err = out.Seek(0, 0)
								So(err, ShouldBeNil)
								cont, err := ioutil.ReadAll(out)
								So(err, ShouldBeNil)
								So(string(cont), ShouldEqual, "History:\n"+
									historyInfoString(&h1))
							})
						})
					})
				})
			})
		})
	})
}

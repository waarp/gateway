package main

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetAccount(t *testing.T) {

	Convey("Testing the account 'get' command", t, func() {
		out = testFile()
		command := &accountGetCommand{}

		Convey("Given a gateway with 1 remote account", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			parentAgent := model.RemoteAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}

			err := db.Create(&parentAgent)
			So(err, ShouldBeNil)

			remoteAccount := model.RemoteAccount{
				Login:         "remote_account",
				Password:      []byte("password"),
				RemoteAgentID: parentAgent.ID,
			}

			err = db.Create(&remoteAccount)
			So(err, ShouldBeNil)

			Convey("Given a valid account ID", func() {
				id := fmt.Sprint(remoteAccount.ID)

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display the account's info", func() {

						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)

						agentID := fmt.Sprint(remoteAccount.RemoteAgentID)
						So(string(cont), ShouldEqual, "Remote account n°1:\n"+
							"      Login: "+remoteAccount.Login+"\n"+
							" Partner ID: "+agentID+"\n",
						)
					})
				})
			})

			Convey("Given an invalid account ID", func() {
				id := "1000"

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "404 - The resource 'http://"+
							addr+admin.APIPath+admin.RemoteAccountsPath+
							"/1000' does not exist")

					})
				})
			})

			Convey("Given no account ID", func() {

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "missing account ID")
					})
				})
			})
		})
	})
}

func TestAddAccount(t *testing.T) {

	Convey("Testing the account 'add' command", t, func() {
		out = testFile()
		command := &accountAddCommand{}

		Convey("Given a gateway", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			parentAgent := model.RemoteAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}

			err := db.Create(&parentAgent)
			So(err, ShouldBeNil)

			Convey("Given valid flags", func() {
				command.Login = "remote_account"
				command.Password = "password"
				command.PartnerID = parentAgent.ID

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then is should display a message saying the account was added", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "The account '"+command.Login+
							"' was successfully added. It can be consulted at "+
							"the address: "+gw.URL+admin.APIPath+
							admin.RemoteAccountsPath+"/1\n")
					})

					Convey("Then the new partner should have been added", func() {
						account := model.RemoteAccount{
							ID:            1,
							Login:         command.Login,
							RemoteAgentID: command.PartnerID,
						}
						err := db.Get(&account)
						So(err, ShouldBeNil)

						clearPwd, err := model.DecryptPassword(account.Password)
						So(err, ShouldBeNil)
						So(string(clearPwd), ShouldEqual, command.Password)
					})
				})
			})

			Convey("Given an invalid server ID", func() {
				command.Login = "remote_account"
				command.Password = "password"
				command.PartnerID = 1000

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "400 - Invalid request: "+
							"No remote agent found with the ID '1000'")
					})
				})
			})
		})
	})
}

func TestDeleteAccount(t *testing.T) {

	Convey("Testing the account 'delete' command", t, func() {
		out = testFile()
		command := &accountDeleteCommand{}

		Convey("Given a gateway with 1 remote account", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			parentAgent := model.RemoteAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}

			err := db.Create(&parentAgent)
			So(err, ShouldBeNil)

			account := model.RemoteAccount{
				RemoteAgentID: parentAgent.ID,
				Login:         "remote_account",
				Password:      []byte("password"),
			}

			err = db.Create(&account)
			So(err, ShouldBeNil)

			Convey("Given a valid account ID", func() {
				id := fmt.Sprint(account.ID)

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then is should display a message saying the account was deleted", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "The account n°"+id+
							" was successfully deleted from the database\n")
					})

					Convey("Then the account should have been removed", func() {
						exists, err := db.Exists(&model.RemoteAccount{ID: account.ID})
						So(err, ShouldBeNil)
						So(exists, ShouldBeFalse)
					})
				})
			})

			Convey("Given an invalid ID", func() {
				id := "1000"

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "404 - The resource 'http://"+
							addr+admin.APIPath+admin.RemoteAccountsPath+
							"/1000' does not exist")
					})

					Convey("Then the partner should still exist", func() {
						exists, err := db.Exists(&model.RemoteAccount{ID: account.ID})
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})
		})
	})
}

func TestUpdateAccount(t *testing.T) {

	Convey("Testing the account 'delete' command", t, func() {
		out = testFile()
		command := &accountUpdateCommand{}

		Convey("Given a gateway with 1 remote account", func() {

			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			parentAgent := model.RemoteAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}

			err := db.Create(&parentAgent)
			So(err, ShouldBeNil)

			account := model.RemoteAccount{
				RemoteAgentID: parentAgent.ID,
				Login:         "remote_account",
				Password:      []byte("password"),
			}

			err = db.Create(&account)
			So(err, ShouldBeNil)

			Convey("Given a valid account ID", func() {
				id := fmt.Sprint(account.ID)

				Convey("Given all valid flags", func() {
					command.Login = "new_remote_account"
					command.Password = "new_password"
					command.PartnerID = parentAgent.ID

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

						err := command.Execute([]string{id})

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then is should display a message saying the account was updated", func() {
							_, err = out.Seek(0, 0)
							So(err, ShouldBeNil)
							cont, err := ioutil.ReadAll(out)
							So(err, ShouldBeNil)
							So(string(cont), ShouldEqual, "The account n°"+id+
								" was successfully updated\n")
						})

						Convey("Then the old account should have been removed", func() {
							exists, err := db.Exists(&account)
							So(err, ShouldBeNil)
							So(exists, ShouldBeFalse)
						})

						Convey("Then the new account should exist", func() {
							newAccount := model.RemoteAccount{
								ID:            account.ID,
								Login:         command.Login,
								RemoteAgentID: command.PartnerID,
							}
							err := db.Get(&newAccount)
							So(err, ShouldBeNil)

							clearPwd, err := model.DecryptPassword(newAccount.Password)
							So(err, ShouldBeNil)
							So(string(clearPwd), ShouldEqual, command.Password)
						})
					})
				})

				Convey("Given an invalid server ID", func() {
					command.Login = "new_remote_account"
					command.Password = "new_password"
					command.PartnerID = 1000

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

						err := command.Execute([]string{id})

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "400 - Invalid request: "+
								"No remote agent found with the ID '1000'")
						})
					})
				})
			})

			Convey("Given an invalid ID", func() {
				id := "1000"

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute([]string{id})

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "404 - The resource 'http://"+
							addr+admin.APIPath+admin.RemoteAccountsPath+
							"/1000' does not exist")
					})

					Convey("Then the partner should stay unchanged", func() {
						exists, err := db.Exists(&account)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})
		})
	})
}

func TestListAccount(t *testing.T) {

	Convey("Testing the account 'list' command", t, func() {
		out = testFile()
		command := &accountListCommand{}
		_, err := flags.ParseArgs(command, []string{"waarp_gateway"})
		So(err, ShouldBeNil)

		Convey("Given a gateway with 2 remote accounts", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			parent1 := model.RemoteAgent{
				Name:        "parent_agent_1",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			err := db.Create(&parent1)
			So(err, ShouldBeNil)

			parent2 := model.RemoteAgent{
				Name:        "remote_agent2",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			err = db.Create(&parent2)
			So(err, ShouldBeNil)

			account1 := model.RemoteAccount{
				RemoteAgentID: parent1.ID,
				Login:         "account1",
				Password:      []byte("password"),
			}
			err = db.Create(&account1)
			So(err, ShouldBeNil)

			account2 := model.RemoteAccount{
				RemoteAgentID: parent2.ID,
				Login:         "account2",
				Password:      []byte("password"),
			}
			err = db.Create(&account2)
			So(err, ShouldBeNil)

			Convey("Given no parameters", func() {

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display the accounts' info", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Remote accounts:\n"+
							"Remote account n°1:\n"+
							"      Login: "+account1.Login+"\n"+
							" Partner ID: "+fmt.Sprint(account1.RemoteAgentID)+"\n"+
							"Remote account n°2:\n"+
							"      Login: "+account2.Login+"\n"+
							" Partner ID: "+fmt.Sprint(account2.RemoteAgentID)+"\n",
						)
					})
				})
			})

			Convey("Given a partner_id parameter", func() {
				command.RemoteAgentID = []uint64{parent1.ID}

				Convey("When executing the command", func() {
					dsn := "http://admin:admin_password@" + gw.Listener.Addr().String()
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should display the accounts' info", func() {
						_, err = out.Seek(0, 0)
						So(err, ShouldBeNil)
						cont, err := ioutil.ReadAll(out)
						So(err, ShouldBeNil)
						So(string(cont), ShouldEqual, "Remote accounts:\n"+
							"Remote account n°1:\n"+
							"      Login: "+account1.Login+"\n"+
							" Partner ID: "+fmt.Sprint(account1.RemoteAgentID)+"\n",
						)
					})
				})
			})
		})
	})
}

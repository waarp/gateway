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
	"golang.org/x/crypto/bcrypt"
)

func TestGetAccess(t *testing.T) {

	Convey("Testing the access 'get' command", t, func() {
		out = testFile()
		command := &accessGetCommand{}

		Convey("Given a gateway with 1 local account", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			parentAgent := model.LocalAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}

			err := db.Create(&parentAgent)
			So(err, ShouldBeNil)

			localAccount := model.LocalAccount{
				Login:        "local_account",
				Password:     []byte("password"),
				LocalAgentID: parentAgent.ID,
			}

			err = db.Create(&localAccount)
			So(err, ShouldBeNil)

			Convey("Given a valid account ID", func() {
				id := fmt.Sprint(localAccount.ID)

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

						agentID := fmt.Sprint(localAccount.LocalAgentID)
						So(string(cont), ShouldEqual, "Local account n°1:\n"+
							"├─Login: "+localAccount.Login+"\n"+
							"└─Server ID: "+agentID+"\n",
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
							addr+admin.APIPath+admin.LocalAccountsPath+
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

func TestAddAccess(t *testing.T) {

	Convey("Testing the access 'add' command", t, func() {
		out = testFile()
		command := &accessAddCommand{}

		Convey("Given a gateway", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			parentAgent := model.LocalAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}

			err := db.Create(&parentAgent)
			So(err, ShouldBeNil)

			Convey("Given valid flags", func() {
				command.Login = "local_account"
				command.Password = "password"
				command.LocalAgentID = parentAgent.ID

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
							admin.LocalAccountsPath+"/1\n")
					})

					Convey("Then the new partner should have been added", func() {
						account := model.LocalAccount{
							ID:           1,
							Login:        command.Login,
							LocalAgentID: command.LocalAgentID,
						}
						err := db.Get(&account)
						So(err, ShouldBeNil)

						err = bcrypt.CompareHashAndPassword(account.Password, []byte(command.Password))
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given an invalid server ID", func() {
				command.Login = "local_account"
				command.Password = "password"
				command.LocalAgentID = 1000

				Convey("When executing the command", func() {
					addr := gw.Listener.Addr().String()
					dsn := "http://admin:admin_password@" + addr
					auth.DSN = dsn

					err := command.Execute(nil)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err.Error(), ShouldEqual, "400 - Invalid request: "+
							"No local agent found with the ID '1000'")
					})
				})
			})
		})
	})
}

func TestDeleteAccess(t *testing.T) {

	Convey("Testing the access 'delete' command", t, func() {
		out = testFile()
		command := &accessDeleteCommand{}

		Convey("Given a gateway with 1 local account", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			parentAgent := model.LocalAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}

			err := db.Create(&parentAgent)
			So(err, ShouldBeNil)

			account := model.LocalAccount{
				LocalAgentID: parentAgent.ID,
				Login:        "local_account",
				Password:     []byte("password"),
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
						exists, err := db.Exists(&model.LocalAccount{ID: account.ID})
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
							addr+admin.APIPath+admin.LocalAccountsPath+
							"/1000' does not exist")
					})

					Convey("Then the partner should still exist", func() {
						exists, err := db.Exists(&model.LocalAccount{ID: account.ID})
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})
		})
	})
}

func TestUpdateAccess(t *testing.T) {

	Convey("Testing the access 'delete' command", t, func() {
		out = testFile()
		command := &accessUpdateCommand{}

		Convey("Given a gateway with 1 local account", func() {

			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			parentAgent := model.LocalAgent{
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}

			err := db.Create(&parentAgent)
			So(err, ShouldBeNil)

			account := model.LocalAccount{
				LocalAgentID: parentAgent.ID,
				Login:        "local_account",
				Password:     []byte("password"),
			}

			err = db.Create(&account)
			So(err, ShouldBeNil)

			Convey("Given a valid account ID", func() {
				id := fmt.Sprint(account.ID)

				Convey("Given all valid flags", func() {
					command.Login = "new_local_account"
					command.Password = "new_password"
					command.LocalAgentID = parentAgent.ID

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
							newAccount := model.LocalAccount{
								ID:           account.ID,
								Login:        command.Login,
								LocalAgentID: command.LocalAgentID,
							}
							err := db.Get(&newAccount)
							So(err, ShouldBeNil)

							err = bcrypt.CompareHashAndPassword(newAccount.Password, []byte(command.Password))
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given an invalid server ID", func() {
					command.Login = "new_local_account"
					command.Password = "new_password"
					command.LocalAgentID = 1000

					Convey("When executing the command", func() {
						addr := gw.Listener.Addr().String()
						dsn := "http://admin:admin_password@" + addr
						auth.DSN = dsn

						err := command.Execute([]string{id})

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "400 - Invalid request: "+
								"No local agent found with the ID '1000'")
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
							addr+admin.APIPath+admin.LocalAccountsPath+
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

func TestListAccess(t *testing.T) {

	Convey("Testing the account 'list' command", t, func() {
		out = testFile()
		command := &accessListCommand{}
		_, err := flags.ParseArgs(command, []string{"waarp_gateway"})
		So(err, ShouldBeNil)

		Convey("Given a gateway with 2 local accounts", func() {
			db := database.GetTestDatabase()
			gw := httptest.NewServer(admin.MakeHandler(discard, db, nil))

			parent1 := model.LocalAgent{
				Name:        "parent_agent_1",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parent1)
			So(err, ShouldBeNil)

			parent2 := model.LocalAgent{
				Name:        "remote_agent2",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err = db.Create(&parent2)
			So(err, ShouldBeNil)

			account1 := model.LocalAccount{
				LocalAgentID: parent1.ID,
				Login:        "account1",
				Password:     []byte("password"),
			}
			err = db.Create(&account1)
			So(err, ShouldBeNil)

			account2 := model.LocalAccount{
				LocalAgentID: parent2.ID,
				Login:        "account2",
				Password:     []byte("password"),
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
						So(string(cont), ShouldEqual, "Local accounts:\n"+
							"Local account n°1:\n"+
							"├─Login: "+account1.Login+"\n"+
							"└─Server ID: "+fmt.Sprint(account1.LocalAgentID)+"\n"+
							"Local account n°2:\n"+
							"├─Login: "+account2.Login+"\n"+
							"└─Server ID: "+fmt.Sprint(account2.LocalAgentID)+"\n",
						)
					})
				})
			})

			Convey("Given a server_id parameter", func() {
				command.LocalAgentID = []uint64{parent1.ID}

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
						So(string(cont), ShouldEqual, "Local accounts:\n"+
							"Local account n°1:\n"+
							"├─Login: "+account1.Login+"\n"+
							"└─Server ID: "+fmt.Sprint(account1.LocalAgentID)+"\n",
						)
					})
				})
			})
		})
	})
}

package model

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRemoteAccountTableName(t *testing.T) {
	Convey("Given a `RemoteAccount` instance", t, func() {
		agent := &RemoteAccount{}

		Convey("When calling the 'TableName' method", func() {
			name := agent.TableName()

			Convey("Then it should return the name of the remote account table", func() {
				So(name, ShouldEqual, "remote_accounts")
			})
		})
	})
}

func TestRemoteAccountValidateInsert(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains 1 remote agent with 1 remote account", func() {
			parentAgent := RemoteAgent{
				Name:        "parent_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parentAgent)
			So(err, ShouldBeNil)

			oldAccount := RemoteAccount{
				RemoteAgentID: parentAgent.ID,
				Login:         "old",
				Password:      []byte("password"),
			}
			err = db.Create(&oldAccount)
			So(err, ShouldBeNil)

			Convey("Given a new remote account", func() {
				newAccount := RemoteAccount{
					RemoteAgentID: parentAgent.ID,
					Login:         "new",
					Password:      []byte("password"),
				}

				Convey("Given that the new account is valid", func() {

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&newAccount).ValidateInsert(ses)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the new account has an ID", func() {
					newAccount.ID = 1000

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&newAccount).ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that IDs are not allowed", func() {
							So(err.Error(), ShouldEqual, "The account's ID cannot "+
								"be entered manually")
						})
					})
				})

				Convey("Given that the new account is missing an agent ID", func() {
					newAccount.RemoteAgentID = 0

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&newAccount).ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that the agent ID is missing", func() {
							So(err.Error(), ShouldEqual, "The account's agentID "+
								"cannot be empty")
						})
					})
				})

				Convey("Given that the new account is missing a login", func() {
					newAccount.Login = ""

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&newAccount).ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that the login is missing", func() {
							So(err.Error(), ShouldEqual, "The account's login "+
								"cannot be empty")
						})
					})
				})

				Convey("Given that the new account has an invalid agent ID", func() {
					newAccount.RemoteAgentID = 1000

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&newAccount).ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that the agent ID is invalid", func() {
							So(err.Error(), ShouldEqual, "No remote agent found "+
								"with the ID '1000'")
						})
					})
				})

				Convey("Given that the new account's login is already taken", func() {
					newAccount.Login = oldAccount.Login

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&newAccount).ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that the login is already taken", func() {
							So(err.Error(), ShouldEqual, "A remote account with "+
								"the same login '"+newAccount.Login+"' already exist")
						})
					})
				})

				Convey("Given that the new account's name is already taken but the"+
					"parent agent is different", func() {
					otherAgent := RemoteAgent{
						Name:        "other",
						Protocol:    "sftp",
						ProtoConfig: []byte("{}"),
					}
					err := db.Create(&otherAgent)
					So(err, ShouldBeNil)

					newAccount.RemoteAgentID = otherAgent.ID
					newAccount.Login = oldAccount.Login

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&newAccount).ValidateInsert(ses)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})
			})
		})
	})
}

func TestRemoteAccountValidateUpdate(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains 1 remote agent with 2 remote accounts", func() {
			parentAgent := RemoteAgent{
				Name:        "parent_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parentAgent)
			So(err, ShouldBeNil)

			oldAccount := RemoteAccount{
				RemoteAgentID: parentAgent.ID,
				Login:         "old",
				Password:      []byte("password"),
			}
			err = db.Create(&oldAccount)
			So(err, ShouldBeNil)

			otherAccount := RemoteAccount{
				RemoteAgentID: parentAgent.ID,
				Login:         "other",
				Password:      []byte("password"),
			}
			err = db.Create(&otherAccount)
			So(err, ShouldBeNil)

			Convey("Given an updated remote account", func() {
				updatedAccount := RemoteAccount{
					RemoteAgentID: parentAgent.ID,
					Login:         "updated",
					Password:      []byte("new_password"),
				}

				Convey("Given that the updated account is valid", func() {

					Convey("When calling the 'ValidateUpdate' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&updatedAccount).ValidateUpdate(ses, oldAccount.ID)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the updated account has an ID", func() {
					updatedAccount.ID = 1000

					Convey("When calling the 'ValidateUpdate' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&updatedAccount).ValidateUpdate(ses, oldAccount.ID)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that IDs are not allowed", func() {
							So(err.Error(), ShouldEqual, "The account's ID cannot "+
								"be entered manually")
						})
					})
				})

				Convey("Given that the updated account has an invalid agent ID", func() {
					updatedAccount.RemoteAgentID = 1000

					Convey("When calling the 'ValidateUpdate' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&updatedAccount).ValidateUpdate(ses, oldAccount.ID)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that the agent ID is invalid", func() {
							So(err.Error(), ShouldEqual, "No remote agent found "+
								"with the ID '1000'")
						})
					})
				})

				Convey("Given that the updated account's login is already taken", func() {
					updatedAccount.Login = oldAccount.Login

					Convey("When calling the 'ValidateUpdate' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&updatedAccount).ValidateUpdate(ses, oldAccount.ID)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that the login is already taken", func() {
							So(err.Error(), ShouldEqual, "A remote account with "+
								"the same login '"+updatedAccount.Login+"' already exist")
						})
					})
				})

				Convey("Given that the updated account's name is already taken but the"+
					"parent agent is different", func() {
					otherAgent := RemoteAgent{
						Name:        "other",
						Protocol:    "sftp",
						ProtoConfig: []byte("{}"),
					}
					err := db.Create(&otherAgent)
					So(err, ShouldBeNil)

					updatedAccount.RemoteAgentID = otherAgent.ID
					updatedAccount.Login = oldAccount.Login

					Convey("When calling the 'ValidateUpdate' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&updatedAccount).ValidateUpdate(ses, oldAccount.ID)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})
			})
		})
	})
}

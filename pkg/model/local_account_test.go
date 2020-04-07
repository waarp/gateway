package model

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	. "github.com/smartystreets/goconvey/convey"
)

func TestLocalAccountTableName(t *testing.T) {
	Convey("Given a `LocalAccount` instance", t, func() {
		agent := &LocalAccount{}

		Convey("When calling the 'TableName' method", func() {
			name := agent.TableName()

			Convey("Then it should return the name of the local account table", func() {
				So(name, ShouldEqual, "local_accounts")
			})
		})
	})
}

func TestLocalAccountBeforeInsert(t *testing.T) {
	Convey("Given a local account entry", t, func() {
		acc := &LocalAccount{
			ID:           1,
			LocalAgentID: 1,
			Login:        "login",
			Password:     []byte("password"),
		}

		Convey("When calling the `BeforeInsert` hook", func() {
			err := acc.BeforeInsert(nil)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the account's password should be hashed", func() {
				hash, err := hashPassword(acc.Password)
				So(err, ShouldBeNil)
				So(string(acc.Password), ShouldEqual, string(hash))
			})
		})
	})
}

func TestLocalAccountBeforeUpdate(t *testing.T) {
	Convey("Given a local account entry", t, func() {
		acc := &LocalAccount{
			ID:           1,
			LocalAgentID: 1,
			Login:        "login",
			Password:     []byte("password"),
		}

		Convey("When calling the `BeforeUpdate` hook", func() {
			err := acc.BeforeUpdate(nil)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the account's password should be hashed", func() {
				hash, err := hashPassword(acc.Password)
				So(err, ShouldBeNil)
				So(string(acc.Password), ShouldEqual, string(hash))
			})
		})
	})
}

func TestLocalAccountBeforeDelete(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a local account entry", func() {
			ag := &LocalAgent{
				Name:        "test agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
			}
			So(db.Create(ag), ShouldBeNil)

			acc := &LocalAccount{
				LocalAgentID: ag.ID,
				Login:        "login",
				Password:     []byte("password"),
			}
			So(db.Create(acc), ShouldBeNil)

			Convey("Given the account has a certificate", func() {
				cert := &Cert{
					OwnerType:   acc.TableName(),
					OwnerID:     acc.ID,
					Name:        "test cert",
					PrivateKey:  []byte("private key"),
					PublicKey:   []byte("public key"),
					Certificate: []byte("certificate"),
				}
				So(db.Create(cert), ShouldBeNil)

				Convey("When calling the `BeforeDelete` hook", func() {
					err := acc.BeforeDelete(db)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the account's certificate should have been deleted", func() {
						exist, err := db.Exists(cert)
						So(err, ShouldBeNil)
						So(exist, ShouldBeFalse)
					})
				})
			})
		})
	})
}

func TestLocalAccountValidateInsert(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains 1 local agent with 1 local account", func() {
			parentAgent := LocalAgent{
				Owner:       "test_gateway",
				Name:        "parent_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
			}
			err := db.Create(&parentAgent)
			So(err, ShouldBeNil)

			oldAccount := LocalAccount{
				LocalAgentID: parentAgent.ID,
				Login:        "old",
				Password:     []byte("password"),
			}
			err = db.Create(&oldAccount)
			So(err, ShouldBeNil)

			Convey("Given a new local account", func() {
				newAccount := LocalAccount{
					LocalAgentID: parentAgent.ID,
					Login:        "new",
					Password:     []byte("password"),
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
					newAccount.LocalAgentID = 0

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
					newAccount.LocalAgentID = 1000

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&newAccount).ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that the agent ID is invalid", func() {
							So(err.Error(), ShouldEqual, "No local agent found "+
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
							So(err.Error(), ShouldEqual, "A local account with "+
								"the same login '"+newAccount.Login+"' already exist")
						})
					})
				})

				Convey("Given that the new account's name is already taken but the"+
					"parent agent is different", func() {
					otherAgent := LocalAgent{
						Owner:       "test_gateway",
						Name:        "other",
						Protocol:    "sftp",
						ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
					}
					err := db.Create(&otherAgent)
					So(err, ShouldBeNil)

					newAccount.LocalAgentID = otherAgent.ID
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

func TestLocalAccountValidateUpdate(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains 1 local agent with 2 local accounts", func() {
			parentAgent := LocalAgent{
				Owner:       "test_gateway",
				Name:        "parent_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
			}
			err := db.Create(&parentAgent)
			So(err, ShouldBeNil)

			oldAccount := LocalAccount{
				LocalAgentID: parentAgent.ID,
				Login:        "old",
				Password:     []byte("password"),
			}
			err = db.Create(&oldAccount)
			So(err, ShouldBeNil)

			otherAccount := LocalAccount{
				LocalAgentID: parentAgent.ID,
				Login:        "other",
				Password:     []byte("password"),
			}
			err = db.Create(&otherAccount)
			So(err, ShouldBeNil)

			Convey("Given an updated local account", func() {
				updatedAccount := LocalAccount{
					LocalAgentID: parentAgent.ID,
					Login:        "updated",
					Password:     []byte("new_password"),
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
					updatedAccount.LocalAgentID = 1000

					Convey("When calling the 'ValidateUpdate' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&updatedAccount).ValidateUpdate(ses, oldAccount.ID)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that the agent ID is invalid", func() {
							So(err.Error(), ShouldEqual, "No local agent found "+
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
							So(err.Error(), ShouldEqual, "A local account with "+
								"the same login '"+updatedAccount.Login+"' already exist")
						})
					})
				})

				Convey("Given that the updated account's name is already taken but the"+
					"parent agent is different", func() {
					otherAgent := LocalAgent{
						Owner:       "test_gateway",
						Name:        "other",
						Protocol:    "sftp",
						ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
					}
					err := db.Create(&otherAgent)
					So(err, ShouldBeNil)

					updatedAccount.LocalAgentID = otherAgent.ID
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

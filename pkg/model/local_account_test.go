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

func TestLocalAccountBeforeDelete(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a local account entry", func() {
			ag := &LocalAgent{Name: "server", Protocol: "dummy", ProtoConfig: []byte(`{}`)}
			So(db.Create(ag), ShouldBeNil)

			acc := &LocalAccount{LocalAgentID: ag.ID, Login: "login", Password: []byte("password")}
			So(db.Create(acc), ShouldBeNil)

			cert := &Cert{
				OwnerType:   acc.TableName(),
				OwnerID:     acc.ID,
				Name:        "test cert",
				PrivateKey:  []byte("private key"),
				PublicKey:   []byte("public key"),
				Certificate: []byte("certificate"),
			}
			So(db.Create(cert), ShouldBeNil)

			rule := &Rule{Name: "rule", IsSend: true, Path: "path"}
			So(db.Create(rule), ShouldBeNil)

			access := &RuleAccess{RuleID: rule.ID, ObjectType: acc.TableName(), ObjectID: acc.ID}
			So(db.Create(access), ShouldBeNil)

			Convey("Given that the account is unused", func() {

				Convey("When calling the `BeforeDelete` hook", func() {
					So(acc.BeforeDelete(db), ShouldBeNil)

					Convey("Then the account's certificates should have been deleted", func() {
						certs, err := db.Query("SELECT * FROM certificates")
						So(err, ShouldBeNil)
						So(certs, ShouldBeEmpty)
					})

					Convey("Then the account's accesses should have been deleted", func() {
						access, err := db.Query("SELECT * FROM rule_access")
						So(err, ShouldBeNil)
						So(access, ShouldBeEmpty)
					})
				})
			})

			Convey("Given that the account is used in a transfer", func() {
				trans := &Transfer{
					RuleID:     rule.ID,
					IsServer:   true,
					AgentID:    ag.ID,
					AccountID:  acc.ID,
					SourceFile: "file.src",
					DestFile:   "file.dst",
				}
				So(db.Create(trans), ShouldBeNil)

				Convey("When calling the `BeforeDelete` hook", func() {
					err := acc.BeforeDelete(db)

					Convey("Then it should say that the account is being used", func() {
						So(err, ShouldBeError, "this account is currently being "+
							"used in a running transfer and cannot be deleted, "+
							"cancel the transfer or wait for it to finish")
					})
				})
			})
		})
	})
}

func TestLocalAccountBeforeInsert(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains 1 local agent with 1 local account", func() {
			parentAgent := &LocalAgent{
				Owner:       "test_gateway",
				Name:        "parent_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
			}
			So(db.Create(parentAgent), ShouldBeNil)

			oldAccount := &LocalAccount{
				LocalAgentID: parentAgent.ID,
				Login:        "old",
				Password:     []byte("password"),
			}
			So(db.Create(oldAccount), ShouldBeNil)

			Convey("Given a new local account", func() {
				newAccount := &LocalAccount{
					LocalAgentID: parentAgent.ID,
					Login:        "new",
					Password:     []byte("password"),
				}

				Convey("Given that the new account is valid", func() {

					Convey("When calling the 'BeforeInsert' function", func() {
						So(newAccount.BeforeInsert(db), ShouldBeNil)

						Convey("Then the account's password should be hashed", func() {
							hash, err := hashPassword(newAccount.Password)
							So(err, ShouldBeNil)
							So(string(newAccount.Password), ShouldEqual, string(hash))
						})
					})
				})

				Convey("Given that the new account has an ID", func() {
					newAccount.ID = 1000

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newAccount.BeforeInsert(db)

						Convey("Then the error should say that IDs are not allowed", func() {
							So(err, ShouldBeError, "the account's ID cannot "+
								"be entered manually")
						})
					})
				})

				Convey("Given that the new account is missing an agent ID", func() {
					newAccount.LocalAgentID = 0

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newAccount.BeforeInsert(db)

						Convey("Then the error should say that the agent ID is missing", func() {
							So(err, ShouldBeError, "the account's agentID "+
								"cannot be empty")
						})
					})
				})

				Convey("Given that the new account is missing a login", func() {
					newAccount.Login = ""

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newAccount.BeforeInsert(db)

						Convey("Then the error should say that the login is missing", func() {
							So(err, ShouldBeError, "the account's login "+
								"cannot be empty")
						})
					})
				})

				Convey("Given that the new account has an invalid agent ID", func() {
					newAccount.LocalAgentID = 1000

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newAccount.BeforeInsert(db)

						Convey("Then the error should say that the agent ID is invalid", func() {
							So(err, ShouldBeError, "no local agent found "+
								"with the ID '1000'")
						})
					})
				})

				Convey("Given that the new account's login is already taken", func() {
					newAccount.Login = oldAccount.Login

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newAccount.BeforeInsert(db)

						Convey("Then the error should say that the login is already taken", func() {
							So(err, ShouldBeError, "a local account with "+
								"the same login '"+newAccount.Login+"' already exist")
						})
					})
				})

				Convey("Given that the new account's name is already taken but the"+
					"parent agent is different", func() {
					otherAgent := &LocalAgent{
						Owner:       "test_gateway",
						Name:        "other",
						Protocol:    "sftp",
						ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
					}
					So(db.Create(otherAgent), ShouldBeNil)

					newAccount.LocalAgentID = otherAgent.ID
					newAccount.Login = oldAccount.Login

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newAccount.BeforeInsert(db)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})
			})
		})
	})
}

func TestLocalAccountBeforeUpdate(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains 1 local agent with 2 local accounts", func() {
			parentAgent := &LocalAgent{
				Owner:       "test_gateway",
				Name:        "parent_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
			}
			So(db.Create(parentAgent), ShouldBeNil)

			oldAccount := &LocalAccount{
				LocalAgentID: parentAgent.ID,
				Login:        "old",
				Password:     []byte("password"),
			}
			So(db.Create(oldAccount), ShouldBeNil)

			otherAccount := &LocalAccount{
				LocalAgentID: parentAgent.ID,
				Login:        "other",
				Password:     []byte("password"),
			}
			So(db.Create(otherAccount), ShouldBeNil)

			Convey("Given an updated local account", func() {
				updatedAccount := &LocalAccount{
					LocalAgentID: parentAgent.ID,
					Login:        "updated",
					Password:     []byte("new_password"),
				}

				Convey("Given that the updated account is valid", func() {

					Convey("When calling the 'BeforeUpdate' function", func() {
						So(updatedAccount.BeforeUpdate(db, oldAccount.ID), ShouldBeNil)

						Convey("Then the account's password should be hashed", func() {
							hash, err := hashPassword(updatedAccount.Password)
							So(err, ShouldBeNil)
							So(string(updatedAccount.Password), ShouldEqual, string(hash))
						})
					})
				})

				Convey("Given that the updated account has an ID", func() {
					updatedAccount.ID = 1000

					Convey("When calling the 'BeforeUpdate' function", func() {
						err := updatedAccount.BeforeUpdate(db, oldAccount.ID)

						Convey("Then the error should say that IDs are not allowed", func() {
							So(err, ShouldBeError, "the account's ID cannot "+
								"be entered manually")
						})
					})
				})

				Convey("Given that the updated account has an invalid agent ID", func() {
					updatedAccount.LocalAgentID = 1000

					Convey("When calling the 'BeforeUpdate' function", func() {
						err := updatedAccount.BeforeUpdate(db, oldAccount.ID)

						Convey("Then the error should say that the agent ID is invalid", func() {
							So(err, ShouldBeError, "no local agent found "+
								"with the ID '1000'")
						})
					})
				})

				Convey("Given that the updated account's login is already taken", func() {
					updatedAccount.Login = otherAccount.Login

					Convey("When calling the 'BeforeUpdate' function", func() {
						err := updatedAccount.BeforeUpdate(db, oldAccount.ID)

						Convey("Then the error should say that the login is already taken", func() {
							So(err, ShouldBeError, "a local account with "+
								"the same login '"+updatedAccount.Login+
								"' already exist")
						})
					})
				})

				Convey("Given that the updated account's name is already taken but the"+
					"parent agent is different", func() {
					otherAgent := &LocalAgent{
						Owner:       "test_gateway",
						Name:        "other",
						Protocol:    "sftp",
						ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
					}
					So(db.Create(otherAgent), ShouldBeNil)

					updatedAccount.LocalAgentID = otherAgent.ID
					updatedAccount.Login = oldAccount.Login

					Convey("When calling the 'BeforeUpdate' function", func() {
						err := updatedAccount.BeforeUpdate(db, oldAccount.ID)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})
			})
		})
	})
}

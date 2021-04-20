package model

import (
	"encoding/json"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

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

func TestRemoteAccountBeforeDelete(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given a remote account entry", func() {
			ag := RemoteAgent{
				Name:        "server",
				Protocol:    dummyProto,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1111",
			}
			So(db.Insert(&ag).Run(), ShouldBeNil)

			acc := RemoteAccount{RemoteAgentID: ag.ID, Login: "login", Password: "password"}
			So(db.Insert(&acc).Run(), ShouldBeNil)

			cert := Crypto{
				OwnerType:   "remote_accounts",
				OwnerID:     acc.ID,
				Name:        "test cert",
				PrivateKey:  testhelpers.ClientKey,
				Certificate: testhelpers.ClientCert,
			}
			So(db.Insert(&cert).Run(), ShouldBeNil)

			rule := Rule{Name: "rule", IsSend: true, Path: "path"}
			So(db.Insert(&rule).Run(), ShouldBeNil)

			access := RuleAccess{RuleID: rule.ID, ObjectType: "remote_accounts", ObjectID: acc.ID}
			So(db.Insert(&access).Run(), ShouldBeNil)

			Convey("Given that the account is unused", func() {

				Convey("When calling the `BeforeDelete` hook", func() {
					So(db.Transaction(func(ses *database.Session) database.Error {
						return acc.BeforeDelete(ses)
					}), ShouldBeNil)

					Convey("Then the account's certificates should have been deleted", func() {
						var certs Cryptos
						So(db.Select(&certs).Run(), ShouldBeNil)
						So(certs, ShouldBeEmpty)
					})

					Convey("Then the account's accesses should have been deleted", func() {
						var perms RuleAccesses
						So(db.Select(&perms).Run(), ShouldBeNil)
						So(perms, ShouldBeEmpty)
					})
				})
			})

			Convey("Given that the account is used in a transfer", func() {
				trans := Transfer{
					RuleID:     rule.ID,
					IsServer:   false,
					AgentID:    ag.ID,
					AccountID:  acc.ID,
					SourceFile: "file.src",
					DestFile:   "file.dst",
				}
				So(db.Insert(&trans).Run(), ShouldBeNil)

				Convey("When calling the `BeforeDelete` hook", func() {
					err := db.Transaction(func(ses *database.Session) database.Error {
						return acc.BeforeDelete(ses)
					})

					Convey("Then it should say that the account is being used", func() {
						So(err, ShouldBeError, database.NewValidationError(
							"this account is currently being used in a running "+
								"transfer and cannot be deleted, cancel "+
								"the transfer or wait for it to finish"))
					})
				})
			})
		})
	})
}

func TestRemoteAccountBeforeWrite(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given the database contains 1 remote agent with 1 remote account", func() {
			parentAgent := RemoteAgent{
				Name:        "parent_agent",
				Protocol:    dummyProto,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}
			So(db.Insert(&parentAgent).Run(), ShouldBeNil)

			oldAccount := RemoteAccount{
				RemoteAgentID: parentAgent.ID,
				Login:         "old",
				Password:      "password",
			}
			So(db.Insert(&oldAccount).Run(), ShouldBeNil)

			Convey("Given a new remote account", func() {
				newAccount := RemoteAccount{
					RemoteAgentID: parentAgent.ID,
					Login:         "new",
					Password:      "password",
				}

				shouldFailWith := func(errDesc string, expErr error) {
					Convey("When calling the 'BeforeWrite' function", func() {
						err := db.Transaction(func(ses *database.Session) database.Error {
							return newAccount.BeforeWrite(ses)
						})

						Convey("Then the error should say that "+errDesc, func() {
							So(err, ShouldBeError, expErr)
						})
					})
				}

				Convey("Given that the new account is valid", func() {
					Convey("When calling the 'BeforeWrite' function", func() {
						err := db.Transaction(func(ses *database.Session) database.Error {
							return newAccount.BeforeWrite(ses)
						})

						Convey("Then it should not return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the new account is missing an agent ID", func() {
					newAccount.RemoteAgentID = 0
					shouldFailWith("the agent ID is missing", database.NewValidationError(
						"the account's agentID cannot be empty"))
				})

				Convey("Given that the new account is missing a login", func() {
					newAccount.Login = ""
					shouldFailWith("the login is missing", database.NewValidationError(
						"the account's login cannot be empty"))
				})

				Convey("Given that the new account has an invalid agent ID", func() {
					newAccount.RemoteAgentID = 1000
					shouldFailWith("the agent ID is invalid", database.NewValidationError(
						"no remote agent found with the ID '%d'", newAccount.RemoteAgentID))
				})

				Convey("Given that the new account's login is already taken", func() {
					newAccount.Login = oldAccount.Login
					shouldFailWith("the login is already taken", database.NewValidationError(
						"a remote account with the same login '%s' already exist",
						newAccount.Login))
				})

				Convey("Given that the new account's name is already taken but the"+
					"parent agent is different", func() {
					otherAgent := RemoteAgent{
						Name:        "other",
						Protocol:    dummyProto,
						ProtoConfig: json.RawMessage(`{}`),
						Address:     "localhost:2022",
					}
					So(db.Insert(&otherAgent).Run(), ShouldBeNil)

					newAccount.RemoteAgentID = otherAgent.ID
					newAccount.Login = oldAccount.Login

					Convey("When calling the 'BeforeWrite' function", func() {
						err := db.Transaction(func(ses *database.Session) database.Error {
							return newAccount.BeforeWrite(ses)
						})

						Convey("Then it should not return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})
			})
		})
	})
}

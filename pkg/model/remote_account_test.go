package model

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestRemoteAccountTableName(t *testing.T) {
	Convey("Given a `RemoteAccount` instance", t, func() {
		agent := &RemoteAccount{}

		Convey("When calling the 'TableName' method", func() {
			name := agent.TableName()

			Convey("Then it should return the name of the remote account table", func() {
				So(name, ShouldEqual, TableRemAccounts)
			})
		})
	})
}

func TestRemoteAccountBeforeDelete(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given a remote account entry", func() {
			ag := RemoteAgent{
				Name: "server", Protocol: testProtocol,
				Address: types.Addr("localhost", 1111),
			}
			So(db.Insert(&ag).Run(), ShouldBeNil)

			acc := RemoteAccount{RemoteAgentID: ag.ID, Login: "foo"}
			So(db.Insert(&acc).Run(), ShouldBeNil)

			accAuth := Credential{
				RemoteAccountID: utils.NewNullInt64(acc.ID),
				Name:            "test cert",
				Type:            testExternalAuth,
				Value:           "val",
			}
			So(db.Insert(&accAuth).Run(), ShouldBeNil)

			rule := Rule{Name: "rule", IsSend: true, Path: "path"}
			So(db.Insert(&rule).Run(), ShouldBeNil)

			access := RuleAccess{RuleID: rule.ID, RemoteAccountID: utils.NewNullInt64(acc.ID)}
			So(db.Insert(&access).Run(), ShouldBeNil)

			Convey("Given that the account is unused", func() {
				Convey("When calling the `BeforeDelete` hook", func() {
					So(acc.BeforeDelete(db), ShouldBeNil)
				})

				Convey("When deleting the account", func() {
					So(db.Delete(&acc).Run(), ShouldBeNil)

					Convey("Then the account should have been deleted", func() {
						var accounts RemoteAccounts
						So(db.Select(&accounts).Run(), ShouldBeNil)
						So(accounts, ShouldBeEmpty)
					})

					Convey("Then the account's certificates should have been deleted", func() {
						var auths Credentials
						So(db.Select(&auths).Run(), ShouldBeNil)
						So(auths, ShouldBeEmpty)
					})

					Convey("Then the account's accesses should have been deleted", func() {
						var perms RuleAccesses
						So(db.Select(&perms).Run(), ShouldBeNil)
						So(perms, ShouldBeEmpty)
					})
				})
			})

			Convey("Given that the account is used in a transfer", func() {
				cli := &Client{Protocol: ag.Protocol}
				So(db.Insert(cli).Run(), ShouldBeNil)

				trans := Transfer{
					RuleID:          rule.ID,
					ClientID:        utils.NewNullInt64(cli.ID),
					RemoteAccountID: utils.NewNullInt64(acc.ID),
					SrcFilename:     "file",
				}
				So(db.Insert(&trans).Run(), ShouldBeNil)

				Convey("When calling the `BeforeDelete` hook", func() {
					err := acc.BeforeDelete(db)

					Convey("Then it should say that the account is being used", func() {
						So(err, ShouldBeError, database.NewValidationError(
							"this account is currently being used in one or more "+
								"running transfers and thus cannot be deleted, "+
								"cancel these transfers or wait for them to finish"))
					})
				})
			})
		})
	})
}

func TestRemoteAccountBeforeWrite(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given the database contains 1 remote agent with 1 remote account", func() {
			parentAgent := RemoteAgent{
				Name: "parent_agent", Protocol: testProtocol,
				Address: types.Addr("localhost", 2022),
			}
			So(db.Insert(&parentAgent).Run(), ShouldBeNil)

			oldAccount := RemoteAccount{
				RemoteAgentID: parentAgent.ID,
				Login:         "old",
			}
			So(db.Insert(&oldAccount).Run(), ShouldBeNil)

			Convey("Given a new remote account", func() {
				newAccount := RemoteAccount{
					RemoteAgentID: parentAgent.ID,
					Login:         "new",
				}

				shouldFailWith := func(errDesc string, expErr error) {
					Convey("When calling the 'BeforeWrite' function", func() {
						err := db.Transaction(func(ses *database.Session) error {
							return newAccount.BeforeWrite(ses)
						})

						Convey("Then the error should say that "+errDesc, func() {
							So(err, ShouldBeError, expErr)
						})
					})
				}

				Convey("Given that the new account is valid", func() {
					Convey("When calling the 'BeforeWrite' function", func() {
						err := db.Transaction(func(ses *database.Session) error {
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
						`no remote agent found with the ID "%d"`, newAccount.RemoteAgentID))
				})

				Convey("Given that the new account's login is already taken", func() {
					newAccount.Login = oldAccount.Login

					shouldFailWith("the login is already taken", database.NewValidationError(
						"a remote account with the same login %q already exist",
						newAccount.Login))
				})

				Convey("Given that the new account's name is already taken but the"+
					"parent agent is different", func() {
					otherAgent := RemoteAgent{
						Name: "other", Protocol: testProtocol,
						Address: types.Addr("localhost", 2022),
					}
					So(db.Insert(&otherAgent).Run(), ShouldBeNil)

					newAccount.RemoteAgentID = otherAgent.ID
					newAccount.Login = oldAccount.Login

					Convey("When calling the 'BeforeWrite' function", func() {
						err := db.Transaction(func(ses *database.Session) error {
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

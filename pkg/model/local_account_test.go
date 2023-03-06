package model

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestLocalAccountTableName(t *testing.T) {
	Convey("Given a `LocalAccount` instance", t, func() {
		agent := &LocalAccount{}

		Convey("When calling the 'TableName' method", func() {
			name := agent.TableName()

			Convey("Then it should return the name of the local account table", func() {
				So(name, ShouldEqual, TableLocAccounts)
			})
		})
	})
}

func TestLocalAccountBeforeDelete(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given a local account entry", func() {
			ag := &LocalAgent{
				Name:     "server",
				Protocol: testProtocol,
				Address:  "localhost:1111",
			}
			So(db.Insert(ag).Run(), ShouldBeNil)

			acc := LocalAccount{LocalAgentID: ag.ID, Login: "foo", PasswordHash: hash("sesame")}
			So(db.Insert(&acc).Run(), ShouldBeNil)

			cert := Crypto{
				LocalAccountID: utils.NewNullInt64(acc.ID),
				Name:           "test cert",
				Certificate:    testhelpers.ClientFooCert,
			}
			So(db.Insert(&cert).Run(), ShouldBeNil)

			rule := Rule{Name: "rule", IsSend: true, Path: "path"}
			So(db.Insert(&rule).Run(), ShouldBeNil)

			access := RuleAccess{RuleID: rule.ID, LocalAccountID: utils.NewNullInt64(acc.ID)}
			So(db.Insert(&access).Run(), ShouldBeNil)

			Convey("Given that the account is unused", func() {
				Convey("When calling the `BeforeDelete` hook", func() {
					So(acc.BeforeDelete(db), ShouldBeNil)
				})

				Convey("When deleting the account", func() {
					So(db.Delete(&acc).Run(), ShouldBeNil)

					Convey("Then the agent's accounts should have been deleted", func() {
						var accounts LocalAccounts
						So(db.Select(&accounts).Run(), ShouldBeNil)
						So(accounts, ShouldBeEmpty)
					})

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
				trans := &Transfer{
					RuleID:         rule.ID,
					LocalAccountID: utils.NewNullInt64(acc.ID),
					SrcFilename:    "file",
				}
				So(db.Insert(trans).Run(), ShouldBeNil)

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

func TestLocalAccountBeforeWrite(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given the database contains 1 local agent with 1 local account", func() {
			parentAgent := LocalAgent{
				Owner:    "test_gateway",
				Name:     "parent_agent",
				Protocol: testProtocol,
				Address:  "localhost:2222",
			}
			So(db.Insert(&parentAgent).Run(), ShouldBeNil)

			Convey("Given a new local account", func() {
				newAccount := &LocalAccount{
					LocalAgentID: parentAgent.ID,
					Login:        "new",
					PasswordHash: hash("sesame"),
				}

				shouldFailWith := func(errDesc string, expErr error) {
					Convey("When calling the 'BeforeValidate' function", func() {
						err := newAccount.BeforeWrite(db)

						Convey("Then the error should say that "+errDesc, func() {
							So(err, ShouldBeError, expErr)
						})
					})
				}

				Convey("Given that the new account is valid", func() {
					Convey("When calling the 'BeforeWrite' function", func() {
						err := newAccount.BeforeWrite(db)

						Convey("Then it should not return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the new account is missing an agent ID", func() {
					newAccount.LocalAgentID = 0

					shouldFailWith("the agent ID is missing", database.NewValidationError(
						"the account's agentID cannot be empty"))
				})

				Convey("Given that the new account is missing a login", func() {
					newAccount.Login = ""

					shouldFailWith("the login is missing", database.NewValidationError(
						"the account's login cannot be empty"))
				})

				Convey("Given that the new account has an invalid agent ID", func() {
					newAccount.LocalAgentID = 1000

					shouldFailWith("the agent ID is invalid", database.NewValidationError(
						"no local agent found with the ID '%d'", newAccount.LocalAgentID))
				})

				Convey("Given that the new account's login is already taken", func() {
					oldAccount := LocalAccount{
						LocalAgentID: parentAgent.ID,
						Login:        "old",
						PasswordHash: hash("sesame"),
					}
					So(db.Insert(&oldAccount).Run(), ShouldBeNil)

					newAccount.Login = oldAccount.Login
					shouldFailWith("the login is already taken", database.NewValidationError(
						"a local account with the same login '%s' already exist",
						newAccount.Login))
				})

				Convey("Given that the new account's name is already taken but the"+
					"parent agent is different", func() {
					otherAgent := LocalAgent{
						Owner:    "test_gateway",
						Name:     "other",
						Protocol: testProtocol,
						Address:  "localhost:2022",
					}
					So(db.Insert(&otherAgent).Run(), ShouldBeNil)

					oldAccount := LocalAccount{
						LocalAgentID: parentAgent.ID,
						Login:        "old",
						PasswordHash: hash("sesame"),
					}
					So(db.Insert(&oldAccount).Run(), ShouldBeNil)

					newAccount.LocalAgentID = otherAgent.ID
					newAccount.Login = oldAccount.Login

					Convey("When calling the 'Validate' function", func() {
						err := newAccount.BeforeWrite(db)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})
			})
		})
	})
}

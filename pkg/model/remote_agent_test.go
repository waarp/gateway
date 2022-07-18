package model

import (
	"encoding/json"
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestRemoteAgentTableName(t *testing.T) {
	Convey("Given a `RemoteAgent` instance", t, func() {
		agent := &RemoteAgent{}

		Convey("When calling the 'TableName' method", func() {
			name := agent.TableName()

			Convey("Then it should return the name of the remote agents table", func() {
				So(name, ShouldEqual, TableRemAgents)
			})
		})
	})
}

func TestRemoteAgentBeforeDelete(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given a remote agent entry", func() {
			ag := RemoteAgent{
				Name:     "partner",
				Protocol: testProtocol,
				Address:  "localhost:6666",
			}
			So(db.Insert(&ag).Run(), ShouldBeNil)

			acc := RemoteAccount{RemoteAgentID: ag.ID, Login: "foo", Password: "sesame"}
			So(db.Insert(&acc).Run(), ShouldBeNil)

			rule := Rule{Name: "rule", IsSend: false, Path: "path"}
			So(db.Insert(&rule).Run(), ShouldBeNil)

			agAccess := RuleAccess{
				RuleID:        rule.ID,
				RemoteAgentID: utils.NewNullInt64(ag.ID),
			}
			So(db.Insert(&agAccess).Run(), ShouldBeNil)

			accAccess := RuleAccess{
				RuleID:          rule.ID,
				RemoteAccountID: utils.NewNullInt64(acc.ID),
			}
			So(db.Insert(&accAccess).Run(), ShouldBeNil)

			certAg := Crypto{
				RemoteAgentID: utils.NewNullInt64(ag.ID),
				Name:          "test agent cert",
				Certificate:   testhelpers.LocalhostCert,
			}
			So(db.Insert(&certAg).Run(), ShouldBeNil)

			certAcc := Crypto{
				RemoteAccountID: utils.NewNullInt64(acc.ID),
				Name:            "test account cert",
				PrivateKey:      testhelpers.ClientFooKey,
				Certificate:     testhelpers.ClientFooCert,
			}
			So(db.Insert(&certAcc).Run(), ShouldBeNil)

			Convey("Given that the agent is unused", func() {
				Convey("When calling the `BeforeDelete` hook", func() {
					So(ag.BeforeDelete(db), ShouldBeNil)
				})

				Convey("When deleting the agent", func() {
					So(db.Delete(&ag).Run(), ShouldBeNil)

					Convey("Then the agent should have been deleted", func() {
						var agents RemoteAgents
						So(db.Select(&agents).Run(), ShouldBeNil)
						So(agents, ShouldBeEmpty)
					})

					Convey("Then the agent's accounts should have been deleted", func() {
						var accounts RemoteAccounts
						So(db.Select(&accounts).Run(), ShouldBeNil)
						So(accounts, ShouldBeEmpty)
					})

					Convey("Then both certificates should have been deleted", func() {
						var certs Cryptos
						So(db.Select(&certs).Run(), ShouldBeNil)
						So(certs, ShouldBeEmpty)
					})

					Convey("Then the rule accesses should have been deleted", func() {
						var perms RuleAccesses
						So(db.Select(&perms).Run(), ShouldBeNil)
						So(perms, ShouldBeEmpty)
					})
				})
			})

			Convey("Given that the agent is used in a transfer", func() {
				trans := &Transfer{
					RuleID:          rule.ID,
					RemoteAccountID: utils.NewNullInt64(acc.ID),
					SrcFilename:     "file",
				}
				So(db.Insert(trans).Run(), ShouldBeNil)

				Convey("When calling the `BeforeDelete` hook", func() {
					err := ag.BeforeDelete(db)

					Convey("Then it should say that the agent is being used", func() {
						So(err, ShouldBeError, database.NewValidationError(
							"this partner is currently being used in one or more "+
								"running transfers and thus cannot be deleted, "+
								"cancel these transfers or wait for them to finish"))
					})
				})
			})
		})
	})
}

func TestRemoteAgentValidate(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given the database contains 1 remote agent", func() {
			oldAgent := RemoteAgent{
				Name:     "old",
				Protocol: testProtocol,
				Address:  "localhost:2022",
			}
			So(db.Insert(&oldAgent).Run(), ShouldBeNil)

			Convey("Given a new remote agent", func() {
				newAgent := &RemoteAgent{
					Name:     "new",
					Protocol: testProtocol,
					Address:  "localhost:2023",
				}

				shouldFailWith := func(expMsg string, args ...any) {
					expErr := fmt.Sprintf(expMsg, args...)

					Convey("When calling the 'BeforeWrite' function", func() {
						err := db.Transaction(func(ses *database.Session) database.Error {
							return newAgent.BeforeWrite(ses)
						})

						Convey("Then the error should say that "+expErr, func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldContainSubstring, expErr)
						})
					})
				}

				Convey("Given that the new agent is valid", func() {
					Convey("When calling the 'BeforeWrite' function", func() {
						err := db.Transaction(func(ses *database.Session) database.Error {
							return newAgent.BeforeWrite(ses)
						})

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the new agent is missing a name", func() {
					newAgent.Name = ""

					shouldFailWith("the agent's name cannot be empty")
				})

				Convey("Given that the new agent's name is already taken", func() {
					newAgent.Name = oldAgent.Name

					shouldFailWith(
						"a remote agent with the same name '%s' already exist",
						newAgent.Name)
				})

				Convey("Given that the new agent is missing an address", func() {
					newAgent.Address = ""

					shouldFailWith("the partner's address cannot be empty")
				})

				Convey("Given that the new agent's address is invalid", func() {
					newAgent.Address = "not_an_address"

					shouldFailWith("'not_an_address' is not a valid partner address")
				})

				Convey("Given that the new agent's protocol is not valid", func() {
					newAgent.Protocol = "not a protocol"

					shouldFailWith(`unknown protocol "not a protocol"`)
				})

				Convey("Given that the new agent's protocol configuration is not valid", func() {
					newAgent.ProtoConfig = json.RawMessage("invalid")

					shouldFailWith("failed to parse the partner protocol configuration")
				})
			})
		})
	})
}

package model

import (
	"encoding/json"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	. "github.com/smartystreets/goconvey/convey"
)

func TestLocalAgentTableName(t *testing.T) {
	Convey("Given a `LocalAgent` instance", t, func() {
		agent := &LocalAgent{}

		Convey("When calling the 'TableName' method", func() {
			name := agent.TableName()

			Convey("Then it should return the name of the local agents table", func() {
				So(name, ShouldEqual, "local_agents")
			})
		})
	})
}

func TestLocalAgentBeforeDelete(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given a local agent entry", func() {
			ag := LocalAgent{
				Name:        "test agent",
				Protocol:    dummyProto,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:6666",
			}
			So(db.Insert(&ag).Run(), ShouldBeNil)

			acc := LocalAccount{LocalAgentID: ag.ID, Login: "login", PasswordHash: hash("password")}
			So(db.Insert(&acc).Run(), ShouldBeNil)

			rule := Rule{Name: "rule", IsSend: false, Path: "path"}
			So(db.Insert(&rule).Run(), ShouldBeNil)

			agAccess := RuleAccess{RuleID: rule.ID, ObjectID: ag.ID, ObjectType: ag.TableName()}
			So(db.Insert(&agAccess).Run(), ShouldBeNil)
			accAccess := RuleAccess{RuleID: rule.ID, ObjectID: acc.ID, ObjectType: acc.TableName()}
			So(db.Insert(&accAccess).Run(), ShouldBeNil)

			certAg := Crypto{
				OwnerType:   "local_agents",
				OwnerID:     ag.ID,
				Name:        "test agent cert",
				PrivateKey:  testhelpers.LocalhostKey,
				Certificate: testhelpers.LocalhostCert,
			}
			So(db.Insert(&certAg).Run(), ShouldBeNil)

			certAcc := Crypto{
				OwnerType:   "local_accounts",
				OwnerID:     acc.ID,
				Name:        "test account cert",
				Certificate: testhelpers.ClientCert,
			}
			So(db.Insert(&certAcc).Run(), ShouldBeNil)

			Convey("Given that the agent is unused", func() {

				Convey("When calling the `BeforeDelete` hook", func() {
					So(db.Transaction(func(ses *database.Session) database.Error {
						return ag.BeforeDelete(ses)
					}), ShouldBeNil)

					Convey("Then the agent's accounts should have been deleted", func() {
						var accounts LocalAccounts
						So(db.Select(&accounts).Run(), ShouldBeNil)
						So(accounts, ShouldBeEmpty)
					})

					Convey("Then both certificates should have been deleted", func() {
						var cryptos Cryptos
						So(db.Select(&cryptos).Run(), ShouldBeNil)
						So(cryptos, ShouldBeEmpty)
					})

					Convey("Then the rule accesses should have been deleted", func() {
						var perms RuleAccesses
						So(db.Select(&perms).Run(), ShouldBeNil)
						So(perms, ShouldBeEmpty)
					})
				})
			})

			Convey("Given that the agent is used in a transfer", func() {
				trans := Transfer{
					RuleID:     rule.ID,
					IsServer:   true,
					AgentID:    ag.ID,
					AccountID:  acc.ID,
					LocalPath:  "file.loc",
					RemotePath: "file.rem",
				}
				So(db.Insert(&trans).Run(), ShouldBeNil)

				Convey("When calling the `BeforeDelete` hook", func() {
					err := db.Transaction(func(ses *database.Session) database.Error {
						return ag.BeforeDelete(ses)
					})

					Convey("Then it should say that the agent is being used", func() {
						So(err, ShouldBeError, database.NewValidationError(
							"this server is currently being used in one or more "+
								"running transfers and thus cannot be deleted, "+
								"cancel these transfers or wait for them to finish"))
					})
				})
			})
		})
	})
}

func TestLocalAgentBeforeWrite(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given the database contains 1 local agent", func() {
			oldAgent := LocalAgent{
				Owner:       "test_gateway",
				Name:        "old",
				Protocol:    dummyProto,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}
			So(db.Insert(&oldAgent).Run(), ShouldBeNil)

			Convey("Given a new local agent", func() {
				newAgent := &LocalAgent{
					Owner:       "test_gateway",
					Name:        "new",
					Root:        "root",
					LocalInDir:  "rcv",
					LocalOutDir: "send",
					LocalTmpDir: "tmp",
					Protocol:    dummyProto,
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:2023",
				}

				shouldFailWith := func(errDesc string, expErr error) {
					Convey("When calling the 'BeforeWrite' function", func() {
						err := db.Transaction(func(ses *database.Session) database.Error {
							return newAgent.BeforeWrite(ses)
						})

						Convey("Then the error should say that "+errDesc, func() {
							So(err, ShouldBeError, expErr)
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
					shouldFailWith("the name is missing", database.NewValidationError(
						"the agent's name cannot be empty"))
				})

				Convey("Given that the new agent's name is already taken", func() {
					newAgent.Name = oldAgent.Name
					shouldFailWith("the name is already taken", database.NewValidationError(
						"a local agent with the same name '%s' already exist",
						newAgent.Name))
				})

				Convey("Given that the new agent is missing an address", func() {
					newAgent.Address = ""
					shouldFailWith("the address is missing", database.NewValidationError(
						"the server's address cannot be empty"))
				})

				Convey("Given that the new agent's address is invalid", func() {
					newAgent.Address = "not_an_address"
					shouldFailWith("the address is invalid", database.NewValidationError(
						"'not_an_address' is not a valid server address"))
				})

				Convey("Given that the new agent's protocol is not valid", func() {
					newAgent.Protocol = "not a protocol"
					shouldFailWith("the protocol is invalid", database.NewValidationError(
						"unknown protocol 'not a protocol'"))
				})

				Convey("Given that the new agent's protocol configuration is not valid", func() {
					newAgent.ProtoConfig = json.RawMessage("invalid")
					shouldFailWith("the configuration is invalid",
						database.NewValidationError("failed to parse protocol "+
							"configuration: invalid character 'i' "+
							"looking for beginning of value"))
				})
			})
		})
	})
}

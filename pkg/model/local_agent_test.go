package model

import (
	"encoding/json"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
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
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a local agent entry", func() {
			ag := &LocalAgent{Name: "test agent", Protocol: "dummy", ProtoConfig: json.RawMessage(`{}`)}
			So(db.Create(ag), ShouldBeNil)

			acc := &LocalAccount{LocalAgentID: ag.ID, Login: "login", Password: json.RawMessage("password")}
			So(db.Create(acc), ShouldBeNil)

			rule := &Rule{Name: "rule", IsSend: false, Path: "path"}
			So(db.Create(rule), ShouldBeNil)

			agAccess := &RuleAccess{RuleID: rule.ID, ObjectID: ag.ID, ObjectType: ag.TableName()}
			So(db.Create(agAccess), ShouldBeNil)
			accAccess := &RuleAccess{RuleID: rule.ID, ObjectID: acc.ID, ObjectType: acc.TableName()}
			So(db.Create(accAccess), ShouldBeNil)

			certAg := &Cert{
				OwnerType:   ag.TableName(),
				OwnerID:     ag.ID,
				Name:        "test agent cert",
				PrivateKey:  json.RawMessage("private key"),
				PublicKey:   json.RawMessage("public key"),
				Certificate: json.RawMessage("certificate"),
			}
			So(db.Create(certAg), ShouldBeNil)

			certAcc := &Cert{
				OwnerType:   acc.TableName(),
				OwnerID:     acc.ID,
				Name:        "test account cert",
				PrivateKey:  json.RawMessage("private key"),
				PublicKey:   json.RawMessage("public key"),
				Certificate: json.RawMessage("certificate"),
			}
			So(db.Create(certAcc), ShouldBeNil)

			Convey("Given that the agent is unused", func() {

				Convey("When calling the `BeforeDelete` hook", func() {
					So(ag.BeforeDelete(db), ShouldBeNil)

					Convey("Then the agent's accounts should have been deleted", func() {
						accounts, err := db.Query("SELECT * FROM local_accounts")
						So(err, ShouldBeNil)
						So(accounts, ShouldBeEmpty)
					})

					Convey("Then both certificates should have been deleted", func() {
						certs, err := db.Query("SELECT * FROM certificates")
						So(err, ShouldBeNil)
						So(certs, ShouldBeEmpty)
					})

					Convey("Then the rule accesses should have been deleted", func() {
						access, err := db.Query("SELECT * FROM rule_access")
						So(err, ShouldBeNil)
						So(access, ShouldBeEmpty)
					})
				})
			})

			Convey("Given that the agent is used in a transfer", func() {
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
					err := ag.BeforeDelete(db)

					Convey("Then it should say that the agent is being used", func() {
						So(err, ShouldBeError, "this server is currently being "+
							"used in a running transfer and cannot be deleted, "+
							"cancel the transfer or wait for it to finish")
					})
				})
			})
		})
	})
}

func TestLocalAgentValidate(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains 1 local agent", func() {
			oldAgent := &LocalAgent{
				Owner:       "test_gateway",
				Name:        "old",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{"address":"address","port":2022}`),
			}
			So(db.Create(oldAgent), ShouldBeNil)

			Convey("Given a new local agent", func() {
				newAgent := &LocalAgent{
					Owner:       "test_gateway",
					Name:        "new",
					Root:        "root",
					InDir:       "rcv",
					OutDir:      "send",
					WorkDir:     "tmp",
					Protocol:    "sftp",
					ProtoConfig: json.RawMessage(`{"address":"address2","port":2023}`),
				}

				Convey("Given that the new agent is valid", func() {

					Convey("When calling the 'Validate' function", func() {
						err := newAgent.Validate(db)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the new agent is missing a name", func() {
					newAgent.Name = ""

					Convey("When calling the 'Validate' function", func() {
						err := newAgent.Validate(db)

						Convey("Then the error should say that the name is missing", func() {
							So(err, ShouldBeError, "the agent's name cannot "+
								"be empty")
						})
					})
				})

				Convey("Given that the new agent's name is already taken", func() {
					newAgent.Name = oldAgent.Name

					Convey("When calling the 'Validate' function", func() {
						err := newAgent.Validate(db)

						Convey("Then the error should say that the name is already taken", func() {
							So(err, ShouldBeError, "a local agent with "+
								"the same name '"+newAgent.Name+"' already exist")
						})
					})
				})

				Convey("Given that the new agent's name is already taken but the"+
					"owner is different", func() {
					So(db.Execute("UPDATE local_agents SET owner='other' WHERE id=?",
						oldAgent.ID), ShouldBeNil)
					newAgent.Name = oldAgent.Name

					Convey("When calling the 'Validate' function", func() {
						err := newAgent.Validate(db)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the new agent's protocol is not valid", func() {
					newAgent.Protocol = "not a protocol"

					Convey("When calling the 'Validate' function", func() {
						err := newAgent.Validate(db)

						Convey("Then the error should say that the protocol is invalid", func() {
							So(err, ShouldBeError, "unknown protocol")
						})
					})
				})

				Convey("Given that the new agent's protocol configuration is not valid", func() {
					newAgent.ProtoConfig = json.RawMessage("invalid")

					Convey("When calling the 'Validate' function", func() {
						err := newAgent.Validate(db)

						Convey("Then the error should say that the configuration is invalid", func() {
							So(err, ShouldBeError, "failed to parse protocol "+
								"configuration: invalid character 'i' looking "+
								"for beginning of value")
						})
					})
				})
			})
		})
	})
}

package model

import (
	"encoding/json"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRemoteAgentTableName(t *testing.T) {
	Convey("Given a `RemoteAgent` instance", t, func() {
		agent := &RemoteAgent{}

		Convey("When calling the 'TableName' method", func() {
			name := agent.TableName()

			Convey("Then it should return the name of the remote agents table", func() {
				So(name, ShouldEqual, "remote_agents")
			})
		})
	})
}

func TestRemoteAgentBeforeDelete(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a remote agent entry", func() {
			ag := &RemoteAgent{
				Name:        "partner",
				Protocol:    "dummy",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:1111",
			}
			So(db.Create(ag), ShouldBeNil)

			acc := &RemoteAccount{RemoteAgentID: ag.ID, Login: "login",
				Password: json.RawMessage("password")}
			So(db.Create(acc), ShouldBeNil)

			rule := &Rule{Name: "rule", IsSend: false, Path: "path"}
			So(db.Create(rule), ShouldBeNil)

			agAccess := &RuleAccess{RuleID: rule.ID, ObjectID: ag.ID,
				ObjectType: ag.TableName()}
			So(db.Create(agAccess), ShouldBeNil)
			accAccess := &RuleAccess{RuleID: rule.ID, ObjectID: acc.ID,
				ObjectType: acc.TableName()}
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
						accounts, err := db.Query("SELECT * FROM remote_accounts")
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
					IsServer:   false,
					AgentID:    ag.ID,
					AccountID:  acc.ID,
					SourceFile: "file.src",
					DestFile:   "file.dst",
				}
				So(db.Create(trans), ShouldBeNil)

				Convey("When calling the `BeforeDelete` hook", func() {
					err := ag.BeforeDelete(db)

					Convey("Then it should say that the agent is being used", func() {
						So(err, ShouldBeError, "this partner is currently being "+
							"used in a running transfer and cannot be deleted, "+
							"cancel the transfer or wait for it to finish")
					})
				})
			})
		})
	})
}

func TestRemoteAgentValidate(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains 1 remote agent", func() {
			oldAgent := &RemoteAgent{
				Name:        "old",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}
			So(db.Create(oldAgent), ShouldBeNil)

			Convey("Given a new remote agent", func() {
				newAgent := &RemoteAgent{
					Name:        "new",
					Protocol:    "sftp",
					ProtoConfig: json.RawMessage(`{}`),
					Address:     "localhost:2023",
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
							So(err, ShouldBeError, "a remote agent with "+
								"the same name '"+newAgent.Name+"' already exist")
						})
					})
				})

				Convey("Given that the new agent is missing an address", func() {
					newAgent.Address = ""

					Convey("When calling the 'Validate' function", func() {
						err := newAgent.Validate(db)

						Convey("Then the error should say that the address is missing", func() {
							So(err, ShouldBeError, "the partner's address cannot be empty")
						})
					})
				})

				Convey("Given that the new agent's address is invalid", func() {
					newAgent.Address = "not_an_address"

					Convey("When calling the 'Validate' function", func() {
						err := newAgent.Validate(db)

						Convey("Then the error should say that the address is invalid", func() {
							So(err, ShouldBeError, "'not_an_address' is not a valid partner address")
						})
					})
				})

				Convey("Given that the new agent's protocol is not valid", func() {
					newAgent.Protocol = "not a protocol"

					Convey("When calling the 'Validate' function", func() {
						err := newAgent.Validate(db)

						Convey("Then the error should say that the protocol is invalid", func() {
							So(err, ShouldBeError, "unknown protocol 'not a protocol'")
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

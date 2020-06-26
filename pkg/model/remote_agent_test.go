package model

import (
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
			ag := &RemoteAgent{Name: "partner", Protocol: "dummy", ProtoConfig: []byte(`{}`)}
			So(db.Create(ag), ShouldBeNil)

			acc := &RemoteAccount{RemoteAgentID: ag.ID, Login: "login", Password: []byte("password")}
			So(db.Create(acc), ShouldBeNil)

			rule := &Rule{Name: "rule", IsSend: false, Path: "path"}
			So(db.Create(rule), ShouldBeNil)

			agAccess := RuleAccess{RuleID: rule.ID, ObjectID: ag.ID, ObjectType: ag.TableName()}
			So(db.Create(agAccess), ShouldBeNil)
			accAccess := RuleAccess{RuleID: rule.ID, ObjectID: acc.ID, ObjectType: acc.TableName()}
			So(db.Create(accAccess), ShouldBeNil)

			certAg := &Cert{
				OwnerType:   ag.TableName(),
				OwnerID:     ag.ID,
				Name:        "test agent cert",
				PrivateKey:  []byte("private key"),
				PublicKey:   []byte("public key"),
				Certificate: []byte("certificate"),
			}
			So(db.Create(certAg), ShouldBeNil)

			certAcc := &Cert{
				OwnerType:   acc.TableName(),
				OwnerID:     acc.ID,
				Name:        "test account cert",
				PrivateKey:  []byte("private key"),
				PublicKey:   []byte("public key"),
				Certificate: []byte("certificate"),
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

func TestRemoteAgentBeforeInsert(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains 1 remote agent", func() {
			oldAgent := &RemoteAgent{
				Name:        "old",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
			}
			So(db.Create(oldAgent), ShouldBeNil)

			Convey("Given a new remote agent", func() {
				newAgent := &RemoteAgent{
					Name:        "new",
					Protocol:    "sftp",
					ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
				}

				Convey("Given that the new agent is valid", func() {

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newAgent.BeforeInsert(db)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the new agent has an ID", func() {
					newAgent.ID = 10

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newAgent.BeforeInsert(db)

						Convey("Then the error should say that IDs are not allowed", func() {
							So(err, ShouldBeError, "the agent's ID cannot "+
								"be entered manually")
						})
					})
				})

				Convey("Given that the new agent is missing a name", func() {
					newAgent.Name = ""

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newAgent.BeforeInsert(db)

						Convey("Then the error should say that the name is missing", func() {
							So(err, ShouldBeError, "the agent's name cannot "+
								"be empty")
						})
					})
				})

				Convey("Given that the new agent's name is already taken", func() {
					newAgent.Name = oldAgent.Name

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newAgent.BeforeInsert(db)

						Convey("Then the error should say that the name is already taken", func() {
							So(err, ShouldBeError, "a remote agent with "+
								"the same name '"+newAgent.Name+"' already exist")
						})
					})
				})

				Convey("Given that the new agent's protocol is not valid", func() {
					newAgent.Protocol = "not a protocol"

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newAgent.BeforeInsert(db)

						Convey("Then the error should say that the protocol is invalid", func() {
							So(err, ShouldBeError, "unknown protocol")
						})
					})
				})

				Convey("Given that the new agent's protocol configuration is not valid", func() {
					newAgent.ProtoConfig = []byte("invalid")

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newAgent.BeforeInsert(db)

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

func TestRemoteAgentBeforeUpdate(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains 2 remote agents", func() {
			oldAgent := &RemoteAgent{
				Name:        "old",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
			}
			So(db.Create(oldAgent), ShouldBeNil)

			otherAgent := &RemoteAgent{
				Name:        "other",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2023}`),
			}
			So(db.Create(otherAgent), ShouldBeNil)

			Convey("Given a new remote agent", func() {
				updatedAgent := &RemoteAgent{
					Name:        "updated",
					Protocol:    "sftp",
					ProtoConfig: []byte(`{"address":"localhost","port":2024}`),
				}

				Convey("Given that the updated agent is valid", func() {

					Convey("When calling the 'BeforeUpdate' function", func() {
						err := updatedAgent.BeforeUpdate(db, oldAgent.ID)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the updated agent has an ID", func() {
					updatedAgent.ID = 10

					Convey("When calling the 'BeforeUpdate' function", func() {
						err := updatedAgent.BeforeUpdate(db, oldAgent.ID)

						Convey("Then the error should say that IDs are not allowed", func() {
							So(err, ShouldBeError, "the agent's ID cannot "+
								"be entered manually")
						})
					})
				})

				Convey("Given that the updated agent's name is already taken", func() {
					updatedAgent.Name = otherAgent.Name

					Convey("When calling the 'BeforeUpdate' function", func() {
						err := updatedAgent.BeforeUpdate(db, oldAgent.ID)

						Convey("Then the error should say that the name is already taken", func() {
							So(err, ShouldBeError, "a remote agent with "+
								"the same name '"+updatedAgent.Name+"' already exist")
						})
					})
				})

				Convey("Given that the updated agent's protocol is not valid", func() {
					updatedAgent.Protocol = "not a protocol"

					Convey("When calling the 'BeforeUpdate' function", func() {
						err := updatedAgent.BeforeUpdate(db, oldAgent.ID)

						Convey("Then the error should say that the protocol is invalid", func() {
							So(err, ShouldBeError, "unknown protocol")
						})
					})
				})

				Convey("Given that the updated agent's protocol configuration is not valid", func() {
					updatedAgent.ProtoConfig = []byte("invalid")

					Convey("When calling the 'BeforeUpdate' function", func() {
						err := updatedAgent.BeforeUpdate(db, oldAgent.ID)

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

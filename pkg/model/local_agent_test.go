package model

import (
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

func TestLocalAgentBeforeInsert(t *testing.T) {
	Convey("Given a local agent entry", t, func() {
		ag := &LocalAgent{
			ID:          1,
			Name:        "test agent",
			Protocol:    "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
		}

		Convey("When calling the `BeforeInsert` hook", func() {
			err := ag.BeforeInsert(nil)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the agent's owner should be set", func() {
				So(ag.Owner, ShouldEqual, database.Owner)
			})
		})
	})
}

func TestLocalAgentBeforeDelete(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a local agent entry", func() {
			ag := &LocalAgent{
				Name:        "test agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			So(db.Create(ag), ShouldBeNil)

			Convey("Given the agent has a certificate, and an account with a certificate", func() {
				certAg := &Cert{
					OwnerType:   ag.TableName(),
					OwnerID:     ag.ID,
					Name:        "test agent cert",
					PrivateKey:  []byte("private key"),
					PublicKey:   []byte("public key"),
					Certificate: []byte("certificate"),
				}
				So(db.Create(certAg), ShouldBeNil)

				acc := &LocalAccount{
					LocalAgentID: ag.ID,
					Login:        "login",
					Password:     []byte("password"),
				}
				So(db.Create(acc), ShouldBeNil)

				certAcc := &Cert{
					OwnerType:   acc.TableName(),
					OwnerID:     acc.ID,
					Name:        "test account cert",
					PrivateKey:  []byte("private key"),
					PublicKey:   []byte("public key"),
					Certificate: []byte("certificate"),
				}
				So(db.Create(certAcc), ShouldBeNil)

				Convey("When calling the `BeforeDelete` hook", func() {
					err := ag.BeforeDelete(db)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the agent's certificate should have been deleted", func() {
						exist, err := db.Exists(certAg)
						So(err, ShouldBeNil)
						So(exist, ShouldBeFalse)
					})

					Convey("Then the agent's account should have been deleted", func() {
						exist, err := db.Exists(acc)
						So(err, ShouldBeNil)
						So(exist, ShouldBeFalse)
					})

					Convey("Then the account's certificate should have been deleted", func() {
						exist, err := db.Exists(certAcc)
						So(err, ShouldBeNil)
						So(exist, ShouldBeFalse)
					})
				})
			})
		})
	})
}

func TestLocalAgentValidateInsert(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains 1 local agent", func() {
			oldAgent := LocalAgent{
				Owner:       "test_gateway",
				Name:        "old",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			err := db.Create(&oldAgent)
			So(err, ShouldBeNil)

			Convey("Given a new local agent", func() {
				newAgent := LocalAgent{
					Owner:       "test_gateway",
					Name:        "new",
					Protocol:    "sftp",
					ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
				}

				Convey("Given that the new agent is valid", func() {

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&newAgent).ValidateInsert(ses)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the new agent has an ID", func() {
					newAgent.ID = 10

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&newAgent).ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that IDs are not allowed", func() {
							So(err.Error(), ShouldEqual, "The agent's ID cannot "+
								"be entered manually")
						})
					})
				})

				Convey("Given that the new agent is missing a name", func() {
					newAgent.Name = ""

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&newAgent).ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that the name is missing", func() {
							So(err.Error(), ShouldEqual, "The agent's name cannot "+
								"be empty")
						})
					})
				})

				Convey("Given that the new agent's name is already taken", func() {
					newAgent.Name = oldAgent.Name

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&newAgent).ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that the name is already taken", func() {
							So(err.Error(), ShouldEqual, "A local agent with "+
								"the same name '"+newAgent.Name+"' already exist")
						})
					})
				})

				Convey("Given that the new agent's name is already taken but the"+
					"owner is different", func() {
					newAgent.Owner = "new_owner"
					newAgent.Name = oldAgent.Name

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&newAgent).ValidateInsert(ses)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the new agent's protocol is not valid", func() {
					newAgent.Protocol = "not a protocol"

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&newAgent).ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that the protocol is invalid", func() {
							So(err.Error(), ShouldEqual, "Invalid agent configuration: "+
								"unknown protocol")
						})
					})
				})

				Convey("Given that the new agent's protocol configuration is not valid", func() {
					newAgent.ProtoConfig = []byte("invalid")

					Convey("When calling the 'ValidateInsert' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&newAgent).ValidateInsert(ses)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that the configuration is invalid", func() {
							So(err.Error(), ShouldEqual, "Invalid agent configuration: "+
								"invalid character 'i' looking for beginning of value")
						})
					})
				})
			})
		})
	})
}

func TestLocalAgentValidateUpdate(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains 2 local agents", func() {
			oldAgent := LocalAgent{
				Owner:       "test_gateway",
				Name:        "old",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			err := db.Create(&oldAgent)
			So(err, ShouldBeNil)

			otherAgent := LocalAgent{
				Owner:       "test_gateway",
				Name:        "other",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2023,"root":"titi"}`),
			}
			err = db.Create(&otherAgent)
			So(err, ShouldBeNil)

			Convey("Given a new local agent", func() {
				updatedAgent := LocalAgent{
					Name:        "updated",
					Protocol:    "sftp",
					ProtoConfig: []byte(`{"address":"localhost","port":2024,"root":"tata"}`),
				}

				Convey("Given that the updated agent is valid", func() {

					Convey("When calling the 'ValidateUpdate' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&updatedAgent).ValidateUpdate(ses, oldAgent.ID)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the updated agent has an ID", func() {
					updatedAgent.ID = 10

					Convey("When calling the 'ValidateUpdate' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&updatedAgent).ValidateUpdate(ses, oldAgent.ID)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that IDs are not allowed", func() {
							So(err.Error(), ShouldEqual, "The agent's ID cannot "+
								"be entered manually")
						})
					})
				})

				Convey("Given that the updated agent has an owner", func() {
					updatedAgent.Owner = "owner"

					Convey("When calling the 'ValidateUpdate' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&updatedAgent).ValidateUpdate(ses, oldAgent.ID)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that owners cannot be changed", func() {
							So(err.Error(), ShouldEqual, "The agent's owner "+
								"cannot be changed")
						})
					})
				})

				Convey("Given that the updated agent's name is already taken", func() {
					updatedAgent.Name = otherAgent.Name

					Convey("When calling the 'ValidateUpdate' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&updatedAgent).ValidateUpdate(ses, oldAgent.ID)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that the name is already taken", func() {
							So(err.Error(), ShouldEqual, "A local agent with "+
								"the same name '"+updatedAgent.Name+"' already exist")
						})
					})
				})

				Convey("Given that the updated agent's protocol is not valid", func() {
					updatedAgent.Protocol = "not a protocol"

					Convey("When calling the 'ValidateUpdate' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&updatedAgent).ValidateUpdate(ses, oldAgent.ID)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that the protocol is invalid", func() {
							So(err.Error(), ShouldEqual, "Invalid agent configuration: "+
								"unknown protocol")
						})
					})
				})

				Convey("Given that the updated agent's protocol configuration is not valid", func() {
					updatedAgent.ProtoConfig = []byte("invalid")

					Convey("When calling the 'ValidateUpdate' function", func() {
						ses, err := db.BeginTransaction()
						So(err, ShouldBeNil)

						err = (&updatedAgent).ValidateUpdate(ses, oldAgent.ID)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
						})

						Convey("Then the error should say that the configuration is invalid", func() {
							So(err.Error(), ShouldEqual, "Invalid agent configuration: "+
								"invalid character 'i' looking for beginning of value")
						})
					})
				})
			})
		})
	})
}

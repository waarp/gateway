package model

import (
	"fmt"
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

func TestRemoteAgentValidateInsert(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains 1 remote agent", func() {
			oldAgent := RemoteAgent{
				Name:        "old",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&oldAgent)
			So(err, ShouldBeNil)

			Convey("Given a new remote agent", func() {
				newAgent := RemoteAgent{
					Name:        "new",
					Protocol:    "sftp",
					ProtoConfig: []byte("{}"),
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
							So(err.Error(), ShouldEqual, "A remote agent with "+
								"the same name '"+newAgent.Name+"' already exist")
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
							So(err.Error(), ShouldEqual, "The agent's protocol "+
								"must be one of: "+fmt.Sprint(validProtocols))
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
							So(err.Error(), ShouldEqual, "The agent's configuration "+
								"is not a valid JSON configuration")
						})
					})
				})
			})
		})
	})
}

func TestRemoteAgentValidateUpdate(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains 2 remote agents", func() {
			oldAgent := RemoteAgent{
				Name:        "old",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&oldAgent)
			So(err, ShouldBeNil)

			otherAgent := RemoteAgent{
				Name:        "other",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err = db.Create(&otherAgent)
			So(err, ShouldBeNil)

			Convey("Given a new remote agent", func() {
				updatedAgent := RemoteAgent{
					Name:        "updated",
					Protocol:    "sftp",
					ProtoConfig: []byte("{\"update\":\"test\"}"),
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
							So(err.Error(), ShouldEqual, "A remote agent with "+
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
							So(err.Error(), ShouldEqual, "The agent's protocol "+
								"must be one of: "+fmt.Sprint(validProtocols))
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
							So(err.Error(), ShouldEqual, "The agent's configuration "+
								"is not a valid JSON configuration")
						})
					})
				})
			})
		})
	})
}

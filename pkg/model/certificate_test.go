package model

import (
	"fmt"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCertTableName(t *testing.T) {
	Convey("Given a `Cert` instance", t, func() {
		agent := &Cert{}

		Convey("When calling the 'TableName' method", func() {
			name := agent.TableName()

			Convey("Then it should return the name of the certificates table", func() {
				So(name, ShouldEqual, "certificates")
			})
		})
	})
}

func TestCertBeforeInsert(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains 1 local agent", func() {
			parentAgent := &LocalAgent{
				Owner:       "test_gateway",
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost", "port":21, "root":"toto"}`),
			}
			So(db.Create(parentAgent), ShouldBeNil)

			Convey("Given a new certificate", func() {
				newCert := &Cert{
					OwnerType:   "local_agents",
					OwnerID:     parentAgent.ID,
					Name:        "cert",
					PrivateKey:  []byte("private key"),
					PublicKey:   []byte("public key"),
					Certificate: []byte("content"),
				}

				Convey("Given that the new agent is valid", func() {

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newCert.BeforeInsert(db)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the new certificate has an ID", func() {
					newCert.ID = 10

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newCert.BeforeInsert(db)

						Convey("Then the error should say that IDs are not allowed", func() {
							So(err, ShouldBeError, "the certificate's ID "+
								"cannot be entered manually")
						})
					})
				})

				Convey("Given that the new certificate is missing an owner type", func() {
					newCert.OwnerType = ""

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newCert.BeforeInsert(db)

						Convey("Then the error should say that the owner type is missing", func() {
							So(err, ShouldBeError, "the certificate's owner "+
								"type is missing")
						})
					})
				})

				Convey("Given that the new certificate is missing an owner ID", func() {
					newCert.OwnerID = 0

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newCert.BeforeInsert(db)

						Convey("Then the error should say that the owner ID is missing", func() {
							So(err, ShouldBeError, "the certificate's owner "+
								"ID is missing")
						})
					})
				})

				Convey("Given that the new certificate is missing a name", func() {
					newCert.Name = ""

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newCert.BeforeInsert(db)

						Convey("Then the error should say that the name is missing", func() {
							So(err, ShouldBeError, "the certificate's name "+
								"cannot be empty")
						})
					})
				})

				Convey("Given that the new certificate is missing a private key", func() {
					newCert.PrivateKey = nil

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newCert.BeforeInsert(db)

						Convey("Then the error should say that the public key is missing", func() {
							So(err, ShouldBeError, "the certificate's private "+
								"key is missing")
						})
					})
				})

				Convey("Given that the new certificate is missing a public key", func() {
					newCert.OwnerType = "remote_agents"
					newCert.PublicKey = nil

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newCert.BeforeInsert(db)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError, "the certificate's public "+
								"key is missing")
						})
					})
				})

				Convey("Given that the new certificate has an invalid owner type", func() {
					newCert.OwnerType = "incorrect"

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newCert.BeforeInsert(db)

						Convey("Then the error should say that the owner type is invalid", func() {
							So(err, ShouldBeError, "the certificate's owner "+
								"type must be one of "+fmt.Sprint(validOwnerTypes))
						})
					})
				})

				Convey("Given that the new certificate has an invalid owner ID", func() {
					newCert.OwnerID = 1000

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newCert.BeforeInsert(db)

						Convey("Then the error should say that the owner ID is invalid", func() {
							So(err, ShouldBeError, "no local_agents found "+
								"with ID '1000'")
						})
					})
				})

				Convey("Given that the new certificate's name is already taken", func() {
					otherCert := &Cert{
						OwnerType:   "local_agents",
						OwnerID:     parentAgent.ID,
						Name:        "other",
						PrivateKey:  []byte("private key"),
						PublicKey:   []byte("public key"),
						Certificate: []byte("content"),
					}
					So(db.Create(otherCert), ShouldBeNil)

					newCert.Name = otherCert.Name

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newCert.BeforeInsert(db)

						Convey("Then the error should say that the name is taken", func() {
							So(err, ShouldBeError, "a certificate with the "+
								"same name '"+newCert.Name+"' already exist")
						})
					})
				})

				Convey("Given that the new certificate's name is already taken "+
					"but the owner is different", func() {
					otherAgent := &LocalAgent{
						Owner:       "test_gateway",
						Name:        "other",
						Protocol:    "sftp",
						ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
					}
					So(db.Create(otherAgent), ShouldBeNil)

					otherCert := &Cert{
						OwnerType:   "local_agents",
						OwnerID:     parentAgent.ID,
						Name:        "other",
						PrivateKey:  []byte("private key"),
						PublicKey:   []byte("public key"),
						Certificate: []byte("content"),
					}
					So(db.Create(otherCert), ShouldBeNil)

					newCert.Name = otherCert.Name
					newCert.OwnerID = otherAgent.ID

					Convey("When calling the 'BeforeInsert' function", func() {
						err := newCert.BeforeInsert(db)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})
			})
		})
	})
}

func TestCertBeforeUpdate(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains 1 local agent & 1 cert", func() {
			parentAgent := &LocalAgent{
				Owner:       "test_gateway",
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost", "port":21, "root":"toto"}`),
			}
			So(db.Create(parentAgent), ShouldBeNil)

			oldCert := &Cert{
				OwnerType:   "local_agents",
				OwnerID:     parentAgent.ID,
				Name:        "old",
				PrivateKey:  []byte("private key"),
				PublicKey:   []byte("public key"),
				Certificate: []byte("content"),
			}
			So(db.Create(oldCert), ShouldBeNil)

			Convey("Given an updated certificate", func() {
				updatedCert := &Cert{
					OwnerType:   "local_agents",
					OwnerID:     parentAgent.ID,
					Name:        "updated",
					PrivateKey:  []byte("new private key"),
					PublicKey:   []byte("new public key"),
					Certificate: []byte("new content"),
				}

				Convey("Given that the updated certificate is valid", func() {

					Convey("When calling the 'BeforeUpdate' function", func() {
						err := updatedCert.BeforeUpdate(db, oldCert.ID)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the updated certificate has an invalid owner type", func() {
					updatedCert.OwnerType = "invalid"

					Convey("When calling the 'BeforeUpdate' function", func() {
						err := updatedCert.BeforeUpdate(db, oldCert.ID)

						Convey("Then the error should say that the owner type is invalid", func() {
							So(err, ShouldBeError, "the certificate's owner "+
								"type must be one of "+fmt.Sprint(validOwnerTypes))
						})
					})
				})

				Convey("Given that the updated certificate has an invalid owner ID", func() {
					updatedCert.OwnerID = 1000

					Convey("When calling the 'BeforeUpdate' function", func() {
						err := updatedCert.BeforeUpdate(db, oldCert.ID)

						Convey("Then the error should say that the owner ID is invalid", func() {
							So(err, ShouldBeError, "no local_agents found "+
								"with ID '1000'")
						})
					})
				})

				Convey("Given that the updated certificate's name is already taken", func() {
					otherCert := &Cert{
						OwnerType:   "local_agents",
						OwnerID:     parentAgent.ID,
						Name:        "other",
						PrivateKey:  []byte("private key"),
						PublicKey:   []byte("public key"),
						Certificate: []byte("content"),
					}
					So(db.Create(otherCert), ShouldBeNil)

					updatedCert.Name = otherCert.Name

					Convey("When calling the 'BeforeUpdate' function", func() {
						err := updatedCert.BeforeUpdate(db, oldCert.ID)

						Convey("Then the error should say that the name is taken", func() {
							So(err, ShouldBeError, "a certificate with the "+
								"same name '"+updatedCert.Name+"' already exist")
						})
					})
				})

				Convey("Given that the updated certificate's owner changed", func() {
					otherAgent := &RemoteAgent{
						Name:        "other",
						Protocol:    "sftp",
						ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
					}
					So(db.Create(otherAgent), ShouldBeNil)

					updatedCert.OwnerType = "remote_agents"
					updatedCert.OwnerID = otherAgent.ID

					Convey("When calling the 'BeforeUpdate' function", func() {
						err := updatedCert.BeforeUpdate(db, oldCert.ID)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})
			})
		})
	})
}

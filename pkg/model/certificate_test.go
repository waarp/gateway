package model

import (
	"encoding/json"
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

func TestCertBeforeWrite(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given the database contains 1 local agent", func() {
			parentAgent := &LocalAgent{
				Owner:       "test_gateway",
				Name:        "parent",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:21",
			}
			So(db.Insert(parentAgent).Run(), ShouldBeNil)

			Convey("Given a new certificate", func() {
				newCert := &Cert{
					OwnerType:   "local_agents",
					OwnerID:     parentAgent.ID,
					Name:        "cert",
					PrivateKey:  []byte("private key"),
					PublicKey:   []byte("public key"),
					Certificate: []byte("content"),
				}

				shouldFailWith := func(errDesc string, expErr error) {
					Convey("When calling the 'BeforeWrite' function", func() {
						err := newCert.BeforeWrite(db)

						Convey("Then the error should say that "+errDesc, func() {
							So(err, ShouldBeError, expErr)
						})
					})
				}

				Convey("Given that the new agent is valid", func() {
					Convey("When calling the 'BeforeWrite' function", func() {
						err := newCert.BeforeWrite(db)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the new certificate is missing an owner type", func() {
					newCert.OwnerType = ""
					shouldFailWith("the owner type is missing", database.NewValidationError(
						"the certificate's owner type is missing"))
				})

				Convey("Given that the new certificate is missing an owner ID", func() {
					newCert.OwnerID = 0
					shouldFailWith("the owner ID is missing", database.NewValidationError(
						"the certificate's owner ID is missing"))
				})

				Convey("Given that the new certificate is missing a name", func() {
					newCert.Name = ""
					shouldFailWith("the name is missing", database.NewValidationError(
						"the certificate's name cannot be empty"))
				})

				Convey("Given that the new certificate is missing a private key", func() {
					newCert.PrivateKey = nil
					shouldFailWith("the public key is missing", database.NewValidationError(
						"the certificate's private key is missing"))
				})

				Convey("Given that the new certificate is missing a public key", func() {
					newCert.OwnerType = "remote_agents"
					newCert.PublicKey = nil
					shouldFailWith("the public key is missing", database.NewValidationError(
						"the certificate's public key is missing"))
				})

				Convey("Given that the new certificate has an invalid owner type", func() {
					newCert.OwnerType = "incorrect"
					shouldFailWith("the owner type is invalid", database.NewValidationError(
						"the certificate's owner type must be one of %s", validOwnerTypes))
				})

				Convey("Given that the new certificate has an invalid owner ID", func() {
					newCert.OwnerID = 1000
					shouldFailWith("the owner ID is invalid", database.NewValidationError(
						"no local_agents found with ID '1000'"))
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
					So(db.Insert(otherCert).Run(), ShouldBeNil)
					newCert.Name = otherCert.Name
					shouldFailWith("the name is taken", database.NewValidationError(
						"a certificate with the same name '%s' already exist",
						newCert.Name))
				})

				Convey("Given that the new certificate's name is already taken "+
					"but the owner is different", func() {
					otherAgent := &LocalAgent{
						Owner:       "test_gateway",
						Name:        "other",
						Protocol:    "sftp",
						ProtoConfig: json.RawMessage(`{}`),
						Address:     "localhost:22",
					}
					So(db.Insert(otherAgent).Run(), ShouldBeNil)

					otherCert := &Cert{
						OwnerType:   "local_agents",
						OwnerID:     parentAgent.ID,
						Name:        "other",
						PrivateKey:  []byte("private key"),
						PublicKey:   []byte("public key"),
						Certificate: []byte("content"),
					}
					So(db.Insert(otherCert).Run(), ShouldBeNil)

					newCert.Name = otherCert.Name
					newCert.OwnerID = otherAgent.ID

					Convey("When calling the 'BeforeWrite' function", func() {
						err := newCert.BeforeWrite(db)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})
			})
		})
	})
}

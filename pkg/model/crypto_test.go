package model

import (
	"encoding/json"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCryptoTableName(t *testing.T) {
	Convey("Given a `Crypto` instance", t, func() {
		agent := &Crypto{}

		Convey("When calling the 'TableName' method", func() {
			name := agent.TableName()

			Convey("Then it should return the name of the certificates table", func() {
				So(name, ShouldEqual, TableCrypto)
			})
		})
	})
}

func TestCryptoBeforeWrite(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given the database contains 1 local agent", func() {
			parentAgent := &LocalAgent{
				Owner:       "test_gateway",
				Name:        "parent",
				Protocol:    dummyProto,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:6666",
			}
			So(db.Insert(parentAgent).Run(), ShouldBeNil)

			Convey("Given new credentials", func() {
				newCert := &Crypto{
					OwnerType:   TableLocAgents,
					OwnerID:     parentAgent.ID,
					Name:        "cert",
					PrivateKey:  testhelpers.LocalhostKey,
					Certificate: testhelpers.LocalhostCert,
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

				Convey("Given that the new credentials are missing an owner type", func() {
					newCert.OwnerType = ""
					shouldFailWith("the owner type is missing", database.NewValidationError(
						"the credentials' owner type is missing"))
				})

				Convey("Given that the new credentials are missing an owner ID", func() {
					newCert.OwnerID = 0
					shouldFailWith("the owner ID is missing", database.NewValidationError(
						"the credentials' owner ID is missing"))
				})

				Convey("Given that the new credentials are missing a name", func() {
					newCert.Name = ""
					shouldFailWith("the name is missing", database.NewValidationError(
						"the credentials' name cannot be empty"))
				})

				Convey("Given that the new credentials are missing a private key", func() {
					newCert.PrivateKey = ""
					shouldFailWith("the private key is missing", database.NewValidationError(
						"the server is missing a private key"))
				})

				Convey("Given that the new credentials have an invalid owner type", func() {
					newCert.OwnerType = "incorrect"
					shouldFailWith("the owner type is invalid", database.NewValidationError(
						"the credentials' owner type must be one of %s", validOwnerTypes))
				})

				Convey("Given that the new credentials have an invalid owner ID", func() {
					newCert.OwnerID = 1000
					shouldFailWith("the owner ID is invalid", database.NewValidationError(
						"no server found with ID '1000'"))
				})

				Convey("Given that the new credentials' name is already taken", func() {
					otherCert := &Crypto{
						OwnerType:   TableLocAgents,
						OwnerID:     parentAgent.ID,
						Name:        "other",
						PrivateKey:  testhelpers.LocalhostKey,
						Certificate: testhelpers.LocalhostCert,
					}
					So(db.Insert(otherCert).Run(), ShouldBeNil)
					newCert.Name = otherCert.Name
					shouldFailWith("the name is taken", database.NewValidationError(
						"credentials with the same name '%s' already exist",
						newCert.Name))
				})

				Convey("Given that the new credentials' name is already taken "+
					"but the owner is different", func() {
					otherAgent := &LocalAgent{
						Owner:       "test_gateway",
						Name:        "other",
						Protocol:    dummyProto,
						ProtoConfig: json.RawMessage(`{}`),
						Address:     "localhost:6666",
					}
					So(db.Insert(otherAgent).Run(), ShouldBeNil)

					otherCert := &Crypto{
						OwnerType:   TableLocAgents,
						OwnerID:     parentAgent.ID,
						Name:        "other",
						PrivateKey:  testhelpers.LocalhostKey,
						Certificate: testhelpers.LocalhostCert,
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

				Convey("Given that the certificate is not valid for the host", func() {
					parentAgent.Address = "localhost:1"
					So(db.Update(parentAgent).Cols("address").Run(), ShouldBeNil)
					shouldFailWith("the certificate host is incorrect", database.NewValidationError(
						"the certificate is not valid for host 'localhost:1'"))
				})
			})
		})
	})
}

package model

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
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
				Owner:    "test_gateway",
				Name:     "parent",
				Protocol: testProtocol,
				Address:  "localhost:6666",
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
						PrivateKey:  testhelpers.OtherLocalhostKey,
						Certificate: testhelpers.OtherLocalhostCert,
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
						Owner:    "test_gateway",
						Name:     "other",
						Protocol: testProtocol,
						Address:  "localhost:6666",
					}
					So(db.Insert(otherAgent).Run(), ShouldBeNil)

					otherCert := &Crypto{
						OwnerType:   TableLocAgents,
						OwnerID:     parentAgent.ID,
						Name:        "other",
						PrivateKey:  testhelpers.OtherLocalhostKey,
						Certificate: testhelpers.OtherLocalhostCert,
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
					parentAgent.Address = "not_localhost:1"
					So(db.Update(parentAgent).Cols("address").Run(), ShouldBeNil)
					shouldFailWith("the certificate host is incorrect",
						database.NewValidationError("certificate is invalid: x509: "+
							"certificate is valid for localhost, not not_localhost"))
				})

				Convey("Given the legacy R66 certificate", func() {
					const protoR66 = "r66"

					newCert.PrivateKey = testhelpers.LegacyR66Key
					newCert.Certificate = testhelpers.LegacyR66Cert

					Convey("Given that the legacy certificate is allowed", func() {
						IsLegacyR66CertificateAllowed = true
						defer func() { IsLegacyR66CertificateAllowed = false }()

						Convey("Given a non-r66 owner", func() {
							err := newCert.BeforeWrite(db)

							Convey("Then it should return an error saying that "+
								"the certificate is invalid", func() {
								So(err, ShouldNotBeNil)
								So(err.Error(), ShouldContainSubstring, "certificate"+
									" is invalid: x509: certificate has expired or"+
									" is not yet valid:")
							})
						})

						Convey("Given an R66 owner", func() {
							config.ProtoConfigs[protoR66] = func() config.ProtoConfig {
								return new(testhelpers.TestProtoConfig)
							}
							defer delete(config.ProtoConfigs, protoR66)

							parentAgent.Protocol = protoR66
							So(db.Update(parentAgent).Cols("protocol").Run(), ShouldBeNil)

							err := newCert.BeforeWrite(db)

							Convey("Then it should NOT return an error", func() {
								So(err, ShouldBeNil)
							})
						})
					})

					Convey("Given that the legacy certificate is NOT allowed", func() {
						Convey("Given an R66 owner", func() {
							config.ProtoConfigs[protoR66] = func() config.ProtoConfig {
								return new(testhelpers.TestProtoConfig)
							}
							defer delete(config.ProtoConfigs, protoR66)

							parentAgent.Protocol = protoR66
							So(db.Update(parentAgent).Cols("protocol").Run(), ShouldBeNil)

							err := newCert.BeforeWrite(db)

							Convey("Then it should return an error saying that "+
								"the certificate is invalid", func() {
								So(err, ShouldNotBeNil)
								So(err.Error(), ShouldContainSubstring, "certificate"+
									" is invalid: x509: certificate has expired or"+
									" is not yet valid:")
							})
						})
					})
				})
			})
		})
	})
}

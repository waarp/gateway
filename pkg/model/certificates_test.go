package model

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCertificatesValidate(t *testing.T) {
	db := database.GetTestDatabase()
	account := Account{
		ID:        1,
		Username:  "account",
		PartnerID: 1,
		Password:  []byte("account"),
	}
	if err := db.Create(&account); err != nil {
		t.Fatal(err)
	}

	Convey("Given the certificate validation function", t, func() {

		Convey("Given a database with 2 certificates", func() {
			cert1 := CertChain{
				ID:         10,
				Name:       "cert1",
				OwnerType:  "ACCOUNT",
				OwnerID:    1,
				PrivateKey: []byte("cert1"),
				PublicKey:  []byte("cert1"),
				Cert:       []byte("cert1"),
			}
			cert2 := CertChain{
				ID:         20,
				Name:       "cert2",
				OwnerType:  "ACCOUNT",
				OwnerID:    1,
				PrivateKey: []byte("cert2"),
				PublicKey:  []byte("cert2"),
				Cert:       []byte("cert2"),
			}
			err := db.Create(&cert1)
			So(err, ShouldBeNil)
			err = db.Create(&cert2)
			So(err, ShouldBeNil)

			Reset(func() {
				err := db.Execute("DELETE FROM 'certificates'")
				So(err, ShouldBeNil)
			})

			Convey("When inserting a 3rd certificate", func() {
				isInsert := true
				cert3 := CertChain{
					ID:         30,
					Name:       "cert3",
					OwnerType:  "ACCOUNT",
					OwnerID:    1,
					PrivateKey: []byte("cert3"),
					PublicKey:  []byte("cert3"),
					Cert:       []byte("cert3"),
				}

				Convey("Given correct values", func() {

					Convey("When calling 'Validate'", func() {
						err := cert3.Validate(db, isInsert)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given an empty name", func() {
					cert3.Name = ""

					Convey("When calling 'Validate'", func() {
						err := cert3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The certificate's name cannot be empty")
						})
					})
				})

				Convey("Given an empty private key", func() {
					cert3.PrivateKey = nil

					Convey("When calling 'Validate'", func() {
						err := cert3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The certificate's private key cannot be empty")
						})
					})
				})

				Convey("Given an empty public key", func() {
					cert3.PublicKey = nil

					Convey("When calling 'Validate'", func() {
						err := cert3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The certificate's public key cannot be empty")
						})
					})
				})

				Convey("Given an empty certificate", func() {
					cert3.Cert = nil

					Convey("When calling 'Validate'", func() {
						err := cert3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "The certificate cannot be empty")
						})
					})
				})

				Convey("Given an already existing ID", func() {
					cert3.ID = cert1.ID

					Convey("When calling 'Validate'", func() {
						err := cert3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"A certificate with the same ID already exist")
						})
					})
				})

				Convey("Given an already existing name", func() {
					cert3.Name = cert2.Name

					Convey("When calling 'Validate'", func() {
						err := cert3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "A certificate "+
								"with the same name already exist for this account")
						})
					})
				})

				Convey("Given a non-existing account id", func() {
					cert3.OwnerID = 2

					Convey("When calling 'Validate'", func() {
						err := cert3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"No account found with ID '2'")
						})
					})
				})
			})

			Convey("When updating one of the certificates", func() {
				isInsert := false
				cert2b := CertChain{
					ID:         cert2.ID,
					Name:       "cert2b",
					OwnerType:  "ACCOUNT",
					OwnerID:    1,
					PrivateKey: []byte("cert2b"),
					PublicKey:  []byte("cert2b"),
					Cert:       []byte("cert2b"),
				}

				Convey("Given correct values", func() {

					Convey("When calling 'Validate'", func() {
						err := cert2b.Validate(db, isInsert)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given an empty name", func() {
					cert2b.Name = ""

					Convey("When calling 'Validate'", func() {
						err := cert2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The certificate's name cannot be empty")
						})
					})
				})

				Convey("Given a non-existing ID", func() {
					cert2b.ID = 25

					Convey("When calling 'Validate'", func() {
						err := cert2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "Unknown certificate ID: '25'")
						})
					})
				})

				Convey("Given an already existing name", func() {
					cert2b.Name = cert1.Name

					Convey("When calling 'Validate'", func() {
						err := cert2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "A certificate "+
								"with the same name already exist for this account")
						})
					})
				})

				Convey("Given a non-existing account id", func() {
					cert2b.OwnerID = 2

					Convey("When calling 'Validate'", func() {
						err := cert2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"No account found with ID '2'")
						})
					})
				})
			})
		})

	})
}

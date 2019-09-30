package model

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	. "github.com/smartystreets/goconvey/convey"
)

func TestAccountValidate(t *testing.T) {
	db := database.GetTestDatabase()
	partner := Partner{
		ID:      1,
		Name:    "partner",
		Address: "address",
		Port:    1,
	}
	if err := db.Create(&partner); err != nil {
		t.Fatal(err)
	}

	Convey("Given the account validation function", t, func() {

		Convey("Given a database with 2 accounts", func() {
			account1 := Account{
				ID:        10,
				Username:  "account1",
				PartnerID: 1,
				Password:  []byte("account1"),
			}
			account2 := Account{
				ID:        20,
				Username:  "account2",
				PartnerID: 1,
				Password:  []byte("account2"),
			}
			err := db.Create(&account1)
			So(err, ShouldBeNil)
			err = db.Create(&account2)
			So(err, ShouldBeNil)

			Reset(func() {
				err := db.Execute("DELETE FROM 'accounts'")
				So(err, ShouldBeNil)
			})

			Convey("When inserting a 3rd account", func() {
				isInsert := true
				account3 := Account{
					ID:        30,
					Username:  "account3",
					PartnerID: 1,
					Password:  []byte("account3"),
				}

				Convey("Given correct values", func() {

					Convey("When calling 'Validate'", func() {
						err := account3.Validate(db, isInsert)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given an empty username", func() {
					account3.Username = ""

					Convey("When calling 'Validate'", func() {
						err := account3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The account's username cannot be empty")
						})
					})
				})

				Convey("Given an empty password", func() {
					account3.Password = []byte{}

					Convey("When calling 'Validate'", func() {
						err := account3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The account's password cannot be empty")
						})
					})
				})

				Convey("Given a nil password", func() {
					account3.Password = nil

					Convey("When calling 'Validate'", func() {
						err := account3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The account's password cannot be empty")
						})
					})
				})

				Convey("Given an already existing ID", func() {
					account3.ID = account1.ID

					Convey("When calling 'Validate'", func() {
						err := account3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"An account with the same ID already exist")
						})
					})
				})

				Convey("Given an already existing username", func() {
					account3.Username = account2.Username

					Convey("When calling 'Validate'", func() {
						err := account3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "An account "+
								"with the same username already exist for this partner")
						})
					})
				})

				Convey("Given a non-existing partner id", func() {
					account3.PartnerID = 2

					Convey("When calling 'Validate'", func() {
						err := account3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"No partner found with id '2'")
						})
					})
				})
			})

			Convey("When updating one of the account", func() {
				isInsert := false
				account2b := Account{
					ID:        account2.ID,
					Username:  "account2b",
					PartnerID: 1,
					Password:  []byte("account2b"),
				}

				Convey("Given correct values", func() {

					Convey("When calling 'Validate'", func() {
						err := account2b.Validate(db, isInsert)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given an empty username", func() {
					account2b.Username = ""

					Convey("When calling 'Validate'", func() {
						err := account2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The account's username cannot be empty")
						})
					})
				})

				Convey("Given an empty password", func() {
					account2b.Password = []byte{}

					Convey("When calling 'Validate'", func() {
						err := account2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The account's password cannot be empty")
						})
					})
				})

				Convey("Given a nil password", func() {
					account2b.Password = nil

					Convey("When calling 'Validate'", func() {
						err := account2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The account's password cannot be empty")
						})
					})
				})

				Convey("Given a non-existing ID", func() {
					account2b.ID = 25

					Convey("When calling 'Validate'", func() {
						err := account2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "Unknown account id: '25'")
						})
					})
				})

				Convey("Given an already existing username", func() {
					account2b.Username = account1.Username

					Convey("When calling 'Validate'", func() {
						err := account2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "An account "+
								"with the same username already exist for this partner")
						})
					})
				})

				Convey("Given a non-existing partner id", func() {
					account2b.PartnerID = 2

					Convey("When calling 'Validate'", func() {
						err := account2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"No partner found with id '2'")
						})
					})
				})
			})
		})

	})
}

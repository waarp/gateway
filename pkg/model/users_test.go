package model

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	. "github.com/smartystreets/goconvey/convey"
)

func TestUserValidate(t *testing.T) {
	db := database.GetTestDatabase()

	Convey("Given the user validation function", t, func() {

		Convey("Given a database with 2 users", func() {
			user1 := User{
				ID:       10,
				Login:    "user1",
				Name:     "User 1",
				Password: []byte("user1"),
			}
			user2 := User{
				ID:       20,
				Login:    "user2",
				Name:     "User 2",
				Password: []byte("user2"),
			}
			err := db.Create(&user1)
			So(err, ShouldBeNil)
			err = db.Create(&user2)
			So(err, ShouldBeNil)

			Reset(func() {
				err := db.Execute("DELETE FROM 'users'")
				So(err, ShouldBeNil)
			})

			Convey("When inserting a 3rd user", func() {
				isInsert := true
				user3 := User{
					ID:       30,
					Login:    "user3",
					Name:     "User 3",
					Password: []byte("user3"),
				}

				Convey("Given correct values", func() {

					Convey("When calling 'Validate'", func() {
						err := user3.Validate(db, isInsert)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given an empty login", func() {
					user3.Login = ""

					Convey("When calling 'Validate'", func() {
						err := user3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The user's login cannot be empty")
						})
					})
				})

				Convey("Given an empty password", func() {
					user3.Password = []byte{}

					Convey("When calling 'Validate'", func() {
						err := user3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The user's password cannot be empty")
						})
					})
				})

				Convey("Given a nil password", func() {
					user3.Password = nil

					Convey("When calling 'Validate'", func() {
						err := user3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The user's password cannot be empty")
						})
					})
				})

				Convey("Given an already existing ID", func() {
					user3.ID = user1.ID

					Convey("When calling 'Validate'", func() {
						err := user3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"A user with the same ID or login already exist")
						})
					})
				})

				Convey("Given an already existing login", func() {
					user3.Login = user2.Login

					Convey("When calling 'Validate'", func() {
						err := user3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"A user with the same ID or login already exist")
						})
					})
				})
			})

			Convey("When updating one of the user", func() {
				isInsert := false
				user2b := User{
					ID:       user2.ID,
					Login:    "user2b",
					Name:     "User 2b",
					Password: []byte("user2b"),
				}

				Convey("Given correct values", func() {

					Convey("When calling 'Validate'", func() {
						err := user2b.Validate(db, isInsert)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given an empty login", func() {
					user2b.Login = ""

					Convey("When calling 'Validate'", func() {
						err := user2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The user's login cannot be empty")
						})
					})
				})

				Convey("Given an empty password", func() {
					user2b.Password = []byte{}

					Convey("When calling 'Validate'", func() {
						err := user2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The user's password cannot be empty")
						})
					})
				})

				Convey("Given a nil password", func() {
					user2b.Password = nil

					Convey("When calling 'Validate'", func() {
						err := user2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The user's password cannot be empty")
						})
					})
				})

				Convey("Given a non-existing ID", func() {
					user2b.ID = 25

					Convey("When calling 'Validate'", func() {
						err := user2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "Unknown user ID: '25'")
						})
					})
				})

				Convey("Given an already existing login", func() {
					user2b.Login = user1.Login

					Convey("When calling 'Validate'", func() {
						err := user2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"A user with the same login already exist")
						})
					})
				})
			})
		})

	})
}

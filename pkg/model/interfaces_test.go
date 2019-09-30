package model

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInterfaceValidate(t *testing.T) {
	db := database.GetTestDatabase()

	Convey("Given the interface validation function", t, func() {

		Convey("Given a database with 2 interfaces", func() {
			interface1 := Interface{
				ID:   1,
				Name: "interface1",
				Type: "sftp",
				Port: 1,
			}
			interface2 := Interface{
				ID:   2,
				Name: "interface2",
				Type: "r66",
				Port: 2,
			}
			err := db.Create(&interface1)
			So(err, ShouldBeNil)
			err = db.Create(&interface2)
			So(err, ShouldBeNil)

			Reset(func() {
				err := db.Execute("DELETE FROM 'interfaces'")
				So(err, ShouldBeNil)
			})

			Convey("When inserting a 3rd account", func() {
				isInsert := true
				interface3 := Interface{
					ID:   3,
					Name: "interface3",
					Type: "http",
					Port: 3,
				}

				Convey("Given correct values", func() {

					Convey("When calling 'Validate'", func() {
						err := interface3.Validate(db, isInsert)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given an empty name", func() {
					interface3.Name = ""

					Convey("When calling 'Validate'", func() {
						err := interface3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The interface's name cannot be empty")
						})
					})
				})

				Convey("Given an invalid type", func() {
					interface3.Type = "not_a_type"

					Convey("When calling 'Validate'", func() {
						err := interface3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "The interface's "+
								"type must be one of [http r66 sftp]")
						})
					})
				})

				Convey("Given an already existing ID", func() {
					interface3.ID = interface1.ID

					Convey("When calling 'Validate'", func() {
						err := interface3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "An interface "+
								"with the same ID or name already exist")
						})
					})
				})

				Convey("Given an already existing name", func() {
					interface3.Name = interface2.Name

					Convey("When calling 'Validate'", func() {
						err := interface3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "An interface "+
								"with the same ID or name already exist")
						})
					})
				})
			})

			Convey("When updating one of the account", func() {
				isInsert := false
				interface2b := Interface{
					ID:   interface2.ID,
					Name: "interface2b",
					Type: "http",
					Port: 20,
				}

				Convey("Given correct values", func() {

					Convey("When calling 'Validate'", func() {
						err := interface2b.Validate(db, isInsert)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given an empty name", func() {
					interface2b.Name = ""

					Convey("When calling 'Validate'", func() {
						err := interface2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The interface's name cannot be empty")
						})
					})
				})

				Convey("Given an invalid type", func() {
					interface2b.Type = "not_a_type"

					Convey("When calling 'Validate'", func() {
						err := interface2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "The interface's "+
								"type must be one of [http r66 sftp]")
						})
					})
				})

				Convey("Given a non-existing ID", func() {
					interface2b.ID = 20

					Convey("When calling 'Validate'", func() {
						err := interface2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "Unknown interface ID: '20'")
						})
					})
				})

				Convey("Given an already existing username", func() {
					interface2b.Name = interface1.Name

					Convey("When calling 'Validate'", func() {
						err := interface2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "An interface "+
								"with the same name already exist")
						})
					})
				})
			})
		})
	})
}

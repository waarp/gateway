package model

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	. "github.com/smartystreets/goconvey/convey"
)

func TestPartnerValidate(t *testing.T) {
	db := database.GetTestDatabase()
	interf := Interface{
		ID:   1,
		Name: "interface",
		Type: "sftp",
		Port: 1,
	}
	if err := db.Create(&interf); err != nil {
		t.Fatal(err)
	}

	Convey("Given the partner validation function", t, func() {

		Convey("Given a database with 2 partners", func() {
			partner1 := Partner{
				ID:          10,
				Name:        "partner1",
				Address:     "address_1",
				Port:        1,
				InterfaceID: 1,
			}
			partner2 := Partner{
				ID:          20,
				Name:        "partner2",
				Address:     "address_2",
				Port:        2,
				InterfaceID: 1,
			}
			err := db.Create(&partner1)
			So(err, ShouldBeNil)
			err = db.Create(&partner2)
			So(err, ShouldBeNil)

			Reset(func() {
				err := db.Execute("DELETE FROM 'partners'")
				So(err, ShouldBeNil)
			})

			Convey("When inserting a 3rd partner", func() {
				isInsert := true
				partner3 := Partner{
					ID:          30,
					Name:        "partner3",
					Address:     "address_3",
					Port:        3,
					InterfaceID: 1,
				}

				Convey("Given correct values", func() {

					Convey("When calling 'Validate'", func() {
						err := partner3.Validate(db, isInsert)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given an empty name", func() {
					partner3.Name = ""

					Convey("When calling 'Validate'", func() {
						err := partner3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The partner's name cannot be empty")
						})
					})
				})

				Convey("Given an empty address", func() {
					partner3.Address = ""

					Convey("When calling 'Validate'", func() {
						err := partner3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The partner's address cannot be empty")
						})
					})
				})

				Convey("Given an already existing ID", func() {
					partner3.ID = partner1.ID

					Convey("When calling 'Validate'", func() {
						err := partner3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"A partner with the same ID already exist")
						})
					})
				})

				Convey("Given an already existing name", func() {
					partner3.Name = partner2.Name

					Convey("When calling 'Validate'", func() {
						err := partner3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "A partner with "+
								"the same name already exist for this interface")
						})
					})
				})

				Convey("Given a non-existing interface id", func() {
					partner3.InterfaceID = 2

					Convey("When calling 'Validate'", func() {
						err := partner3.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"No interface found with id '2'")
						})
					})
				})
			})

			Convey("When updating one of the partners", func() {
				isInsert := false
				partner2b := Partner{
					ID:          partner2.ID,
					Name:        "partner2b",
					Address:     "address_2b",
					Port:        3,
					InterfaceID: 1,
				}

				Convey("Given correct values", func() {

					Convey("When calling 'Validate'", func() {
						err := partner2b.Validate(db, isInsert)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given an empty name", func() {
					partner2b.Name = ""

					Convey("When calling 'Validate'", func() {
						err := partner2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"The partner's name cannot be empty")
						})
					})
				})

				Convey("Given a non-existing ID", func() {
					partner2b.ID = 25

					Convey("When calling 'Validate'", func() {
						err := partner2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "Unknown partner ID: '25'")
						})
					})
				})

				Convey("Given an already existing name", func() {
					partner2b.Name = partner1.Name

					Convey("When calling 'Validate'", func() {
						err := partner2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual, "A partner with "+
								"the same name already exist for this interface")
						})
					})
				})

				Convey("Given a non-existing interface id", func() {
					partner2b.InterfaceID = 2

					Convey("When calling 'Validate'", func() {
						err := partner2b.Validate(db, isInsert)

						Convey("Then it should return an error", func() {
							So(err, ShouldBeError)
							So(err.Error(), ShouldEqual,
								"No interface found with id '2'")
						})
					})
				})
			})
		})

	})
}

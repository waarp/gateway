package model

import (
	"fmt"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRuleTableName(t *testing.T) {
	Convey("Given a `rule` instance", t, func() {
		rule := &Rule{}

		Convey("When calling the 'TableName' method", func() {
			name := rule.TableName()

			Convey("Then it should return the name of the rule table", func() {
				So(name, ShouldEqual, "rules")
			})
		})

	})
}

func TestRuleBeforeInsert(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a rule entry", func() {
			r := &Rule{
				Name:   "Test",
				IsSend: true,
				Path:   "/path",
			}
			So(db.Create(r), ShouldBeNil)

			Convey("Given a rule with a different a name", func() {
				r2 := &Rule{
					Name:   "Test2",
					IsSend: true,
					Path:   "/path2",
				}

				Convey("When calling `BeforeUpdate`", func() {
					err := r2.BeforeInsert(db)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given a rule with the same name but with different send", func() {
				r2 := &Rule{
					Name:   r.Name,
					IsSend: !r.IsSend,
					Path:   "/path2",
				}

				Convey("When calling `BeforeUpdate`", func() {
					err := r2.BeforeInsert(db)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given a rule with the same name and same send", func() {
				r2 := &Rule{
					Name:   r.Name,
					IsSend: r.IsSend,
					Path:   "/path2",
				}

				Convey("When calling `BeforeUpdate`", func() {
					err := r2.BeforeInsert(db)

					Convey("Then the error should say that rule already exist", func() {
						So(err, ShouldBeError, fmt.Sprintf("a rule named '%s' "+
							"with send = %t already exist", r.Name, r.IsSend))
					})
				})
			})

			Convey("Given a rule without a path", func() {
				r2 := &Rule{
					Name:   "Test2",
					IsSend: false,
				}

				Convey("When calling `BeforeUpdate`", func() {
					err := r2.BeforeInsert(db)

					Convey("Then it should return an error saying that the path cannot be empty", func() {
						So(err, ShouldBeError, "the rule's path cannot be empty")
					})
				})
			})
		})
	})
}

func TestRuleBeforeUpdate(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given two rule entry", func() {
			r := &Rule{
				Name:   "rule1",
				IsSend: true,
				Path:   "/path",
			}
			So(db.Create(r), ShouldBeNil)

			r2 := &Rule{
				Name:   "rule2",
				IsSend: true,
				Path:   "/path2",
			}
			So(db.Create(r2), ShouldBeNil)

			Convey("When updating with invalid data", func() {
				update := &Rule{Name: r2.Name, IsSend: r2.IsSend}

				Convey("When calling the `BeforeUpdate` function", func() {
					err := update.BeforeUpdate(db, r.ID)

					Convey("Then the error should say that the name is already used", func() {
						So(err, ShouldBeError, fmt.Sprintf("a rule named '%s' "+
							"with send = %t already exist", update.Name, r.IsSend))
					})

				})
			})

			Convey("When updating with valid data", func() {
				update := &Rule{Name: "toto", IsSend: true}

				Convey("When calling the `BeforeUpdate` function", func() {
					err := update.BeforeUpdate(db, r.ID)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})
				})
			})
		})
	})

}

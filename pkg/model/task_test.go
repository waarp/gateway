package model

import (
	"fmt"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTaskTableName(t *testing.T) {
	Convey("Given a Task instance", t, func() {
		task := &Task{}

		Convey("When calling the `TableName` method", func() {
			name := task.TableName()

			Convey("Then it should return the name of the task table", func() {
				So(name, ShouldEqual, "tasks")
			})
		})
	})
}

func TestTaskValidateInsert(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a rule entry", func() {
			r := &Rule{
				Name: "Test",
				Send: true,
			}
			So(db.Create(r), ShouldBeNil)

			t := &Task{
				RuleID: r.ID,
				Chain:  "PRE",
				Rank:   0,
				Type:   "DUMMY",
				Args:   "{}",
			}
			So(db.Create(t), ShouldBeNil)

			Convey("Given a task with an invalid RuleID", func() {
				t2 := &Task{
					RuleID: 0,
					Chain:  "PRE",
					Rank:   0,
					Type:   "DUMMY",
					Args:   "{}",
				}

				Convey("When calling the `ValidateInsert` method", func() {
					err := t2.ValidateInsert(db)

					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)
					})

					Convey("Then the error should say 'No rule found'", func() {
						So(err.Error(), ShouldEqual,
							fmt.Sprintf("No rule found with ID %d", t2.RuleID))
					})
				})
			})

			Convey("Given a task with an invalid Chain", func() {
				t2 := &Task{
					RuleID: r.ID,
					Chain:  "XXX",
					Rank:   0,
					Type:   "DUMMY",
					Args:   "{}",
				}

				Convey("When calling the `ValidateInsert` method", func() {
					err := t2.ValidateInsert(db)

					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)
					})

					Convey("Then the error should say 'No rule found'", func() {
						So(err.Error(), ShouldEqual,
							fmt.Sprintf("%s is not a valid task chain", t2.Chain))
					})
				})
			})

			Convey("Given a task which would overwrite another task", func() {
				t2 := &Task{
					RuleID: t.RuleID,
					Chain:  t.Chain,
					Rank:   t.Rank,
					Type:   "DUMMY",
					Args:   "{}",
				}

				Convey("When calling the `ValidateInsert` method", func() {
					err := t2.ValidateInsert(db)

					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)
					})

					Convey("Then the error should say 'No rule found'", func() {
						So(err.Error(), ShouldEqual,
							fmt.Sprintf("Rule %d already has a task in %s at %d",
								t2.RuleID, t2.Chain, t2.Rank))
					})
				})
			})
		})
	})
}

func TestTaskValidateUpdate(t *testing.T) {
	Convey("Given a Task instance", t, func() {
		task := &Task{}

		Convey("When calling the `ValidateUpdate` method", func() {
			err := task.ValidateUpdate(nil)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})

			Convey("Then the error should say that operation is unallowed", func() {
				So(err.Error(), ShouldEqual, "Unallowed operation")
			})
		})
	})
}

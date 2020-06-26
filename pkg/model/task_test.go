package model

import (
	"fmt"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	ValidTasks["TESTSUCCESS"] = &testTaskSuccess{}
	ValidTasks["TESTFAIL"] = &testTaskFail{}
}

type testTaskSuccess struct{}

func (t *testTaskSuccess) Validate(map[string]string) error {
	return nil
}

type testTaskFail struct{}

func (t *testTaskFail) Validate(map[string]string) error {
	return nil
}

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

func TestTaskBeforeInsert(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a rule entry", func() {
			r := &Rule{
				Name:   "rule1",
				IsSend: true,
				Path:   "path",
			}
			So(db.Create(r), ShouldBeNil)

			t := &Task{
				RuleID: r.ID,
				Chain:  ChainPre,
				Rank:   0,
				Type:   "TESTSUCCESS",
				Args:   []byte("{}"),
			}
			So(db.Create(t), ShouldBeNil)

			Convey("Given a task with an invalid RuleID", func() {
				t2 := &Task{
					RuleID: 0,
					Chain:  ChainPre,
					Rank:   0,
					Type:   "TESTSUCCESS",
					Args:   []byte("{}"),
				}

				Convey("When calling the `BeforeInsert` method", func() {
					err := t2.BeforeInsert(db)

					Convey("Then the error should say the rule was not found", func() {
						So(err, ShouldBeError, "no rule found with ID "+
							fmt.Sprint(t2.RuleID))
					})
				})
			})

			Convey("Given a task with an invalid Chain", func() {
				t2 := &Task{
					RuleID: r.ID,
					Chain:  "XXX",
					Rank:   0,
					Type:   "TESTSUCCESS",
					Args:   []byte("{}"),
				}

				Convey("When calling the `BeforeInsert` method", func() {
					err := t2.BeforeInsert(db)

					Convey("Then the error should say that the chain is invalid", func() {
						So(err, ShouldBeError, fmt.Sprintf(
							"%s is not a valid task chain", t2.Chain))
					})
				})
			})

			Convey("Given a task which would overwrite another task", func() {
				t2 := &Task{
					RuleID: t.RuleID,
					Chain:  t.Chain,
					Rank:   t.Rank,
					Type:   "TESTSUCCESS",
					Args:   []byte("{}"),
				}

				Convey("When calling the `BeforeInsert` method", func() {
					err := t2.BeforeInsert(db)

					Convey("Then the error should say that the task already exist", func() {
						So(err, ShouldBeError, fmt.Sprintf("rule %d already has a task in %s at %d",
							t2.RuleID, t2.Chain, t2.Rank))
					})
				})
			})
		})
	})
}

func TestTaskBeforeUpdate(t *testing.T) {
	Convey("Given a Task instance", t, func() {
		task := &Task{}

		Convey("When calling the `BeforeUpdate` method", func() {
			err := task.BeforeUpdate(nil, 0)

			Convey("Then the error should say that operation is not allowed", func() {
				So(err, ShouldBeError, "operation not allowed")
			})
		})
	})
}

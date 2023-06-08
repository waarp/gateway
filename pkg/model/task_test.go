package model

import (
	"context"
	"fmt"
	"testing"

	"code.waarp.fr/lib/log"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

var (
	errExec  = fmt.Errorf("execution failed")
	errValid = fmt.Errorf("validation failed")
)

//nolint:gochecknoinits // init is used by design
func init() {
	ValidTasks["TESTSUCCESS"] = &testTaskSuccess{}
	ValidTasks["TESTFAIL"] = &testTaskFail{}
}

type testTaskSuccess struct{}

func (t *testTaskSuccess) Validate(map[string]string) error {
	return nil
}

func (t *testTaskSuccess) Run(context.Context, map[string]string, *database.DB,
	*log.Logger, *TransferContext,
) error {
	return nil
}

type testTaskFail struct{}

func (t *testTaskFail) Validate(map[string]string) error {
	return errValid
}

func (t *testTaskFail) Run(context.Context, map[string]string, *database.DB,
	*log.Logger, *TransferContext,
) error {
	return errExec
}

func TestTaskTableName(t *testing.T) {
	Convey("Given a Task instance", t, func() {
		task := &Task{}

		Convey("When calling the `TableName` method", func() {
			name := task.TableName()

			Convey("Then it should return the name of the task table", func() {
				So(name, ShouldEqual, TableTasks)
			})
		})
	})
}

func TestTaskBeforeInsert(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given a rule entry", func() {
			r := Rule{
				Name:   "rule1",
				IsSend: true,
				Path:   "path",
			}
			So(db.Insert(&r).Run(), ShouldBeNil)

			t := Task{
				RuleID: r.ID,
				Chain:  ChainPre,
				Rank:   0,
				Type:   "TESTSUCCESS",
				Args:   []byte("{}"),
			}
			So(db.Insert(&t).Run(), ShouldBeNil)

			Convey("Given a task with an invalid RuleID", func() {
				t2 := Task{
					RuleID: 0,
					Chain:  ChainPre,
					Rank:   0,
					Type:   "TESTSUCCESS",
					Args:   []byte("{}"),
				}

				Convey("When calling the `BeforeWrite` method", func() {
					err := t2.BeforeWrite(db)

					Convey("Then the error should say the rule was not found", func() {
						So(err, ShouldBeError, database.NewValidationError(
							"no rule found with ID %d", t2.RuleID))
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

				Convey("When calling the `BeforeWrite` method", func() {
					err := t2.BeforeWrite(db)

					Convey("Then the error should say that the chain is invalid", func() {
						So(err, ShouldBeError, database.NewValidationError(
							"%s is not a valid task chain", t2.Chain))
					})
				})
			})

			Convey("Given a task which would overwrite another task", func() {
				t2 := Task{
					RuleID: t.RuleID,
					Chain:  t.Chain,
					Rank:   t.Rank,
					Type:   "TESTSUCCESS",
					Args:   []byte("{}"),
				}

				Convey("When calling the `BeforeWrite` method", func() {
					err := t2.BeforeWrite(db)

					Convey("Then the error should say that the task already exist", func() {
						So(err, ShouldBeError, database.NewValidationError(
							"rule %d already has a task in %s at %d",
							t2.RuleID, t2.Chain, t2.Rank))
					})
				})
			})
		})
	})
}

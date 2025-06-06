package database

import (
	"github.com/smartystreets/goconvey/convey"
)

type testValid struct {
	ID     int64  `xorm:"pk autoincr 'id'"`
	String string `xorm:"notnull 'string'"`
	Hooks  string `xorm:"-"`
}

func (*testValid) TableName() string   { return "test_valid" }
func (*testValid) Appellation() string { return "test struct" }
func (t *testValid) GetID() int64      { return t.ID }

func (t *testValid) BeforeWrite(Access) error {
	t.Hooks = "write hook"

	return nil
}

func (t *testValid) BeforeDelete(Access) error {
	t.Hooks = "delete hook"

	return nil
}

type validList []*testValid

func (*validList) TableName() string { return "test_valid" }
func (*validList) Elem() string      { return "test struct" }

type testValid2 struct {
	ID     int64  `xorm:"pk 'id'"`
	String string `xorm:"notnull 'string'"`
	Hooks  string `xorm:"-"`
}

func (*testValid2) TableName() string   { return "test_valid_2" }
func (*testValid2) Appellation() string { return "test valid 2" }
func (t *testValid2) GetID() int64      { return t.ID }

func (t *testValid2) BeforeWrite(Access) error {
	t.Hooks = "write hook"

	return nil
}

func (t *testValid2) BeforeDelete(Access) error {
	t.Hooks = "delete hook"

	return nil
}

type testWriteFail struct {
	ID    int64  `xorm:"pk 'id'"`
	Hooks string `xorm:"-"`
}

func (*testWriteFail) TableName() string   { return "test_write_fail" }
func (*testWriteFail) Appellation() string { return "test write fail" }
func (t *testWriteFail) GetID() int64      { return t.ID }

func (t *testWriteFail) BeforeWrite(Access) error {
	t.Hooks = "write hook"

	return NewValidationError("write hook failed")
}

type testDeleteFail struct {
	ID    int64  `xorm:"pk 'id'"`
	Hooks string `xorm:"-"`
}

func (*testDeleteFail) TableName() string   { return "test_delete_fail" }
func (*testDeleteFail) Appellation() string { return "test delete fail" }
func (t *testDeleteFail) GetID() int64      { return t.ID }

func (t *testDeleteFail) BeforeWrite(Access) error {
	t.Hooks = "write hook"

	return nil
}

func (t *testDeleteFail) BeforeDelete(db Access) error {
	t.Hooks = "delete hook"

	convey.So(db.Insert(&testDeleteFail{ID: 1000}).Run(), convey.ShouldBeNil)

	return NewValidationError("delete hook failed")
}

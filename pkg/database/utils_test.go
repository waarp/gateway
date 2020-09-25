package database

import (
	"database/sql"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
)

const tblName = "bean_test"

func init() {
	logConf := conf.LogConfig{
		Level: "DEBUG",
		LogTo: "stdout",
	}
	_ = log.InitBackend(logConf)
}

type queryFunc func(query) (Iterator, error)
type execFunc func(command) (sql.Result, error)

// Deprecated:
type testBean struct {
	ID     uint64 `xorm:"pk 'id'"`
	String string `xorm:"notnull 'string'"`

	signals string `xorm:"-"`
}

func (*testBean) TableName() string   { return tblName }
func (*testBean) Table() string       { return tblName }
func (*testBean) Appellation() string { return tblName }
func (*testBean) ElemName() string    { return tblName }

func (t *testBean) GetID() uint64 {
	return t.ID
}

func (t *testBean) Validate(Accessor) error {
	t.signals = "validation hook"
	return nil
}

func (t *testBean) BeforeWrite(*Session) error {
	t.signals = "write hook"
	return nil
}

func (t *testBean) BeforeDelete(Accessor) error {
	t.signals = "delete hook"
	return nil
}

// Deprecated:
type invalidBean struct{}

func (*invalidBean) Table() string              { return "invalid" }
func (*invalidBean) GetID() uint64              { return 0 }
func (*invalidBean) TableName() string          { return "invalid" }
func (*invalidBean) ElemName() string           { return "invalid" }
func (*invalidBean) Appellation() string        { return "invalid" }
func (*invalidBean) BeforeWrite(*Session) error { return nil }

type testValid struct {
	ID     uint64 `xorm:"pk 'id'"`
	String string `xorm:"notnull 'string'"`
	Hooks  string `xorm:"-"`
}

func (*testValid) Table() string       { return "test_valid" }
func (*testValid) TableName() string   { return "test_valid" }
func (*testValid) Appellation() string { return "test struct" }
func (t *testValid) GetID() uint64     { return t.ID }

func (t *testValid) BeforeWrite(*Session) error {
	t.Hooks = "write hook"
	return nil
}

func (t *testValid) BeforeDelete(*Session) error {
	t.Hooks = "delete hook"
	return nil
}

type testValid2 struct {
	ID     uint64 `xorm:"pk 'id'"`
	String string `xorm:"notnull 'string'"`
	Hooks  string `xorm:"notnull 'hooks'"`
}

func (*testValid2) Table() string       { return "test_valid_2" }
func (*testValid2) TableName() string   { return "test_valid_2" }
func (*testValid2) Appellation() string { return "test valid 2" }
func (t *testValid2) GetID() uint64     { return t.ID }

func (t *testValid2) BeforeWrite(*Session) error {
	t.Hooks = "write hook"
	return nil
}

func (t *testValid2) BeforeDelete(*Session) error {
	t.Hooks = "delete hook"
	return nil
}

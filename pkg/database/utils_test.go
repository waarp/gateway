package database

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
)

func init() {
	logConf := conf.LogConfig{
		Level: "DEBUG",
		LogTo: "stdout",
	}
	_ = log.InitBackend(logConf)
}

type testBean struct {
	ID     uint64 `xorm:"pk 'id'"`
	String string `xorm:"notnull 'string'"`

	signals string `xorm:"-"`
}

func (*testBean) TableName() string { return tblName }
func (*testBean) ElemName() string  { return tblName }

func (t *testBean) GetID() uint64 {
	return t.ID
}

func (t *testBean) Validate(Accessor) error {
	t.signals = "validation hook"
	return nil
}

func (t *testBean) BeforeDelete(Accessor) error {
	t.signals = "delete hook"
	return nil
}

type invalidBean struct{}

func (*invalidBean) GetID() uint64     { return 0 }
func (*invalidBean) TableName() string { return "invalid" }
func (*invalidBean) ElemName() string  { return "invalid" }

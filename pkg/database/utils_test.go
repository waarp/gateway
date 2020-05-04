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

func (*testBean) TableName() string {
	return tblName
}

func (t *testBean) BeforeInsert(Accessor) error {
	t.signals = "insert hook"
	return nil
}

func (t *testBean) BeforeUpdate(Accessor, uint64) error {
	t.signals = "update hook"
	return nil
}

func (t *testBean) BeforeDelete(Accessor) error {
	t.signals = "delete hook"
	return nil
}

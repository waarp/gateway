// +build test_full test_db_mysql

package database

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	. "github.com/smartystreets/goconvey/convey"
)

var mysqlTestDatabase *Db

func init() {
	mysqlConfig := &conf.ServerConfig{}
	mysqlConfig.Database.Type = mysql
	mysqlConfig.Database.User = "waarp"
	mysqlConfig.Database.Name = "waarp_gatewayd_test"
	mysqlConfig.Database.Address = "localhost"
	mysqlConfig.Database.AESPassphrase = "/tmp/aes_passphrase"

	mysqlTestDatabase = &Db{Conf: mysqlConfig}
}

func TestMySQL(t *testing.T) {
	db := mysqlTestDatabase
	if err := db.Start(); err != nil {
		t.Fatal(err)
	}
	if err := db.engine.CreateTables(&testBean{}); err != nil {
		t.Fatal(err)
	}

	Convey("Given a MySQL service", t, func() {
		testDatabase(mysqlTestDatabase)
	})
}

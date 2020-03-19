// +build test_full test_db_mysql

package database

import (
	"fmt"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	mysqlTestDatabase *DB
	mysqlConfig       *conf.ServerConfig
)

func init() {
	mysqlConfig = &conf.ServerConfig{}
	mysqlConfig.Database.Type = mysql
	mysqlConfig.Database.User = "waarp"
	mysqlConfig.Database.Name = "waarp_gatewayd_test"
	mysqlConfig.Database.Address = "localhost"
	mysqlConfig.Database.AESPassphrase = fmt.Sprintf("%s%smysql_test_passphrase.aes",
		os.TempDir(), string(os.PathSeparator))

	mysqlTestDatabase = &DB{Conf: mysqlConfig}
}

func TestMySQL(t *testing.T) {
	defer func() { _ = os.Remove(mysqlConfig.Database.AESPassphrase) }()
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

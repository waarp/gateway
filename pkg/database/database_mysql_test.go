//go:build test_full || test_db_mysql
// +build test_full test_db_mysql

package database

import (
	"fmt"
	"os"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	mysqlTestDatabase *DB
	mysqlConfig       *conf.ServerConfig
)

//nolint:gochecknoinits // init is used by design
func init() {
	mysqlConfig = &conf.ServerConfig{}
	mysqlConfig.Database.Type = MySQL
	mysqlConfig.Database.User = "root"
	mysqlConfig.Database.Name = "waarp_gateway_test"
	mysqlConfig.Database.Address = "localhost:3306"
	mysqlConfig.Database.AESPassphrase = fmt.Sprintf("%s%smysql_test_passphrase.aes",
		os.TempDir(), string(os.PathSeparator))

	mysqlTestDatabase = &DB{Conf: mysqlConfig}
}

func TestMySQL(t *testing.T) {
	db := mysqlTestDatabase
	if err := db.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.engine.Close(); err != nil {
			t.Logf("Failed to close database: %s", err)
		}
		if err := os.Remove(mysqlConfig.Database.AESPassphrase); err != nil {
			t.Logf("Failed to delete passphrase file: %s", err)
		}
	}()

	Convey("Given a MySQL service", t, func() {
		testDatabase(mysqlTestDatabase)
	})
}

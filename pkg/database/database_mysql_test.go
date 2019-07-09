// +build test_full test_db_mysql

package database

import (
	"fmt"
	"testing"
	"time"

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

	mysqlTestDatabase = &Db{Conf: mysqlConfig}
}

func TestMySQL(t *testing.T) {
	start := time.Now()

	db := mysqlTestDatabase
	if err := db.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanDatabase(t, db)
		dur := time.Since(start)
		fmt.Printf("\nMySQL test finished in %s\n", dur)
	}()

	Convey("Given a MySQL service", t, func() {
		testDatabase(mysqlTestDatabase)
	})
}

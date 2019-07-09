// +build test_full test_db_postgresql

package database

import (
	"fmt"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	. "github.com/smartystreets/goconvey/convey"
)

var psqlTestDatabase *Db

func init() {
	psqlConfig := &conf.ServerConfig{}
	psqlConfig.Database.Type = postgres
	psqlConfig.Database.User = "waarp"
	psqlConfig.Database.Name = "waarp_gatewayd_test" + "' sslmode='disable"
	psqlConfig.Database.Address = "localhost"

	psqlTestDatabase = &Db{Conf: psqlConfig}
}

func TestPostgreSQL(t *testing.T) {
	start := time.Now()

	db := psqlTestDatabase
	if err := db.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanDatabase(t, db)
		dur := time.Since(start)
		fmt.Printf("\nPostgreSQL test finished in %s\n", dur)
	}()

	Convey("Given a PostgreSQL service", t, func() {
		testDatabase(psqlTestDatabase)
	})
}

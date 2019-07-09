// +build test_full test_db_oracle

package database

import (
	"fmt"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	. "github.com/smartystreets/goconvey/convey"
)

var oracleTestDatabase *Db

func init() {
	oracleConfig := &conf.ServerConfig{}
	oracleConfig.Database.Type = oracle
	oracleConfig.Database.User = "waarp"
	oracleConfig.Database.Password = "password"
	oracleConfig.Database.Name = "XE"
	oracleConfig.Database.Address = "localhost"

	oracleTestDatabase = &Db{Conf: oracleConfig}
}

func TestOracleDB(t *testing.T) {
	start := time.Now()

	db := oracleTestDatabase
	if err := db.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanDatabase(t, db)
		dur := time.Since(start)
		fmt.Printf("\nOracleDB test finished in %s\n", dur)
	}()

	Convey("Given an Oracledb service", t, func() {
		testDatabase(oracleTestDatabase)
	})
}

// +build test_db_oracle

package database

import (
	"fmt"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	oracleTestDatabase *DB
	oracleConfig       *conf.ServerConfig
)

func init() {
	oracleConfig = &conf.ServerConfig{}
	oracleConfig.Database.Type = oracle
	oracleConfig.Database.User = "waarp"
	oracleConfig.Database.Password = "password"
	oracleConfig.Database.Name = "XE"
	oracleConfig.Database.Address = "localhost"
	oracleConfig.Database.AESPassphrase = fmt.Sprintf("%s%soracle_test_passphrase.aes",
		os.TempDir(), string(os.PathSeparator))

	oracleTestDatabase = &DB{Conf: oracleConfig}
}

func TestOracleDB(t *testing.T) {
	db := oracleTestDatabase
	if err := db.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.engine.Close(); err != nil {
			t.Logf("Failed to close database: %s", err)
		}
		if err := os.Remove(sqliteConfig.Database.AESPassphrase); err != nil {
			t.Logf("Failed to delete passphrase file: %s", err)
		}
	}()

	Convey("Given an Oracledb service", t, func() {
		testDatabase(oracleTestDatabase)
	})
}

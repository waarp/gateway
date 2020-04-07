// +build test_full test_db_oracle

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
	defer func() { _ = os.Remove(oracleConfig.Database.AESPassphrase) }()
	db := oracleTestDatabase
	if err := db.Start(); err != nil {
		t.Fatal(err)
	}
	if err := db.engine.CreateTables(&testBean{}); err != nil {
		t.Fatal(err)
	}

	Convey("Given an Oracledb service", t, func() {
		testDatabase(oracleTestDatabase)
	})
}

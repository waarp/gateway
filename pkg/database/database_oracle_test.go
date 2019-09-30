// +build test_full test_db_oracle

package database

import (
	"testing"

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
	oracleConfig.Database.AESPassphrase = "/tmp/aes_passphrase"

	oracleTestDatabase = &Db{Conf: oracleConfig}
}

func TestOracleDB(t *testing.T) {
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

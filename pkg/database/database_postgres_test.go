// +build test_full test_db_postgresql

package database

import (
	"testing"

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
	psqlConfig.Database.AESPassphrase = "/tmp/aes_passphrase"

	psqlTestDatabase = &Db{Conf: psqlConfig}
}

func TestPostgreSQL(t *testing.T) {
	db := psqlTestDatabase
	if err := db.Start(); err != nil {
		t.Fatal(err)
	}
	if err := db.engine.CreateTables(&testBean{}); err != nil {
		t.Fatal(err)
	}

	Convey("Given a PostgreSQL service", t, func() {
		testDatabase(psqlTestDatabase)
	})
}

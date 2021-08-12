// +build test_full test_db_postgresql

package database

import (
	"fmt"
	"os"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	psqlTestDatabase *DB
	psqlConfig       *conf.ServerConfig
)

func init() {
	psqlConfig = &conf.ServerConfig{}
	psqlConfig.Database.Type = PostgreSQL
	psqlConfig.Database.User = "postgres"
	psqlConfig.Database.Name = "waarp_gateway_test"
	psqlConfig.Database.Address = "localhost:5432"
	psqlConfig.Database.AESPassphrase = fmt.Sprintf("%s%spsql_test_passphrase.aes",
		os.TempDir(), string(os.PathSeparator))

	psqlTestDatabase = &DB{Conf: psqlConfig}
}

func TestPostgreSQL(t *testing.T) {
	db := psqlTestDatabase
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

	Convey("Given a PostgreSQL service", t, func() {
		testDatabase(psqlTestDatabase)
	})
}

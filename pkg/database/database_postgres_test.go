//go:build test_db_postgresql
// +build test_db_postgresql

package database

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
)

func TestPostgreSQL(t *testing.T) {
	conf.GlobalConfig.Log.Level = "CRITICAL"
	conf.GlobalConfig.Log.LogTo = "stdout"
	conf.GlobalConfig.Database.Type = PostgreSQL
	conf.GlobalConfig.Database.User = "postgres"
	conf.GlobalConfig.Database.Name = "waarp_gateway_test"
	conf.GlobalConfig.Database.Address = "localhost:5432"
	conf.GlobalConfig.Database.AESPassphrase = filepath.Join(os.TempDir(), "pgsql_test_passphrase.aes")

	db := &DB{}
	if err := db.start(false); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.engine.Close(); err != nil {
			t.Logf("Failed to close database: %s", err)
		}
		if err := os.Remove(conf.GlobalConfig.Database.AESPassphrase); err != nil {
			t.Logf("Failed to delete passphrase file: %s", err)
		}
	}()

	Convey("Given a PostgreSQL service", t, func() {
		testDatabase(db)
	})
}

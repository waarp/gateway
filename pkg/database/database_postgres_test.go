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
	config := &conf.ServerConfig{}
	config.Log.Level = "CRITICAL"
	config.Log.LogTo = "stdout"
	config.Database.Type = PostgreSQL
	config.Database.User = "postgres"
	config.Database.Password = "postgres"
	config.Database.Name = "waarp_gateway_test"
	config.Database.Address = "localhost:5432"
	config.Database.AESPassphrase = filepath.Join(os.TempDir(), "pgsql_test_passphrase.aes")

	db := NewDB(config)
	if err := db.start(false); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.engine.Close(); err != nil {
			t.Logf("Failed to close database: %v", err)
		}
		if err := os.Remove(db.Config.Database.AESPassphrase); err != nil {
			t.Logf("Failed to delete passphrase file: %v", err)
		}
	}()

	Convey("Given a PostgreSQL service", t, func() {
		testDatabase(db)
	})
}

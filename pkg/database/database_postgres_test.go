// +build test_full test_db_postgresql

package database

import (
	"os"
	"path/filepath"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	. "github.com/smartystreets/goconvey/convey"
)

func TestPostgreSQL(t *testing.T) {
	conf.GlobalConfig.ServerConf.Log.Level = "CRITICAL"
	conf.GlobalConfig.ServerConf.Log.LogTo = "stdout"
	conf.GlobalConfig.ServerConf.Database.Type = PostgreSQL
	conf.GlobalConfig.ServerConf.Database.User = "postgres"
	conf.GlobalConfig.ServerConf.Database.Name = "waarp_gateway_test"
	conf.GlobalConfig.ServerConf.Database.Address = "localhost:5432"
	conf.GlobalConfig.ServerConf.Database.AESPassphrase = filepath.Join(os.TempDir(), "pgsql_test_passphrase.aes")

	db := &DB{}
	if err := db.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.engine.Close(); err != nil {
			t.Logf("Failed to close database: %s", err)
		}
		if err := os.Remove(conf.GlobalConfig.ServerConf.Database.AESPassphrase); err != nil {
			t.Logf("Failed to delete passphrase file: %s", err)
		}
	}()

	Convey("Given a PostgreSQL service", t, func() {
		testDatabase(db)
	})
}

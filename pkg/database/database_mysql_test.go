//go:build test_db_mysql
// +build test_db_mysql

package database

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
)

func TestMySQL(t *testing.T) {
	config := &conf.ServerConfig{}
	config.Log.Level = "CRITICAL"
	config.Log.LogTo = "stdout"
	config.Database.Type = MySQL
	config.Database.User = "root"
	config.Database.Name = "waarp_gateway_test"
	config.Database.Address = "localhost:3306"
	config.Database.AESPassphrase = filepath.Join(os.TempDir(), "mysql_test_passphrase.aes")

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

	Convey("Given a MySQL service", t, func() {
		testDatabase(db)
	})
}

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
	conf.GlobalConfig.Log.Level = "CRITICAL"
	conf.GlobalConfig.Log.LogTo = "stdout"
	conf.GlobalConfig.Database.Type = MySQL
	conf.GlobalConfig.Database.User = "root"
	conf.GlobalConfig.Database.Name = "waarp_gateway_test"
	conf.GlobalConfig.Database.Address = "localhost:3306"
	conf.GlobalConfig.Database.AESPassphrase = filepath.Join(os.TempDir(), "mysql_test_passphrase.aes")

	db := &DB{}
	if err := db.start(false); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := db.engine.Close(); err != nil {
			t.Logf("Failed to close database: %v", err)
		}
		if err := os.Remove(conf.GlobalConfig.Database.AESPassphrase); err != nil {
			t.Logf("Failed to delete passphrase file: %v", err)
		}
	}()

	Convey("Given a MySQL service", t, func() {
		testDatabase(db)
	})
}

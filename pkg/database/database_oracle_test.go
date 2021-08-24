// +build test_db_oracle

package database

import (
	"os"
	"path/filepath"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	. "github.com/smartystreets/goconvey/convey"
)

func TestOracleDB(t *testing.T) {
	conf.GlobalConfig.Log.Level = "CRITICAL"
	conf.GlobalConfig.Log.LogTo = "stdout"
	conf.GlobalConfig.Database.Type = "oracle"
	conf.GlobalConfig.Database.User = "waarp"
	conf.GlobalConfig.Database.Password = "password"
	conf.GlobalConfig.Database.Name = "XE"
	conf.GlobalConfig.Database.Address = "localhost"
	conf.GlobalConfig.Database.AESPassphrase = filepath.Join(os.TempDir(), "oracle_test_passphrase.aes")

	db := &DB{}
	if err := db.Start(); err != nil {
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

	Convey("Given an Oracledb service", t, func() {
		testDatabase(db)
	})
}

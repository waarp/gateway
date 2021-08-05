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
	conf.GlobalConfig.ServerConf.Log.Level = "CRITICAL"
	conf.GlobalConfig.ServerConf.Log.LogTo = "stdout"
	conf.GlobalConfig.ServerConf.Database.Type = "oracle"
	conf.GlobalConfig.ServerConf.Database.User = "waarp"
	conf.GlobalConfig.ServerConf.Database.Password = "password"
	conf.GlobalConfig.ServerConf.Database.Name = "XE"
	conf.GlobalConfig.ServerConf.Database.Address = "localhost"
	conf.GlobalConfig.ServerConf.Database.AESPassphrase = filepath.Join(os.TempDir(), "oracle_test_passphrase.aes")

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

	Convey("Given an Oracledb service", t, func() {
		testDatabase(db)
	})
}

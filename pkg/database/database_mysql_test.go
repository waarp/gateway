//go:build test_db_mysql
// +build test_db_mysql

package database

import (
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
	config.Database.AESPassphrase = filepath.Join(t.TempDir(), "mysql_test_passphrase.aes")

	db := NewDB(config)
	if err := db.start(false); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := db.engine.Exec("DROP DATABASE IF EXISTS waarp_gateway_test").Error; err != nil {
			t.Logf("Failed to drop database: %v", err)
		}

		if err := db.engine.Exec("CREATE DATABASE waarp_gateway_test").Error; err != nil {
			t.Logf("Failed to restore database: %v", err)
		}

		if err := db.close(); err != nil {
			t.Logf("Failed to close database: %v", err)
		}
	}()

	checkCharset(t, db)

	Convey("Given a MySQL service", t, func() {
		testDatabase(db)
	})
}

func checkCharset(tb testing.TB, db *DB) {
	tb.Helper()

	res := struct{ Client, Conn, Res string }{}
	row := db.engine.Raw(`SELECT
		@@character_set_client     AS client,
		@@character_set_connection AS conn,
		@@character_set_results    AS res`)
	if err := row.Scan(&res).Error; err != nil {
		tb.Fatal(err)
	}

	if res.Client != "utf8mb4" {
		tb.Fatalf("Unexpected client charset %q", res)
	}

	if res.Conn != "utf8mb4" {
		tb.Fatalf("Unexpected connection charset %q", res)
	}

	if res.Res != "utf8mb4" {
		tb.Fatalf("Unexpected connection charset %q", res)
	}
}

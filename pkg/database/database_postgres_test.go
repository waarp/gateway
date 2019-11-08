// +build test_full test_db_postgresql

package database

import (
	"fmt"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	psqlTestDatabase *Db
	psqlConfig       *conf.ServerConfig
)

func init() {
	psqlConfig = &conf.ServerConfig{}
	psqlConfig.Database.Type = postgres
	psqlConfig.Database.User = "waarp"
	psqlConfig.Database.Name = "waarp_gatewayd_test" + "' sslmode='disable"
	psqlConfig.Database.Address = "localhost"
	psqlConfig.Database.AESPassphrase = fmt.Sprintf("%s%spsql_test_passphrase.aes",
		os.TempDir(), string(os.PathSeparator))

	psqlTestDatabase = &Db{Conf: psqlConfig}
}

func TestPostgreSQL(t *testing.T) {
	defer func() { _ = os.Remove(psqlConfig.Database.AESPassphrase) }()
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

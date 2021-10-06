// +build test_full test_db_mysql

package migrations

import (
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/migration"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMySQLMigrations(t *testing.T) {
	const dbType = migration.MySQL

	Convey("Given an un-migrated MySQL database engine", t, func(c C) {
		eng := getSQLiteEngine(c)

		testVer0_4_2RemoveHistoryRemoteIdUnique(eng, dbType)
	})
}

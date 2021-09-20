package migrations

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/migration"
)

func TestSQLiteMigrations(t *testing.T) {
	const dbType = migration.SQLite

	Convey("Given an un-migrated SQLite database engine", t, func(c C) {
		eng := getSQLiteEngine(c)

		testVer0_4_2RemoveHistoryRemoteIdUnique(eng, dbType)
	})
}

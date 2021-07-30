package migrations

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/migration"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSQLiteMigrations(t *testing.T) {
	const dbType = migration.SQLite

	Convey("Given an un-migrated SQLite database engine", t, func(c C) {
		eng := getSQLiteEngine(c)

		testVer0_5_0RemoveRulePathSlash(eng, dbType)
		testVer0_5_0CheckRulePathAncestor(eng, dbType)
	})
}

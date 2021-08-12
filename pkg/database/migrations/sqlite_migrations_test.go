package migrations

import (
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/migration"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSQLiteMigrations(t *testing.T) {
	const dbType = migration.SQLite

	Convey("Given an un-migrated SQLite database engine", t, func(c C) {
		_ = getSQLiteEngine(c)

	})
}

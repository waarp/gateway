package migrations

import (
	"database/sql"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/migration"
	. "github.com/smartystreets/goconvey/convey"
)

func setupDatabaseUpTo(eng *migration.Engine, migrationIndex int) {
	So(eng.Upgrade(Migrations[:migrationIndex]), ShouldNotBeNil)
}

func tableShouldHaveColumn(db *sql.DB, table string, cols ...string) {
	rows, err := db.Query("SELECT * FROM " + table)
	So(err, ShouldBeNil)
	defer rows.Close()

	names, err := rows.Columns()
	So(err, ShouldBeNil)
	for _, col := range cols {
		So(names, ShouldContain, col)
	}
}

func tableShouldNotHaveColumn(db *sql.DB, table string, cols ...string) {
	rows, err := db.Query("SELECT * FROM " + table)
	So(err, ShouldBeNil)
	defer rows.Close()

	names, err := rows.Columns()
	So(err, ShouldBeNil)
	for _, col := range cols {
		So(names, ShouldNotContain, col)
	}
}

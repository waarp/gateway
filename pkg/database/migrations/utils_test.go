package migrations

import (
	"database/sql"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/migration"
	. "github.com/smartystreets/goconvey/convey"
)

func setupDatabaseUpTo(eng *migration.Engine, target *migration.Script) {
	index := -1
	for i, mig := range Migrations {
		if mig.Script == target {
			index = i
			break
		}
	}
	So(index, ShouldBeGreaterThanOrEqualTo, 0)
	So(eng.Upgrade(Migrations[:index]), ShouldNotBeNil)
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

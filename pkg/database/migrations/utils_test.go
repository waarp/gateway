package migrations

import (
	"database/sql"
	"fmt"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/migration"
)

func setupDatabaseUpTo(eng *migration.Engine, target migration.Script) {
	index := -1

	for i, mig := range Migrations {
		if mig.Script == target {
			index = i

			break
		}
	}

	So(index, ShouldBeGreaterThanOrEqualTo, 0)
	So(eng.Upgrade(Migrations[:index]), ShouldBeNil)
}

func doesIndexExist(db *sql.DB, dialect, table, index string) bool {
	var (
		rows *sql.Rows
		err  error
	)

	switch dialect {
	case migration.SQLite:
		rows, err = db.Query("SELECT name FROM sqlite_master WHERE type=? AND name=?", "index", index)
	case migration.PostgreSQL:
		rows, err = db.Query("SELECT indexname FROM pg_indexes WHERE indexname=$1", index)
	case migration.MySQL:
		rows, err = db.Query("SHOW INDEX FROM "+table+" WHERE Key_name=?", index)
	default:
		panic(fmt.Sprintf("unknown database engine '%s'", dialect))
	}

	So(err, ShouldBeNil)
	So(rows.Err(), ShouldBeNil)

	defer rows.Close()

	return rows.Next()
}

// func tableShouldHaveColumn(db *sql.DB, table string, cols ...string) {
// 	rows, err := db.Query("SELECT * FROM " + table)
// 	So(err, ShouldBeNil)
// 	So(rows.Err(), ShouldBeNil)
//
// 	defer rows.Close()
//
// 	names, err := rows.Columns()
// 	So(err, ShouldBeNil)
//
// 	for _, col := range cols {
// 		So(names, ShouldContain, col)
// 	}
// }

// func tableShouldNotHaveColumn(db *sql.DB, table string, cols ...string) {
// 	rows, err := db.Query("SELECT * FROM " + table)
// 	So(err, ShouldBeNil)
// 	So(rows.Err(), ShouldBeNil)
//
// 	defer rows.Close()
//
// 	names, err := rows.Columns()
// 	So(err, ShouldBeNil)
//
// 	for _, col := range cols {
// 		So(names, ShouldNotContain, col)
// 	}
// }

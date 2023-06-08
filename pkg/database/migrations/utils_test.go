package migrations

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	"code.waarp.fr/lib/migration"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	. "github.com/smartystreets/goconvey/convey"
	"modernc.org/sqlite"
	sqliteLib "modernc.org/sqlite/lib"
)

func setupDatabaseUpTo(eng *testEngine, target Script) {
	index := -1

	for i, mig := range Migrations {
		if mig.Script == target {
			index = i

			break
		}
	}

	So(index, ShouldBeGreaterThanOrEqualTo, 0)
	So(eng.Engine.Upgrade(makeMigration(Migrations[:index])), ShouldBeNil)
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

func tableShouldHaveColumns(db *sql.DB, table string, cols ...string) {
	rows, err := db.Query("SELECT * FROM " + table)
	So(err, ShouldBeNil)
	So(rows.Err(), ShouldBeNil)

	defer rows.Close()

	names, err := rows.Columns()
	So(err, ShouldBeNil)

	for _, col := range cols {
		So(names, ShouldContain, col)
	}
}

func tableShouldNotHaveColumns(db *sql.DB, table string, cols ...string) {
	rows, err := db.Query("SELECT * FROM " + table)
	So(err, ShouldBeNil)
	So(rows.Err(), ShouldBeNil)

	defer rows.Close()

	names, err := rows.Columns()
	So(err, ShouldBeNil)

	for _, col := range cols {
		So(names, ShouldNotContain, col)
	}
}

func shouldBeUniqueViolationError(err error) {
	var sqlErr *sqlite.Error
	if errors.As(err, &sqlErr) {
		So(sqlErr.Code(), ShouldEqual, sqliteLib.SQLITE_CONSTRAINT_UNIQUE)

		return
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		So(pgErr.Code, ShouldEqual, pgerrcode.UniqueViolation)

		return
	}

	var myErr *mysql.MySQLError
	if errors.As(err, &myErr) {
		So(myErr.Number, ShouldEqual, 1062)

		return
	}

	panic(fmt.Sprintf("unknown error type %T", err))
}

func doesTableExist(db *sql.DB, dbType, table string) bool {
	var (
		row  *sql.Row
		name string
	)

	switch dbType {
	case migration.SQLite:
		row = db.QueryRow(`SELECT name FROM sqlite_master WHERE
			type='table' AND name=?`, table)
	case migration.PostgreSQL:
		row = db.QueryRow(`SELECT tablename FROM pg_tables WHERE tablename=$1`, table)
	case migration.MySQL:
		row = db.QueryRow(fmt.Sprintf(`SHOW TABLES LIKE '%s'`, table))
	default:
		panic(fmt.Sprintf("unknown database type: %s", dbType))
	}

	if err := row.Scan(&name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false
		}

		So(err, ShouldBeNil)
	}

	return true
}

func queryAndParse[T any](db *sql.DB, result *[]T, query string, args ...any) {
	rows, err := db.Query(query, args...)
	So(err, ShouldBeNil)

	defer rows.Close()

	typ := reflect.TypeOf(*new(T))
	isStruct := typ.Kind() == reflect.Struct

	for rows.Next() {
		row := new(T)

		if !isStruct {
			So(rows.Scan(row), ShouldBeNil)
		} else {
			cols := make([]any, typ.NumField())
			for i := range cols {
				cols[i] = reflect.ValueOf(row).Elem().Field(i).Addr().Interface()
			}

			So(rows.Scan(cols...), ShouldBeNil)
		}

		*result = append(*result, *row)
	}

	So(rows.Err(), ShouldBeNil)
}

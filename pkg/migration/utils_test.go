package migration

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/mattn/go-sqlite3"

	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	isTest = true
}

type testEngine interface {
	sqlFormatter
	Actions
}

type testInterface struct {
	str string
}

func (v *testInterface) Value() (driver.Value, error) {
	return v.str, nil
}

func (v *testInterface) Scan(i interface{}) error {
	switch typ := i.(type) {
	case string:
		v.str = typ
		return nil
	case []byte:
		v.str = string(typ)
		return nil
	default:
		return fmt.Errorf("invalid type %T", i)
	}
}

func tempFilename() string {
	f, err := ioutil.TempFile(os.TempDir(), "test_migration_database_*.db")
	So(err, ShouldBeNil)
	So(f.Close(), ShouldBeNil)
	So(os.Remove(f.Name()), ShouldBeNil)
	return f.Name()
}

func isTableNotFound(err error) bool {
	var sqlErr sqlite3.Error
	if errors.As(err, &sqlErr) {
		return strings.HasPrefix(sqlErr.Error(), "no such table:")
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgerrcode.UndefinedTable
	}

	var myErr *mysql.MySQLError
	if errors.As(err, &myErr) {
		return myErr.Number == 1051 || myErr.Number == 1146
	}

	panic(fmt.Sprintf("unknown error type %T", err))
}

func isColumnNotFound(err error) bool {
	var sqlErr sqlite3.Error
	if errors.As(err, &sqlErr) {
		return strings.HasPrefix(sqlErr.Error(), "no such column:")
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgerrcode.UndefinedColumn
	}

	var myErr *mysql.MySQLError
	if errors.As(err, &myErr) {
		return myErr.Number == 1054
	}

	panic(fmt.Sprintf("unknown error type %T: %s", err, err))
}

func isColumnAlreadyExist(err error) bool {
	var sqlErr sqlite3.Error
	if errors.As(err, &sqlErr) {
		return strings.Contains(sqlErr.Error(), "duplicate column name:")
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgerrcode.DuplicateColumn
	}

	var myErr *mysql.MySQLError
	if errors.As(err, &myErr) {
		return myErr.Number == 1060
	}

	panic(fmt.Sprintf("unknown error type %T", err))
}

func doesTableExist(db *sql.DB, table string) bool {
	rows, err := db.Query("SELECT * FROM " + table)
	if err != nil {
		if isTableNotFound(err) {
			return false
		}
		panic(err)
	}
	So(rows.Close(), ShouldBeNil)
	return true
}

func tableShouldHaveColumns(db *sql.DB, table string, cols ...string) {
	rows, err := db.Query("SELECT * FROM " + table)
	So(err, ShouldBeNil)
	defer rows.Close()

	names, err := rows.Columns()
	So(err, ShouldBeNil)
	So(names, ShouldResemble, cols)
}

func removeTypeLength(typ string) string {
	return strings.Split(typ, "(")[0]
}

func colShouldHaveType(engine testEngine, table, col string, exp sqlType) {
	rows, err := engine.Query("SELECT %s FROM %s", col, table)
	So(err, ShouldBeNil)
	defer rows.Close()

	typ, err := engine.sqlTypeToDBType(exp)
	So(err, ShouldBeNil)
	typ = removeTypeLength(typ)
	types, err := rows.ColumnTypes()
	So(err, ShouldBeNil)

	So(types[0].DatabaseTypeName(), ShouldEqual, typ)
}

func getTimeFormats(eng Actions) (date, ts, tsz string) {
	date = "2006-01-02"
	ts = "2006-01-02 15:04:05.999999999"
	tsz = "2006-01-02 15:04:05.999999999Z07:00"
	if _, ok := eng.(*mySQLDialect); ok {
		ts = "2006-01-02 15:04:05"
	}
	if _, ok := eng.(*postgreDialect); ok {
		date = time.RFC3339
		ts = "2006-01-02T15:04:05.999999Z07:00"
		tsz = ts
	}
	return
}

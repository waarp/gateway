package migrations

import (
	"database/sql"

	. "github.com/smartystreets/goconvey/convey"
)

func testVer0_4_0InitDatabase(eng *testEngine, dialect string) {
	dbShouldBeEmpty := func() {
		var row *sql.Row

		switch dialect {
		case SQLite:
			row = eng.DB.QueryRow(`SELECT name FROM sqlite_master WHERE 
                name<>'sqlite_sequence'`)
		case PostgreSQL:
			row = eng.DB.QueryRow(`SELECT tablename FROM pg_tables 
                 WHERE schemaname=current_schema()`)
		case MySQL:
			row = eng.DB.QueryRow(`SHOW TABLES`)
		}

		var name string
		err := row.Scan(&name)
		So(err, ShouldWrap, sql.ErrNoRows)
	}

	Convey("Given the 0.4.0 database initialization", func() {
		setupDatabaseUpTo(eng, ver0_4_0InitDatabase{})
		dbShouldBeEmpty()

		Convey("When applying the migration", func() {
			So(eng.Upgrade(ver0_4_0InitDatabase{}), ShouldBeNil)

			Convey("Then it should have initialized the database", func() {
				for _, table := range initTableList() {
					doesTableExist(eng.DB, dialect, table)
				}
			})

			Convey("When reversing the migration", func() {
				So(eng.Downgrade(ver0_4_0InitDatabase{}), ShouldBeNil)

				Convey("Then it should have reset the database", func() {
					dbShouldBeEmpty()
				})
			})
		})
	})
}

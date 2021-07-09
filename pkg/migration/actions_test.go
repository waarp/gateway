package migration

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func testSQLCreateTable(t *testing.T, dbms string, initDB func(C) *sql.DB,
	getEngine func(*sql.DB) testEngine) {

	Convey(fmt.Sprintf("Given a %s database", dbms), t, func(c C) {
		db := initDB(c)
		_, err := db.Exec("CREATE TABLE titi (id INTEGER PRIMARY KEY)")
		So(err, ShouldBeNil)

		Convey(fmt.Sprintf("Given a %s dialect engine", dbms), func() {
			engine := getEngine(db)

			Convey("When adding a table with various columns and constraints", func() {
				defs := []Definition{
					Col("i64", BigInt, PrimaryKey, AutoIncr),
					Col("i32", Integer, NotNull, ForeignKey("titi", "id")),
					Col("i16", SmallInt, NotNull),
					Col("i8", TinyInt),
					Col("flo", Float, Unique),
					Col("dou", Double),
					Col("bol", Boolean),
					Col("vc1", Varchar(4)),
					Col("str", Text, Default("txt")),
					Col("bin", Binary(4)),
					Col("blo", Blob),
					Col("dat", Date),
					Col("ts", Timestamp),
					Col("tsz", Timestampz),
					MultiUnique("i32", "i8"),
				}
				So(engine.CreateTable("toto", defs...), ShouldBeNil)

				Convey("Then it should have added the table with the correct types", func() {
					for _, def := range defs {
						if col, ok := def.(Column); ok {
							Convey(fmt.Sprintf("Then column %s should have type %s", col.Name,
								col.Type.code.String()), func() {
								colShouldHaveType(engine, "toto", col.Name, col.Type)
							})
						}
					}
				})
			})
		})
	})
}

func testSQLRenameTable(t *testing.T, dbms string, initDB func(C) *sql.DB,
	getEngine func(*sql.DB) testEngine) {

	Convey(fmt.Sprintf("Given a %s database with 2 tables", dbms), t, func(c C) {
		db := initDB(c)
		_, err := db.Exec("CREATE TABLE toto (str TEXT)")
		So(err, ShouldBeNil)
		_, err = db.Exec("CREATE TABLE tutu (str TEXT)")
		So(err, ShouldBeNil)

		Convey(fmt.Sprintf("Given a %s dialect engine", dbms), func() {
			engine := getEngine(db)

			Convey("When renaming the table", func() {
				So(engine.RenameTable("toto", "tata"), ShouldBeNil)

				Convey("Then the table should have been renamed", func() {
					So(doesTableExist(db, "tata"), ShouldBeTrue)
					So(doesTableExist(db, "toto"), ShouldBeFalse)
				})
			})

			Convey("When renaming a non-existing table", func() {
				err := engine.RenameTable("titi", "tata")
				So(isTableNotFound(err), ShouldBeTrue)

				Convey("Then the existing table should be unchanged", func() {
					So(doesTableExist(db, "toto"), ShouldBeTrue)
				})
			})

			Convey("When renaming a table to an already existing name", func() {
				err := engine.RenameTable("toto", "tutu")
				So(err, ShouldNotBeNil)

				Convey("Then the existing tables should be unchanged", func() {
					So(doesTableExist(db, "toto"), ShouldBeTrue)
					So(doesTableExist(db, "tutu"), ShouldBeTrue)
				})
			})
		})
	})
}

func testSQLDropTable(t *testing.T, dbms string, initDB func(C) *sql.DB,
	getEngine func(*sql.DB) testEngine) {

	Convey(fmt.Sprintf("Given a %s database with a table", dbms), t, func(c C) {
		db := initDB(c)
		_, err := db.Exec("CREATE TABLE toto (str TEXT)")
		So(err, ShouldBeNil)

		Convey(fmt.Sprintf("Given a %s dialect engine", dbms), func() {
			engine := getEngine(db)

			Convey("When dropping the table", func() {
				So(engine.DropTable("toto"), ShouldBeNil)

				Convey("Then the table should have been removed", func() {
					So(doesTableExist(db, "toto"), ShouldBeFalse)
				})
			})

			Convey("When dropping a non-existing table", func() {
				err := engine.DropTable("titi")
				So(isTableNotFound(err), ShouldBeTrue)

				Convey("Then the existing table should be unchanged", func() {
					So(doesTableExist(db, "toto"), ShouldBeTrue)
				})
			})
		})
	})
}

func testSQLRenameColumn(t *testing.T, dbms string, initDB func(C) *sql.DB,
	getEngine func(*sql.DB) testEngine) {

	Convey(fmt.Sprintf("Given a %s database with a table", dbms), t, func(c C) {
		db := initDB(c)
		_, err := db.Exec("CREATE TABLE toto (str TEXT, id BIGINT)")
		So(err, ShouldBeNil)

		Convey(fmt.Sprintf("Given a %s dialect engine", dbms), func() {
			engine := getEngine(db)

			Convey("When renaming a column", func() {
				So(engine.RenameColumn("toto", "str", "new_str"), ShouldBeNil)

				Convey("Then the column should have been renamed", func() {
					tableShouldHaveColumns(db, "toto", "new_str", "id")
				})
			})

			Convey("When renaming to an already existing name", func() {
				err := engine.RenameColumn("toto", "str", "id")
				So(isColumnAlreadyExist(err), ShouldBeTrue)

				Convey("Then the table should be unchanged", func() {
					tableShouldHaveColumns(db, "toto", "str", "id")
					colShouldHaveType(engine, "toto", "str", Text)
				})
			})

			Convey("When renaming a non-existing column", func() {
				err := engine.RenameColumn("toto", "NA", "new_str")
				So(isColumnNotFound(err), ShouldBeTrue)

				Convey("Then the existing column should be unchanged", func() {
					tableShouldHaveColumns(db, "toto", "str", "id")
				})
			})

			Convey("When renaming a column from a non-existing table", func() {
				err := engine.RenameColumn("titi", "str", "new_str")
				So(isTableNotFound(err), ShouldBeTrue)

				Convey("Then the existing table should be unchanged", func() {
					tableShouldHaveColumns(db, "toto", "str", "id")
					So(doesTableExist(db, "titi"), ShouldBeFalse)
				})
			})
		})
	})
}

func testSQLChangeColumnType(t *testing.T, dbms string, initDB func(C) *sql.DB,
	getEngine func(*sql.DB) testEngine) {

	Convey(fmt.Sprintf("Given a %s database with a table", dbms), t, func(c C) {
		db := initDB(c)

		Convey(fmt.Sprintf("Given a %s dialect engine", dbms), func() {
			engine := getEngine(db)
			So(engine.CreateTable("toto", Col("str", Varchar(10))), ShouldBeNil)
			So(engine.AddRow("toto", Cells{"str": Cel(Varchar(10), "sesame")}), ShouldBeNil)

			Convey("When changing a column's type", func() {
				So(engine.ChangeColumnType("toto", "str", Varchar(10), Text), ShouldBeNil)

				Convey("Then the column's type should have changed", func() {
					colShouldHaveType(engine, "toto", "str", Text)
				})
			})

			Convey("When the type conversion is not possible", func() {
				err := engine.ChangeColumnType("toto", "str", Varchar(10), Double)
				So(err, ShouldBeError, "cannot convert from type varchar to type double")

				Convey("Then the table should be unchanged", func() {
					colShouldHaveType(engine, "toto", "str", Varchar(10))
				})
			})
		})
	})
}

func testSQLAddColumn(t *testing.T, dbms string, initDB func(C) *sql.DB,
	getEngine func(*sql.DB) testEngine) {

	Convey(fmt.Sprintf("Given a %s database with a table", dbms), t, func(c C) {
		db := initDB(c)
		_, err := db.Exec("CREATE TABLE toto (str TEXT)")
		So(err, ShouldBeNil)

		Convey(fmt.Sprintf("Given a %s dialect engine", dbms), func() {
			engine := getEngine(db)

			Convey("When adding a column", func() {
				So(engine.AddColumn("toto", "id", Integer, NotNull), ShouldBeNil)

				Convey("Then the column should have been added", func() {
					tableShouldHaveColumns(db, "toto", "str", "id")
					colShouldHaveType(engine, "toto", "id", Integer)
				})
			})

			Convey("When adding an already existing column", func() {
				err := engine.AddColumn("toto", "str", Integer)
				So(isColumnAlreadyExist(err), ShouldBeTrue)

				Convey("Then the table should be unchanged", func() {
					tableShouldHaveColumns(db, "toto", "str")
					colShouldHaveType(engine, "toto", "str", Text)
				})
			})

			Convey("When adding a column to a non-existing table", func() {
				err := engine.AddColumn("titi", "id", Integer)
				So(isTableNotFound(err), ShouldBeTrue)

				Convey("Then the existing table should be unchanged", func() {
					tableShouldHaveColumns(db, "toto", "str")
					So(doesTableExist(db, "titi"), ShouldBeFalse)
				})
			})
		})
	})
}

func testSQLDropColumn(t *testing.T, dbms string, initDB func(C) *sql.DB,
	getEngine func(*sql.DB) testEngine) {

	Convey(fmt.Sprintf("Given a %s database with a table", dbms), t, func(c C) {
		db := initDB(c)
		_, err := db.Exec("CREATE TABLE toto (str TEXT, id BIGINT)")
		So(err, ShouldBeNil)

		Convey(fmt.Sprintf("Given a %s dialect engine", dbms), func() {
			engine := getEngine(db)

			Convey("When dropping a column", func() {
				So(engine.DropColumn("toto", "id"), ShouldBeNil)

				Convey("Then the column should have been dropped", func() {
					tableShouldHaveColumns(db, "toto", "str")
				})
			})

			Convey("When dropping from a non-existing table", func() {
				err := engine.DropColumn("titi", "id")
				So(isTableNotFound(err), ShouldBeTrue)

				Convey("Then the existing table should be unchanged", func() {
					tableShouldHaveColumns(db, "toto", "str", "id")

				})
			})

			Convey("When dropping a non-existing column", func() {
				err := engine.DropColumn("toto", "NA")
				So(isColumnNotFound(err), ShouldBeTrue)

				Convey("Then the existing table should be unchanged", func() {
					tableShouldHaveColumns(db, "toto", "str", "id")
				})
			})
		})
	})
}

func testSQLAddRow(t *testing.T, dbms string, initDB func(C) *sql.DB,
	getEngine func(*sql.DB) testEngine) {

	Convey(fmt.Sprintf("Given a %s database", dbms), t, func(c C) {
		db := initDB(c)

		Convey(fmt.Sprintf("Given a %s dialect engine", dbms), func() {
			engine := getEngine(db)
			dateFormat, tsFormat, tszFormat := getTimeFormats(engine)

			Convey("Given a table with various types", func() {
				So(engine.CreateTable("toto",
					Col("i64", BigInt),
					Col("i32", Integer),
					Col("i16", SmallInt),
					Col("i8", TinyInt),
					Col("flo", Float),
					Col("dou", Double),
					Col("bol", Boolean),
					Col("vc1", Varchar(4)),
					Col("str", Text),
					Col("bin", Binary(4)),
					Col("blo", Blob),
					Col("dat", Date),
					Col("ts", Timestamp),
					Col("tsz", Timestampz),
				), ShouldBeNil)

				Convey("When adding a row", func() {
					tDat := time.Date(1970, 1, 1, 1, 0, 0, 0, time.UTC)
					tTs := time.Date(1980, 1, 1, 1, 0, 0, 111111111, time.UTC)
					tTsz := time.Date(1990, 1, 1, 1, 0, 0, 222222222, time.Local)

					So(engine.AddRow("toto", Cells{
						"i64": Cel(BigInt, int64(64)),
						"i32": Cel(Integer, int32(32)),
						"i16": Cel(SmallInt, int16(16)),
						"i8":  Cel(TinyInt, int8(8)),
						"flo": Cel(Float, float32(1.1)),
						"dou": Cel(Double, float64(2.2)),
						"bol": Cel(Boolean, true),
						"vc1": Cel(Varchar(4), "abcd"),
						"str": Cel(Text, &testInterface{str: "message"}),
						"bin": Cel(Binary(4), []byte{0x00, 0xFF, 0x00, 0xFF}),
						"blo": Cel(Blob, []byte{0x00, 0xFF, 0xFF, 0x00}),
						"dat": Cel(Date, tDat),
						"ts":  Cel(Timestamp, tTs),
						"tsz": Cel(Timestampz, tTsz),
					}), ShouldBeNil)

					Convey("Then the row should have been added", func() {
						rows, err := db.Query("SELECT * FROM toto")
						So(err, ShouldBeNil)
						defer func() { So(rows.Close(), ShouldBeNil) }()

						So(rows.Next(), ShouldBeTrue)

						var (
							i64          int64
							i32          int32
							i16          int16
							i8           int8
							fl           float32
							dou          float64
							bo           bool
							vc, str      string
							bin, blo     []byte
							dat, ts, tsz string
						)

						So(rows.Scan(&i64, &i32, &i16, &i8, &fl, &dou, &bo, &vc,
							&str, &bin, &blo, &dat, &ts, &tsz), ShouldBeNil)

						So(i64, ShouldEqual, 64)
						So(i32, ShouldEqual, 32)
						So(i16, ShouldEqual, 16)
						So(i8, ShouldEqual, 8)
						So(fl, ShouldEqual, 1.1)
						So(dou, ShouldEqual, 2.2)
						So(bo, ShouldEqual, true)
						So(vc, ShouldEqual, "abcd")
						So(str, ShouldEqual, "message")
						So(bin, ShouldResemble, []byte{0x0, 0xFF, 0x00, 0xFF})
						So(blo, ShouldResemble, []byte{0x0, 0xFF, 0xFF, 0x00})
						SkipSo(dat, ShouldEqual, tDat.Format(dateFormat))
						SkipSo(ts, ShouldEqual, tTs.Format(tsFormat))
						So(tsz, ShouldEqual, tTsz.Format(tszFormat))
					})
				})
			})
		})
	})
}

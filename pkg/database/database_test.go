package database

import (
	"context"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"github.com/go-xorm/builder"
	"github.com/go-xorm/xorm"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
)

const tblName = "test"

var (
	sqliteTestDatabase *Db
	signals            string
)

type testBean struct {
	ID     uint64 `xorm:"pk 'id'"`
	String string `xorm:"notnull 'string'"`
}

func (*testBean) TableName() string {
	return tblName
}

func (*testBean) BeforeInsert(Accessor) error {
	signals = "insert hook"
	return nil
}

func (*testBean) BeforeUpdate(Accessor) error {
	signals = "update hook"
	return nil
}

func (*testBean) BeforeDelete(Accessor) error {
	signals = "delete hook"
	return nil
}

func init() {
	BcryptRounds = bcrypt.MinCost

	sqliteConfig := &conf.ServerConfig{}
	sqliteConfig.Database.Type = sqlite
	sqliteConfig.Database.Name = "file::memory:?mode=memory&cache=shared"
	sqliteConfig.Database.AESPassphrase = "/tmp/aes_passphrase"

	sqliteTestDatabase = &Db{Conf: sqliteConfig}
}

func testGet(db *Db) {
	getBean := &testBean{
		ID:     1,
		String: "get",
	}

	runTests := func(acc Accessor) {
		Convey("With an existing ID", func() {
			result := &testBean{ID: getBean.ID}
			err := acc.Get(result)

			Convey("Then the parameter should contain the result", func() {
				So(err, ShouldBeNil)
				So(result, ShouldResemble, getBean)
			})
		})

		Convey("With an unknown ID", func() {
			err := acc.Get(&testBean{ID: 1000})

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, ErrNotFound)
			})
		})

		Convey("With a nil key", func() {
			err := acc.Get(nil)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, ErrNilRecord)
			})
		})
	}

	Convey("When calling the 'Get' method", func() {
		_, err := db.engine.InsertOne(getBean)
		So(err, ShouldBeNil)

		Convey("Using the standalone accessor", func() {
			runTests(db)
		})

		Convey("Using the transaction accessor", func() {
			ses, err := db.BeginTransaction()
			So(err, ShouldBeNil)

			Reset(ses.session.Close)

			runTests(ses)
		})

	})
}

func testSelect(db *Db) {
	selectBean1 := testBean{
		ID:     1,
		String: "select",
	}
	selectBean2 := testBean{
		ID:     2,
		String: selectBean1.String,
	}
	selectBean3 := testBean{
		ID:     3,
		String: selectBean1.String,
	}
	selectBean4 := testBean{
		ID:     4,
		String: selectBean1.String,
	}
	selectBean5 := testBean{
		ID:     5,
		String: selectBean1.String,
	}

	runTests := func(acc Accessor) {
		filters := &Filters{}
		result := []testBean{}

		Convey("With just a condition", func() {
			filters.Conditions = builder.Eq{"id": selectBean1.ID}
			err := acc.Select(&result, filters)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then it should return all the valid entries", func() {
					filtered := []testBean{selectBean1}
					So(result, ShouldResemble, filtered)
				})
			})
		})

		Convey("With a limit of 2", func() {
			filters.Limit = 2
			limited := []testBean{selectBean1, selectBean2}
			err := acc.Select(&result, filters)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then it should return 2 entries at most", func() {
					So(result, ShouldResemble, limited)
				})
			})
		})

		Convey("With an offset of 1", func() {
			filters.Limit = 10
			filters.Offset = 1
			err := acc.Select(&result, filters)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then it should return all valid entries except the first one", func() {
					offset := []testBean{selectBean2, selectBean3, selectBean4,
						selectBean5}
					So(result, ShouldResemble, offset)
				})
			})
		})

		Convey("With an order", func() {
			filters.Order = "id DESC"
			err := acc.Select(&result, filters)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then it should return all valid entries sorted in the specified order", func() {
					ordered := []testBean{selectBean5, selectBean4, selectBean3,
						selectBean2, selectBean1}
					So(result, ShouldResemble, ordered)
				})
			})
		})

		Convey("With no filters", func() {
			all := []testBean{selectBean1, selectBean2, selectBean3, selectBean4,
				selectBean5}
			err := acc.Select(&result, nil)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then it should return all entries", func() {
					So(result, ShouldResemble, all)
				})
			})
		})

		Convey("With invalid filters", func() {
			err := acc.Select(&result, &Filters{Conditions: builder.In("error", 1)})

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("With a nil result slice", func() {
			err := acc.Select(nil, filters)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, ErrNilRecord)
			})
		})

	}

	Convey("When calling the 'Select' method", func() {

		_, err := db.engine.Insert(&selectBean1, &selectBean2,
			&selectBean3, &selectBean4, &selectBean5)
		So(err, ShouldBeNil)

		Convey("Using the standalone accessor", func() {
			runTests(db)
		})

		Convey("Using the transaction accessor", func() {
			ses, err := db.BeginTransaction()
			So(err, ShouldBeNil)

			Reset(ses.session.Close)

			runTests(ses)
		})
	})

}

func testCreate(db *Db) {
	existingBean := testBean{
		ID:     1,
		String: "existing",
	}
	createBean := testBean{
		ID:     2,
		String: "create",
	}

	runTests := func(acc Accessor) {
		Convey("With a valid record", func() {
			err := db.Create(&createBean)

			Reset(func() { signals = "" })

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then the record should have been inserted", func() {
					exists, err := acc.Exists(&createBean)
					So(err, ShouldBeNil)
					So(exists, ShouldBeTrue)
				})

				Convey("Then the `BeforeInsert` hook should have been called", func() {
					So(signals, ShouldEqual, "insert hook")
				})
			})
		})

		Convey("With a existing record", func() {
			err := acc.Create(&existingBean)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("With an unknown record type", func() {
			err := acc.Create(&struct{}{})

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, xorm.ErrTableNotFound)
			})
		})

		Convey("With a nil record ", func() {
			err := acc.Create(nil)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, ErrNilRecord)
			})
		})
	}

	Convey("When calling the 'Create' method", func() {

		_, err := db.engine.InsertOne(&existingBean)
		So(err, ShouldBeNil)

		Convey("Using the standalone accessor", func() {
			runTests(db)
		})

		Convey("Using the transaction accessor", func() {
			ses, err := db.BeginTransaction()
			So(err, ShouldBeNil)

			Reset(ses.session.Close)

			runTests(ses)
		})
	})
}

func testUpdate(db *Db) {

	updateBeanBefore := testBean{
		ID:     1,
		String: "update",
	}
	updateBeanAfter := testBean{
		ID:     2,
		String: "updated",
	}

	runTests := func(acc Accessor) {
		Convey("With an existing record", func() {
			err := db.Update(&updateBeanAfter, updateBeanBefore.ID, false)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then the new record should be present in the database", func() {
					existsAfter, err := acc.Exists(&updateBeanAfter)
					So(err, ShouldBeNil)
					So(existsAfter, ShouldBeTrue)
				})

				Convey("Then the old record should no longer be present in the database", func() {
					existsBefore, err := acc.Exists(&updateBeanBefore)
					So(err, ShouldBeNil)
					So(existsBefore, ShouldBeFalse)
				})

				Convey("Then the `BeforeUpdate` hook should have been called", func() {
					So(signals, ShouldEqual, "update hook")
				})
			})
		})

		Convey("With an unknown record", func() {
			err := acc.Update(&updateBeanAfter, 1000, false)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, ErrNotFound)
			})
		})

		Convey("With an invalid record type", func() {
			err := acc.Update(&struct{}{}, updateBeanBefore.ID, false)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, xorm.ErrTableNotFound)
			})
		})

		Convey("With an nil record", func() {
			err := acc.Update(nil, updateBeanBefore.ID, false)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, ErrNilRecord)
			})
		})
	}

	Convey("When calling the 'Update' method", func() {
		_, err := db.engine.InsertOne(&updateBeanBefore)
		So(err, ShouldBeNil)

		Convey("Using the standalone accessor", func() {
			runTests(db)
		})

		Convey("Using the transaction accessor", func() {
			ses, err := db.BeginTransaction()
			So(err, ShouldBeNil)

			Reset(ses.session.Close)

			runTests(ses)
		})
	})
}

func testDelete(db *Db) {

	deleteBean := testBean{
		ID:     1,
		String: "delete",
	}

	runTests := func(acc Accessor) {
		Convey("With a valid record", func() {
			err := db.Delete(&deleteBean)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then the record should no longer be present in the database", func() {
					exists, err := acc.Exists(&deleteBean)
					So(err, ShouldBeNil)
					So(exists, ShouldBeFalse)
				})

				Convey("Then the `BeforeDelete` hook should have been called", func() {
					So(signals, ShouldEqual, "delete hook")
				})
			})
		})

		Convey("With an unknown record", func() {
			err := acc.Delete(&testBean{ID: 1000})

			Convey("Then it should not change anything", func() {
				So(err, ShouldBeError, ErrNotFound)
			})
		})

		Convey("With an invalid record type", func() {
			err := acc.Delete(&struct{ Test string }{Test: "test"})

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("With a nil record", func() {
			err := acc.Delete(nil)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, ErrNilRecord)
			})
		})
	}

	Convey("When calling the 'Delete' method", func() {
		_, err := db.engine.InsertOne(&deleteBean)
		So(err, ShouldBeNil)

		Convey("Using the standalone accessor", func() {
			runTests(db)
		})

		Convey("Using the session accessor", func() {
			ses, err := db.BeginTransaction()
			So(err, ShouldBeNil)

			Reset(ses.session.Close)

			runTests(ses)
		})

	})
}

func testExist(db *Db) {
	existBean := testBean{
		ID:     1,
		String: "exists",
	}

	runTests := func(acc Accessor) {
		Convey("With an existing record", func() {
			exists, err := acc.Exists(&existBean)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then it should return true", func() {
					So(exists, ShouldBeTrue)
				})
			})
		})

		Convey("With a non-existing record", func() {
			exists, err := acc.Exists(&testBean{ID: 1000})

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then it should return false", func() {
					So(exists, ShouldBeFalse)
				})
			})
		})

		Convey("With an invalid record type", func() {
			_, err := acc.Exists(&struct{}{})

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, xorm.ErrTableNotFound)
			})
		})

		Convey("With an nil record", func() {
			_, err := acc.Exists(nil)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, ErrNilRecord)
			})
		})
	}

	Convey("When calling the 'Exists' method", func() {
		_, err := db.engine.InsertOne(&existBean)
		So(err, ShouldBeNil)

		Convey("Using the standalone accessor", func() {
			runTests(db)
		})

		Convey("Using the transaction accessor", func() {
			ses, err := db.BeginTransaction()
			So(err, ShouldBeNil)

			Reset(ses.session.Close)

			runTests(ses)
		})
	})
}

func testExecute(db *Db) {

	execBean := testBean{
		ID:     1,
		String: "execute",
	}
	execInsert := builder.Eq{
		"id":     execBean.ID,
		"string": execBean.String,
	}

	runTests := func(acc Accessor) {
		Convey("With a valid SQL `INSERT` command", func() {
			err := db.Execute(builder.Insert(execInsert).Into(tblName))

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then it should have inserted the entry", func() {
					exists, err := acc.Exists(&execBean)
					So(err, ShouldBeNil)
					So(exists, ShouldBeTrue)
				})
			})
		})

		Convey("With an invalid custom SQL command", func() {
			invalid := "SELECT * FROM 'unknown'"
			err := acc.Execute(invalid)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("With an invalid command type", func() {
			invalid := 10
			err := acc.Execute(invalid)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("With a nil SQL command", func() {
			err := acc.Execute(nil)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, xorm.ErrUnSupportedType)
			})
		})
	}

	Convey("When calling the 'Execute' method", func() {

		Convey("Using the standalone accessor", func() {
			runTests(db)
		})

		Convey("Using the transaction accessor", func() {
			ses, err := db.BeginTransaction()
			So(err, ShouldBeNil)

			Reset(ses.session.Close)

			runTests(ses)
		})
	})
}

func testQuery(db *Db) {

	runTests := func(acc Accessor, count func(...interface{}) (int64, error)) {
		Convey("With a valid custom SQL `SELECT` query", func() {
			res, err := db.Query(builder.Select().From(tblName))

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)

				Convey("Then it should execute the command without error", func() {
					count, err := count(&testBean{})
					So(err, ShouldBeNil)
					So(len(res), ShouldEqual, count)
				})
			})
		})

		Convey("With an invalid SQL query", func() {
			invalid := builder.Select().From("unknown")
			_, err := acc.Query(invalid)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("With an invalid query type", func() {
			invalid := 10
			_, err := acc.Query(invalid)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("With a nil SQL query", func() {
			_, err := acc.Query(nil)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, xorm.ErrUnSupportedType)
			})
		})
	}

	Convey("When calling the 'Query' method", func() {

		Convey("Using the standalone accessor", func() {
			runTests(db, db.engine.Count)
		})

		Convey("Using the transaction accessor", func() {
			ses, err := db.BeginTransaction()
			So(err, ShouldBeNil)

			Reset(ses.session.Close)

			runTests(ses, ses.session.Count)
		})
	})
}

func testTrans(db *Db) {
	bean := testBean{
		ID:     1,
		String: "test trans",
	}

	Convey("Given a new transaction", func() {
		ses, err := db.BeginTransaction()
		So(err, ShouldBeNil)

		Convey("Given a pending insertion operation", func() {
			err := ses.Create(&bean)
			So(err, ShouldBeNil)

			Convey("When calling the 'Commit' method", func() {
				err := ses.Commit()

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)

					Convey("Then the new insertion should have been committed", func() {
						exists, err := db.engine.Exist(&bean)
						So(err, ShouldBeNil)
						So(exists, ShouldBeTrue)
					})
				})
			})

			Convey("When calling the 'Rollback' method", func() {
				ses.Rollback()

				Convey("Then the new insertion should NOT have been committed", func() {
					exists, err := db.engine.Exist(&bean)
					So(err, ShouldBeNil)
					So(exists, ShouldBeFalse)
				})
			})
		})
	})
}

func testDatabase(db *Db) {
	Reset(func() {
		_, err := db.engine.Exec("DELETE FROM " + tblName)
		So(err, ShouldBeNil)
	})

	testGet(db)
	testSelect(db)
	testCreate(db)
	testUpdate(db)
	testDelete(db)
	testExist(db)
	testExecute(db)
	testQuery(db)
	testTrans(db)
}

func TestSqlite(t *testing.T) {
	db := sqliteTestDatabase
	if err := db.Start(); err != nil {
		t.Fatal(err)
	}
	if err := db.engine.CreateTables(&testBean{}); err != nil {
		t.Fatal(err)
	}

	Convey("Given a Sqlite service", t, func() {
		testDatabase(db)
	})
}

func TestDatabaseStartWithNoPassPhraseFile(t *testing.T) {
	Convey("Given a test database", t, func() {
		db := GetTestDatabase()
		db.Stop(context.Background())
		os.Remove(db.Conf.Database.AESPassphrase)

		Convey("Given there is no passphrase file", func() {
			_, err := os.Stat(db.Conf.Database.AESPassphrase)
			So(os.IsNotExist(err), ShouldBeTrue)

			Convey("When the database service is started", func() {
				err := db.Start()
				So(err, ShouldBeNil)

				Convey("Then there is a new passphrase file", func() {
					stats, err := os.Stat(db.Conf.Database.AESPassphrase)
					So(err, ShouldBeNil)
					So(stats, ShouldNotBeNil)

					Convey("Then the permissions of the files are secure", func() {
						So(int(stats.Mode().Perm()), ShouldEqual, 0600)
					})
				})
			})
		})
	})
}

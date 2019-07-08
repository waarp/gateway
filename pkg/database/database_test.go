package database

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"github.com/go-xorm/builder"
	"github.com/go-xorm/xorm"
	. "github.com/smartystreets/goconvey/convey"
)

var sqliteTestDatabase *Db

func init() {
	sqliteConfig := &conf.ServerConfig{}
	sqliteConfig.Database.Type = sqlite
	sqliteConfig.Database.Name = "file::memory:?mode=memory&cache=shared"

	sqliteTestDatabase = &Db{Conf: sqliteConfig}
}

func testGet(acc Accessor, env *testEnv) {

	Convey("When calling the 'Get' method", func() {

		Convey("With an existing key", func() {
			err := acc.Get(env.get)

			Convey("Then the parameter should contain the result", func() {
				So(err, ShouldBeNil)
				So(env.get, ShouldResemble, env.getExpected)
			})
		})

		Convey("With an unknown key", func() {
			err := acc.Get(env.fail)

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
	})
}

func testSelect(acc Accessor, env *testEnv) {

	Convey("When calling the 'Select' method", func() {
		filters := &Filters{
			Limit:      2,
			Offset:     1,
			Order:      env.selectOrder,
			Conditions: env.selectCondition,
			Args:       env.selectArgs,
		}

		Convey("With valid filters", func() {
			err := acc.Select(env.selectResult, filters)

			Convey("Then it should return all the valid entries", func() {
				So(err, ShouldBeNil)
				So(env.selectResult, ShouldResemble, env.selectExpected)
			})
		})

		Convey("With nil filters", func() {
			err := acc.Select(env.selectResult, nil)

			Convey("Then it should return all entries", func() {
				So(err, ShouldBeNil)
				expected, err := acc.Query(builder.Select().From(env.table))
				So(err, ShouldBeNil)
				nbRes := reflect.Indirect(reflect.ValueOf(env.selectResult)).Len()
				So(nbRes, ShouldEqual, len(expected))
			})
		})

		Convey("With invalid filters", func() {
			err := acc.Select(env.selectResult, &Filters{Conditions: "error"})

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
	})

}

func createFails(acc Accessor, env *testEnv) {

	Convey("With a existing record", func() {
		err := acc.Create(env.getExpected)

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

func testDbCreate(db *Db, env *testEnv) {

	Convey("When calling the 'Create' method", func() {

		Convey("With a valid record", func() {
			err := db.Create(env.create)

			Convey("Then the record should be inserted without error", func() {
				So(err, ShouldBeNil)
				exists, err := db.engine.Exist(env.create)
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
			})
		})

		createFails(db, env)
	})
}

func testSesCreate(ses *Session, env *testEnv) {
	Convey("When calling the 'Create' method", func() {

		Convey("With a valid record", func() {
			err := ses.Create(env.create)

			Convey("Then the record should be inserted without error", func() {
				So(err, ShouldBeNil)
				err := ses.session.Commit()
				So(err, ShouldBeNil)
				exists, err := ses.session.Exist(env.create)
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
			})
		})

		createFails(ses, env)
	})
}

func updateFails(acc Accessor, env *testEnv) {
	Convey("With an unknown record", func() {
		err := acc.Update(env.fail, env.fail)

		Convey("Then it should return an error", func() {
			So(err, ShouldBeError)
		})
	})

	Convey("With an invalid record type", func() {
		err := acc.Update(&struct{}{}, &struct{}{})

		Convey("Then it should return an error", func() {
			So(err, ShouldBeError, xorm.ErrTableNotFound)
		})
	})

	Convey("With an nil record", func() {
		err := acc.Update(nil, nil)

		Convey("Then it should return an error", func() {
			So(err, ShouldBeError, ErrNilRecord)
		})
	})
}

func testDbUpdate(db *Db, env *testEnv) {

	Convey("When calling the 'Update' method", func() {

		Convey("With an existing record", func() {
			err := db.Update(env.updateBefore, env.updateAfter)

			Convey("Then the record should be updated without error", func() {
				So(err, ShouldBeNil)
				exists, err := db.engine.Exist(env.updateAfter)
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
			})
		})

		updateFails(db, env)
	})
}

func testSesUpdate(ses *Session, env *testEnv) {

	Convey("When calling the 'Update' method", func() {

		Convey("With an existing record", func() {
			err := ses.Update(env.updateBefore, env.updateAfter)

			Convey("Then the record should be updated without error", func() {
				So(err, ShouldBeNil)
				err := ses.session.Commit()
				So(err, ShouldBeNil)
				exists, err := ses.session.Exist(env.updateAfter)
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
			})
		})

		updateFails(ses, env)
	})
}

func deleteFails(acc Accessor, env *testEnv) {

	Convey("With an unknown record", func() {
		err := acc.Delete(env.fail)

		Convey("Then it should not change anything", func() {
			So(err, ShouldBeNil)
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

func testDbDelete(db *Db, env *testEnv) {

	Convey("When calling the 'Delete' method", func() {

		Convey("With a valid record", func() {
			err := db.Delete(env.delete)

			Convey("Then the record should be deleted without error", func() {
				So(err, ShouldBeNil)
				exists, err := db.engine.Exist(env.delete)
				So(err, ShouldBeNil)
				So(exists, ShouldBeFalse)
			})
		})

		deleteFails(db, env)
	})
}

func testSesDelete(ses *Session, env *testEnv) {

	Convey("When calling the 'Delete' method", func() {

		Convey("With a valid record", func() {
			err := ses.Delete(env.delete)

			Convey("Then the record should be deleted without error", func() {
				So(err, ShouldBeNil)
				err := ses.session.Commit()
				So(err, ShouldBeNil)
				exists, err := ses.session.Exist(env.delete)
				So(err, ShouldBeNil)
				So(exists, ShouldBeFalse)
			})
		})

		deleteFails(ses, env)
	})
}

func testExist(acc Accessor, env *testEnv) {

	Convey("When calling the 'Exists' method", func() {

		Convey("With an existing record", func() {
			exists, err := acc.Exists(env.get)

			Convey("Then it should return true", func() {
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
			})
		})

		Convey("With a non-existing record", func() {
			exists, err := acc.Exists(env.fail)

			Convey("Then it should return false", func() {
				So(err, ShouldBeNil)
				So(exists, ShouldBeFalse)
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
	})
}

func execFails(acc Accessor) {

	Convey("With an invalid custom SQL command", func() {
		invalid := "SELECT * FROM 'unknown'"
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

func testDbExecute(db *Db, env *testEnv) {

	Convey("When calling the 'Execute' method", func() {

		Convey("With a valid SQL command", func() {
			err := db.Execute(builder.Insert(env.exec).Into(env.table))

			Convey("Then it should execute the command without error", func() {
				So(err, ShouldBeNil)
				exists, err := db.engine.Exist(env.execExpected)
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
			})
		})

		execFails(db)
	})
}

func testSesExecute(ses *Session, env *testEnv) {

	Convey("When calling the 'Execute' method", func() {

		Convey("With a valid SQL command", func() {
			err := ses.Execute(builder.Insert(env.exec).Into(env.table))

			Convey("Then it should execute the command without error", func() {
				So(err, ShouldBeNil)
				err := ses.session.Commit()
				So(err, ShouldBeNil)
				exists, err := ses.session.Exist(env.execExpected)
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
			})
		})

		execFails(ses)
	})
}

func queryFails(acc Accessor) {
	Convey("With an invalid SQL query", func() {
		invalid := builder.Select().From("unknown")
		res, err := acc.Query(invalid)

		Convey("Then it should return an error", func() {
			So(err, ShouldBeError)
			So(res, ShouldBeNil)
		})
	})

	Convey("With a nil SQL query", func() {
		_, err := acc.Query(nil)

		Convey("Then it should return an error", func() {
			So(err, ShouldBeError, xorm.ErrUnSupportedType)
		})
	})
}

func testDbQuery(db *Db, env *testEnv) {

	Convey("When calling the 'Query' method", func() {

		Convey("With a valid custom SQL query", func() {
			res, err := db.Query(builder.Select().From(env.table))

			Convey("Then it should execute the command without error", func() {
				So(err, ShouldBeNil)
				count, err := db.engine.Count(env.empty)
				So(err, ShouldBeNil)
				So(len(res), ShouldEqual, count)
			})
		})

		queryFails(db)
	})
}

func testSesQuery(ses *Session, env *testEnv) {

	Convey("When calling the 'Query' method", func() {

		Convey("With a valid custom SQL query", func() {
			res, err := ses.Query(builder.Select().From(env.table))

			Convey("Then it should execute the command without error", func() {
				So(err, ShouldBeNil)
				count, err := ses.session.Count(env.empty)
				So(err, ShouldBeNil)
				So(len(res), ShouldEqual, count)
			})
		})

		queryFails(ses)
	})
}

func testCommit(ses *Session, env *testEnv) {

	Convey("When calling the 'Commit' method", func() {
		_, err := ses.session.Insert(env.commit)

		Convey("Then the changes should take effect", func() {
			So(err, ShouldBeNil)
			err := ses.Commit()
			So(err, ShouldBeNil)
			exists, err := ses.session.Exist(env.commit)
			So(err, ShouldBeNil)
			So(exists, ShouldBeTrue)
		})
	})
}

func testRollback(ses *Session, env *testEnv) {

	Convey("When calling the 'Rollback' method", func() {
		_, err := ses.session.Insert(env.commit)

		Convey("Then the changes should not take effect", func() {
			So(err, ShouldBeNil)
			ses.Rollback()
			exists, err := ses.session.Exist(env.commit)
			So(err, ShouldBeNil)
			So(exists, ShouldBeFalse)
		})
	})
}

func testDb(db *Db, env *testEnv) {

	Convey("Using the standalone Accessor", func() {
		testGet(db, env)
		testSelect(db, env)
		testDbCreate(db, env)
		testDbUpdate(db, env)
		testDbDelete(db, env)
		testExist(db, env)
		testDbExecute(db, env)
		testDbQuery(db, env)
	})
}

func testSession(db *Db, env *testEnv) {

	Convey("Using the transaction Accessor", func() {
		ses, err := db.BeginTransaction()
		So(err, ShouldBeNil)

		Reset(ses.session.Close)

		testGet(ses, env)
		testSelect(ses, env)
		testSesCreate(ses, env)
		testSesUpdate(ses, env)
		testSesDelete(ses, env)
		testExist(ses, env)
		testSesExecute(ses, env)
		testSesQuery(ses, env)
		testCommit(ses, env)
		testRollback(ses, env)
	})
}

func setupTable(db *Db, env *testEnv) {
	err := db.engine.CreateTables(env.empty)
	So(err, ShouldBeNil)

	_, err = db.engine.Insert(env.getExpected, env.select1, env.select2, env.select3,
		env.select4, env.select5, env.updateBefore, env.delete)
	So(err, ShouldBeNil)
}

func cleanTable(t *testing.T, db *Db, env *testEnv) func() {
	return func() {
		if err := db.engine.DropTables(env.empty); err != nil {
			t.Fatal(err)
		}
	}
}

func testTable(t *testing.T, db *Db, table string, f func() *testEnv) {

	Convey("Testing the '"+table+"' table", func() {
		env := f()
		setupTable(db, env)

		Reset(cleanTable(t, db, env))

		testDb(db, env)
		testSession(db, env)
	})
}

func testDatabase(t *testing.T, db *Db) {
	testTable(t, db, (&User{}).TableName(), usersTestEnv)
	//testTable(t, db, Partner{}.TableName(), partnersTestEnv)
	//testTable(t, db, Account{}.TableName(), accountsTestEnv)
	//testTable(t, db, CertChain{}.TableName(), certsTestEnv)
}

func cleanDatabase(t *testing.T, db *Db) {
	_ = db.engine.DropTables(&User{})
	//_ = db.engine.DropTables(&Partner{})
	//_ = db.engine.DropTables(&Account{})
	//_ = db.engine.DropTables(&CertChain{})
	_ = db.Stop(context.Background())
	if r := recover(); r != nil {
		t.Fatal(r)
	}
}

func TestSqlite(t *testing.T) {
	start := time.Now()

	db := sqliteTestDatabase
	if err := db.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanDatabase(t, db)
		dur := time.Since(start)
		fmt.Printf("\nSqlite test finished in %s\n", dur)
	}()

	Convey("Given a Sqlite service", t, func() {
		testDatabase(t, db)
	})
}

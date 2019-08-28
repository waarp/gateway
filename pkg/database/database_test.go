package database

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
	"github.com/go-xorm/xorm"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
)

var sqliteTestDatabase *Db

func init() {
	model.BcryptRounds = bcrypt.MinCost

	sqliteConfig := &conf.ServerConfig{}
	sqliteConfig.Database.Type = sqlite
	sqliteConfig.Database.Name = "file::memory:?mode=memory&cache=shared"

	sqliteTestDatabase = &Db{Conf: sqliteConfig}
}

func testGet(db *Db) {
	getTestUser := &model.User{
		Login:    "get",
		Password: []byte("get_password"),
	}

	runTests := func(acc Accessor) {
		Convey("With an existing key", func() {
			result := &model.User{Login: getTestUser.Login}
			err := acc.Get(result)

			Convey("Then the parameter should contain the result", func() {
				So(err, ShouldBeNil)
				So(result, ShouldResemble, getTestUser)
			})
		})

		Convey("With an unknown key", func() {
			err := acc.Get(&model.User{Login: "unknown"})

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
		_, err := db.engine.InsertOne(getTestUser)
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
	selectTestUser1 := &model.User{
		Login:    "select1",
		Password: []byte("select_password"),
	}
	selectTestUser2 := &model.User{
		Login:    "select2",
		Password: selectTestUser1.Password,
	}
	selectTestUser3 := &model.User{
		Login:    "select3",
		Password: selectTestUser1.Password,
	}
	selectTestUser4 := &model.User{
		Login:    "select4",
		Password: selectTestUser1.Password,
	}
	selectTestUser5 := &model.User{
		Login:    "select5",
		Password: selectTestUser1.Password,
	}

	runTests := func(acc Accessor) {
		filters := &Filters{Conditions: builder.In("login", selectTestUser1.Login,
			selectTestUser2.Login, selectTestUser4.Login, selectTestUser5.Login)}
		result := &[]*model.User{}

		Convey("With just a condition", func() {
			filtered := &[]*model.User{selectTestUser1, selectTestUser2,
				selectTestUser4, selectTestUser5}
			err := acc.Select(result, filters)

			Convey("Then it should return all the valid entries", func() {
				So(err, ShouldBeNil)
				So(result, ShouldResemble, filtered)
			})
		})

		Convey("With a condition and a limit", func() {
			filters.Limit = 2
			limited := &[]*model.User{selectTestUser1, selectTestUser2}
			err := acc.Select(result, filters)

			Convey("Then it should return `limit` amount of entries at most", func() {
				So(err, ShouldBeNil)
				So(result, ShouldResemble, limited)
			})
		})

		Convey("With a condition, an offset", func() {
			filters.Limit = 10
			filters.Offset = 1
			offset := &[]*model.User{selectTestUser2, selectTestUser4, selectTestUser5}
			err := acc.Select(result, filters)

			Convey("Then it should return all valid entries except the `offset` first ones", func() {
				So(err, ShouldBeNil)
				So(result, ShouldResemble, offset)
			})
		})

		Convey("With a condition and an order", func() {
			filters.Order = "login DESC"
			ordered := &[]*model.User{selectTestUser5, selectTestUser4, selectTestUser2, selectTestUser1}
			err := acc.Select(result, filters)

			Convey("Then it should return all valid entries sorted in the specified order", func() {
				So(err, ShouldBeNil)
				So(result, ShouldResemble, ordered)
			})
		})

		Convey("With no filters", func() {
			err := acc.Select(result, nil)

			Convey("Then it should return all entries", func() {
				So(err, ShouldBeNil)
				expected, err := acc.Query(builder.Select().From("users"))
				So(err, ShouldBeNil)
				nbRes := reflect.Indirect(reflect.ValueOf(result)).Len()
				So(nbRes, ShouldEqual, len(expected))
			})
		})

		Convey("With invalid filters", func() {
			err := acc.Select(result, &Filters{Conditions: builder.In("error", 1)})

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

		_, err := db.engine.Insert(selectTestUser1, selectTestUser2,
			selectTestUser3, selectTestUser4, selectTestUser5)
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
	createTestUserFail := &model.User{
		Login:    "existing",
		Password: []byte("create_fail_password"),
	}
	createTestUserSuccess := &model.User{
		Login:    "create",
		Password: []byte("create_success_password"),
	}

	runTests := func(acc Accessor) {
		Convey("With a valid record", func() {
			err := db.Create(createTestUserSuccess)

			Convey("Then the record should be inserted without error", func() {
				So(err, ShouldBeNil)
				exists, err := acc.Exists(createTestUserSuccess)
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
			})
		})

		Convey("With a existing record", func() {
			err := acc.Create(createTestUserFail)

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

		_, err := db.engine.InsertOne(createTestUserFail)
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

	updateTestUserBefore := &model.User{
		Login:    "update",
		Password: []byte("update_password"),
	}
	updateTestUserAfter := &model.User{
		Login:    "updated",
		Password: []byte("updated_password"),
	}

	runTests := func(acc Accessor) {
		Convey("With an existing record", func() {
			err := db.Update(updateTestUserBefore, updateTestUserAfter)

			Convey("Then the record should be updated without error", func() {
				So(err, ShouldBeNil)

				existsAfter, err := acc.Exists(updateTestUserAfter)
				So(err, ShouldBeNil)
				So(existsAfter, ShouldBeTrue)

				existsBefore, err := acc.Exists(updateTestUserBefore)
				So(err, ShouldBeNil)
				So(existsBefore, ShouldBeFalse)
			})
		})

		Convey("With an unknown record", func() {
			err := acc.Update(&model.User{Login: "unknown"}, &model.User{})

			Convey("Then it should do nothing", func() {
				So(err, ShouldBeNil)

				existsAfter, err := acc.Exists(updateTestUserAfter)
				So(err, ShouldBeNil)
				So(existsAfter, ShouldBeFalse)

				existsBefore, err := acc.Exists(updateTestUserBefore)
				So(err, ShouldBeNil)
				So(existsBefore, ShouldBeTrue)
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

	Convey("When calling the 'Update' method", func() {
		_, err := db.engine.InsertOne(updateTestUserBefore)
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

	deleteTestUser := &model.User{
		Login:    "delete",
		Password: []byte("delete_password"),
	}

	runTests := func(acc Accessor) {
		Convey("With a valid record", func() {
			err := db.Delete(deleteTestUser)

			Convey("Then the record should be deleted without error", func() {
				So(err, ShouldBeNil)
				exists, err := acc.Exists(deleteTestUser)
				So(err, ShouldBeNil)
				So(exists, ShouldBeFalse)
			})
		})

		Convey("With an unknown record", func() {
			err := acc.Delete(&model.User{Login: "unknown"})

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

	Convey("When calling the 'Delete' method", func() {
		_, err := db.engine.InsertOne(deleteTestUser)
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
	existTestUser := &model.User{
		Login:    "exists",
		Password: []byte("exists_password"),
	}

	runTests := func(acc Accessor) {
		Convey("With an existing record", func() {
			exists, err := acc.Exists(existTestUser)

			Convey("Then it should return true", func() {
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
			})
		})

		Convey("With a non-existing record", func() {
			exists, err := acc.Exists(&model.User{Login: "unknown"})

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
	}

	Convey("When calling the 'Exists' method", func() {
		_, err := db.engine.InsertOne(existTestUser)
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

	execTestUser := &model.User{
		Login:    "execute",
		Password: []byte("execute_password"),
	}
	execInsert := builder.Eq{
		"login":    "execute",
		"password": []byte("execute_password"),
	}

	runTests := func(acc Accessor) {
		Convey("With a valid SQL command", func() {
			err := db.Execute(builder.Insert(execInsert).Into("users"))

			Convey("Then it should execute the command without error", func() {
				So(err, ShouldBeNil)
				exists, err := acc.Exists(execTestUser)
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
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
		Convey("With a valid custom SQL query", func() {
			res, err := db.Query(builder.Select().From("users"))

			Convey("Then it should execute the command without error", func() {
				So(err, ShouldBeNil)
				count, err := count(&model.User{})
				So(err, ShouldBeNil)
				So(len(res), ShouldEqual, count)
			})
		})

		Convey("With an invalid SQL query", func() {
			invalid := builder.Select().From("unknown")
			res, err := acc.Query(invalid)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
				So(res, ShouldBeNil)
			})
		})

		Convey("With an invalid query type", func() {
			invalid := 10
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

func testCommit(db *Db) {
	commitTestUser := &model.User{
		Login:    "commit",
		Password: []byte("commit_password"),
	}

	Convey("When calling the 'Commit' method", func() {
		ses, err := db.BeginTransaction()
		So(err, ShouldBeNil)

		Reset(ses.session.Close)

		Convey("Using the transaction accessor", func() {
			_, err := ses.session.Insert(commitTestUser)

			Convey("Then the changes should take effect", func() {
				So(err, ShouldBeNil)
				err := ses.Commit()
				So(err, ShouldBeNil)
				exists, err := db.engine.Exist(commitTestUser)
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
			})
		})
	})
}

func testRollback(db *Db) {
	rollbackTestUser := &model.User{
		Login:    "rollback",
		Password: []byte("rollback_password"),
	}

	Convey("When calling the 'Rollback' method", func() {
		ses, err := db.BeginTransaction()
		So(err, ShouldBeNil)

		Reset(ses.session.Close)

		Convey("Using the transaction accessor", func() {
			_, err := ses.session.Insert(rollbackTestUser)

			Convey("Then the changes should be dropped", func() {
				So(err, ShouldBeNil)
				ses.Rollback()
				exists, err := db.engine.Exist(rollbackTestUser)
				So(err, ShouldBeNil)
				So(exists, ShouldBeFalse)
			})
		})
	})
}

func testDatabase(db *Db) {
	testGet(db)
	testSelect(db)
	testCreate(db)
	testUpdate(db)
	testDelete(db)
	testExist(db)
	testExecute(db)
	testQuery(db)
	testCommit(db)
	testRollback(db)
}

func cleanDatabase(t *testing.T, db *Db) {
	for _, table := range model.Tables {
		_ = db.engine.DropTables(table)
	}
	_ = db.Stop(context.Background())
	if r := recover(); r != nil {
		fmt.Println(r)
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

		Reset(func() {
			_, err := db.engine.Exec("DELETE FROM 'users'")
			So(err, ShouldBeNil)
		})

		testDatabase(db)
	})
}

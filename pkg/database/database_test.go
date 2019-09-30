package database

import (
	"reflect"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"github.com/go-xorm/builder"
	"github.com/go-xorm/xorm"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
)

const tblName = "test"

var sqliteTestDatabase *Db

type testBean struct {
	StrPK string `xorm:"pk 'str_pk'"`
	ByteA []byte `xorm:"notnull 'bytea'"`
}

func (*testBean) TableName() string {
	return tblName
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
		StrPK: "get",
		ByteA: []byte("get"),
	}

	runTests := func(acc Accessor) {
		Convey("With an existing key", func() {
			result := &testBean{StrPK: getBean.StrPK}
			err := acc.Get(result)

			Convey("Then the parameter should contain the result", func() {
				So(err, ShouldBeNil)
				So(result, ShouldResemble, getBean)
			})
		})

		Convey("With an unknown key", func() {
			err := acc.Get(&testBean{StrPK: "unknown"})

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
	selectBean1 := &testBean{
		StrPK: "select1",
		ByteA: []byte("select"),
	}
	selectBean2 := &testBean{
		StrPK: "select2",
		ByteA: selectBean1.ByteA,
	}
	selectBean3 := &testBean{
		StrPK: "select3",
		ByteA: selectBean1.ByteA,
	}
	selectBean4 := &testBean{
		StrPK: "select4",
		ByteA: selectBean1.ByteA,
	}
	selectBean5 := &testBean{
		StrPK: "select5",
		ByteA: selectBean1.ByteA,
	}

	runTests := func(acc Accessor) {
		filters := &Filters{Conditions: builder.In("str_pk", selectBean1.StrPK,
			selectBean2.StrPK, selectBean4.StrPK, selectBean5.StrPK)}
		result := &[]*testBean{}

		Convey("With just a condition", func() {
			filtered := &[]*testBean{selectBean1, selectBean2,
				selectBean4, selectBean5}
			err := acc.Select(result, filters)

			Convey("Then it should return all the valid entries", func() {
				So(err, ShouldBeNil)
				So(result, ShouldResemble, filtered)
			})
		})

		Convey("With a condition and a limit", func() {
			filters.Limit = 2
			limited := &[]*testBean{selectBean1, selectBean2}
			err := acc.Select(result, filters)

			Convey("Then it should return `limit` amount of entries at most", func() {
				So(err, ShouldBeNil)
				So(result, ShouldResemble, limited)
			})
		})

		Convey("With a condition, an offset", func() {
			filters.Limit = 10
			filters.Offset = 1
			offset := &[]*testBean{selectBean2, selectBean4, selectBean5}
			err := acc.Select(result, filters)

			Convey("Then it should return all valid entries except the `offset` first ones", func() {
				So(err, ShouldBeNil)
				So(result, ShouldResemble, offset)
			})
		})

		Convey("With a condition and an order", func() {
			filters.Order = "str_pk DESC"
			ordered := &[]*testBean{selectBean5, selectBean4, selectBean2, selectBean1}
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
				expected, err := acc.Query(builder.Select().From(tblName))
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

		_, err := db.engine.Insert(selectBean1, selectBean2,
			selectBean3, selectBean4, selectBean5)
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
	existingBean := &testBean{
		StrPK: "existing",
		ByteA: []byte("existing"),
	}
	createBean := &testBean{
		StrPK: "create",
		ByteA: []byte("create"),
	}

	runTests := func(acc Accessor) {
		Convey("With a valid record", func() {
			err := db.Create(createBean)

			Convey("Then the record should be inserted without error", func() {
				So(err, ShouldBeNil)
				exists, err := acc.Exists(createBean)
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
			})
		})

		Convey("With a existing record", func() {
			err := acc.Create(existingBean)

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

		_, err := db.engine.InsertOne(existingBean)
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

	updateBeanBefore := &testBean{
		StrPK: "update",
		ByteA: []byte("update"),
	}
	updateBeanAfter := &testBean{
		StrPK: "updated",
		ByteA: []byte("updated"),
	}

	runTests := func(acc Accessor) {
		Convey("With an existing record", func() {
			err := db.Update(updateBeanBefore, updateBeanAfter)

			Convey("Then the record should be updated without error", func() {
				So(err, ShouldBeNil)

				existsAfter, err := acc.Exists(updateBeanAfter)
				So(err, ShouldBeNil)
				So(existsAfter, ShouldBeTrue)

				existsBefore, err := acc.Exists(updateBeanBefore)
				So(err, ShouldBeNil)
				So(existsBefore, ShouldBeFalse)
			})
		})

		Convey("With an unknown record", func() {
			err := acc.Update(&testBean{StrPK: "unknown"}, updateBeanAfter)

			Convey("Then it should do nothing", func() {
				So(err, ShouldBeNil)

				existsAfter, err := acc.Exists(updateBeanAfter)
				So(err, ShouldBeNil)
				So(existsAfter, ShouldBeFalse)

				existsBefore, err := acc.Exists(updateBeanBefore)
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
		_, err := db.engine.InsertOne(updateBeanBefore)
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

	deleteBean := &testBean{
		StrPK: "delete",
		ByteA: []byte("delete"),
	}

	runTests := func(acc Accessor) {
		Convey("With a valid record", func() {
			err := db.Delete(deleteBean)

			Convey("Then the record should be deleted without error", func() {
				So(err, ShouldBeNil)
				exists, err := acc.Exists(deleteBean)
				So(err, ShouldBeNil)
				So(exists, ShouldBeFalse)
			})
		})

		Convey("With an unknown record", func() {
			err := acc.Delete(&testBean{StrPK: "unknown"})

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
		_, err := db.engine.InsertOne(deleteBean)
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
	existBean := &testBean{
		StrPK: "exists",
		ByteA: []byte("exists"),
	}

	runTests := func(acc Accessor) {
		Convey("With an existing record", func() {
			exists, err := acc.Exists(existBean)

			Convey("Then it should return true", func() {
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
			})
		})

		Convey("With a non-existing record", func() {
			exists, err := acc.Exists(&testBean{StrPK: "unknown"})

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
		_, err := db.engine.InsertOne(existBean)
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

	execBean := &testBean{
		StrPK: "execute",
		ByteA: []byte("execute"),
	}
	execInsert := builder.Eq{
		"str_pk": execBean.StrPK,
		"bytea":  execBean.ByteA,
	}

	runTests := func(acc Accessor) {
		Convey("With a valid SQL command", func() {
			err := db.Execute(builder.Insert(execInsert).Into(tblName))

			Convey("Then it should execute the command without error", func() {
				So(err, ShouldBeNil)
				exists, err := acc.Exists(execBean)
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
			res, err := db.Query(builder.Select().From(tblName))

			Convey("Then it should execute the command without error", func() {
				So(err, ShouldBeNil)
				count, err := count(&testBean{})
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
	commitBean := &testBean{
		StrPK: "commit",
		ByteA: []byte("commit"),
	}

	Convey("When calling the 'Commit' method", func() {
		ses, err := db.BeginTransaction()
		So(err, ShouldBeNil)

		Reset(ses.session.Close)

		Convey("Using the transaction accessor", func() {
			_, err := ses.session.Insert(commitBean)

			Convey("Then the changes should take effect", func() {
				So(err, ShouldBeNil)
				err := ses.Commit()
				So(err, ShouldBeNil)
				exists, err := db.engine.Exist(commitBean)
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
			})
		})
	})
}

func testRollback(db *Db) {
	rollbackBean := &testBean{
		StrPK: "rollback",
		ByteA: []byte("rollback"),
	}

	Convey("When calling the 'Rollback' method", func() {
		ses, err := db.BeginTransaction()
		So(err, ShouldBeNil)

		Reset(ses.session.Close)

		Convey("Using the transaction accessor", func() {
			_, err := ses.session.Insert(rollbackBean)

			Convey("Then the changes should be dropped", func() {
				So(err, ShouldBeNil)
				ses.Rollback()
				exists, err := db.engine.Exist(rollbackBean)
				So(err, ShouldBeNil)
				So(exists, ShouldBeFalse)
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
	testCommit(db)
	testRollback(db)
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

package database

import (
	"os"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	errWriteHook  = NewValidationError("write hook failed")
	errDeleteHook = NewValidationError("delete hook failed")
)

type testValid struct {
	ID     int64
	String string
	hooks  string
}

func (*testValid) TableName() string   { return "test_valid" }
func (*testValid) Appellation() string { return "test struct" }
func (t *testValid) GetID() int64      { return t.ID }

func (t *testValid) BeforeWrite(Access) error {
	t.hooks = "write hook"

	return nil
}

func (t *testValid) BeforeDelete(Access) error {
	t.hooks = "delete hook"

	return nil
}

type validList []*testValid

func (*validList) TableName() string { return "test_valid" }
func (*validList) Elem() string      { return "test struct" }

type testValid2 struct {
	ID     int64
	String string
	hooks  string
}

func (*testValid2) TableName() string   { return "test_valid_2" }
func (*testValid2) Appellation() string { return "test valid 2" }
func (t *testValid2) GetID() int64      { return t.ID }

func (t *testValid2) BeforeWrite(Access) error {
	t.hooks = "write hook"

	return nil
}

func (t *testValid2) BeforeDelete(Access) error {
	t.hooks = "delete hook"

	return nil
}

type testWriteFail struct {
	ID    int64
	Hooks string `gorm:"-"`
}

func (*testWriteFail) TableName() string   { return "test_write_fail" }
func (*testWriteFail) Appellation() string { return "test write fail" }
func (t *testWriteFail) GetID() int64      { return t.ID }

func (t *testWriteFail) BeforeWrite(Access) error {
	t.Hooks = "write hook"

	return errWriteHook
}

type testDeleteFail struct {
	ID    int64
	hooks string
}

func (*testDeleteFail) TableName() string   { return "test_delete_fail" }
func (*testDeleteFail) Appellation() string { return "test delete fail" }
func (t *testDeleteFail) GetID() int64      { return t.ID }

func (t *testDeleteFail) BeforeWrite(Access) error {
	t.hooks = "write hook"

	return nil
}

func (t *testDeleteFail) BeforeDelete(db Access) error {
	t.hooks = "delete hook"

	convey.So(db.Insert(&testDeleteFail{ID: 1000}).Run(), convey.ShouldBeNil)

	return errDeleteHook
}

func newGormDB(tb testing.TB) *gorm.DB {
	tb.Helper()

	config := conf.ServerConfig{}
	config.Database.Name = tb.Name()
	dialector, err := memDBInfo(&config.Database)
	require.NoError(tb, err)

	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(tb, err)

	registerAEAD(db, testAEAD)

	return db
}

func newGormPostgresDB(tb testing.TB) *gorm.DB {
	tb.Helper()

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		tb.Skip("POSTGRES_DSN not set")
	}

	dialector := postgres.Open(dsn)
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(tb, err)

	tb.Cleanup(func() {
		require.NoError(tb, db.Exec("DROP SCHEMA public CASCADE").Error)
		require.NoError(tb, db.Exec("CREATE SCHEMA public").Error)
	})

	registerAEAD(db, testAEAD)

	return db
}

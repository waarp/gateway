package migrations

import (
	"database/sql"
	"fmt"
	"testing"

	"code.waarp.fr/lib/migration"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

type testEngine struct {
	*migration.Engine
	DB *sql.DB
}

func (t *testEngine) makeMigration(scripts []script) migration.Migration {
	mig := make(migration.Migration, len(scripts))

	for i := range scripts {
		mig[i] = migration.Script{
			Description: fmt.Sprintf("%T", scripts[i]),
			Up:          scripts[i].Up,
			Down:        scripts[i].Down,
		}
	}

	return mig
}

func (t *testEngine) Upgrade(scripts ...script) error {
	toApply := t.makeMigration(scripts)

	return t.Engine.Upgrade(toApply) //nolint:wrapcheck //this is just for tests
}

func (t *testEngine) Downgrade(scripts ...script) error {
	toApply := t.makeMigration(scripts)

	return t.Engine.Downgrade(toApply) //nolint:wrapcheck //this is just for tests
}

func getSQLiteEngine(c C) *testEngine {
	logger := testhelpers.TestLogger(c, "test_sqlite_engine")
	db := testhelpers.GetTestSqliteDB(c)

	_, err := db.Exec(SqliteCreationScript)
	So(err, ShouldBeNil)

	eng, err := migration.NewEngine(db, migration.SQLite, logger, nil)
	So(err, ShouldBeNil)

	return &testEngine{Engine: eng, DB: db}
}

func TestSQLiteCreationScript(t *testing.T) {
	Convey("Given a SQLite database", t, func(c C) {
		db := testhelpers.GetTestSqliteDB(c)

		Convey("Given the change to initialize version 0.0.0 of the database", func() {
			Convey("When executing the change", func() {
				_, err := db.Exec(SqliteCreationScript)

				Convey("Then it should not return an error", func() {
					So(err, ShouldBeNil)
				})
			})
		})
	})
}

func testMigrations(eng *testEngine, dbType string) {
	// 0.4.2
	testVer0_4_2RemoveHistoryRemoteIDUnique(eng, dbType)

	// 0.5.0
	testVer0_5_0RemoveRulePathSlash(eng, dbType)
	testVer0_5_0CheckRulePathAncestor(eng, dbType)
	testVer0_5_0LocalAgentChangePaths(eng)
	testVer0_5_0LocalAgentsPathsRename(eng)
	testVer0_5_0LocalAgentDisallowReservedNames(eng)
	testVer0_5_0RuleNewPathCols(eng)
	testVer0_5_0RulePathChanges(eng)
	testVer0_5_0AddFilesize(eng)
	testVer0_5_0TransferChangePaths(eng)
	testVer0_5_0TransferFormatLocalPath(eng)
	testVer0_5_0HistoryChangePaths(eng)
	testVer0_5_0LocalAccountsPasswordDecode(eng)
	testVer0_5_0UserPasswordChange(eng, dbType)

	// 0.5.2
	testVer0_5_2FillRemoteTransferID(eng)

	// 0.6.0
	testVer0_6_0AddTransferInfoIsHistory(eng)
}

func TestSQLiteMigrations(t *testing.T) {
	Convey("Given an un-migrated SQLite database engine", t, func(c C) {
		testMigrations(getSQLiteEngine(c), migration.SQLite)
	})
}

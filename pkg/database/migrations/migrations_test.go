package migrations

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func listTables(tb testing.TB, eng *testEngine) []string {
	tb.Helper()

	var (
		rows *sql.Rows
		err  error
	)

	switch eng.Dialect {
	case PostgreSQL:
		rows, err = eng.DB.Query(`SELECT tablename FROM pg_catalog.pg_tables
				WHERE schemaname != 'pg_catalog' AND schemaname != 'information_schema'`)
	case SQLite:
		rows, err = eng.DB.Query(`SELECT name FROM sqlite_master
				WHERE type='table' AND name != 'sqlite_sequence'`)
	case MySQL:
		rows, err = eng.DB.Query(`SHOW TABLES`)
	default:
		tb.Fatalf("Unsupported database type: %s", eng.Dialect)
	}

	require.NoError(tb, err)

	defer rows.Close()

	var tables []string

	for rows.Next() {
		var table string

		require.NoError(tb, rows.Scan(&table))

		tables = append(tables, table)
	}

	require.NoError(tb, rows.Err())

	return tables
}

func checkTableEmpty(tb testing.TB, db *sql.DB, table string) {
	tb.Helper()

	var count int64

	require.NoError(tb, db.QueryRow(`SELECT COUNT(*) FROM `+table).Scan(&count))
	require.Zerof(tb, count,
		"Table %s was expected to be empty, but wasn't "+
			"(most likely a migration test did not clean after itself)", table)
}

// testMigrations tests all the migrations in order. As such, be careful that
// tests are run in the same order as the migration list. Once done, the test
// should also return the migration tested, so that it can be applied for the
// next test. Each test should also return the database to its original (empty)
// state once it's done. If it doesn't, the "apply" function will fail the test.
//
//nolint:thelper //this is NOT a helper function, t.Helper() should NOT be called here
func testMigrations(t *testing.T, eng *testEngine) {
	apply := func(change Change) {
		tables := listTables(t, eng)
		for _, table := range tables {
			if table != "version" {
				checkTableEmpty(t, eng.DB, table)
			}
		}

		require.NoError(t, eng.Upgrade(change))
	}

	// 0.4.0
	apply(testVer0_4_0InitDatabase(t, eng))

	// 0.4.2
	apply(testVer0_4_2RemoveHistoryRemoteIDUnique(t, eng))

	// 0.5.0
	apply(testVer0_5_0RemoveRulePathSlash(t, eng))
	apply(testVer0_5_0CheckRulePathAncestor(t, eng))
	apply(testVer0_5_0LocalAgentChangePaths(t, eng))
	apply(testVer0_5_0LocalAgentsPathsRename(t, eng))
	apply(testVer0_5_0LocalAgentDisallowReservedNames(t, eng))
	apply(testVer0_5_0RuleNewPathCols(t, eng))
	apply(testVer0_5_0RulePathChanges(t, eng))
	apply(testVer0_5_0AddFilesize(t, eng))
	apply(testVer0_5_0TransferChangePaths(t, eng))
	apply(testVer0_5_0TransferFormatLocalPath(t, eng))
	apply(testVer0_5_0HistoryChangePaths(t, eng))
	apply(testVer0_5_0LocalAccountsPasswordDecode(t, eng))
	apply(testVer0_5_0UserPasswordChange(t, eng))

	// 0.5.2
	apply(testVer0_5_2FillRemoteTransferID(t, eng))

	// 0.6.0
	apply(testVer0_6_0AddTransferInfoIsHistory(t, eng))

	// 0.7.0
	apply(testVer0_7_0AddLocalAgentEnabled(t, eng))
	apply(testVer0_7_0RevampUsersTable(t, eng))
	apply(testVer0_7_0RevampLocalAgentTable(t, eng))
	apply(testVer0_7_0RevampRemoteAgentTable(t, eng))
	apply(testVer0_7_0RevampLocalAccountsTable(t, eng))
	apply(testVer0_7_0RevampRemoteAccountsTable(t, eng))
	apply(testVer0_7_0RevampRulesTable(t, eng))
	apply(testVer0_7_0RevampTasksTable(t, eng))
	apply(testVer0_7_0RevampHistoryTable(t, eng))
	apply(testVer0_7_0RevampTransfersTable(t, eng))
	apply(testVer0_7_0RevampTransferInfoTable(t, eng))
	apply(testVer0_7_0RevampCryptoTable(t, eng))
	apply(testVer0_7_0RevampRuleAccessTable(t, eng))
	apply(testVer0_7_0AddLocalAgentsAddressUnique(t, eng))
	apply(testVer0_7_0AddNormalizedTransfersView(t, eng))

	// 0.7.5
	apply(testVer0_7_5SplitR66TLS(t, eng))

	// 0.8.0
	apply(testVer0_8_0DropNormalizedTransfersView(t, eng))
	apply(testVer0_8_0AddTransferFilename(t, eng))
	apply(testVer0_8_0AddHistoryFilename(t, eng))
	apply(testVer0_8_0UpdateNormalizedTransfersView(t, eng))

	// 0.9.0
	apply(testVer0_9_0AddCloudInstances(t, eng))
	apply(testVer0_9_0LocalPathToURL(t, eng))
	apply(testVer0_9_0FixLocalServerEnabled(t, eng))
	apply(testVer0_9_0AddClientsTable(t, eng))
	apply(testVer0_9_0AddRemoteAgentOwner(t, eng))
	apply(testVer0_9_0DuplicateRemoteAgents(t, eng))
	apply(testVer0_9_0RelinkTransfers(t, eng))
	apply(testVer0_9_0AddTransfersClientID(t, eng))
	apply(testVer0_9_0AddHistoryClient(t, eng))
	apply(testVer0_9_0AddNormalizedTransfersView(t, eng))
	apply(testVer0_9_0AddCredTable(t, eng))
	apply(testVer0_9_0FillCredTable(t, eng))
	apply(testVer0_9_0RemoveOldCreds(t, eng))
	apply(testVer0_9_0MoveR66ServerCreds(t, eng))
	apply(testVer0_9_0AddAuthorities(t, eng))

	// 0.10.0
	apply(testVer0_10_0AddSNMPMonitors(t, eng))
	apply(testVer0_10_0AddLocalAccountIPAddr(t, eng))
	apply(testVer0_10_0AddTransferIndex(t, eng))

	// 0.11.0
	apply(testVer0_11_0AddSNMPServerConf(t, eng))

	// 0.12.0
	apply(testVer0_12_0AddCryptoKeys(t, eng))
	apply(testVer0_12_0DropRemoteTransferIdUnique(t, eng))

	// 0.12.1
	apply(testVer0_12_0AddCryptoKeysOwner(t, eng))

	// 0.13.0
	apply(testVer0_13_0AddTransferAutoResume(t, eng))
	apply(testVer0_13_0AddClientAutoResume(t, eng))
}

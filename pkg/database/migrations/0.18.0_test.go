package migrations

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readTransfersView returns the columns of the "normalized_transfers" view and
// its rows ordered by id, every value as the database driver writes it.
func readTransfersView(tb testing.TB, eng *testEngine) ([]string, [][]sql.NullString) {
	tb.Helper()

	rows, queryErr := eng.DB.Query(`SELECT * FROM normalized_transfers ORDER BY id`)
	require.NoError(tb, queryErr)

	defer rows.Close()

	columns, colErr := rows.Columns()
	require.NoError(tb, colErr)

	var content [][]sql.NullString

	for rows.Next() {
		row := make([]sql.NullString, len(columns))
		dest := make([]any, len(columns))

		for i := range row {
			dest[i] = &row[i]
		}

		require.NoError(tb, rows.Scan(dest...))

		content = append(content, row)
	}

	require.NoError(tb, rows.Err())

	return columns, content
}

// transfersViewIsUnionAll says whether the database holds the
// "normalized_transfers" view as a UNION ALL of its two halves.
func transfersViewIsUnionAll(tb testing.TB, eng *testEngine) bool {
	tb.Helper()

	var row *sql.Row

	switch eng.Dialect {
	case SQLite:
		row = eng.DB.QueryRow(`SELECT sql FROM sqlite_master
			WHERE type='view' AND name='normalized_transfers'`)
	case PostgreSQL:
		row = eng.DB.QueryRow(`SELECT pg_get_viewdef('normalized_transfers'::regclass)`)
	case MySQL:
		row = eng.DB.QueryRow(`SELECT VIEW_DEFINITION FROM information_schema.VIEWS
			WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='normalized_transfers'`)
	default:
		tb.Fatalf("Unsupported database type: %s", eng.Dialect)
	}

	var definition string

	require.NoError(tb, row.Scan(&definition))
	require.Contains(tb, strings.ToUpper(definition), "UNION",
		"The view should put its two halves together")

	return strings.Contains(strings.ToUpper(definition), "UNION ALL")
}

func testVer0_18_0TransfersViewUnionAll(t *testing.T, eng *testEngine) Change {
	mig := Migrations[68]

	t.Run("When applying the 0.18.0 transfers view change", func(t *testing.T) {
		// Running transfers, as a server and as a client, and finished ones.
		eng.NoError(t, `INSERT INTO clients(id, owner, name, protocol)
			VALUES (2222, 'bbb', 'sftp', 'sftp')`)
		eng.NoError(t, `INSERT INTO rules(id, name, is_send, path)
			VALUES (1, 'push', TRUE, '/push'), (2, 'pull', FALSE, '/pull')`)
		eng.NoError(t, `INSERT INTO local_agents(id,owner,name,protocol,address)
			VALUES (10, 'aaa', 'sftp_serv', 'sftp', '1.1.1.1:1111')`)
		eng.NoError(t, `INSERT INTO local_accounts(id,local_agent_id,login)
			VALUES (100, 10, 'toto')`)
		eng.NoError(t, `INSERT INTO remote_agents(id,owner,name,protocol,address)
			VALUES (20, 'bbb', 'sftp_part', 'sftp', '2.2.2.2:2222')`)
		eng.NoError(t, `INSERT INTO remote_accounts(id,remote_agent_id,login)
			VALUES (200, 20, 'tata')`)
		eng.NoError(t, `INSERT INTO transfers(id,owner,remote_transfer_id,rule_id,
				client_id,local_account_id,remote_account_id,src_filename,dest_filename)
			VALUES (1000, 'aaa', 'abcd', 1, NULL, 100, NULL, '/src/1', '/dst/1'),
				(2000, 'bbb', 'efgh', 2, 2222, NULL, 200, '/src/2', '/dst/2')`)
		eng.NoError(t, `INSERT INTO transfer_history(id,owner,is_server,
				is_send,remote_transfer_id,rule,client,account,agent,protocol,
				src_filename,dest_filename,filesize,start,stop,status,step)
			VALUES (3000,'ccc',FALSE,TRUE,'xyz','push','r66_client','tutu',
					'r66_part','r66','/src/3','/dst/3',123,'2021-01-03 01:00:00',
					'2021-01-03 02:00:00','CANCELLED','StepData'),
				(4000,'ccc',TRUE,FALSE,'uvw','pull','','titi',
					'r66_serv','r66','/src/4','/dst/4',456,'2021-01-04 01:00:00',
					'2021-01-04 02:00:00','DONE','StepNone')`)

		t.Cleanup(func() {
			eng.NoError(t, `DELETE FROM transfer_history`)
			eng.NoError(t, `DELETE FROM transfers`)
			eng.NoError(t, `DELETE FROM remote_accounts`)
			eng.NoError(t, `DELETE FROM remote_agents`)
			eng.NoError(t, `DELETE FROM local_accounts`)
			eng.NoError(t, `DELETE FROM local_agents`)
			eng.NoError(t, `DELETE FROM rules`)
			eng.NoError(t, `DELETE FROM clients`)
		})

		columnsBefore, rowsBefore := readTransfersView(t, eng)

		require.Len(t, rowsBefore, 4, "The view should give the four transfers")
		require.False(t, transfersViewIsUnionAll(t, eng),
			"The view should be a UNION before the migration")

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then the view should be a UNION ALL", func(t *testing.T) {
			assert.True(t, transfersViewIsUnionAll(t, eng))
		})

		t.Run("Then the view should give the same columns and the same rows", func(t *testing.T) {
			columnsAfter, rowsAfter := readTransfersView(t, eng)

			assert.Equal(t, columnsBefore, columnsAfter)
			assert.Equal(t, rowsBefore, rowsAfter)
		})

		t.Run("Then the view should give no transfer twice", func(t *testing.T) {
			var total, distinct int

			require.NoError(t, eng.DB.QueryRow(`SELECT COUNT(*), COUNT(DISTINCT id)
				FROM normalized_transfers`).Scan(&total, &distinct))
			assert.Equal(t, 4, total)
			assert.Equal(t, total, distinct)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig), "Reverting the migration should not fail")

			t.Run("Then the view should be a UNION again", func(t *testing.T) {
				assert.False(t, transfersViewIsUnionAll(t, eng))
			})

			t.Run("Then the view should give the same columns and the same rows", func(t *testing.T) {
				columnsAfter, rowsAfter := readTransfersView(t, eng)

				assert.Equal(t, columnsBefore, columnsAfter)
				assert.Equal(t, rowsBefore, rowsAfter)
			})
		})
	})

	return mig
}

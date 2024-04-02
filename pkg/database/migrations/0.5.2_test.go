package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_5_2FillRemoteTransferID(t *testing.T, eng *testEngine) Change {
	mig := Migrations[15]

	t.Run("When applying the 0.5.2 remote transfer id change", func(t *testing.T) {
		eng.NoError(t, `INSERT INTO transfers (id,owner,remote_transfer_id,
        	is_server,rule_id,agent_id,account_id,local_path,remote_path,filesize,
			start,status,step,progression,task_number,error_code,error_details) VALUES
			(1234,'', '', false,1,0,0,'/loc_path1','/rem_path1',1111,'2000-01-02 04:05:06 -7:00',
			 	'','',0,0,'',''),
			(5678,'', 'ABCD', true,1,0,0,'/loc_path2','/rem_path2',2222,'2000-01-03 04:05:06 -7:00',
			 	'','',0,0,'','')`)

		t.Cleanup(func() {
			eng.NoError(t, `DELETE FROM transfers`)
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have filled the remote transfer id", func(t *testing.T) {
			rows, err := eng.DB.Query(`SELECT id,remote_transfer_id 
				FROM transfers ORDER BY id`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				id       int
				remoteID string
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &remoteID))
			assert.Equal(t, 1234, id)
			assert.Equal(t, "1234", remoteID)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &remoteID))
			assert.Equal(t, 5678, id)
			assert.Equal(t, "ABCD", remoteID)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have re-emptied the remote transfer id", func(t *testing.T) {
				rows, err := eng.DB.Query(`SELECT id,remote_transfer_id 
					FROM transfers ORDER BY id`)
				require.NoError(t, err)

				defer rows.Close()

				var (
					id       int
					remoteID string
				)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &remoteID))
				assert.Equal(t, 1234, id)
				assert.Zero(t, remoteID)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &remoteID))
				assert.Equal(t, 5678, id)
				assert.Equal(t, "ABCD", remoteID)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})
		})
	})

	return mig
}

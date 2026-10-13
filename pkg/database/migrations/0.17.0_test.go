package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_17_0TransferInfoValueType(t *testing.T, eng *testEngine) Change {
	mig := Migrations[66]

	// Migration only affects MySQL, skip other RDBMS.
	if eng.Dialect != MySQL {
		return mig
	}

	t.Run("When applying the 0.17.0 transfer info value type change", func(t *testing.T) {
		require.Equal(t, "TEXT", getColumnType(t, eng.DB, eng.Dialect, "transfer_info", "value"),
			`The "value" column should be of type TEXT before the migration`)

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should have changed the column type", func(t *testing.T) {
			assert.Equal(t, "LONGTEXT", getColumnType(t, eng.DB, eng.Dialect, "transfer_info", "value"),
				`The "value" column should now be of type LONGTEXT`)
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have restored the column type", func(t *testing.T) {
				assert.Equal(t, "TEXT", getColumnType(t, eng.DB, eng.Dialect, "transfer_info", "value"),
					`The "value" column should be of type TEXT again`)
			})
		})
	})

	return mig
}

func testVer0_17_0RemoveCloudCredsInOpts(t *testing.T, eng *testEngine) Change {
	mig := Migrations[67]

	t.Run("When applying the 0.17.0 secret removal", func(t *testing.T) {
		eng.NoError(t, `INSERT INTO cloud_instances (owner, name, type, options) VALUES
        	('gw', 's3', 's3', '{"other":"value","access_key_id":"key","secret_access_key":"secret"}'),
        	('gw', 'gcs', 'gcs', '{"other":"value","service_account_credentials":"account","service_account_file":"file"}'),
        	('gw', 'onedrive', 'onedrive', '{"other":"value","client_id":"client","client_secret":"secret"}'),
        	('gw', 'azblob', 'azblob', '{"other":"value","account":"account","key":"secret"}')
		`)
		t.Cleanup(func() { eng.NoError(t, "DELETE FROM cloud_instances") })

		require.NoError(t, eng.Upgrade(mig), "The migration should not fail")

		t.Run("Then it should have removed the secret values", func(t *testing.T) {
			rows, err := eng.DB.Query("SELECT options FROM cloud_instances")
			require.NoError(t, err)
			defer rows.Close()

			for rows.Next() {
				var opts jsonObj
				require.NoError(t, rows.Scan(&opts))
				assert.Equal(t, jsonObj{"other": "value"}, opts)
			}

			require.NoError(t, rows.Err())
		})
	})

	return mig
}

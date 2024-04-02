package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testVer0_7_5SplitR66TLS(t *testing.T, eng *testEngine) Change {
	mig := Migrations[32]

	t.Run("Given the 0.7.5 R66 agents split", func(t *testing.T) {
		// ### Local agents ###
		eng.NoError(t, `INSERT INTO local_agents(id,owner,name,protocol,address,proto_config) 
			VALUES (1,'waarp_gw','gw_r66_1','r66','localhost:1','{"isTLS":true}'),
			       (2,'waarp_gw','gw_r66_2','r66','localhost:2','{"isTLS":false}')`)

		// ### Remote agents ###
		eng.NoError(t, `INSERT INTO remote_agents(id,name,protocol,address,proto_config) 
			VALUES (3,'waarp_r66_1','r66','localhost:3','{"isTLS":true}'),
			       (4,'waarp_r66_2','r66','localhost:4','{"isTLS":false}')`)

		t.Cleanup(func() {
			eng.NoError(t, "DELETE FROM local_agents")
			eng.NoError(t, "DELETE FROM remote_agents")
		})

		require.NoError(t, eng.Upgrade(mig),
			"The migration should not fail")

		t.Run("Then it should have split the R66 agents", func(t *testing.T) {
			rows, err := eng.DB.Query(`
					SELECT id,protocol,proto_config FROM local_agents UNION ALL 
					SELECT id,protocol,proto_config FROM remote_agents ORDER BY id`)
			require.NoError(t, err)

			defer rows.Close()

			var (
				id          int
				proto, conf string
			)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &proto, &conf))
			assert.Equal(t, 1, id)
			assert.Equal(t, "r66-tls", proto)
			assert.Equal(t, `{"isTLS":true}`, conf)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &proto, &conf))
			assert.Equal(t, 2, id)
			assert.Equal(t, "r66", proto)
			assert.Equal(t, `{"isTLS":false}`, conf)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &proto, &conf))
			assert.Equal(t, 3, id)
			assert.Equal(t, "r66-tls", proto)
			assert.Equal(t, `{"isTLS":true}`, conf)

			require.True(t, rows.Next())
			require.NoError(t, rows.Scan(&id, &proto, &conf))
			assert.Equal(t, 4, id)
			assert.Equal(t, "r66", proto)
			assert.Equal(t, `{"isTLS":false}`, conf)

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
		})

		t.Run("When reverting the migration", func(t *testing.T) {
			// Adding new R66-TLS agents without "isTLS" to test if these
			// cases are handled properly when migrating down.
			eng.NoError(t, `INSERT INTO remote_agents(id,name,protocol,address,proto_config) 
				VALUES (5,'new_waarp_r66-tls','r66-tls','localhost:5','{}'),
					   (6,'new_waarp_r66','r66','localhost:6','{}')`)

			require.NoError(t, eng.Downgrade(mig),
				"Reverting the migration should not fail")

			t.Run("Then it should have restored the R66 agents", func(t *testing.T) {
				rows, err := eng.DB.Query(`
					SELECT id,protocol,proto_config FROM local_agents UNION ALL 
					SELECT id,protocol,proto_config FROM remote_agents ORDER BY id`)
				require.NoError(t, err)

				defer rows.Close()

				var (
					id          int
					proto, conf string
				)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &proto, &conf))
				assert.Equal(t, 1, id)
				assert.Equal(t, "r66", proto)
				assert.Equal(t, `{"isTLS":true}`, conf)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &proto, &conf))
				assert.Equal(t, 2, id)
				assert.Equal(t, "r66", proto)
				assert.Equal(t, `{"isTLS":false}`, conf)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &proto, &conf))
				assert.Equal(t, 3, id)
				assert.Equal(t, "r66", proto)
				assert.Equal(t, `{"isTLS":true}`, conf)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &proto, &conf))
				assert.Equal(t, 4, id)
				assert.Equal(t, "r66", proto)
				assert.Equal(t, `{"isTLS":false}`, conf)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &proto, &conf))
				assert.Equal(t, 5, id)
				assert.Equal(t, "r66", proto)
				assert.Equal(t, `{"isTLS":true}`, conf)

				require.True(t, rows.Next())
				require.NoError(t, rows.Scan(&id, &proto, &conf))
				assert.Equal(t, 6, id)
				assert.Equal(t, "r66", proto)
				assert.Equal(t, `{"isTLS":false}`, conf)

				require.False(t, rows.Next())
				require.NoError(t, rows.Err())
			})
		})
	})

	return mig
}

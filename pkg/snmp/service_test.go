package snmp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
)

func TestStartStop(t *testing.T) {
	// Setup test database and server configuration
	db := dbtest.TestDatabase(t)
	config := ServerConfig{
		LocalUDPAddress: "127.0.0.1:2222",
		Community:       "public",
	}
	require.NoError(t, db.Insert(&config).Run())
	monitor := MonitorConfig{
		Name:       "monitor",
		Version:    Version2,
		UDPAddress: "1.2.3.4:5",
		Community:  "public",
		UseInforms: true,
	}
	require.NoError(t, db.Insert(&monitor).Run())

	// Start and stop the service
	s := &Service{DB: db}
	require.NoError(t, s.Start())
	require.NoError(t, s.Stop(t.Context()))

	// Update the configuration in the database
	config.LocalUDPAddress = "127.0.0.1:2222"
	require.NoError(t, db.Update(&config).Run())
	monitor.Name = "monitor2"
	require.NoError(t, db.Update(&monitor).Run())

	// Restart the service
	require.NoError(t, s.Start())
	t.Cleanup(func() {
		require.NoError(t, s.Stop(t.Context()))
	})

	// Check that the config was updated
	assert.Equal(t, config.LocalUDPAddress, s.server.Address().String())
	require.Len(t, s.monitors, 1)
	assert.Equal(t, monitor, *s.monitors[0])
}

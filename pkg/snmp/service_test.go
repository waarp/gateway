package snmp

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestStartStop(t *testing.T) {
	t.Parallel()

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
		require.NoError(t, s.Stop(context.Background()))
	})

	// Check that the config was updated
	assert.Equal(t, config.LocalUDPAddress, s.GetServerAddr())
	require.Len(t, s.monitors, 1)
	assert.Equal(t, monitor, *s.monitors[0])
}

func TestReloadServerConfAfterConfigDeletion(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)
	config := ServerConfig{LocalUDPAddress: "127.0.0.1:0", Community: "public"}
	require.NoError(t, db.Insert(&config).Run())

	s := &Service{DB: db}
	require.NoError(t, s.Start())
	t.Cleanup(func() { require.NoError(t, s.Stop(context.Background())) })
	require.NotEmpty(t, s.GetServerAddr())

	require.NoError(t, db.Delete(&config).Run())
	require.NoError(t, s.ReloadServerConf(t.Context()))
	assert.Empty(t, s.GetServerAddr())
	state, _ := s.State()
	assert.Equal(t, utils.StateRunning, state)
}

func TestReloadServerConfFromErrorState(t *testing.T) {
	t.Parallel()

	// Setup
	db := dbtest.TestDatabase(t)
	config := ServerConfig{LocalUDPAddress: "127.0.0.1:0", Community: "public"}
	require.NoError(t, db.Insert(&config).Run())
	s := &Service{DB: db}
	s.state.Set(utils.StateError, "dummy error")

	// Reload server config
	require.NoError(t, s.ReloadServerConf(t.Context()))
	t.Cleanup(func() { require.NoError(t, s.Stop(context.Background())) })

	// Check state and server address
	state, _ := s.State()
	assert.Equal(t, utils.StateRunning, state)
	assert.NotEmpty(t, s.GetServerAddr())
}

func TestConcurrentReloadServerConf(t *testing.T) {
	db := dbtest.TestDatabase(t)
	config := ServerConfig{LocalUDPAddress: "127.0.0.1:0", Community: "public"}
	require.NoError(t, db.Insert(&config).Run())

	s := &Service{DB: db}
	require.NoError(t, s.Start())
	t.Cleanup(func() { require.NoError(t, s.Stop(context.Background())) })

	const reloads = 5
	var wg sync.WaitGroup
	errs := make(chan error, reloads)
	for range reloads {
		wg.Go(func() {
			errs <- s.ReloadServerConf(t.Context())
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	state, _ := s.State()
	assert.Equal(t, utils.StateRunning, state)
	assert.NotEmpty(t, s.GetServerAddr())
}

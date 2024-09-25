// Package dbtest an in-memory database implementation for testing.
package dbtest

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/migrations"
)

//nolint:gochecknoinits //init is required here
func init() {
	database.SupportedRBMS[database.SQLite] = memDBInfo
}

func memDBInfo() *database.DBInfo {
	config := conf.GlobalConfig.Database
	values := url.Values{}
	dsn := url.URL{
		Scheme:   "file",
		OmitHost: true,
		Path:     config.Address,
	}

	values.Set("mode", "memory")
	values.Set("cache", "shared")
	values.Set("_txlock", "immediate")
	values.Add("_pragma", "busy_timeout(10000)")
	values.Add("_pragma", "foreign_keys(ON)")
	values.Add("_pragma", "journal_mode(MEMORY)")
	values.Add("_pragma", "synchronous(OFF)")

	dsn.RawQuery = values.Encode()

	return &database.DBInfo{
		Driver:    migrations.SqliteDriver,
		DSN:       dsn.String(),
		ConnLimit: 1,
	}
}

func makeTestGCM(tb testing.TB) {
	tb.Helper()

	if database.GCM != nil {
		return
	}

	const aesKeySize = 16
	key := make([]byte, aesKeySize)

	_, err := rand.Read(key)
	require.NoError(tb, err, "cannot generate AES key")

	ciph, err := aes.NewCipher(key)
	require.NoError(tb, err, "cannot create AES cipher block")

	database.GCM, err = cipher.NewGCM(ciph)
	require.NoError(tb, err, "cannot initialize AES-GCM cipher")
}

// FIXME: remove the global config var, it causes a race condition between tests
// which use different databases. It requires the addition of this mutex to be
// able to run tests concurrently.
//
//nolint:gochecknoglobals //required here to avoid race condition between tests
var confLock sync.Mutex

func TestDatabase(tb testing.TB) *database.DB {
	tb.Helper()

	confLock.Lock()
	tb.Cleanup(confLock.Unlock)

	const shutdownTimeout = 5 * time.Second

	dbName := strings.ReplaceAll(tb.Name(), "/", "_")

	conf.GlobalConfig.Database = conf.DatabaseConfig{
		Type:    database.SQLite,
		Address: dbName,
	}

	makeTestGCM(tb)

	db := &database.DB{}
	require.NoError(tb, db.Start(), "cannot start database")

	tb.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		require.NoError(tb, db.Stop(ctx), "cannot stop database")
	})

	return db
}

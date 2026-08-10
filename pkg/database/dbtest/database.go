// Package dbtest an in-memory database implementation for testing.
package dbtest

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
)

const memDBType = "memory"

//nolint:gochecknoinits //init is required here
func init() {
	database.SupportedRBMS[memDBType] = memDBInfo
}

//nolint:gochecknoglobals //we use a global var here to save time and not have to reinstantiate it for every test
var testAEAD = makeTestGCM()

func memDBInfo(config *conf.DatabaseConfig) (gorm.Dialector, error) {
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

	db, err := sql.Open(database.SQLiteDriver, dsn.String())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(1)

	return sqlite.New(sqlite.Config{
		DSN:  dsn.String(),
		Conn: db,
	}), nil
}

func makeTestGCM() cipher.AEAD {
	const aesKeySize = 16
	key := make([]byte, aesKeySize)

	if _, err := rand.Read(key); err != nil {
		panic(err)
	}

	ciph, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	gcm, err := cipher.NewGCM(ciph)
	if err != nil {
		panic(err)
	}

	return gcm
}

func TestDatabase(tb testing.TB) *database.DB {
	tb.Helper()

	const shutdownTimeout = 5 * time.Second

	// dbName := strings.ReplaceAll(tb.Name(), "/", "_")
	dbName := filepath.Join(tb.TempDir(), "test.db")

	config := &conf.ServerConfig{}
	config.GatewayName = "gw-test"
	config.Database = conf.DatabaseConfig{
		Type:    database.SQLite,
		Address: dbName,
	}
	config.Paths.FilePerms = 0o600
	config.Paths.DirPerms = 0o700

	db := &database.DB{
		Logger: logtest.GetTestLogger(tb, logtest.WithLevel("TRACE")),
		Config: config,
	}
	db.ChangeAEAD(testAEAD)
	require.NoError(tb, db.Start(), "cannot start database")

	tb.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		require.NoError(tb, db.Stop(ctx), "cannot stop database")
	})

	return db
}

func InsertAsOwner(db *database.DB, owner string, obj database.InsertBean) error {
	oldOwner := db.Config.GatewayName
	db.Config.GatewayName = owner

	defer func() { db.Config.GatewayName = oldOwner }()

	//nolint:wrapcheck //wrapping adds nothing here
	return db.Insert(obj).Run()
}

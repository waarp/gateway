package database

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
)

// A model with a child table which has no primary key of its own, like
// model.Transfer and its transfer infos.
type updParent struct {
	ID       int64  `gorm:"column:id"`
	Name     string `gorm:"column:name"`
	Size     int64  `gorm:"column:size"`
	Children updChildren `gorm:"foreignKey:ParentID"`
}

func (*updParent) TableName() string   { return "upd_parent" }
func (*updParent) Appellation() string { return "test parent" }
func (u *updParent) GetID() int64      { return u.ID }

type updChild struct {
	ParentID int64  `gorm:"column:parent_id"`
	Name     string `gorm:"column:name"`
}

func (*updChild) TableName() string { return "upd_child" }

type updChildren []updChild

func newUpdateTestDB(tb testing.TB, logs io.Writer) *DB {
	tb.Helper()

	const shutdownTimeout = 5 * time.Second

	config := &conf.ServerConfig{}
	config.GatewayName = "gw-test"
	config.Database = conf.DatabaseConfig{
		Type:    SQLite,
		Address: filepath.Join(tb.TempDir(), "test.db"),
	}
	config.Paths.FilePerms = 0o600
	config.Paths.DirPerms = 0o700

	db := &DB{
		Logger: logtest.GetTestLogger(tb,
			logtest.WithWriter(logs), logtest.WithLevel("TRACE")),
		Config: config,
		AEAD:   testAEAD,
	}

	require.NoError(tb, db.Start())
	tb.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		require.NoError(tb, db.Stop(ctx))
	})

	require.NoError(tb, db.engine.AutoMigrate(&updParent{}, &updChild{}))

	return db
}

// An UPDATE must never write the bean's associations. GORM would upsert them
// with an ON CONFLICT clause which, the child having no primary key, carries no
// conflict target. SQLite and MySQL accept that, PostgreSQL rejects it with
// SQLSTATE 42601.
func TestUpdateIgnoresAssociations(t *testing.T) {
	t.Parallel()

	logs := &bytes.Buffer{}
	db := newUpdateTestDB(t, logs)

	parent := &updParent{Name: "old name", Size: 12}
	require.NoError(t, db.engine.Create(parent).Error)

	// The models fill their associations in their write hooks, so an entry
	// comes back from an insert (or from an eager select) with them populated.
	parent.Children = updChildren{{ParentID: parent.ID, Name: "child"}}
	parent.Name = "new name"
	parent.Size = 0

	logs.Reset()
	require.NoError(t, db.Update(parent).Run())

	assert.NotContains(t, logs.String(), "ON CONFLICT")

	var children updChildren
	require.NoError(t, db.engine.Find(&children).Error)
	assert.Empty(t, children, "the update must not have written the association")

	var after updParent
	require.NoError(t, db.engine.First(&after, parent.ID).Error)
	assert.Equal(t, "new name", after.Name)
	assert.Zero(t, after.Size, "a full update must also write the zero values")
}

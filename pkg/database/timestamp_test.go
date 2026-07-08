package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTimestamp(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		getDB func(testing.TB) *gorm.DB
	}{
		{"SQLite", newGormDB},
		{"PostgreSQL", newGormPostgresDB},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := tc.getDB(t)
			sqlDB, err := db.DB()
			require.NoError(t, err)

			type testStruct struct {
				Start time.Time `gorm:"serializer:timestamp;type:timestamp"`
			}

			require.NoError(t, db.AutoMigrate(&testStruct{}))

			date := time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local)
			obj := testStruct{Start: date}
			require.NoError(t, db.Create(&obj).Error)

			t.Run("test write", func(t *testing.T) {
				var ts time.Time
				row := sqlDB.QueryRow("SELECT * FROM test_structs")
				require.NoError(t, row.Scan(&ts))

				assert.Equal(t, date.UTC(), ts)
			})

			t.Run("test read", func(t *testing.T) {
				var check testStruct
				require.NoError(t, db.First(&check).Error)

				assert.Equal(t, date, check.Start)
			})
		})
	}
}

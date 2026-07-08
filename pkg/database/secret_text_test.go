package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretText(t *testing.T) {
	t.Parallel()

	db := newGormDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)

	type testStruct struct {
		Password string `gorm:"serializer:secret"`
	}

	require.NoError(t, db.AutoMigrate(&testStruct{}))

	obj := testStruct{Password: "sesame"}
	require.NoError(t, db.Create(&obj).Error)

	t.Run("test write", func(t *testing.T) {
		var cipherText string
		row := sqlDB.QueryRow("SELECT * FROM test_structs")
		require.NoError(t, row.Scan(&cipherText))

		plain, err := AESDecrypt(testAEAD, cipherText)
		require.NoError(t, err)

		assert.Equal(t, "sesame", plain)
	})

	t.Run("test read", func(t *testing.T) {
		var check testStruct
		require.NoError(t, db.First(&check).Error)

		assert.Equal(t, "sesame", check.Password)
	})
}

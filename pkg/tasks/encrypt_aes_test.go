package tasks

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestEncryptAESCFB(t *testing.T) {
	testEncryptAES(t, string(encryptModeCFB))
}

func TestEncryptAESCTR(t *testing.T) {
	testEncryptAES(t, string(encryptModeCTR))
}

func TestEncryptAESOFB(t *testing.T) {
	testEncryptAES(t, string(encryptModeOFB))
}

func testEncryptAES(t *testing.T, mode string) {
	t.Helper()

	const testFileContent = `Lorem ipsum dolor sit amet, consectetur adipiscing
elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim
ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea
commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit
esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat
non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`

	dir := t.TempDir()
	filePath := fs.JoinPath(dir, "aes_test.txt")
	require.NoError(t, fs.WriteFullFile(filePath, []byte(testFileContent)))

	transCtx := &model.TransferContext{
		Transfer: &model.Transfer{LocalPath: filePath},
	}

	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)

	outputFile1 := filePath + ".cipher"
	outputFile2 := filePath + ".plaintext"

	// 32 chars in base64 => 24 bytes long key => AES-192
	const key = "0123456789abcdefhijklABCDEFHIJKL"

	cryptoKey := model.CryptoKey{
		Name: "aes-key",
		Type: model.CryptoKeyTypeAES,
		Key:  key,
	}
	require.NoError(t, db.Insert(&cryptoKey).Run())

	encryptParams := map[string]string{
		"aesKeyName": cryptoKey.Name,
		"outputFile": outputFile1,
		"mode":       mode,
	}
	decryptParams := map[string]string{
		"aesKeyName": cryptoKey.Name,
		"outputFile": outputFile2,
		"mode":       mode,
	}

	encrypt := func() error {
		return (&encryptAES{}).Run(context.Background(), encryptParams, db,
			logger, transCtx)
	}

	decrypt := func() error {
		return (&decryptAES{}).Run(context.Background(), decryptParams, db,
			logger, transCtx)
	}

	t.Run(mode+" mode encrypt", func(t *testing.T) {
		require.NoError(t, encrypt(), "The task should not fail")

		assert.Equal(t, outputFile1, transCtx.Transfer.LocalPath,
			"The file path should have changed")

		encryptedContent, rErr := fs.ReadFullFile(outputFile1)
		require.NoError(t, rErr)

		assert.NotEqual(t, testFileContent, string(encryptedContent),
			"The content should have been encrypted")

		_, statErr := fs.Stat(filePath)
		assert.ErrorIs(t, statErr, os.ErrNotExist,
			"The original file should have been deleted")

		t.Run(mode+" mode decrypt", func(t *testing.T) {
			require.NoError(t, decrypt(), "The task should not fail")

			assert.Equal(t, outputFile2, transCtx.Transfer.LocalPath,
				"The file path should have changed")

			decryptedContent, rErr := fs.ReadFullFile(outputFile2)
			require.NoError(t, rErr)

			assert.Equal(t, testFileContent, string(decryptedContent),
				"The content should have been decrypted")

			_, statErr := fs.Stat(outputFile1)
			assert.ErrorIs(t, statErr, os.ErrNotExist,
				"The original file should have been deleted")
		})
	})
}

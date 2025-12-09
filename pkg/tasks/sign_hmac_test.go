package tasks

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestSignHMACSHA256(t *testing.T) {
	testSignHMAC(t, SignMethodHMACSHA256)
}

func TestSignHMACSHA384(t *testing.T) {
	testSignHMAC(t, SignMethodHMACSHA384)
}

func TestSignHMACSHA512(t *testing.T) {
	testSignHMAC(t, SignMethodHMACSHA512)
}

func TestSignHMACMD5(t *testing.T) {
	testSignHMAC(t, SignMethodHMACMD5)
}

func testSignHMAC(t *testing.T, method string) {
	t.Helper()

	const testFileContent = `Lorem ipsum dolor sit amet, consectetur adipiscing
elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim
ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea
commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit
esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat
non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`

	dir := t.TempDir()
	filePath := fs.JoinPath(dir, "hmac_test.txt")
	require.NoError(t, fs.WriteFullFile(filePath, []byte(testFileContent)))

	transCtx := &model.TransferContext{
		Transfer: &model.Transfer{LocalPath: filePath},
	}

	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)
	outputFile := filePath + ".hmac"

	const key = "0123456789ABCDEF"

	hmacKey := model.CryptoKey{
		Name: "hmac-key",
		Type: model.CryptoKeyTypeHMAC,
		Key:  key,
	}
	require.NoError(t, db.Insert(&hmacKey).Run())

	signParams := map[string]string{
		"outputFile": outputFile,
		"keyName":    hmacKey.Name,
		"method":     method,
	}

	verifyParams := map[string]string{
		"signatureFile": outputFile,
		"keyName":       hmacKey.Name,
		"method":        method,
	}

	doSign := func() error {
		return (&sign{}).Run(t.Context(), signParams, db, logger, transCtx, nil)
	}

	doVerify := func() error {
		return (&verify{}).Run(t.Context(), verifyParams, db, logger, transCtx, nil)
	}

	t.Run("When signing with "+method, func(t *testing.T) {
		require.NoError(t, doSign(), "Then the task should not fail")

		t.Run("When verifying the signature", func(t *testing.T) {
			require.NoError(t, doVerify(), "Then the task should not fail")
		})
	})
}

package tasks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestSignHMACSHA256(t *testing.T) {
	testSignHMAC(t, string(hmacAlgoSHA256))
}

func TestSignHMACSHA384(t *testing.T) {
	testSignHMAC(t, string(hmacAlgoSHA384))
}

func TestSignHMACSHA512(t *testing.T) {
	testSignHMAC(t, string(hmacAlgoSHA512))
}

func TestSignHMACMD5(t *testing.T) {
	testSignHMAC(t, string(hmacAlgoMD5))
}

func testSignHMAC(t *testing.T, algo string) {
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
	outputFile := filePath + ".hmac"

	const key = "0123456789ABCDEF"

	signParams := map[string]string{
		"algorithm":  algo,
		"outputFile": outputFile,
		"key":        key,
	}

	verifyParams := map[string]string{
		"algorithm":     algo,
		"signatureFile": outputFile,
		"key":           key,
	}

	sign := func() error {
		return (&signHMAC{}).Run(context.Background(), signParams, nil,
			logger, transCtx)
	}

	verify := func() error {
		return (&verifyHMAC{}).Run(context.Background(), verifyParams, nil,
			logger, transCtx)
	}

	t.Run("When signing with "+algo, func(t *testing.T) {
		require.NoError(t, sign(), "Then the task should not fail")

		t.Run("When verifying the signature", func(t *testing.T) {
			require.NoError(t, verify(), "Then the task should not fail")
		})
	})
}

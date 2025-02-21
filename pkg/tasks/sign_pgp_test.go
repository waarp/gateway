package tasks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestSignPGP(t *testing.T) {
	const testFileContent = `Lorem ipsum dolor sit amet, consectetur adipiscing
elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim
ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea
commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit
esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat
non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`

	db := dbtest.TestDatabase(t)
	root := t.TempDir()
	filePath := fs.JoinPath(root, "pgp_test.txt")
	require.NoError(t, fs.WriteFullFile(filePath, []byte(testFileContent)))

	pgpTestKey := &model.CryptoKey{
		Name: "test_key",
		Type: model.CryptoKeyTypePGPPrivate,
		Key:  testhelpers.TestPGPPrivateKey,
	}
	require.NoError(t, db.Insert(pgpTestKey).Run())

	transCtx := &model.TransferContext{
		Transfer: &model.Transfer{LocalPath: filePath},
	}

	logger := testhelpers.GetTestLogger(t)
	outputPath := filePath + ".signature"

	signParams := map[string]string{
		"pgpKeyName": pgpTestKey.Name,
		"outputFile": outputPath,
	}

	verifyParams := map[string]string{
		"pgpKeyName":    pgpTestKey.Name,
		"signatureFile": outputPath,
	}

	sign := func() error {
		return (&signPGP{}).Run(context.Background(), signParams, db,
			logger, transCtx)
	}

	verify := func() error {
		return (&verifyPGP{}).Run(context.Background(), verifyParams, db,
			logger, transCtx)
	}

	t.Run("When signing with PGP", func(t *testing.T) {
		require.NoError(t, sign(), "Then the task should not fail")

		t.Run("When verifying the signature", func(t *testing.T) {
			require.NoError(t, verify(), "Then the task should not fail")
		})
	})
}

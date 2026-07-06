package filewatcher

import (
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

var errNotImplemented = errors.New("not implemented")

func doListerTests(t *testing.T, lister Lister, filePermMask, dirPermMask fs.FileMode) {
	t.Helper()
	getFile := func(tb testing.TB, name string) fs.FileInfo {
		tb.Helper()
		info, err := memFs.Stat(name)
		require.NoError(t, err)

		return maskFileInfo{
			FileInfo: info,
			fMask:    filePermMask,
			dMask:    dirPermMask,
		}
	}

	t.Run("Root", func(t *testing.T) {
		result, err := lister.List("*")
		require.NoError(t, err)
		require.Len(t, result, 2)

		checkEqualInfo(t, getFile(t, "file1.txt"), result[0])
		checkEqualInfo(t, getFile(t, "file2.txt"), result[1])
	})

	t.Run("Subdir", func(t *testing.T) {
		result, err := lister.List("dirA/*")
		require.NoError(t, err)
		require.Len(t, result, 2)

		checkEqualInfo(t, getFile(t, "dirA/fileA1.txt"), result[0])
		checkEqualInfo(t, getFile(t, "dirA/fileA2.txt"), result[1])
	})
}

func checkEqualInfo(tb testing.TB, expected, actual fs.FileInfo) {
	tb.Helper()

	assert.Equal(tb, expected.Name(), actual.Name())
	assert.Equal(tb, expected.Size(), actual.Size())
	assert.Equal(tb, expected.Mode(), actual.Mode())
	assert.Equal(tb, expected.ModTime().UTC(), actual.ModTime().UTC())
	assert.Equal(tb, expected.IsDir(), actual.IsDir())
}

type testContext struct {
	db     *database.DB
	logger *log.Logger
	dialer *protoutils.TraceDialer
	models *testModels
}

func makeTestContext(tb testing.TB, models *testModels) *testContext {
	tb.Helper()

	db := dbtest.TestDatabase(tb)
	logger := logtest.GetTestLogger(tb)
	dialer, _ := protoutils.NewDialerFor("")
	dialer.Timeout = time.Second

	// partner
	require.NoError(tb, db.Insert(models.partner).Run())
	// partner credentials
	for _, cred := range models.partnerCredentials {
		cred.RemoteAgentID = models.partner.NullableID()
		require.NoError(tb, db.Insert(cred).Run())
	}

	// user
	models.user.RemoteAgentID = models.partner.ID
	require.NoError(tb, db.Insert(models.user).Run())
	// user credentials
	for _, cred := range models.userCredentials {
		cred.RemoteAccountID = models.user.NullableID()
		require.NoError(tb, db.Insert(cred).Run())
	}

	return &testContext{
		db:     db,
		logger: logger,
		dialer: dialer,
		models: models,
	}
}

type testModels struct {
	partner            *model.RemoteAgent
	partnerCredentials []*model.Credential
	user               *model.RemoteAccount
	userCredentials    []*model.Credential
}

func (t *testModels) asTransCtx() *model.TransferContext {
	return &model.TransferContext{
		Rule: &model.Rule{
			Name:   "default",
			IsSend: false,
		},
		Client: &model.Client{
			Name:     t.partner.Protocol + "-client",
			Protocol: t.partner.Protocol,
		},
		RemoteAgent:        t.partner,
		RemoteAgentCreds:   t.partnerCredentials,
		RemoteAccount:      t.user,
		RemoteAccountCreds: t.userCredentials,
	}
}

type maskFileInfo struct {
	fs.FileInfo
	fMask, dMask fs.FileMode
}

func (m maskFileInfo) Mode() fs.FileMode {
	if m.IsDir() {
		return m.FileInfo.Mode() | m.dMask
	}
	return m.FileInfo.Mode() | m.fMask
}

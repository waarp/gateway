package protoutils_test

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocolstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used by design
func init() {
	protocols.Register(testProtocol, protocolstest.TestModule{})
}

func TestGetRealPathConfinement(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)
	root := t.TempDir()
	db.Config.Paths.GatewayHome = root

	server := &model.LocalAgent{
		Name: "server", Protocol: testProtocol,
		RootDir: "srv_root", SendDir: "out",
		Address: types.Addr("localhost", 0),
	}
	require.NoError(t, db.Insert(server).Run())

	account := &model.LocalAccount{LocalAgentID: server.ID, Login: "toto"}
	require.NoError(t, db.Insert(account).Run())
	require.NoError(t, db.Get(&account.LocalAgent, "id=?", server.ID).Run())

	rule := &model.Rule{Name: "send", IsSend: true, Path: "send", LocalDir: "send_dir"}
	require.NoError(t, db.Insert(rule).Run())

	ruleDir := fs.JoinPath(root, "srv_root", "send_dir")

	inside := map[string]string{
		"/send":             ruleDir,
		"/send/file":        fs.JoinPath(ruleDir, "file"),
		"/send/sub/dir":     fs.JoinPath(ruleDir, "sub", "dir"),
		"/send/sub/../file": fs.JoinPath(ruleDir, "file"),
		"/send/..file":      fs.JoinPath(ruleDir, "..file"),
	}

	outside := []string{
		"/send/..",
		"/send/../file",
		"/send/sub/../../file",
		"/send/file:/etc",
		"/send/s3:bucket/key",
	}

	if runtime.GOOS == "windows" {
		outside = append(outside, `/send/..\..\file`)
	} else {
		inside[`/send/..\..\file`] = fs.JoinPath(ruleDir, `..\..\file`)
	}

	for name, expected := range inside {
		t.Run("accepts "+name, func(t *testing.T) {
			t.Parallel()

			realPath, err := protoutils.GetRealPath(false, db, logger, account, name)
			require.NoError(t, err)
			assert.Equal(t, expected, realPath)
		})
	}

	for _, name := range outside {
		t.Run("refuses "+name, func(t *testing.T) {
			t.Parallel()

			// No real path may be returned: either the path is refused, or it
			// matches no rule at all (and is then served as a virtual directory).
			realPath, err := protoutils.GetRealPath(false, db, logger, account, name)
			assert.Empty(t, realPath)

			if err != nil {
				require.ErrorIs(t, err, protoutils.ErrPermissionDenied)
			}
		})
	}
}

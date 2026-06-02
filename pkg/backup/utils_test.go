package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/authtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/modeltest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used by design
func init() {
	modeltest.AddDummyProtoConfig(testProtocol)
	modeltest.AddDummyProtoConfig(r66TLS)

	modeltest.AddDummyTask("COPY")
	modeltest.AddDummyTask("MOVE")
	modeltest.AddDummyTask("DELETE")

	authtest.AddDummyAuthHandler(r66LegacyCert, r66TLS)
}

func discard() *log.Logger {
	back, err := log.NewBackend(log.LevelTrace, log.Discard, "", "")
	convey.So(err, convey.ShouldBeNil)

	return back.NewLogger("discard")
}

func hash(pwd string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	convey.So(err, convey.ShouldBeNil)

	return string(h)
}

func localPath(full string) string {
	if runtime.GOOS == "windows" {
		full = "C:" + full
	}

	return full
}

func mustAddr(addr string) types.Address {
	a, err := types.NewAddress(addr)
	convey.So(err, convey.ShouldBeNil)

	return *a
}

func assertHasHash(tb testing.TB, creds []file.Credential, hash string) {
	tb.Helper()
	assert.Contains(tb, creds, file.Credential{
		Name:  auth.Password,
		Type:  auth.Password,
		Value: hash,
	})
}

func assertHasHashOf(tb testing.TB, creds []file.Credential, pswd string) {
	tb.Helper()

	for _, cred := range creds {
		if cred.Type == auth.Password {
			assert.True(tb, utils.IsHashOf(cred.Value, pswd))

			return
		}
	}

	assert.Failf(tb, "no password %q credential found", pswd)
}

func TestMakeJSON(t *testing.T) {
	const size = 100
	data := file.Data{
		Remotes: make([]file.RemoteAgent, size),
	}

	for i := range size {
		data.Remotes[i] = file.RemoteAgent{
			Name:     fmt.Sprintf("partner-%d", i),
			Address:  fmt.Sprintf("127.0.0.1:%d", i),
			Protocol: "r66-tls",
			Configuration: map[string]any{
				"serverPassword": "sesame",
			},
			Accounts: []file.RemoteAccount{{
				Login:    "toto",
				Password: "sesame",
			}},
			Certificates: []file.Certificate{{
				Name:        "cert",
				Certificate: testhelpers.LocalhostCert,
			}},
		}
	}

	file, err := os.Create("test.json")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, file.Close())
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	require.NoError(t, encoder.Encode(data))
}

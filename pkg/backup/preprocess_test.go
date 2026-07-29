package backup

import (
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	r66lib "code.waarp.fr/lib/r66"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreprocess(t *testing.T) {
	t.Parallel()

	const (
		pswd     = "sesame"
		pswdHash = "$2a$12$6Q/oyEIPlw9UzdzN79srU.w4JjZNbwHxEj87GmJ4sxamR9TSAepNe"
	)

	r66Pswd := string(r66lib.CryptPass([]byte(pswd)))

	data := &file.Data{
		Locals: []file.LocalAgent{{
			Name:     "sftp-server",
			Protocol: "sftp",
			Accounts: []file.LocalAccount{{
				Login:    "password",
				Password: pswd,
			}, {
				Login:        "hash",
				PasswordHash: pswdHash,
			}, {
				Login:       "cred",
				Credentials: []file.Credential{pswdCred(pswd)},
			}, {
				Login:       "cred-hash",
				Credentials: []file.Credential{pswdCred(pswdHash)},
			}, {
				Login:       "password&cred",
				Password:    "old_sesame",
				Credentials: []file.Credential{pswdCred(pswd)},
			}},
		}},
		Remotes: []file.RemoteAgent{
			{
				Name:          "r66-conf-pswd",
				Protocol:      "r66",
				Configuration: map[string]any{"serverPassword": pswd},
			}, {
				Name:          "r66-conf-hash",
				Protocol:      "r66",
				Configuration: map[string]any{"serverPassword": pswdHash},
			}, {
				Name:        "r66-cred-pswd",
				Protocol:    "r66",
				Credentials: []file.Credential{pswdCred(pswd)},
			}, {
				Name:        "r66-cred-hash",
				Protocol:    "r66",
				Credentials: []file.Credential{pswdCred(pswdHash)},
			}, {
				Name:     "r66-conf&cred-pswd",
				Protocol: "r66",
				Configuration: map[string]any{
					"serverPassword": "old_sesame",
				},
				Credentials: []file.Credential{pswdCred(pswd)},
			},
		},
		Users: []file.User{
			{
				Username: "with-password",
				Password: pswd,
			}, {
				Username:     "with-hash",
				PasswordHash: pswdHash,
			},
		},
	}
	require.NoError(t, preprocessImport(data))

	t.Run("Servers", func(t *testing.T) {
		t.Parallel()
		require.Len(t, data.Locals, 1)

		t.Run("Local accounts", func(t *testing.T) {
			t.Parallel()
			sftpServer := data.Locals[0]
			require.Len(t, sftpServer.Accounts, 5)

			t.Run("With password", func(t *testing.T) {
				t.Parallel()
				withPassword := sftpServer.Accounts[0]
				assertHasHashOf(t, withPassword.Credentials, pswd)
			})

			t.Run("With hash", func(t *testing.T) {
				t.Parallel()
				withHash := sftpServer.Accounts[1]
				assertHasHash(t, withHash.Credentials, pswdHash)
			})

			t.Run("With credential password", func(t *testing.T) {
				t.Parallel()
				withCredPswd := sftpServer.Accounts[2]
				assertHasHashOf(t, withCredPswd.Credentials, pswd)
			})

			t.Run("With credential hash", func(t *testing.T) {
				t.Parallel()
				withCredHash := sftpServer.Accounts[3]
				assertHasHash(t, withCredHash.Credentials, pswdHash)
			})

			t.Run("With both", func(t *testing.T) {
				t.Parallel()
				withBoth := sftpServer.Accounts[4]
				assertHasHashOf(t, withBoth.Credentials, pswd)
			})
		})
	})

	t.Run("Partners", func(t *testing.T) {
		t.Parallel()
		require.Len(t, data.Remotes, 5)

		t.Run("With conf password", func(t *testing.T) {
			t.Parallel()
			confPassword := data.Remotes[0]
			assertHasHashOf(t, confPassword.Credentials, r66Pswd)
			assert.NotContains(t, confPassword.Configuration, "serverPassword")
		})

		t.Run("With conf hash", func(t *testing.T) {
			t.Parallel()
			confHash := data.Remotes[1]
			assertHasHash(t, confHash.Credentials, pswdHash)
			assert.NotContains(t, confHash.Configuration, "serverPassword")
		})

		t.Run("With credential password", func(t *testing.T) {
			t.Parallel()
			withCredPswd := data.Remotes[2]
			assertHasHashOf(t, withCredPswd.Credentials, r66Pswd)
		})

		t.Run("With credential hash", func(t *testing.T) {
			t.Parallel()
			withCredHash := data.Remotes[3]
			assertHasHash(t, withCredHash.Credentials, pswdHash)
		})

		t.Run("With both", func(t *testing.T) {
			t.Parallel()
			withBoth := data.Remotes[4]
			assertHasHashOf(t, withBoth.Credentials, r66Pswd)
		})
	})

	t.Run("Users", func(t *testing.T) {
		t.Parallel()
		require.Len(t, data.Users, 2)

		t.Run("With password", func(t *testing.T) {
			t.Parallel()
			withPassword := data.Users[0]
			assert.True(t, utils.IsHashOf(withPassword.PasswordHash, pswd))
		})

		t.Run("With hash", func(t *testing.T) {
			t.Parallel()
			withHash := data.Users[1]
			assert.Equal(t, pswdHash, withHash.PasswordHash)
		})
	})
}

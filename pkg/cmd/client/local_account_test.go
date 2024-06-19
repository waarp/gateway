package wg

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalAccountGet(t *testing.T) {
	const (
		server = "foo"

		login    = "bar"
		send1    = "send1"
		send2    = "send2"
		receive1 = "receive1"
		receive2 = "receive2"
		cred1    = "cred1"
		cred2    = "cred2"

		path = "/api/servers/" + server + "/accounts/" + login
	)

	t.Run(`Testing the local account "get" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &LocAccGet{}

		Server = server
		defer resetVars()

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"login":       login,
				"credentials": []string{cred1, cred2},
				"authorizedRules": map[string]any{
					"sending":   []string{send1, send2},
					"reception": []string{receive1, receive2},
				},
			},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, login),
					"Then it should not return an error")

				assert.Equal(t,
					expectedOutput(t, result.body,
						`‣Account "{{.login}}"`,
						`  •Credentials: {{ join .credentials }}`,
						`  •Authorized rules:`,
						`    ⁃Send: {{ join .authorizedRules.sending }}`,
						`    ⁃Receive: {{ join .authorizedRules.reception }}`,
					),
					w.String(),
					"Then it should display the account",
				)
			})
		})
	})
}

func TestLocalAccountAdd(t *testing.T) {
	const (
		server = "foo"

		login    = "bar"
		password = "sesame"

		path     = "/api/servers/" + server + "/accounts"
		location = path + "/" + login
	)

	t.Run(`Testing the local account "add" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &LocAccAdd{}

		Server = server
		defer resetVars()

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body: map[string]any{
				"login":    login,
				"password": password,
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--login", login,
					"--password", password,
				),
					"Then it should not return an error",
				)

				assert.Equal(t,
					fmt.Sprintf("The account %q was successfully added.\n", login),
					w.String(),
					"Then it should display a message saying the account was added",
				)
			})
		})
	})
}

func TestLocalAccountDelete(t *testing.T) {
	const (
		server = "foo"
		login  = "bar"

		path = "/api/servers/" + server + "/accounts/" + login
	)

	t.Run(`Testing the local account "delete" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &LocAccDelete{}

		Server = server
		defer resetVars()

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusNoContent}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, login),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The account %q was successfully deleted.\n", login),
					w.String(),
					"Then it should display a message saying the account was deleted",
				)
			})
		})
	})
}

func TestLocalAccountUpdate(t *testing.T) {
	const (
		server = "foo"

		oldLogin = "bar"
		login    = "baz"
		password = "sesame"

		path     = "/api/servers/" + server + "/accounts/" + oldLogin
		location = "/api/servers/" + server + "/accounts/" + login
	)

	t.Run(`Testing the local account "update" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &LocAccUpdate{}

		Server = server
		defer resetVars()

		expected := &expectedRequest{
			method: http.MethodPatch,
			path:   path,
			body: map[string]any{
				"login":    login,
				"password": password,
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--login", login,
					"--password", password,
					oldLogin,
				),
					"Then it should not return an error",
				)

				assert.Equal(t,
					fmt.Sprintf("The account %q was successfully updated.\n", login),
					w.String(),
					"Then it should display a message saying the account was updated",
				)
			})
		})
	})
}

func TestLocalAccountsList(t *testing.T) {
	const (
		server = "foo"
		path   = "/api/servers/" + server + "/accounts"

		sort   = "login+"
		limit  = "10"
		offset = "5"

		login1 = "bar1"
		login2 = "bar2"
	)

	t.Run(`Testing the local account "list" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &LocAccList{}

		Server = server
		defer resetVars()

		expected := &expectedRequest{
			method: http.MethodGet,
			values: map[string][]string{
				"limit":  {limit},
				"offset": {offset},
				"sort":   {sort},
			},
			path: path,
		}

		localAccounts := []map[string]any{
			{"login": login1},
			{"login": login2},
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body:   map[string]any{"localAccounts": localAccounts},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--limit", limit, "--offset", offset, "--sort", sort,
				),
					"Then it should not return an error",
				)

				assert.Equal(t,
					expectedOutput(t, localAccounts,
						`=== Accounts of server "{{getServer}}" ===`,
						`{{- with (index . 0) }}`,
						`‣Account "{{.login}}"`,
						`  •Credentials: <none>`,
						`  •Authorized rules:`,
						`    ⁃Send: <none>`,
						`    ⁃Receive: <none>`,
						`{{- end }}`,
						`{{- with (index . 1) }}`,
						`‣Account "{{.login}}"`,
						`  •Credentials: <none>`,
						`  •Authorized rules:`,
						`    ⁃Send: <none>`,
						`    ⁃Receive: <none>`,
						`{{- end }}`,
					),
					w.String(),
					"Then it should display the accounts of the server",
				)
			})
		})
	})
}

func TestLocalAccountAuthorize(t *testing.T) {
	const (
		server = "foo"
		login  = "bar"

		rule = "push"
		way  = directionSend

		path = "/api/servers/" + server + "/accounts/" + login + "/authorize/" +
			rule + "/" + way
	)

	t.Run(`Testing the local account "authorize" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &LocAccAuthorize{}

		Server = server
		defer resetVars()

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusOK}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, login, rule, way),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The local account %q is now allowed to use the %s rule %q for transfers.\n",
						login, way, rule),
					w.String(),
					"Then it should display a message saying the account can now use the rule",
				)
			})
		})
	})
}

func TestLocalAccountRevoke(t *testing.T) {
	const (
		server = "foo"
		login  = "bar"

		rule = "pull"
		way  = directionRecv

		path = "/api/servers/" + server + "/accounts/" + login + "/revoke/" +
			rule + "/" + way
	)

	t.Run(`Testing the local account "revoke" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &LocAccRevoke{}

		Server = server
		defer resetVars()

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusOK}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, login, rule, way),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The local account %q is no longer allowed to use the %s rule %q for transfers.\n",
						login, way, rule),
					w.String(),
					"Then it should display a message saying the account can no longer use the rule",
				)
			})
		})
	})
}

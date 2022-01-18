package wg

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoteAccountGet(t *testing.T) {
	const (
		partner = "foo"

		login    = "bar"
		send1    = "send1"
		send2    = "send2"
		receive1 = "receive1"
		receive2 = "receive2"
		cred1    = "cred1"
		cred2    = "cred2"

		path = "/api/partners/" + partner + "/accounts/" + login
	)

	t.Run(`Testing the remote account "get" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &RemAccGet{}

		Partner = partner
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
					fmt.Sprintf("── Account %q\n", login)+
						fmt.Sprintf("   ├─ Credentials: %s, %s\n", cred1, cred2)+
						fmt.Sprintf("   ╰─ Authorized rules\n")+
						fmt.Sprintf("      ├─ Send: %s, %s\n", send1, send2)+
						fmt.Sprintf("      ╰─ Receive: %s, %s\n", receive1, receive2),
					w.String(),
					"Then it should display the account",
				)
			})
		})
	})
}

func TestRemoteAccountAdd(t *testing.T) {
	const (
		partner = "foo"

		login    = "bar"
		password = "sesame"

		path     = "/api/partners/" + partner + "/accounts"
		location = path + "/" + login
	)

	t.Run(`Testing the remote account "add" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &RemAccAdd{}

		Partner = partner
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

func TestRemoteAccountDelete(t *testing.T) {
	const (
		partner = "foo"
		login   = "bar"

		path = "/api/partners/" + partner + "/accounts/" + login
	)

	t.Run(`Testing the remote account "delete" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &RemAccDelete{}

		Partner = partner
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

func TestRemoteAccountUpdate(t *testing.T) {
	const (
		partner = "foo"

		oldLogin = "bar"
		login    = "baz"
		password = "sesame"

		path     = "/api/partners/" + partner + "/accounts/" + oldLogin
		location = path + "/" + login
	)

	t.Run(`Testing the remote account "update" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &RemAccUpdate{}

		Partner = partner
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

func TestRemoteAccountList(t *testing.T) {
	const (
		partner = "foo"
		path    = "/api/partners/" + partner + "/accounts"

		sort   = "login+"
		limit  = "10"
		offset = "5"

		login1 = "bar1"
		login2 = "bar2"
	)

	t.Run(`Testing the remote account "list" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &RemAccList{}

		Partner = partner
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

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"remoteAccounts": []any{
					map[string]any{"login": login1},
					map[string]any{"login": login2},
				},
			},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--limit", limit, "--offset", offset, "--sort", sort,
				),
					"Then it should not return an error",
				)

				assert.Equal(t, fmt.Sprintf("Accounts of partner %q:\n", partner)+
					fmt.Sprintf("╭─ Account %q\n", login1)+
					fmt.Sprintf("│  ├─ Credentials: <none>\n")+
					fmt.Sprintf("│  ╰─ Authorized rules\n")+
					fmt.Sprintf("│     ├─ Send: <none>\n")+
					fmt.Sprintf("│     ╰─ Receive: <none>\n")+
					fmt.Sprintf("╰─ Account %q\n", login2)+
					fmt.Sprintf("   ├─ Credentials: <none>\n")+
					fmt.Sprintf("   ╰─ Authorized rules\n")+
					fmt.Sprintf("      ├─ Send: <none>\n")+
					fmt.Sprintf("      ╰─ Receive: <none>\n"),
					w.String(),
					"Then it should display the accounts of the partner",
				)
			})
		})
	})
}

func TestRemoteAccountAuthorize(t *testing.T) {
	const (
		partner = "foo"
		login   = "bar"

		rule = "push"
		way  = directionSend

		path = "/api/partners/" + partner + "/accounts/" + login + "/authorize/" +
			rule + "/" + way
	)

	t.Run(`Testing the remote account "authorize" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &RemAccAuthorize{}

		Partner = partner
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
					fmt.Sprintf("The remote account %q is now allowed to use the %s rule %q for transfers.\n",
						login, way, rule),
					w.String(),
					"Then it should display a message saying the account can now use the rule",
				)
			})
		})
	})
}

func TestRemoteAccountRevoke(t *testing.T) {
	const (
		partner = "foo"
		login   = "bar"

		rule = "pull"
		way  = directionRecv

		path = "/api/partners/" + partner + "/accounts/" + login + "/revoke/" +
			rule + "/" + way
	)

	t.Run(`Testing the remote account "revoke" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &RemAccRevoke{}

		Partner = partner

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
					fmt.Sprintf("The remote account %q is no longer allowed to use the %s rule %q for transfers.\n",
						login, way, rule),
					w.String(),
					"Then it should display a message saying the account can no longer use the rule",
				)
			})
		})
	})
}

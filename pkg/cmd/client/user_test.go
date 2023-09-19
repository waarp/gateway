package wg

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserGet(t *testing.T) {
	const (
		username     = "foo"
		permTrans    = "-r-"
		permServers  = "-w-"
		permPartners = "--d"
		permRules    = "rw-"
		permUsers    = "r-w"
		permAdmin    = "-wd"

		path = "/api/users/" + username
	)

	t.Run(`Testing the user "get" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &UserGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"username": username,
				"perms": map[string]any{
					"transfers":      permTrans,
					"servers":        permServers,
					"partners":       permPartners,
					"rules":          permRules,
					"users":          permUsers,
					"administration": permAdmin,
				},
			},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, username),
					"Then is should not return an error")

				assert.Equal(t,
					fmt.Sprintf("── User %q\n", username)+
						fmt.Sprintf("   ╰─ Permissions\n")+
						fmt.Sprintf("      ├─ Transfers: %s\n", permTrans)+
						fmt.Sprintf("      ├─ Servers: %s\n", permServers)+
						fmt.Sprintf("      ├─ Partners: %s\n", permPartners)+
						fmt.Sprintf("      ├─ Rules: %s\n", permRules)+
						fmt.Sprintf("      ├─ Users: %s\n", permUsers)+
						fmt.Sprintf("      ╰─ Administration: %s\n", permAdmin),
					w.String(),
					"Then it should display the user's info",
				)
			})
		})
	})
}

func TestUserAdd(t *testing.T) {
	const (
		username = "foo"
		password = "sesame"

		permsTrans    = "=r"
		permsServers  = "=w"
		permsPartners = "=d"
		permsRules    = "=rw"
		permsUsers    = "=rd"
		permsAdmin    = "=wd"

		permsFull = "T" + permsTrans + ",S" + permsServers +
			",P" + permsPartners + ",R" + permsRules +
			",U" + permsUsers + ",A" + permsAdmin

		path     = "/api/users"
		location = path + "/" + username
	)

	t.Run(`Testing the user "add" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &UserAdd{}

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body: map[string]any{
				"username": username,
				"password": password,
				"perms": map[string]any{
					"transfers":      permsTrans,
					"servers":        permsServers,
					"partners":       permsPartners,
					"rules":          permsRules,
					"users":          permsUsers,
					"administration": permsAdmin,
				},
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: http.Header{"Location": {location}},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--username", username,
					"--password", password,
					"--rights", permsFull),
					"Then is should not return an error",
				)

				assert.Equal(t,
					fmt.Sprintf("The user %q was successfully added.\n", username),
					w.String(),
					"Then it should display a message saying the user was added",
				)
			})
		})
	})
}

func TestUserDelete(t *testing.T) {
	const (
		username = "foo"

		path = "/api/users/" + username
	)

	t.Run(`Testing the user "delete" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &UserDelete{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusNoContent}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, username),
					"Then is should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The user %q was successfully deleted.\n", username),
					w.String(),
					"Then it should display a message saying the user was deleted",
				)
			})
		})
	})
}

func TestUserUpdate(t *testing.T) {
	const (
		oldName  = "foo"
		username = "bar"
		password = "sesame"

		permsTrans    = "+r"
		permsServers  = "-w"
		permsPartners = "=d"
		permsRules    = "=rw"
		permsUsers    = "+rd"
		permsAdmin    = "-wd"

		permsFull = "T" + permsTrans + ",S" + permsServers +
			",P" + permsPartners + ",R" + permsRules +
			",U" + permsUsers + ",A" + permsAdmin

		path     = "/api/users/" + oldName
		location = "/api/users/" + username
	)

	t.Run(`Testing the user "delete" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &UserUpdate{}

		expected := &expectedRequest{
			method: http.MethodPatch,
			path:   path,
			body: map[string]any{
				"username": username,
				"password": password,
				"perms": map[string]any{
					"transfers":      permsTrans,
					"servers":        permsServers,
					"partners":       permsPartners,
					"rules":          permsRules,
					"users":          permsUsers,
					"administration": permsAdmin,
				},
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: http.Header{"Location": {location}},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--username", username,
					"--password", password,
					"--rights", permsFull,
					oldName),
					"Then is should not return an error",
				)

				assert.Equal(t,
					fmt.Sprintf("The user %q was successfully updated.\n", username),
					w.String(),
					"Then it should display a message saying the user was updated",
				)
			})
		})
	})
}

func TestUserList(t *testing.T) {
	const (
		path = "/api/users"

		sort   = "username+"
		limit  = "10"
		offset = "5"

		user1 = "user1"
		user2 = "user2"
	)

	t.Run(`Testing the user "list" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &UserList{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
			values: url.Values{
				"sort":   {sort},
				"limit":  {limit},
				"offset": {offset},
			},
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"users": []any{
					map[string]any{"username": user1},
					map[string]any{"username": user2},
				},
			},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--limit", limit, "--offset", offset, "--sort", sort,
				),
					"Then it should not return an error")

				assert.Equal(t,
					"Users:\n"+
						fmt.Sprintf("╭─ User %q\n", user1)+
						fmt.Sprintf("│  ╰─ Permissions\n")+
						fmt.Sprintf("│     ├─ Transfers: ---\n")+
						fmt.Sprintf("│     ├─ Servers: ---\n")+
						fmt.Sprintf("│     ├─ Partners: ---\n")+
						fmt.Sprintf("│     ├─ Rules: ---\n")+
						fmt.Sprintf("│     ├─ Users: ---\n")+
						fmt.Sprintf("│     ╰─ Administration: ---\n")+
						fmt.Sprintf("╰─ User %q\n", user2)+
						fmt.Sprintf("   ╰─ Permissions\n")+
						fmt.Sprintf("      ├─ Transfers: ---\n")+
						fmt.Sprintf("      ├─ Servers: ---\n")+
						fmt.Sprintf("      ├─ Partners: ---\n")+
						fmt.Sprintf("      ├─ Rules: ---\n")+
						fmt.Sprintf("      ├─ Users: ---\n")+
						fmt.Sprintf("      ╰─ Administration: ---\n"),
					w.String(),
					"Then it should display the users",
				)
			})
		})
	})
}

package wg

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestClientAdd(t *testing.T) {
	const (
		clientName     = "test_client"
		clientProtocol = "test_protocol"
		clientAddress  = "1.2.3.4:5"

		clientConfigStrKey  = "str_key"
		clientConfigStrVal  = "str_val"
		clientConfigNumKey  = "int_key"
		clientConfigNumVal  = 1.0
		clientConfigBoolKey = "bool_key"
		clientConfigBoolVal = true

		path     = "/api/clients"
		location = path + "/" + clientName
	)

	expectedBody := map[string]any{
		"name":         clientName,
		"protocol":     clientProtocol,
		"localAddress": clientAddress,
		"protoConfig": map[string]any{
			clientConfigStrKey:  clientConfigStrVal,
			clientConfigNumKey:  clientConfigNumVal,
			clientConfigBoolKey: clientConfigBoolVal,
		},
	}

	t.Run(`Given the client "add" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ClientAdd{}

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body:   expectedBody,
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				assert.NoError(t, executeCommand(t, w, command,
					"--name", clientName,
					"--protocol", clientProtocol,
					"--local-address", clientAddress,
					"--config", clientConfigStrKey+":"+clientConfigStrVal,
					"--config", clientConfigNumKey+":"+utils.FormatFloat(clientConfigNumVal),
					"--config", clientConfigBoolKey+":"+strconv.FormatBool(clientConfigBoolVal),
				),
					"Then it should not return an error",
				)

				assert.Equal(t,
					fmt.Sprintf("The client %q was successfully added.\n", clientName),
					w.String(),
					"Then it should display a message saying the client was added",
				)
			})
		})
	})
}

func TestClientList(t *testing.T) {
	const (
		path = "/api/clients"

		sort     = "name+"
		limit    = "10"
		offset   = "5"
		protocol = "proto1"

		client1name    = "cli1"
		client1proto   = "proto1"
		client1addr    = "addr1"
		client1enabled = true

		client2name    = "cli2"
		client2proto   = "proto2"
		client2addr    = "addr2"
		client2enabled = false
	)

	var (
		status1 = enabledStatus(client1enabled)
		status2 = enabledStatus(client2enabled)
	)

	t.Run(`Given the client "list" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ClientList{}

		expected := &expectedRequest{
			method: http.MethodGet,
			values: map[string][]string{
				"limit":    {limit},
				"offset":   {offset},
				"sort":     {sort},
				"protocol": {protocol},
			},
			path: path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"clients": []any{
					map[string]any{
						"name":         client1name,
						"enabled":      client1enabled,
						"protocol":     client1proto,
						"localAddress": client1addr,
					},
					map[string]any{
						"name":         client2name,
						"enabled":      client2enabled,
						"protocol":     client2proto,
						"localAddress": client2addr,
					},
				},
			},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				assert.NoError(t, executeCommand(t, w, command,
					"--limit", limit, "--offset", offset,
					"--sort", sort, "--protocol", protocol,
				),
					"Then it should not return an error",
				)

				assert.Equal(t,
					fmt.Sprintf("Clients:\n")+
						fmt.Sprintf("╭─ Client %q [%s]\n", client1name, status1)+
						fmt.Sprintf("│  ├─ Protocol: %s\n", client1proto)+
						fmt.Sprintf("│  ├─ Local address: %s\n", client1addr)+
						fmt.Sprintf("│  ╰─ Configuration: <empty>\n")+
						fmt.Sprintf("╰─ Client %q [%s]\n", client2name, status2)+
						fmt.Sprintf("   ├─ Protocol: %s\n", client2proto)+
						fmt.Sprintf("   ├─ Local address: %s\n", client2addr)+
						fmt.Sprintf("   ╰─ Configuration: <empty>\n"),
					w.String(),
					"Then it should display the clients",
				)
			})
		})
	})
}

func TestClientGet(t *testing.T) {
	const (
		clientName    = "test_client"
		clientEnabled = true
		clientProto   = "proto"
		clientAddr    = "addr"

		key1 = "key1"
		val1 = "val1"
		key2 = "key2"
		val2 = "val2"

		path = "/api/clients/" + clientName
	)

	status := enabledStatus(clientEnabled)

	responseBody := map[string]any{
		"name":         clientName,
		"enabled":      clientEnabled,
		"protocol":     clientProto,
		"localAddress": clientAddr,
		"protoConfig":  map[string]any{key1: val1, key2: val2},
	}

	t.Run(`Given the client "get" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ClientGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body:   responseBody,
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				assert.NoError(t, executeCommand(t, w, command, clientName),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("── Client %q [%s]\n", clientName, status)+
						fmt.Sprintf("   ├─ Protocol: %s\n", clientProto)+
						fmt.Sprintf("   ├─ Local address: %s\n", clientAddr)+
						fmt.Sprintf("   ╰─ Configuration\n")+
						fmt.Sprintf("      ├─ %s: %s\n", key1, val1)+
						fmt.Sprintf("      ╰─ %s: %s\n", key2, val2),
					w.String(),
					"Then it should display the client",
				)
			})
		})
	})
}

func TestClientUpdate(t *testing.T) {
	const (
		oldClientName     = "old_test_client"
		newClientName     = "new_test_client"
		newClientProtocol = "new_test_protocol"
		newClientAddress  = "9.8.7.6:5"

		newClientConfigKey = "new_key"
		newClientConfigVal = "new_val"

		path     = "/api/clients/" + oldClientName
		location = "/api/clients/" + newClientName
	)

	expectedBody := map[string]any{
		"name":         newClientName,
		"protocol":     newClientProtocol,
		"localAddress": newClientAddress,
		"protoConfig":  map[string]any{newClientConfigKey: newClientConfigVal},
	}

	t.Run(`Given the client "update" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ClientUpdate{}

		expected := &expectedRequest{
			method: http.MethodPatch,
			path:   path,
			body:   expectedBody,
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				assert.NoError(t, executeCommand(t, w, command, oldClientName,
					"--name", newClientName,
					"--protocol", newClientProtocol,
					"--local-address", newClientAddress,
					"--config", newClientConfigKey+":"+newClientConfigVal,
				),
					"Then it should not return an error",
				)

				assert.Equal(t,
					fmt.Sprintf("The client %q was successfully updated.\n", newClientName),
					w.String(),
					"Then it should display a message saying the client was updated",
				)
			})
		})
	})
}

func TestClientDelete(t *testing.T) {
	const (
		clientName = "test_client"
		path       = "/api/clients/" + clientName
	)

	t.Run(`Given the client "delete" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ClientDelete{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusNoContent,
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				assert.NoError(t, executeCommand(t, w, command, clientName),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The client %q was successfully deleted.\n", clientName),
					w.String(),
					"Then it should display a message saying the client was deleted",
				)
			})
		})
	})
}

func TestClientEnable(t *testing.T) {
	const (
		clientName = "test_client"
		path       = "/api/clients/" + clientName + "/enable"
	)

	t.Run(`Given the client "enable" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ClientEnable{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				assert.NoError(t, executeCommand(t, w, command, clientName),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The client %q was successfully enabled.\n", clientName),
					w.String(),
					"Then it should display a message saying the client was enabled",
				)
			})
		})
	})
}

func TestClientDisable(t *testing.T) {
	const (
		clientName = "test_client"
		path       = "/api/clients/" + clientName + "/disable"
	)

	t.Run(`Given the client "disable" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ClientDisable{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				assert.NoError(t, executeCommand(t, w, command, clientName),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The client %q was successfully disabled.\n", clientName),
					w.String(),
					"Then it should display a message saying the client was disabled",
				)
			})
		})
	})
}

func TestClientStart(t *testing.T) {
	const (
		clientName = "test_client"
		path       = "/api/clients/" + clientName + "/start"
	)

	t.Run(`Given the client "start" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ClientStart{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				assert.NoError(t, executeCommand(t, w, command, clientName),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The client %q was successfully started.\n", clientName),
					w.String(),
					"Then it should display a message saying the client was started",
				)
			})
		})
	})
}

func TestClientStop(t *testing.T) {
	const (
		clientName = "test_client"
		path       = "/api/clients/" + clientName + "/stop"
	)

	t.Run(`Given the client "stop" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ClientStop{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				assert.NoError(t, executeCommand(t, w, command, clientName),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The client %q was successfully stopped.\n", clientName),
					w.String(),
					"Then it should display a message saying the client was stopped",
				)
			})
		})
	})
}

func TestClientRestart(t *testing.T) {
	const (
		clientName = "test_client"
		path       = "/api/clients/" + clientName + "/restart"
	)

	t.Run(`Given the client "restart" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ClientRestart{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				assert.NoError(t, executeCommand(t, w, command, clientName),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The client %q was successfully restarted.\n", clientName),
					w.String(),
					"Then it should display a message saying the client was restarted",
				)
			})
		})
	})
}

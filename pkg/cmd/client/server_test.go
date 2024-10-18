package wg

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerGet(t *testing.T) {
	const (
		name    = "foo"
		proto   = "bar"
		enabled = true
		addr    = "localhost:1"
		root    = "root/dir"
		recvDir = "recv/dir"
		sendDir = "send/dir"
		tempDir = "temp/dir"

		key1 = "key1"
		key2 = "key2"
		val1 = "val1"
		val2 = "val2"

		send1    = "send1"
		send2    = "send2"
		receive1 = "receive1"
		receive2 = "receive2"

		cred1 = "cred1"
		cred2 = "cred2"

		path = "/api/servers/" + name
	)

	t.Run(`Testing the server "get" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ServerGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"name":          name,
				"protocol":      proto,
				"enabled":       enabled,
				"address":       addr,
				"credentials":   []string{cred1, cred2},
				"rootDir":       root,
				"receiveDir":    recvDir,
				"sendDir":       sendDir,
				"tmpReceiveDir": tempDir,
				"protoConfig":   map[string]any{key1: val1, key2: val2},
				"authorizedRules": map[string][]string{
					"sending":   {send1, send2},
					"reception": {receive1, receive2},
				},
			},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, name),
					"Then it should not return an error")

				outputData := maps.Clone(result.body)
				outputData["status"] = enabledStatus(enabled)

				assert.Equal(t,
					expectedOutput(t, outputData,
						`‣Server "{{.name}}" [{{.status}}]`,
						`  •Protocol: {{.protocol}}`,
						`  •Address: {{.address}}`,
						`  •Credentials: {{ join .credentials }}`,
						`  •Root directory: {{.rootDir}}`,
						`  •Receive directory: {{.receiveDir}}`,
						`  •Send directory: {{.sendDir}}`,
						`  •Temp receive directory: {{.tmpReceiveDir}}`,
						`  •Configuration:`,
						`    {{- range $option, $value := .protoConfig }}`,
						`    ⁃{{$option}}: {{$value}}`,
						`    {{- end }}`,
						`  •Authorized rules:`,
						`    ⁃Send: {{ join .authorizedRules.sending }}`,
						`    ⁃Receive: {{ join .authorizedRules.reception }}`,
					),
					w.String(),
					"Then it should display the server's information",
				)
			})
		})
	})
}

func TestServerAdd(t *testing.T) {
	const (
		name    = "foo"
		proto   = "bar"
		addr    = "localhost:1"
		root    = "root/dir"
		recvDir = "recv/dir"
		sendDir = "send/dir"
		tempDir = "temp/dir"

		key1 = "key1"
		key2 = "key2"
		val1 = "val1"
		val2 = "val2"

		path     = "/api/servers"
		location = path + "/" + name
	)

	t.Run(`Testing the server "add" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ServerAdd{}

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body: map[string]any{
				"name":          name,
				"protocol":      proto,
				"address":       addr,
				"rootDir":       root,
				"receiveDir":    recvDir,
				"sendDir":       sendDir,
				"tmpReceiveDir": tempDir,
				"protoConfig":   map[string]any{key1: val1, key2: val2},
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
					"--name", name,
					"--protocol", proto,
					"--address", addr,
					"--root-dir", root,
					"--receive-dir", recvDir,
					"--send-dir", sendDir,
					"--tmp-dir", tempDir,
					"--config", key1+":"+val1,
					"--config", key2+":"+val2,
				),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The server %q was successfully added.\n", name),
					w.String(),
					"Then it should display a message saying the server was added",
				)
			})
		})
	})
}

func TestServersList(t *testing.T) {
	const (
		name1    = "foo"
		proto1   = "proto1"
		enabled1 = true
		addr1    = "localhost:1"

		name2    = "bar"
		proto2   = "proto2"
		enabled2 = false
		addr2    = "localhost:2"

		path = "/api/servers"

		sort     = "name+"
		limit    = "10"
		offset   = "5"
		protocol = "proto1"
	)

	t.Run(`Testing the server "list" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ServerList{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
			values: map[string][]string{
				"limit":    {limit},
				"offset":   {offset},
				"sort":     {sort},
				"protocol": {protocol},
			},
		}

		servers := []map[string]any{{
			"name":     name1,
			"protocol": proto1,
			"enabled":  enabled1,
			"address":  addr1,
		}, {
			"name":     name2,
			"protocol": proto2,
			"enabled":  enabled2,
			"address":  addr2,
		}}

		result := &expectedResponse{
			status: http.StatusOK,
			body:   map[string]any{"servers": servers},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--limit", limit, "--offset", offset,
					"--sort", sort, "--protocol", protocol),
					"Then it should not return an error")

				outputData := slices.Clone(servers)
				outputData[0]["status"] = enabledStatus(enabled1)
				outputData[1]["status"] = enabledStatus(enabled2)

				assert.Equal(t,
					expectedOutput(t, outputData,
						`=== Servers ===`,
						`{{- with index . 0 }}`,
						`‣Server "{{.name}}" [{{.status}}]`,
						`  •Protocol: {{.protocol}}`,
						`  •Address: {{.address}}`,
						`  •Credentials: <none>`,
						`  •Configuration: <empty>`,
						`  •Authorized rules:`,
						`    ⁃Send: <none>`,
						`    ⁃Receive: <none>`,
						`{{- end }}`,
						`{{- with index . 1 }}`,
						`‣Server "{{.name}}" [{{.status}}]`,
						`  •Protocol: {{.protocol}}`,
						`  •Address: {{.address}}`,
						`  •Credentials: <none>`,
						`  •Configuration: <empty>`,
						`  •Authorized rules:`,
						`    ⁃Send: <none>`,
						`    ⁃Receive: <none>`,
						`{{- end }}`,
					),
					w.String(),
					"Then it should display the list of servers",
				)
			})
		})
	})
}

func TestServerDelete(t *testing.T) {
	const (
		name = "foobar"

		path = "/api/servers/" + name
	)

	t.Run(`Testing the server "delete" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ServerDelete{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusNoContent}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, name),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The server %q was successfully deleted.\n", name),
					w.String(),
					"Then it should display a message saying the server was deleted",
				)
			})
		})
	})
}

func TestServerUpdate(t *testing.T) {
	const (
		oldName = "foobar"
		name    = "foo"
		proto   = "bar"
		addr    = "localhost:1"
		root    = "root/dir"
		recvDir = "recv/dir"
		sendDir = "send/dir"
		tempDir = "temp/dir"

		key1 = "key1"
		key2 = "key2"
		val1 = "val1"
		val2 = "val2"

		path     = "/api/servers/" + oldName
		location = "/api/servers/" + name
	)

	t.Run(`Testing the server "add" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ServerUpdate{}

		expected := &expectedRequest{
			method: http.MethodPatch,
			path:   path,
			body: map[string]any{
				"name":          name,
				"protocol":      proto,
				"address":       addr,
				"rootDir":       root,
				"receiveDir":    recvDir,
				"sendDir":       sendDir,
				"tmpReceiveDir": tempDir,
				"protoConfig":   map[string]any{key1: val1, key2: val2},
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
					"--name", name,
					"--protocol", proto,
					"--address", addr,
					"--root-dir", root,
					"--receive-dir", recvDir,
					"--send-dir", sendDir,
					"--tmp-dir", tempDir,
					"--config", key1+":"+val1,
					"--config", key2+":"+val2,
					oldName,
				),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The server %q was successfully updated.\n", name),
					w.String(),
					"Then it should display a message saying the server was updated",
				)
			})
		})
	})
}

func TestServerAuthorize(t *testing.T) {
	const (
		name = "foo"
		rule = "push"
		way  = directionSend

		path = "/api/servers/" + name + "/authorize/" + rule + "/" + way
	)

	t.Run(`Testing the server "authorize" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ServerAuthorize{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusOK}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, name, rule, way),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The server %q is now allowed to use the %s rule %q for transfers.\n",
						name, way, rule),
					w.String(),
					"Then it should display a message saying the server can now use the rule",
				)
			})
		})
	})
}

func TestServerRevoke(t *testing.T) {
	const (
		name = "foo"
		rule = "pull"
		way  = directionRecv

		path = "/api/servers/" + name + "/revoke/" + rule + "/" + way
	)

	t.Run(`Testing the server "revoke" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ServerRevoke{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusOK}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, name, rule, way),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The server %q is no longer allowed to use the %s rule %q for transfers.\n",
						name, way, rule),
					w.String(),
					"Then it should display a message saying the server can no longer use the rule",
				)
			})
		})
	})
}

func TestServerEnableDisable(t *testing.T) {
	const (
		name = "test_server"

		enablePath  = "/api/servers/" + name + "/enable"
		disablePath = "/api/servers/" + name + "/disable"
	)

	t.Run(`Given the server "enable" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ServerEnable{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   enablePath,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, name),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The server %q was successfully enabled.\n", name),
					w.String(),
					"Then it should display a message saying the server was enabled",
				)
			})
		})
	})

	t.Run(`Given the server "disable" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ServerDisable{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   disablePath,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, name),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The server %q was successfully disabled.\n", name),
					w.String(),
					"Then it should display a message saying the server was disabled",
				)
			})
		})
	})
}

func TestServerStart(t *testing.T) {
	const (
		name = "test_server"

		path = "/api/servers/" + name + "/start"
	)

	t.Run(`Testing the server "start" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ServerStart{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, name),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The server %q was successfully started.\n", name),
					w.String(),
					"Then it should display a message saying the server was started",
				)
			})
		})
	})
}

func TestServerStop(t *testing.T) {
	const (
		name = "test_server"

		path = "/api/servers/" + name + "/stop"
	)

	t.Run(`Testing the server "stop" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ServerStop{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, name),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The server %q was successfully stopped.\n", name),
					w.String(),
					"Then it should display a message saying the server was stopped",
				)
			})
		})
	})
}

func TestServerRestart(t *testing.T) {
	const (
		name = "test_server"

		path = "/api/servers/" + name + "/restart"
	)

	t.Run(`Testing the server "restart" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &ServerRestart{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, name),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The server %q was successfully restarted.\n", name),
					w.String(),
					"Then it should display a message saying the server was restarted",
				)
			})
		})
	})
}

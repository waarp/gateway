package wg

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

func TestDisplayClient(t *testing.T) {
	Convey("Given a REST client instance", t, func() {
		client := &api.OutClient{
			Name:         "test_client",
			Protocol:     "test_protocol",
			LocalAddress: "client.test.address:port",
			ProtoConfig: map[string]any{
				"key1": "val1", "key2": 2, "key3": true, "long_key4": "long_val4",
			},
		}

		Convey("When displaying the client", func() {
			w := &strings.Builder{}
			DisplayClient(w, client)

			Convey("Then it should have displayed the client's info", func() {
				So(w.String(), ShouldResemble, `── Client "test_client"
   ├─ Protocol: test_protocol
   ├─ Local address: client.test.address:port
   ╰─ Configuration:
      ├─ key1: val1
      ├─ key2: 2
      ├─ key3: true
      ╰─ long_key4: long_val4
`)
			})
		})
	})
}

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

	Convey(`Given the client "add" command`, t, func() {
		w := &strings.Builder{}
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

		Convey("Given a dummy gateway REST interface", func() {
			testServer(expected, result)

			Convey("When executing the command", func() {
				So(executeCommand(w, command,
					"--name", clientName,
					"--protocol", clientProtocol,
					"--local-address", clientAddress,
					"--config", clientConfigStrKey+":"+clientConfigStrVal,
					"--config", clientConfigNumKey+":"+utils.FormatFloat(clientConfigNumVal),
					"--config", clientConfigBoolKey+":"+strconv.FormatBool(clientConfigBoolVal),
				), ShouldBeNil)

				Convey("Then it should display a message saying the client was added", func() {
					So(w.String(), ShouldEqual, "The client "+clientName+
						" was successfully added.\n")
				})
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
	)

	responseBody := map[string]any{
		"clients": []any{
			map[string]any{
				"name":         "cli1",
				"protocol":     "proto1",
				"localAddress": "addr1",
				"protoConfig":  map[string]any{"key1": "val1"},
			},
			map[string]any{
				"name":         "cli2",
				"protocol":     "proto2",
				"localAddress": "addr2",
				"protoConfig":  map[string]any{"key2": "val2"},
			},
		},
	}

	Convey(`Given the client "list" command`, t, func() {
		w := &strings.Builder{}
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
			body:   responseBody,
		}

		Convey("Given a dummy gateway REST interface", func() {
			testServer(expected, result)

			Convey("When executing the command", func() {
				So(executeCommand(w, command,
					"--limit", limit, "--offset", offset,
					"--sort", sort, "--protocol", protocol,
				), ShouldBeNil)

				Convey("Then it should display the clients", func() {
					So(w.String(), ShouldEqual, `Clients:
╭─ Client "cli1"
│  ├─ Protocol: proto1
│  ├─ Local address: addr1
│  ╰─ Configuration:
│     ╰─ key1: val1
╰─ Client "cli2"
   ├─ Protocol: proto2
   ├─ Local address: addr2
   ╰─ Configuration:
      ╰─ key2: val2
`)
				})
			})
		})
	})
}

func TestClientGet(t *testing.T) {
	const (
		clientName = "test_client"
		path       = "/api/clients/" + clientName
	)

	responseBody := map[string]any{
		"name":         clientName,
		"protocol":     "proto",
		"localAddress": "addr",
		"protoConfig":  map[string]any{"key": "val"},
	}

	Convey(`Given the client "get" command`, t, func() {
		w := &strings.Builder{}
		command := &ClientGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body:   responseBody,
		}

		Convey("Given a dummy gateway REST interface", func() {
			testServer(expected, result)

			Convey("When executing the command", func() {
				So(executeCommand(w, command, clientName), ShouldBeNil)

				Convey("Then it should display the client", func() {
					So(w.String(), ShouldEqual, `── Client "test_client"
   ├─ Protocol: proto
   ├─ Local address: addr
   ╰─ Configuration:
      ╰─ key: val
`)
				})
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

	Convey(`Given the client "update" command`, t, func() {
		w := &strings.Builder{}
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

		Convey("Given a dummy gateway REST interface", func() {
			testServer(expected, result)

			Convey("When executing the command", func() {
				So(executeCommand(w, command, oldClientName,
					"--name", newClientName,
					"--protocol", newClientProtocol,
					"--local-address", newClientAddress,
					"--config", newClientConfigKey+":"+newClientConfigVal,
				), ShouldBeNil)

				Convey("Then it should display a message saying the client was updated", func() {
					So(w.String(), ShouldEqual, "The client "+newClientName+
						" was successfully updated.\n")
				})
			})
		})
	})
}

func TestClientDelete(t *testing.T) {
	const (
		clientName = "test_client"
		path       = "/api/clients/" + clientName
	)

	Convey(`Given the client "delete" command`, t, func() {
		w := &strings.Builder{}
		command := &ClientDelete{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusNoContent,
		}

		Convey("Given a dummy gateway REST interface", func() {
			testServer(expected, result)

			Convey("When executing the command", func() {
				So(executeCommand(w, command, clientName), ShouldBeNil)

				Convey("Then it should have deleted the client", func() {
					So(w.String(), ShouldEqual, fmt.Sprintf(
						"The client %s was successfully deleted.\n", clientName))
				})
			})
		})
	})
}

func TestClientEnable(t *testing.T) {
	const (
		clientName = "test_client"
		path       = "/api/clients/" + clientName + "/enable"
	)

	Convey(`Given the client "enable" command`, t, func() {
		w := &strings.Builder{}
		command := &ClientEnable{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		Convey("Given a dummy gateway REST interface", func() {
			testServer(expected, result)

			Convey("When executing the command", func() {
				So(executeCommand(w, command, clientName), ShouldBeNil)

				Convey("Then it should have enables the client", func() {
					So(w.String(), ShouldEqual, fmt.Sprintf(
						"The client %s was successfully enabled.\n", clientName))
				})
			})
		})
	})
}

func TestClientDisable(t *testing.T) {
	const (
		clientName = "test_client"
		path       = "/api/clients/" + clientName + "/disable"
	)

	Convey(`Given the client "disable" command`, t, func() {
		w := &strings.Builder{}
		command := &ClientDisable{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		Convey("Given a dummy gateway REST interface", func() {
			testServer(expected, result)

			Convey("When executing the command", func() {
				So(executeCommand(w, command, clientName), ShouldBeNil)

				Convey("Then it should have disabled the client", func() {
					So(w.String(), ShouldEqual, fmt.Sprintf(
						"The client %s was successfully disabled.\n", clientName))
				})
			})
		})
	})
}

func TestClientStart(t *testing.T) {
	const (
		clientName = "test_client"
		path       = "/api/clients/" + clientName + "/start"
	)

	Convey(`Given the client "start" command`, t, func() {
		w := &strings.Builder{}
		command := &ClientStart{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		Convey("Given a dummy gateway REST interface", func() {
			testServer(expected, result)

			Convey("When executing the command", func() {
				So(executeCommand(w, command, clientName), ShouldBeNil)

				Convey("Then it should have started the client", func() {
					So(w.String(), ShouldEqual, fmt.Sprintf(
						"The client %s was successfully started.\n", clientName))
				})
			})
		})
	})
}

func TestClientStop(t *testing.T) {
	const (
		clientName = "test_client"
		path       = "/api/clients/" + clientName + "/stop"
	)

	Convey(`Given the client "stop" command`, t, func() {
		w := &strings.Builder{}
		command := &ClientStop{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		Convey("Given a dummy gateway REST interface", func() {
			testServer(expected, result)

			Convey("When executing the command", func() {
				So(executeCommand(w, command, clientName), ShouldBeNil)

				Convey("Then it should have stopped the client", func() {
					So(w.String(), ShouldEqual, fmt.Sprintf(
						"The client %s was successfully stopped.\n", clientName))
				})
			})
		})
	})
}

func TestClientRestart(t *testing.T) {
	const (
		clientName = "test_client"
		path       = "/api/clients/" + clientName + "/restart"
	)

	Convey(`Given the client "restart" command`, t, func() {
		w := &strings.Builder{}
		command := &ClientRestart{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		Convey("Given a dummy gateway REST interface", func() {
			testServer(expected, result)

			Convey("When executing the command", func() {
				So(executeCommand(w, command, clientName), ShouldBeNil)

				Convey("Then it should have restarted the client", func() {
					So(w.String(), ShouldEqual, fmt.Sprintf(
						"The client %s was successfully restarted.\n", clientName))
				})
			})
		})
	})
}

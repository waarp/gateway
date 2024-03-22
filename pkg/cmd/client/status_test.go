package wg

import (
	"net/http"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

func TestRequestStatus(t *testing.T) {
	const (
		version = "waarp-gatewayd/dev"
		date    = "Mon, 02 Jan 2006 15:04:05 GMT"
		wgDate  = "Mon, 02 Jan 2006 16:04:05 CET"

		path = "/api/about"
	)

	responseBody := map[string]any{
		"coreServices": []map[string]any{
			{"name": "Core Service 1", "state": "Offline"},
			{"name": "Core Service 2", "state": "Running"},
			{"name": "Core Service 3", "state": "Offline"},
			{"name": "Core Service 4", "state": "Error", "reason": "error n°4"},
			{"name": "Core Service 5", "state": "Error", "reason": "error n°5"},
			{"name": "Core Service 6", "state": "Running"},
		},
		"servers": []map[string]any{
			{"name": "Server 1", "state": "Running"},
			{"name": "Server 2", "state": "Error", "reason": "error n°2"},
			{"name": "Server 3", "state": "Offline"},
			{"name": "Server 4", "state": "Offline"},
			{"name": "Server 5", "state": "Running"},
			{"name": "Server 6", "state": "Error", "reason": "error n°6"},
		},
		"clients": []map[string]any{
			{"name": "Client 1", "state": "Error", "reason": "error n°1"},
			{"name": "Client 2", "state": "Offline"},
			{"name": "Client 3", "state": "Error", "reason": "error n°3"},
			{"name": "Client 4", "state": "Running"},
			{"name": "Client 5", "state": "Running"},
			{"name": "Client 6", "state": "Offline"},
		},
	}

	Convey("Testing the 'status' command", t, func() {
		w := &strings.Builder{}
		command := &Status{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body:   responseBody,
			headers: map[string][]string{
				"Server":       {version},
				"Date":         {date},
				api.DateHeader: {wgDate},
			},
		}

		Convey("Given a dummy gateway REST interface", func(c C) {
			testServer(expected, result)

			Convey("When executing the command", func() {
				So(executeCommand(w, command), ShouldBeNil)

				Convey("Then it should display the services' status", func() {
					So(w.String(), ShouldEqual, "Server info: "+version+"\n"+
						"Local date: "+wgDate+"\n"+
						"\n"+
						"Core services:\n"+
						"[Error]   Core Service 4 (error n°4)\n"+
						"[Error]   Core Service 5 (error n°5)\n"+
						"[Active]  Core Service 2\n"+
						"[Active]  Core Service 6\n"+
						"[Offline] Core Service 1\n"+
						"[Offline] Core Service 3\n"+
						"\n"+
						"Servers:\n"+
						"[Error]   Server 2 (error n°2)\n"+
						"[Error]   Server 6 (error n°6)\n"+
						"[Active]  Server 1\n"+
						"[Active]  Server 5\n"+
						"[Offline] Server 3\n"+
						"[Offline] Server 4\n"+
						"\n"+
						"Clients:\n"+
						"[Error]   Client 1 (error n°1)\n"+
						"[Error]   Client 3 (error n°3)\n"+
						"[Active]  Client 4\n"+
						"[Active]  Client 5\n"+
						"[Offline] Client 2\n"+
						"[Offline] Client 6\n"+
						"\n",
					)
				})
			})
		})
	})
}

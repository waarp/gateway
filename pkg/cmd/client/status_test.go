package wg

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestRequestStatus(t *testing.T) {
	const (
		version = "waarp-gatewayd/dev"
		date    = "Mon, 02 Jan 2006 15:04:05 GMT"
		wgDate  = "Mon, 02 Jan 2006 16:04:05 CET"

		core1 = "Core service 1"
		core2 = "Core service 2"
		core3 = "Core service 3"
		core4 = "Core service 4"
		core5 = "Core service 5"
		core6 = "Core service 6"

		server1 = "Server 1"
		server2 = "Server 2"
		server3 = "Server 3"
		server4 = "Server 4"
		server5 = "Server 5"
		server6 = "Server 6"

		client1 = "Client 1"
		client2 = "Client 2"
		client3 = "Client 3"
		client4 = "Client 4"
		client5 = "Client 5"
		client6 = "Client 6"

		reason1 = "error n°1"
		reason2 = "error n°2"
		reason3 = "error n°3"
		reason4 = "error n°4"
		reason5 = "error n°5"
		reason6 = "error n°6"

		path = "/api/about"
	)

	var (
		offline = utils.StateOffline.String()
		running = utils.StateRunning.String()
		inError = utils.StateError.String()
	)

	responseBody := map[string]any{
		"coreServices": []map[string]any{
			{"name": core1, "state": offline},
			{"name": core2, "state": running},
			{"name": core3, "state": offline},
			{"name": core4, "state": inError, "reason": reason4},
			{"name": core5, "state": inError, "reason": reason5},
			{"name": core6, "state": running},
		},
		"servers": []map[string]any{
			{"name": server1, "state": running},
			{"name": server2, "state": inError, "reason": reason2},
			{"name": server3, "state": offline},
			{"name": server4, "state": offline},
			{"name": server5, "state": running},
			{"name": server6, "state": inError, "reason": reason6},
		},
		"clients": []map[string]any{
			{"name": client1, "state": inError, "reason": reason1},
			{"name": client2, "state": offline},
			{"name": client3, "state": inError, "reason": reason3},
			{"name": client4, "state": running},
			{"name": client5, "state": running},
			{"name": client6, "state": offline},
		},
	}

	t.Run("Testing the 'status' command", func(t *testing.T) {
		w := newTestOutput()
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

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command),
					"Then it should not return an error")

				t.Run("Then it should display the services' status", func(t *testing.T) {
					assert.Equal(t, ""+
						fmt.Sprintf("╭─ Server version: %s\n", version)+
						fmt.Sprintf("├─ Local date: %s\n", wgDate)+
						fmt.Sprintf("├─ Core services\n")+
						fmt.Sprintf("│  ├─ [ERROR]   %s (%s)\n", core4, reason4)+
						fmt.Sprintf("│  ├─ [ERROR]   %s (%s)\n", core5, reason5)+
						fmt.Sprintf("│  ├─ [ACTIVE]  %s\n", core2)+
						fmt.Sprintf("│  ├─ [ACTIVE]  %s\n", core6)+
						fmt.Sprintf("│  ├─ [OFFLINE] %s\n", core1)+
						fmt.Sprintf("│  ╰─ [OFFLINE] %s\n", core3)+
						fmt.Sprintf("├─ Servers\n")+
						fmt.Sprintf("│  ├─ [ERROR]   %s (%s)\n", server2, reason2)+
						fmt.Sprintf("│  ├─ [ERROR]   %s (%s)\n", server6, reason6)+
						fmt.Sprintf("│  ├─ [ACTIVE]  %s\n", server1)+
						fmt.Sprintf("│  ├─ [ACTIVE]  %s\n", server5)+
						fmt.Sprintf("│  ├─ [OFFLINE] %s\n", server3)+
						fmt.Sprintf("│  ╰─ [OFFLINE] %s\n", server4)+
						fmt.Sprintf("╰─ Clients\n")+
						fmt.Sprintf("   ├─ [ERROR]   %s (%s)\n", client1, reason1)+
						fmt.Sprintf("   ├─ [ERROR]   %s (%s)\n", client3, reason3)+
						fmt.Sprintf("   ├─ [ACTIVE]  %s\n", client4)+
						fmt.Sprintf("   ├─ [ACTIVE]  %s\n", client5)+
						fmt.Sprintf("   ├─ [OFFLINE] %s\n", client2)+
						fmt.Sprintf("   ╰─ [OFFLINE] %s\n", client6),
						w.String(),
						"Then it should display the gateway's information",
					)
				})
			})
		})
	})
}

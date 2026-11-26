package wg

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

// The 'status' command shows the environment label the gateway gives in a
// header of its answer, decoded; it says nothing of it when there is none.
func TestStatusEnvironment(t *testing.T) {
	const (
		version = "waarp-gatewayd/dev"
		wgDate  = "Mon, 02 Jan 2006 16:04:05 CET"
	)

	for _, test := range []struct {
		name     string
		header   []string
		expected string
	}{
		{
			name:   "Given a gateway with an environment label",
			header: []string{"Pr%C3%A9-prod%202"},
			expected: "-Server version: " + version + "\n" +
				"-Environment: Pré-prod 2\n" +
				"-Local date: " + wgDate + "\n",
		},
		{
			name:   "Given a gateway without an environment label",
			header: nil,
			expected: "-Server version: " + version + "\n" +
				"-Local date: " + wgDate + "\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			headers := map[string][]string{"Server": {version}, api.DateHeader: {wgDate}}
			if test.header != nil {
				headers[api.EnvironmentHeader] = test.header
			}

			testServer(t,
				&expectedRequest{method: http.MethodGet, path: "/api/about"},
				&expectedResponse{status: http.StatusOK, body: map[string]any{}, headers: headers})

			w := newTestOutput()
			require.NoError(t, executeCommand(t, w, &Status{}))

			assert.Equal(t, test.expected+"-Core services: <none>\n\n-Servers: <none>\n\n-Clients: <none>\n\n", w.String())
		})
	}
}

package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
)

// The environment label of the gateway is given by a header of every answer of
// the API, next to its version and its date: percent-encoded, a header holding
// ASCII characters only. Without a label, the answers are what they were.
func TestServerInfoEnvironment(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		label    conf.EnvironmentLabel
		expected []string
	}{
		{label: "", expected: nil},
		{label: "Production", expected: []string{"Production"}},
		{label: "Pré-prod 2", expected: []string{"Pr%C3%A9-prod%202"}},
	} {
		t.Run(string(test.label), func(t *testing.T) {
			t.Parallel()

			handler := ServerInfoMiddleware(test.label)(http.HandlerFunc(
				func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/about", nil))

			assert.Equal(t, test.expected, w.Header().Values(api.EnvironmentHeader))
			assert.NotEmpty(t, w.Header().Get("Server"))
			assert.NotEmpty(t, w.Header().Get(api.DateHeader))
		})
	}
}

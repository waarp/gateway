package wg

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fwFlow     = "test-flow"
	fwFlow2    = "test-flow-2"
	fwInterval = "5m0s"
	fwPattern  = "*.txt"
	fwPartner  = "test-partner"
	fwAccount  = "test-account"
	fwClient   = "test-client"
	fwRule     = "test-rule"
)

func TestFilewatcherGet(t *testing.T) {
	const path = filewatchersAPIPath + "/" + fwFlow

	w := newTestOutput()
	command := &FilewatcherGet{}

	expected := &expectedRequest{
		method: http.MethodGet,
		path:   path,
	}

	response := &expectedResponse{
		status: http.StatusOK,
		body: map[string]any{
			"flow":             fwFlow,
			"disabled":         false,
			"interval":         fwInterval,
			"pattern":          fwPattern,
			"noDuplicateCheck": false,
			"partner":          fwPartner,
			"account":          fwAccount,
			"client":           fwClient,
			"rule":             fwRule,
		},
	}

	testServer(t, expected, response)

	require.NoError(t, executeCommand(t, w, command, fwFlow),
		"Then it should not return an error")

	assert.Equal(t,
		expectedOutput(t, response.body,
			`-Filewatcher "{{.flow}}"`,
			`  -Disabled: {{.disabled}}`,
			`  -Interval: {{.interval}}`,
			`  -Pattern: {{.pattern}}`,
			`  -No duplicate check: {{.noDuplicateCheck}}`,
			`  -Partner: {{.partner}}`,
			`  -Account: {{.account}}`,
			`  -Client: {{.client}}`,
			`  -Rule: {{.rule}}`,
		),
		w.String(),
		"Then it should display the filewatcher's info",
	)
}

func TestFilewatcherAdd(t *testing.T) {
	const (
		path     = filewatchersAPIPath
		location = path + "/" + fwFlow
	)

	w := newTestOutput()
	command := &FilewatcherAdd{}

	expected := &expectedRequest{
		method: http.MethodPost,
		path:   path,
		body: map[string]any{
			"flow":     fwFlow,
			"disabled": true,
			"interval": fwInterval,
			"pattern":  fwPattern,
			"partner":  fwPartner,
			"account":  fwAccount,
			"client":   fwClient,
			"rule":     fwRule,
		},
	}

	response := &expectedResponse{
		status:  http.StatusCreated,
		headers: map[string][]string{"Location": {location}},
	}

	testServer(t, expected, response)

	require.NoError(t, executeCommand(t, w, command,
		"--flow", fwFlow,
		"--disabled",
		"--interval", fwInterval,
		"--pattern", fwPattern,
		"--partner", fwPartner,
		"--account", fwAccount,
		"--client", fwClient,
		"--rule", fwRule,
	), "Then it should not return an error")

	assert.Equal(t,
		fmt.Sprintf("The filewatcher %q was successfully added.\n", fwFlow),
		w.String(),
		"Then it should display a message saying the filewatcher was added",
	)
}

func TestFilewatcherDelete(t *testing.T) {
	const path = filewatchersAPIPath + "/" + fwFlow

	w := newTestOutput()
	command := &FilewatcherDelete{}

	expected := &expectedRequest{
		method: http.MethodDelete,
		path:   path,
	}

	response := &expectedResponse{status: http.StatusNoContent}

	testServer(t, expected, response)

	require.NoError(t, executeCommand(t, w, command, fwFlow),
		"Then it should not return an error")

	assert.Equal(t,
		fmt.Sprintf("The filewatcher %q was successfully deleted.\n", fwFlow),
		w.String(),
		"Then it should display a message saying the filewatcher was deleted",
	)
}

func TestFilewatcherUpdate(t *testing.T) {
	const (
		newFWFlow    = "updated-flow"
		newFWPattern = "*.csv"

		path     = filewatchersAPIPath + "/" + fwFlow
		location = filewatchersAPIPath + "/" + newFWFlow
	)

	w := newTestOutput()
	command := &FilewatcherUpdate{}

	expected := &expectedRequest{
		method: http.MethodPatch,
		path:   path,
		body: map[string]any{
			"flow":     newFWFlow,
			"disabled": true,
			"pattern":  newFWPattern,
		},
	}

	response := &expectedResponse{
		status:  http.StatusCreated,
		headers: map[string][]string{"Location": {location}},
	}

	testServer(t, expected, response)

	require.NoError(t, executeCommand(t, w, command,
		fwFlow,
		"--flow", newFWFlow,
		"--disabled",
		"--pattern", newFWPattern,
	), "Then it should not return an error")

	assert.Equal(t,
		fmt.Sprintf("The filewatcher %q was successfully updated.\n", newFWFlow),
		w.String(),
		"Then it should display a message saying the filewatcher was updated",
	)
}

func TestFilewatcherStart(t *testing.T) {
	const path = filewatchersAPIPath + "/" + fwFlow + "/start"

	w := newTestOutput()
	command := &FilewatcherStart{}

	expected := &expectedRequest{
		method: http.MethodPut,
		path:   path,
	}

	response := &expectedResponse{status: http.StatusAccepted}

	testServer(t, expected, response)

	require.NoError(t, executeCommand(t, w, command, fwFlow),
		"Then it should not return an error")

	assert.Equal(t,
		fmt.Sprintf("The filewatcher %q was successfully started.\n", fwFlow),
		w.String(),
		"Then it should display a message saying the filewatcher was started",
	)
}

func TestFilewatcherStop(t *testing.T) {
	const path = filewatchersAPIPath + "/" + fwFlow + "/stop"

	w := newTestOutput()
	command := &FilewatcherStop{}

	expected := &expectedRequest{
		method: http.MethodPut,
		path:   path,
	}

	response := &expectedResponse{status: http.StatusAccepted}

	testServer(t, expected, response)

	require.NoError(t, executeCommand(t, w, command, fwFlow),
		"Then it should not return an error")

	assert.Equal(t,
		fmt.Sprintf("The filewatcher %q was successfully stopped.\n", fwFlow),
		w.String(),
		"Then it should display a message saying the filewatcher was stopped",
	)
}

func TestFilewatcherFire(t *testing.T) {
	const path = filewatchersAPIPath + "/" + fwFlow + "/fire"

	w := newTestOutput()
	command := &FilewatcherFire{}

	expected := &expectedRequest{
		method: http.MethodPut,
		path:   path,
	}

	response := &expectedResponse{status: http.StatusAccepted}

	testServer(t, expected, response)

	require.NoError(t, executeCommand(t, w, command, fwFlow),
		"Then it should not return an error")

	assert.Equal(t,
		fmt.Sprintf("The filewatcher %q was successfully fired.\n", fwFlow),
		w.String(),
		"Then it should display a message saying the filewatcher was fired",
	)
}

func TestFilewatcherList(t *testing.T) {
	const (
		path   = filewatchersAPIPath
		sort   = "flow-"
		limit  = "10"
		offset = "5"
	)

	w := newTestOutput()
	command := &FilewatcherList{}

	expected := &expectedRequest{
		method: http.MethodGet,
		path:   path,
		values: url.Values{
			"sort":   {sort},
			"limit":  {limit},
			"offset": {offset},
		},
	}

	fws := []map[string]any{
		{
			"flow":             fwFlow,
			"disabled":         false,
			"interval":         fwInterval,
			"pattern":          fwPattern,
			"noDuplicateCheck": false,
			"partner":          fwPartner,
			"account":          fwAccount,
			"client":           fwClient,
			"rule":             fwRule,
		},
		{
			"flow":             fwFlow2,
			"disabled":         false,
			"interval":         fwInterval,
			"pattern":          fwPattern,
			"noDuplicateCheck": false,
			"partner":          fwPartner,
			"account":          fwAccount,
			"client":           fwClient,
			"rule":             fwRule,
		},
	}

	response := &expectedResponse{
		status: http.StatusOK,
		body:   map[string]any{"filewatchers": fws},
	}

	testServer(t, expected, response)

	require.NoError(t, executeCommand(t, w, command,
		"--sort", sort,
		"--limit", limit,
		"--offset", offset,
	), "Then it should not return an error")

	assert.Equal(t,
		expectedOutput(t, fws,
			`=== Filewatchers ===`,
			`{{- with index . 0 }}`,
			`-Filewatcher "{{.flow}}"`,
			`  -Disabled: {{.disabled}}`,
			`  -Interval: {{.interval}}`,
			`  -Pattern: {{.pattern}}`,
			`  -No duplicate check: {{.noDuplicateCheck}}`,
			`  -Partner: {{.partner}}`,
			`  -Account: {{.account}}`,
			`  -Client: {{.client}}`,
			`  -Rule: {{.rule}}`,
			`{{- end }}`,
			`{{- with index . 1 }}`,
			`-Filewatcher "{{.flow}}"`,
			`  -Disabled: {{.disabled}}`,
			`  -Interval: {{.interval}}`,
			`  -Pattern: {{.pattern}}`,
			`  -No duplicate check: {{.noDuplicateCheck}}`,
			`  -Partner: {{.partner}}`,
			`  -Account: {{.account}}`,
			`  -Client: {{.client}}`,
			`  -Rule: {{.rule}}`,
			`{{- end }}`,
		),
		w.String(),
		"Then it should display the filewatchers",
	)
}

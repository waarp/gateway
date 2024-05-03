package wg

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddressOverrideSet(t *testing.T) {
	const (
		path = "/api/override/addresses"

		target = "localhost"
		redir  = "127.0.0.1"
	)

	t.Run(`Testing the address override "set" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &OverrideAddressSet{}

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body:   map[string]any{target: redir},
		}

		result := &expectedResponse{status: http.StatusCreated}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, "-t", target, "-r", redir))

				assert.Equal(t,
					fmt.Sprintf("The indirection for address %q was successfully set to %q.\n", target, redir),
					w.String(),
					"Then it should display a message saying the indirection was added")
			})
		})
	})
}

func TestAddressOverridesList(t *testing.T) {
	const (
		path = "/api/override/addresses"

		target1 = "localhost"
		redir1  = "127.0.0.1"
		target2 = "waarp.fr"
		redir2  = "1.2.3.4"
	)

	t.Run(`Testing the address override "list" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &OverrideAddressList{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body:   map[string]any{target1: redir1, target2: redir2},
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command))

				assert.Equal(t,
					expectedOutput(t, result.body,
						`=== Address indirections ===`,
						`{{- range $target, $redir := . }}`,
						`‣Address "{{$target}}" redirects to "{{$redir}}"`,
						`{{- end }}`,
					),
					w.String(),
					"Then is should display the indirections",
				)
			})
		})
	})
}

func TestAddressOverrideDelete(t *testing.T) {
	const (
		target = "localhost"
		path   = "/api/override/addresses/" + target
	)

	t.Run(`Testing the address override "delete" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &OverrideAddressDelete{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusNoContent}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, target))

				assert.Equal(t,
					fmt.Sprintf("The indirection for address %q was successfully deleted.\n", target),
					w.String(),
					"Then is should display a message saying the indirection was deleted")
			})
		})
	})
}

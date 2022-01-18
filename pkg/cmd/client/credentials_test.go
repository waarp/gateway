package wg

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestCredentialAdd(t *testing.T) {
	const (
		serverName = "test-server"

		credName = "server_cert"
		credType = auth.TLSCertificate
		credVal1 = testhelpers.LocalhostCert
		credVal2 = testhelpers.LocalhostCert

		path     = "/api/servers/" + serverName + "/credentials"
		location = path + "/" + credName
	)

	t.Run(`Given the credentials "add" command`, func(t *testing.T) {
		command := &credentialAdd{}

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body: map[string]any{
				"name":   credName,
				"type":   credType,
				"value":  credVal1,
				"value2": credVal2,
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			Server = serverName
			defer resetVars()

			testServer(t, expected, result)

			t.Run("When executing the command with values", func(t *testing.T) {
				w := newTestOutput()

				require.NoError(t, executeCommand(t, w, command,
					"--name", credName,
					"--type", credType,
					"--value", credVal1,
					"--secondary-value", credVal2,
				), "Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The %q credential was successfully added.\n", credName),
					w.String(),
					"Then it should display a message saying the credential was added",
				)
			})

			t.Run("When executing the command with file paths", func(t *testing.T) {
				w := newTestOutput()
				file1 := writeFile(t, "val1.file", credVal1)
				file2 := writeFile(t, "val2.file", credVal2)

				require.NoError(t, executeCommand(t, w, command,
					"--name", credName,
					"--type", credType,
					"--value", file1,
					"--secondary-value", file2,
				), "Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The %q credential was successfully added.\n", credName),
					w.String(),
					"Then it should display a message saying the credential was added",
				)
			})
		})
	})
}

func TestCredentialDelete(t *testing.T) {
	const (
		serverName = "test-server"

		credName = "server_cert"
		path     = "/api/servers/" + serverName + "/credentials/" + credName
	)

	t.Run(`Given the credential "delete"" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &credentialDelete{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusNoContent,
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			Server = serverName
			defer resetVars()

			testServer(t, expected, result)

			t.Run("When executing the command with file paths", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, credName),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The %q credential was successfully removed.\n", credName),
					w.String(),
					"Then it should display a message saying the credential was deleted",
				)
			})
		})
	})
}

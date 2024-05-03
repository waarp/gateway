package wg

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloudGet(t *testing.T) {
	const (
		cloudName = "cloud_name"
		cloudType = "cloud_type"
		cloudKey  = "cloud_key"

		opt1, key1 = "opt1", "key1"
		opt2, key2 = "opt2", "key2"

		path = cloudsAPIPath + "/" + cloudName
	)

	t.Run(`Testing the cloud "get" command`, func(t *testing.T) {
		w := &strings.Builder{}
		command := &CloudGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		response := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"name":    cloudName,
				"type":    cloudType,
				"key":     cloudKey,
				"options": map[string]string{opt1: key1, opt2: key2},
			},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, response)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, cloudName),
					"Then it should not return an error")

				assert.Equal(t,
					expectedOutput(t, response.body,
						`‣Cloud instance "{{.name}}" ({{.type}})`,
						`  •Key: {{.key}}`,
						`  •Options:`,
						`    {{- range $option, $value := .options }}`,
						`    ⁃{{$option}}: {{$value}}`,
						`    {{- end }}`,
					),
					w.String(),
					"Then it should display the cloud's info",
				)
			})
		})
	})
}

func TestCloudAdd(t *testing.T) {
	const (
		cloudName   = "cloud_name"
		cloudType   = "cloud_type"
		cloudKey    = "cloud_key"
		cloudSecret = "cloud_secret"

		opt1, key1 = "opt1", "key1"
		opt2, key2 = "opt2", "key2"

		path     = "/api/clouds"
		location = path + "/" + cloudName
	)

	t.Run(`Testing the cloud "add" command`, func(t *testing.T) {
		w := &strings.Builder{}
		command := &CloudAdd{}

		expRequest := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body: map[string]any{
				"name":    cloudName,
				"type":    cloudType,
				"key":     cloudKey,
				"secret":  cloudSecret,
				"options": map[string]any{opt1: key1, opt2: key2},
			},
		}
		expResponse := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		testServer(t, expRequest, expResponse)

		t.Run("When executing the command", func(t *testing.T) {
			require.NoError(t, executeCommand(t, w, command,
				"--name", cloudName,
				"--type", cloudType,
				"--key", cloudKey,
				"--secret", cloudSecret,
				"--options", fmt.Sprintf("%s:%s", opt1, key1),
				"--options", fmt.Sprintf("%s:%s", opt2, key2),
			),
				"Then it should not return an error",
			)

			assert.Equal(t,
				fmt.Sprintf("The cloud instance %q was successfully added.\n", cloudName),
				w.String(),
				"Then it should display a message saying the cloud instance was added",
			)
		})
	})
}

func TestCloudDelete(t *testing.T) {
	const (
		cloudName = "cloud_name"

		path = "/api/clouds/" + cloudName
	)

	t.Run(`Testing the cloud "delete" command`, func(t *testing.T) {
		w := &strings.Builder{}
		command := &CloudDelete{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		response := &expectedResponse{status: http.StatusNoContent}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, response)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, cloudName),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The cloud instance %q was successfully deleted.\n", cloudName),
					w.String(),
					"Then it should display a message saying the cloud instance was deleted",
				)
			})
		})
	})
}

func TestCloudUpdate(t *testing.T) {
	const (
		oldCloudName = "old_cloud_name"

		cloudName   = "cloud_name"
		cloudType   = "cloud_type"
		cloudKey    = "cloud_key"
		cloudSecret = "cloud_secret"

		opt1, key1 = "opt1", "key1"
		opt2, key2 = "opt2", "key2"

		path     = "/api/clouds/" + oldCloudName
		location = "/api/clouds/" + cloudName
	)

	t.Run(`Testing the cloud "update" command`, func(t *testing.T) {
		w := &strings.Builder{}
		command := &CloudUpdate{}

		expected := &expectedRequest{
			method: http.MethodPatch,
			path:   path,
			body: map[string]any{
				"name":    cloudName,
				"type":    cloudType,
				"key":     cloudKey,
				"secret":  cloudSecret,
				"options": map[string]any{opt1: key1, opt2: key2},
			},
		}
		response := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, response)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					oldCloudName,
					"--name", cloudName,
					"--type", cloudType,
					"--key", cloudKey,
					"--secret", cloudSecret,
					"--options", fmt.Sprintf("%s:%s", opt1, key1),
					"--options", fmt.Sprintf("%s:%s", opt2, key2),
				),
					"Then it should not return an error",
				)

				assert.Equal(t,
					fmt.Sprintf("The cloud instance %q was successfully updated.\n", cloudName),
					w.String(),
					"Then it should display a message saying the cloud instance was updated",
				)
			})
		})
	})
}

func TestCloudList(t *testing.T) {
	const (
		path = "/api/clouds"

		sort   = "name-"
		limit  = "10"
		offset = "5"

		cloud1     = "cloud1"
		cloud1Type = "cloud1_type"

		cloud2     = "cloud2"
		cloud2Type = "cloud2_type"
	)

	t.Run(`Testing the cloud "list" command`, func(t *testing.T) {
		w := &strings.Builder{}
		command := &CloudList{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
			values: url.Values{
				"sort":   {sort},
				"limit":  {limit},
				"offset": {offset},
			},
		}

		clouds := []map[string]any{
			{"name": cloud1, "type": cloud1Type},
			{"name": cloud2, "type": cloud2Type},
		}

		response := &expectedResponse{
			status: http.StatusOK,
			body:   map[string]any{"clouds": clouds},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, response)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--sort", sort,
					"--limit", limit,
					"--offset", offset,
				),
					"Then it should not return an error",
				)

				assert.Equal(t,
					expectedOutput(t, clouds,
						`=== Cloud instances ===`,
						`{{- with index . 0 }}`,
						`‣Cloud instance "{{.name}}" ({{.type}})`,
						`  •Key: <none>`,
						`  •Options: <none>`,
						`{{- end }}`,
						`{{- with index . 1 }}`,
						`‣Cloud instance "{{.name}}" ({{.type}})`,
						`  •Key: <none>`,
						`  •Options: <none>`,
						`{{- end }}`,
					),
					w.String(),
					"Then it should display the clients",
				)
			})
		})
	})
}

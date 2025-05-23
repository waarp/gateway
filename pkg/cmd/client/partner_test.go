package wg

import (
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPartnerGet(t *testing.T) {
	const (
		partner = "foobar"
		proto   = "proto"
		addr    = "1.2.3.4"
		key1    = "key1"
		val1    = "val1"
		key2    = "key2"
		val2    = "val2"
		send1   = "send1"
		send2   = "send2"
		rcv1    = "rcv1"
		rcv2    = "rcv2"
		cred1   = "cred1"
		cred2   = "cred2"

		path = "/api/partners/" + partner
	)

	t.Run(`Testing the partner "get" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &PartnerGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"name":        partner,
				"protocol":    proto,
				"address":     addr,
				"credentials": []string{cred1, cred2},
				"protoConfig": map[string]any{key1: val1, key2: val2},
				"authorizedRules": map[string]any{
					"sending":   []string{send1, send2},
					"reception": []string{rcv1, rcv2},
				},
			},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, partner),
					"Then it should not return an error")

				outputData := maps.Clone(result.body)

				assert.Equal(t,
					expectedOutput(t, outputData,
						`‣Partner "{{.name}}"`,
						`  •Protocol: {{.protocol}}`,
						`  •Address: {{.address}}`,
						`  •Credentials: {{ join .credentials }}`,
						`  •Configuration:`,
						`    {{- range $key, $value := .protoConfig }}`,
						`    ⁃{{$key}}: {{$value}}`,
						`    {{- end }}`,
						`  •Authorized rules:`,
						`    ⁃Send: {{ join .authorizedRules.sending }}`,
						`    ⁃Receive: {{ join .authorizedRules.reception }}`,
					),
					w.String(),
					"Then it should display the partner's details",
				)
			})
		})
	})
}

func TestPartnerAdd(t *testing.T) {
	const (
		partner = "foobar"
		proto   = "proto"
		addr    = "1.2.3.4"
		key     = "key"
		val     = "val"

		path     = "/api/partners"
		location = path + "/" + partner
	)

	t.Run(`Testing the partner "add" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &PartnerAdd{}

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body: map[string]any{
				"name":        partner,
				"protocol":    proto,
				"address":     addr,
				"protoConfig": map[string]any{key: val},
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
					"--name", partner, "--protocol", proto, "--address", addr,
					"--config", key+":"+val),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The partner %q was successfully added.\n", partner),
					w.String(),
					"Then it should display a message saying the partner was added",
				)
			})
		})
	})
}

func TestPartnersList(t *testing.T) {
	const (
		path = "/api/partners"

		sort     = "name+"
		limit    = "10"
		offset   = "5"
		protocol = "proto1"

		partner1 = "partner1"
		proto1   = "proto1"
		addr1    = "1.2.3.4"

		partner2 = "partner2"
		proto2   = "proto2"
		addr2    = "4.3.2.1"
	)

	t.Run(`Testing the partner "list" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &PartnerList{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
			values: url.Values{
				"sort":     []string{sort},
				"limit":    []string{limit},
				"offset":   []string{offset},
				"protocol": []string{protocol},
			},
		}

		partners := []map[string]any{{
			"name":     partner1,
			"protocol": proto1,
			"address":  addr1,
		}, {
			"name":     partner2,
			"protocol": proto2,
			"address":  addr2,
		}}

		result := &expectedResponse{
			status: http.StatusOK,
			body:   map[string]any{"partners": partners},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--sort", sort, "--limit", limit,
					"--offset", offset, "--protocol", protocol),
					"Then it should not return an error")

				outputData := slices.Clone(partners)

				assert.Equal(t,
					expectedOutput(t, outputData,
						`=== Partners ===`,
						`{{- with index . 0 }}`,
						`‣Partner "{{.name}}"`,
						`  •Protocol: {{.protocol}}`,
						`  •Address: {{.address}}`,
						`  •Credentials: <none>`,
						`  •Configuration: <empty>`,
						`  •Authorized rules:`,
						`    ⁃Send: <none>`,
						`    ⁃Receive: <none>`,
						`{{- end }}`,
						`{{- with index . 1 }}`,
						`‣Partner "{{.name}}"`,
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
					"Then it should display the partners' info",
				)
			})
		})
	})
}

func TestPartnerDelete(t *testing.T) {
	const (
		partner = "foobar"
		path    = "/api/partners/" + partner
	)

	t.Run(`Testing the partner "delete" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &PartnerDelete{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusNoContent}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, partner),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The partner %q was successfully deleted.\n", partner),
					w.String(),
					"Then it should display a message saying the partner was deleted")
			})
		})
	})
}

func TestPartnerUpdate(t *testing.T) {
	const (
		oldName = "barfoo"
		partner = "foobar"
		proto   = "proto"
		addr    = "1.2.3.4"
		key     = "key"
		val     = "val"

		path     = "/api/partners/" + oldName
		location = "/api/partners/" + partner
	)

	t.Run(`Testing the partner "update" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &PartnerUpdate{}

		expected := &expectedRequest{
			method: http.MethodPatch,
			path:   path,
			body: map[string]any{
				"name":        partner,
				"protocol":    proto,
				"address":     addr,
				"protoConfig": map[string]any{key: val},
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
					"--name", partner, "--protocol", proto, "--address", addr,
					"--config", key+":"+val,
					oldName),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The partner %q was successfully updated.\n", partner),
					w.String(),
					"Then it should display a message saying the partner was updated",
				)
			})
		})
	})
}

func TestPartnerAuthorize(t *testing.T) {
	const (
		partner = "foobar"
		rule    = "push"
		way     = directionSend

		path = "/api/partners/" + partner + "/authorize/" + rule + "/" + way
	)

	t.Run(`Testing the partner "authorize" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &PartnerAuthorize{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusOK}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, partner, rule, way),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The partner %q is now allowed to use the %s rule %q for transfers.\n",
						partner, way, rule),
					w.String(),
					"Then it should display a message saying the partner can now use the rule",
				)
			})
		})
	})
}

func TestPartnerRevoke(t *testing.T) {
	const (
		partner = "foobar"
		rule    = "pull"
		way     = directionRecv

		path = "/api/partners/" + partner + "/revoke/" + rule + "/" + way
	)

	t.Run(`Testing the partner "revoke" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &PartnerRevoke{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusOK}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, partner, rule, way),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The partner %q is no longer allowed to use the %s rule %q for transfers.\n",
						partner, way, rule),
					w.String(),
					"Then it should display a message saying the partner can no longer use the rule",
				)
			})
		})
	})
}

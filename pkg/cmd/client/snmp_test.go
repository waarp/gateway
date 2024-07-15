package wg

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
)

func TestSnmpMonitorAdd(t *testing.T) {
	const (
		monitorName      = "snmp-monitor"
		monitorVersion   = "SNMPv3"
		monitorAddress   = "1.2.3.4:5"
		monitorCommunity = "public"

		monitorNotifType  = "inform"
		monitorUseInforms = true

		monitorV3SecLevel      = snmp.V3SecurityAuthPriv
		monitorContextName     = "wg-ctx"
		monitorContextEngineID = "123456"

		monitorAuthEngineID   = "654321"
		monitorAuthUsername   = "toto"
		monitorAuthProtocol   = "SHA"
		monitorAuthPassphrase = "sesame"

		monitorPrivProtocol   = "DES"
		monitorPrivPassphrase = "foobar"

		path     = "/api/snmp/monitors"
		location = path + "/" + monitorName
	)

	t.Run(`Testing the SNMP monitor "add" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &SnmpMonitorAdd{}

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body: map[string]any{
				"name":            monitorName,
				"version":         monitorVersion,
				"udpAddress":      monitorAddress,
				"community":       monitorCommunity,
				"useInforms":      monitorUseInforms,
				"contextName":     monitorContextName,
				"contextEngineID": monitorContextEngineID,
				"snmpv3Security":  monitorV3SecLevel,
				"authEngineID":    monitorAuthEngineID,
				"authUsername":    monitorAuthUsername,
				"authProtocol":    monitorAuthProtocol,
				"authPassphrase":  monitorAuthPassphrase,
				"privProtocol":    monitorPrivProtocol,
				"privPassphrase":  monitorPrivPassphrase,
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
					"--name", monitorName,
					"--version", monitorVersion,
					"--address", monitorAddress,
					"--community", monitorCommunity,
					"--notif-type", monitorNotifType,
					"--context-name", monitorContextName,
					"--context-engine-id", monitorContextEngineID,
					"--snmpv3-sec", monitorV3SecLevel,
					"--auth-engine-id", monitorAuthEngineID,
					"--auth-username", monitorAuthUsername,
					"--auth-protocol", monitorAuthProtocol,
					"--auth-passphrase", monitorAuthPassphrase,
					"--priv-protocol", monitorPrivProtocol,
					"--priv-passphrase", monitorPrivPassphrase,
				),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The SNMP monitor %q was successfully added.\n", monitorName),
					w.String(),
					"Then it should display a message saying the monitor was added",
				)
			})
		})
	})
}

func TestSnmpMonitorList(t *testing.T) {
	const (
		monitor1Name       = "snmp-monitor-1"
		monitor1Version    = "SNMPv2"
		monitor1Address    = "1.2.3.4:5"
		monitor1Community  = "public"
		monitor1UseInforms = true

		monitor2Name       = "snmp-monitor-2"
		monitor2Version    = "SNMPv3"
		monitor2Address    = "2.3.4.5:6"
		monitor2Community  = "waarp"
		monitor2UseInforms = false
		monitor2v3Security = snmp.V3SecurityNoAuthNoPriv
		monitor2Username   = "toto"

		sort   = "name+"
		limit  = "10"
		offset = "5"

		path = "/api/snmp/monitors"
	)

	t.Run(`Testing the SNMP monitor "list" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &SnmpMonitorList{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
			values: url.Values{
				"limit":  {limit},
				"offset": {offset},
				"sort":   {sort},
			},
		}

		monitors := []map[string]any{{
			"name":       monitor1Name,
			"version":    monitor1Version,
			"udpAddress": monitor1Address,
			"community":  monitor1Community,
			"useInforms": monitor1UseInforms,
		}, {
			"name":           monitor2Name,
			"version":        monitor2Version,
			"udpAddress":     monitor2Address,
			"community":      monitor2Community,
			"useInforms":     monitor2UseInforms,
			"snmpv3Security": monitor2v3Security,
			"authUsername":   monitor2Username,
		}}

		result := &expectedResponse{
			status: http.StatusOK,
			body:   map[string]any{"monitors": monitors},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--limit", limit,
					"--offset", offset,
					"--sort", sort),
					"Then it should not return an error")

				outputData := slices.Clone(monitors)
				outputData[0]["notifType"] = snmpNotifType(monitor1UseInforms)
				outputData[1]["notifType"] = snmpNotifType(monitor2UseInforms)

				assert.Equal(t,
					expectedOutput(t, outputData,
						`=== SNMP monitors ===`,
						`{{- with index . 0 }}`,
						`‣SNMP monitor "{{.name}}"`,
						`  •SNMP version: {{.version}}`,
						`  •UDP address: {{.udpAddress}}`,
						`  •Community: {{.community}}`,
						`  •Notification type: {{.notifType}}`,
						`{{- end }}`,
						`{{- with index . 1 }}`,
						`‣SNMP monitor "{{.name}}"`,
						`  •SNMP version: {{.version}}`,
						`  •UDP address: {{.udpAddress}}`,
						`  •Community: {{.community}}`,
						`  •Notification type: {{.notifType}}`,
						`  •SNMPv3 security: {{.snmpv3Security}}`,
						`  •SNMPv3 username: {{.authUsername}}`,
						`{{- end }}`,
					),
					w.String(),
					"Then it should display the monitors' details",
				)
			})
		})
	})
}

func TestSnmpMonitorGet(t *testing.T) {
	const (
		monitorName       = "snmp-monitor"
		monitorVersion    = "SNMPv3"
		monitorAddress    = "1.2.3.4:5"
		monitorCommunity  = "public"
		monitorUseInforms = true

		monitorV3Security      = snmp.V3SecurityAuthPriv
		monitorContextName     = "snmp-context"
		monitorContextEngineID = "123456"

		monitorAuthEngineID   = "654321"
		monitorAuthUsername   = "toto"
		monitorAuthProtocol   = snmp.AuthSHA
		monitorAuthPassphrase = "sesame"
		monitorPrivProtocol   = snmp.PrivDES
		monitorPrivPassphrase = "foobar"

		path = "/api/snmp/monitors/" + monitorName
	)

	t.Run(`Testing the SNMP monitor "get" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &SnmpMonitorGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"name":            monitorName,
				"version":         monitorVersion,
				"udpAddress":      monitorAddress,
				"community":       monitorCommunity,
				"useInforms":      monitorUseInforms,
				"snmpv3Security":  monitorV3Security,
				"contextName":     monitorContextName,
				"contextEngineID": monitorContextEngineID,
				"authEngineID":    monitorAuthEngineID,
				"authUsername":    monitorAuthUsername,
				"authProtocol":    monitorAuthProtocol,
				"authPassphrase":  monitorAuthPassphrase,
				"privProtocol":    monitorPrivProtocol,
				"privPassphrase":  monitorPrivPassphrase,
			},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, monitorName),
					"Then it should not return an error")

				outputData := maps.Clone(result.body)
				outputData["notifType"] = snmpNotifType(monitorUseInforms)

				assert.Equal(t,
					expectedOutput(t, outputData,
						`‣SNMP monitor "{{.name}}"`,
						`  •SNMP version: {{.version}}`,
						`  •UDP address: {{.udpAddress}}`,
						`  •Community: {{.community}}`,
						`  •Notification type: {{.notifType}}`,
						`  •SNMPv3 security: {{.snmpv3Security}}`,
						`  •SNMPv3 context name: {{.contextName}}`,
						`  •SNMPv3 context engine ID: {{.contextEngineID}}`,
						`  •SNMPv3 auth engine ID: {{.authEngineID}}`,
						`  •SNMPv3 username: {{.authUsername}}`,
						`  •SNMPv3 authentication protocol: {{.authProtocol}}`,
						`  •SNMPv3 authentication passphrase: {{.authPassphrase}}`,
						`  •SNMPv3 privacy protocol: {{.privProtocol}}`,
						`  •SNMPv3 privacy passphrase: {{.privPassphrase}}`,
					),
					w.String(),
					"Then it should display the monitor's details",
				)
			})
		})
	})
}

func TestSnmpMonitorUpdate(t *testing.T) {
	const (
		oldMonitorName = "old-snmp-monitor"

		monitorName      = "snmp-monitor"
		monitorVersion   = "SNMPv2"
		monitorAddress   = "1.2.3.4:5"
		monitorCommunity = "public"
		monitorNotifType = "trap"

		path     = "/api/snmp/monitors/" + oldMonitorName
		location = "/api/snmp/monitors/" + monitorName
	)

	t.Run(`Testing the SNMP monitor "update command`, func(t *testing.T) {
		w := newTestOutput()
		command := &SnmpMonitorUpdate{}

		expected := &expectedRequest{
			method: http.MethodPatch,
			path:   path,
			body: map[string]any{
				"name":       monitorName,
				"version":    monitorVersion,
				"udpAddress": monitorAddress,
				"community":  monitorCommunity,
				"useInforms": monitorNotifType == "inform",
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
					oldMonitorName,
					"--name", monitorName,
					"--version", monitorVersion,
					"--address", monitorAddress,
					"--community", monitorCommunity,
					"--notif-type", monitorNotifType),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The SNMP monitor %q was successfully updated.\n", monitorName),
					w.String(),
					"Then it should display a message saying the monitor was updated",
				)
			})
		})
	})
}

func TestSnmpMonitorDelete(t *testing.T) {
	const (
		monitor = "snmp-monitor"
		path    = "/api/snmp/monitors/" + monitor
	)

	t.Run(`Testing the SNMP monitor "delete" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &SnmpMonitorDelete{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusNoContent}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, monitor),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The SNMP monitor %q was successfully deleted.\n", monitor),
					w.String(),
					"Then it should display a message saying the monitor was deleted")
			})
		})
	})
}

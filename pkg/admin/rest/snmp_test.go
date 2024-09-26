package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestAddSNMPMonitor(t *testing.T) {
	const (
		monitorName            = "snmp-monitor"
		monitorVersion         = "SNMPv3"
		monitorAddress         = "1.2.3.4:5"
		monitorCommunity       = "public"
		monitorUseInforms      = true
		monitorV3Security      = snmp.V3SecurityAuthPriv
		monitorContextName     = "ctxName"
		monitorContextEngineID = "123456"
		monitorAuthEngineID    = "654321"
		monitorAuthUsername    = "toto"
		monitorAuthProtocol    = snmp.AuthSHA
		monitorAuthPassphrase  = "opensesame"
		monitorPrivProtocol    = snmp.PrivAES
		monitorPrivPassphrase  = "foobar"
	)

	t.Run("When creating a valid new SNMP monitor", func(t *testing.T) {
		logger := testhelpers.GetTestLogger(t)
		db := dbtest.TestDatabase(t)
		handle := addSnmpMonitor(logger, db)

		reqBody := bytes.Buffer{}
		encoder := json.NewEncoder(&reqBody)

		input := map[string]any{
			"name":            monitorName,
			"version":         monitorVersion,
			"udpAddress":      monitorAddress,
			"community":       monitorCommunity,
			"useInforms":      monitorUseInforms,
			"contextName":     monitorContextName,
			"contextEngineID": monitorContextEngineID,
			"snmpv3Security":  monitorV3Security,
			"authEngineID":    monitorAuthEngineID,
			"authUsername":    monitorAuthUsername,
			"authProtocol":    monitorAuthProtocol,
			"authPassphrase":  monitorAuthPassphrase,
			"privProtocol":    monitorPrivProtocol,
			"privPassphrase":  monitorPrivPassphrase,
		}
		require.NoError(t, encoder.Encode(input))

		expectedDBMonitor := snmp.MonitorConfig{
			ID:              1,
			Name:            monitorName,
			Owner:           conf.GlobalConfig.GatewayName,
			Version:         monitorVersion,
			UDPAddress:      monitorAddress,
			Community:       monitorCommunity,
			UseInforms:      monitorUseInforms,
			ContextName:     monitorContextName,
			ContextEngineID: monitorContextEngineID,
			SNMPv3Security:  monitorV3Security,
			AuthEngineID:    monitorAuthEngineID,
			AuthUsername:    monitorAuthUsername,
			AuthProtocol:    monitorAuthProtocol,
			AuthPassphrase:  monitorAuthPassphrase,
			PrivProtocol:    monitorPrivProtocol,
			PrivPassphrase:  monitorPrivPassphrase,
		}
		expectedLoc := path.Join(SNMPMonitorsPath, monitorName)

		req := httptest.NewRequest(http.MethodPost, SNMPMonitorsPath, &reqBody)
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code,
			`Then the response code should be "201 Created"`)
		assert.Empty(t, w.Body.String(), `Then the response body should be empty`)
		assert.Equal(t, expectedLoc, w.Header().Get("Location"),
			`Then the response location header should have been set correctly`)
		assert.Empty(t, w.Body.String(), `Then the response body should be empty`)

		var dbMonitor snmp.MonitorConfig

		require.NoError(t, db.Get(&dbMonitor, "name=? AND owner=?", monitorName,
			conf.GlobalConfig.GatewayName).Run())
		assert.Equal(t, expectedDBMonitor, dbMonitor,
			`Then the SNMP monitor should have been inserted in the database`)
	})
}

func TestListSNMPMonitors(t *testing.T) {
	t.Run("When retrieving a list of SNMP monitors", func(t *testing.T) {
		logger := testhelpers.GetTestLogger(t)
		db := dbtest.TestDatabase(t)
		handle := listSnmpMonitors(logger, db)

		dbMonitor1 := snmp.MonitorConfig{
			Name:       "snmp-monitor-1",
			Version:    "SNMPv2",
			UDPAddress: "1.2.3.4:5",
			Community:  "public",
		}
		require.NoError(t, db.Insert(&dbMonitor1).Run())

		dbMonitor2 := snmp.MonitorConfig{
			Name:         "snmp-monitor-2",
			Version:      "SNMPv3",
			UDPAddress:   "9.8.7.6:5",
			Community:    "waarp",
			UseInforms:   true,
			AuthUsername: "gw",
		}
		require.NoError(t, db.Insert(&dbMonitor2).Run())

		expectedResponse := marshal(t, map[string]any{
			"snmpMonitors": []any{
				map[string]any{
					"name":       dbMonitor1.Name,
					"version":    dbMonitor1.Version,
					"udpAddress": dbMonitor1.UDPAddress,
					"community":  dbMonitor1.Community,
					"useInforms": dbMonitor1.UseInforms,
				},
				map[string]any{
					"name":           dbMonitor2.Name,
					"version":        dbMonitor2.Version,
					"udpAddress":     dbMonitor2.UDPAddress,
					"community":      dbMonitor2.Community,
					"useInforms":     dbMonitor2.UseInforms,
					"authUsername":   dbMonitor2.AuthUsername,
					"snmpv3Security": dbMonitor2.SNMPv3Security,
				},
			},
		})

		req := httptest.NewRequest(http.MethodGet, SNMPMonitorsPath, nil)
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, `Then the response code should be "200 OK"`)
		assert.JSONEq(t, expectedResponse, w.Body.String(),
			`Then the list of SNMP monitors should have been returned`)
	})
}

func TestGetSnmpMonitor(t *testing.T) {
	t.Run("When retrieving an existing SNMP monitor", func(t *testing.T) {
		logger := testhelpers.GetTestLogger(t)
		db := dbtest.TestDatabase(t)
		handle := getSnmpMonitor(logger, db)

		dbMonitor := snmp.MonitorConfig{
			Name:            "snmp-monitor",
			Owner:           "",
			Version:         "SNMPv3",
			UDPAddress:      "1.2.3.4:5",
			UseInforms:      false,
			SNMPv3Security:  snmp.V3SecurityAuthPriv,
			ContextName:     "wg-ctx",
			ContextEngineID: "123",
			AuthEngineID:    "567",
			AuthUsername:    "toto",
			AuthProtocol:    snmp.AuthMD5,
			AuthPassphrase:  "sesame",
			PrivProtocol:    snmp.PrivDES,
			PrivPassphrase:  "foobar",
		}
		require.NoError(t, db.Insert(&dbMonitor).Run())

		expectedResponse := marshal(t, map[string]any{
			"name":            dbMonitor.Name,
			"version":         dbMonitor.Version,
			"udpAddress":      dbMonitor.UDPAddress,
			"useInforms":      dbMonitor.UseInforms,
			"contextName":     dbMonitor.ContextName,
			"contextEngineID": dbMonitor.ContextEngineID,
			"snmpv3Security":  dbMonitor.SNMPv3Security,
			"authEngineID":    dbMonitor.AuthEngineID,
			"authUsername":    dbMonitor.AuthUsername,
			"authProtocol":    dbMonitor.AuthProtocol,
			"authPassphrase":  dbMonitor.AuthPassphrase,
			"privProtocol":    dbMonitor.PrivProtocol,
			"privPassphrase":  dbMonitor.PrivPassphrase,
		})

		req := makeRequest(t, http.MethodGet, nil, SNMPMonitorPath, dbMonitor.Name)
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, `Then the response code should be "200 OK"`)
		assert.JSONEq(t, expectedResponse, w.Body.String(),
			`Then the list of SNMP monitors should have been returned`)
	})
}

func TestUpdateSnmpMonitor(t *testing.T) {
	const (
		newMonitorName       = "snmpv3-monitor"
		newMonitorVersion    = "SNMPv3"
		newMonitorUseInforms = true
		newAuthUsername      = "toto"
		newV3Security        = snmp.V3SecurityNoAuthNoPriv
	)

	t.Run("When updating an existing SNMP monitor", func(t *testing.T) {
		logger := testhelpers.GetTestLogger(t)
		db := dbtest.TestDatabase(t)
		handle := updateSnmpMonitor(logger, db)

		reqBody := bytes.Buffer{}
		encoder := json.NewEncoder(&reqBody)

		oldDBMonitor := snmp.MonitorConfig{
			Name:       "snmp-monitor",
			Version:    "SNMPv2",
			UDPAddress: "1.2.3.4:5",
			Community:  "public",
		}
		require.NoError(t, db.Insert(&oldDBMonitor).Run())

		input := map[string]any{
			"name":         newMonitorName,
			"version":      newMonitorVersion,
			"useInforms":   newMonitorUseInforms,
			"authUsername": newAuthUsername,
		}
		require.NoError(t, encoder.Encode(input))

		expectedDBMonitor := snmp.MonitorConfig{
			ID:             oldDBMonitor.ID,
			Owner:          oldDBMonitor.Owner,
			Name:           newMonitorName,
			UDPAddress:     oldDBMonitor.UDPAddress,
			Version:        newMonitorVersion,
			Community:      oldDBMonitor.Community,
			UseInforms:     newMonitorUseInforms,
			SNMPv3Security: newV3Security,
			AuthUsername:   newAuthUsername,
		}
		expectedLoc := path.Join(SNMPMonitorsPath, expectedDBMonitor.Name)

		req := makeRequest(t, http.MethodPatch, &reqBody, SNMPMonitorPath, oldDBMonitor.Name)
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code,
			`Then the response code should be "201 Created"`)
		assert.Empty(t, w.Body.String(), `Then the response body should be empty`)
		assert.Equal(t, expectedLoc, w.Header().Get("Location"),
			`Then the response location header should have been set correctly`)
		assert.Empty(t, w.Body.String(), `Then the response body should be empty`)

		var dbMonitor snmp.MonitorConfig

		require.NoError(t, db.Get(&dbMonitor, "name=? AND owner=?", newMonitorName,
			conf.GlobalConfig.GatewayName).Run())
		assert.Equal(t, expectedDBMonitor, dbMonitor,
			`Then the SNMP monitor should have been updated in the database`)
	})
}

func TestDeleteSnmpMonitor(t *testing.T) {
	t.Run("When deleting an existing SNMP monitor", func(t *testing.T) {
		logger := testhelpers.GetTestLogger(t)
		db := dbtest.TestDatabase(t)
		handle := deleteSnmpMonitor(logger, db)

		dbMonitor := snmp.MonitorConfig{
			Name:       "snmp-monitor",
			Version:    "SNMPv2",
			UDPAddress: "1.2.3.4:5",
			Community:  "public",
		}
		require.NoError(t, db.Insert(&dbMonitor).Run())

		req := makeRequest(t, http.MethodDelete, nil, SNMPMonitorPath, dbMonitor.Name)
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code, `Then the response code should be "204 No Content"`)
		assert.Zero(t, w.Body.String(), `Then the response body should be blank`)

		var dbCheck snmp.MonitorConfig
		var dbErr *database.NotFoundError

		require.ErrorAs(t, db.Get(&dbCheck, "true").Run(), &dbErr,
			`Then the SNMP monitor should have been deleted from the database`)
	})
}

func setupSnmpService(tb testing.TB, db *database.DB) {
	tb.Helper()

	snmp.GlobalService = &snmp.Service{DB: db}
	require.NoError(tb, snmp.GlobalService.Start())

	tb.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		require.NoError(tb, snmp.GlobalService.Stop(ctx))
		snmp.GlobalService = nil
	})
}

func TestGetSNMPServer(t *testing.T) {
	t.Run("When retrieving the existing SNMP server config", func(t *testing.T) {
		logger := testhelpers.GetTestLogger(t)
		db := dbtest.TestDatabase(t)
		handle := getSnmpService(logger, db)

		dbSnmpConf := snmp.ServerConfig{
			LocalUDPAddress:      "127.0.0.1:1610",
			Community:            "waarp-gw",
			SNMPv3Only:           false,
			SNMPv3Username:       "toto",
			SNMPv3AuthProtocol:   snmp.AuthSHA,
			SNMPv3AuthPassphrase: "123456789",
			SNMPv3PrivProtocol:   snmp.PrivAES,
			SNMPv3PrivPassphrase: "987654321",
		}
		require.NoError(t, db.Insert(&dbSnmpConf).Run())

		expectedResponse := marshal(t, map[string]any{
			"localUDPAddress":  dbSnmpConf.LocalUDPAddress,
			"community":        dbSnmpConf.Community,
			"v3Only":           dbSnmpConf.SNMPv3Only,
			"v3Username":       dbSnmpConf.SNMPv3Username,
			"v3AuthProtocol":   dbSnmpConf.SNMPv3AuthProtocol,
			"v3AuthPassphrase": dbSnmpConf.SNMPv3AuthPassphrase,
			"v3PrivProtocol":   dbSnmpConf.SNMPv3PrivProtocol,
			"v3PrivPassphrase": dbSnmpConf.SNMPv3PrivPassphrase,
		})

		req := httptest.NewRequest(http.MethodGet, SNMPServerPath, nil)
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, `Then the response code should be "200 OK"`)
		assert.JSONEq(t, expectedResponse, w.Body.String(),
			`Then the SNMP server config should have been returned`)
	})
}

func TestAddNewSNMPServer(t *testing.T) {
	const (
		localUDPAddress  = "127.0.0.1:1610"
		snmpCommunity    = "waarp-gw"
		v3Only           = false
		v3Username       = "toto"
		v3AuthProtocol   = snmp.AuthSHA
		v3AuthPassphrase = "123456789"
		v3PrivProtocol   = snmp.PrivAES
		v3PrivPassphrase = "987654321"
	)

	t.Run("When adding a new valid SNMP server", func(t *testing.T) {
		logger := testhelpers.GetTestLogger(t)
		db := dbtest.TestDatabase(t)
		handle := setSnmpService(logger, db)
		setupSnmpService(t, db)

		reqBody := bytes.Buffer{}
		encoder := json.NewEncoder(&reqBody)

		input := map[string]any{
			"localUDPAddress":  localUDPAddress,
			"community":        snmpCommunity,
			"v3Only":           v3Only,
			"v3Username":       v3Username,
			"v3AuthProtocol":   v3AuthProtocol,
			"v3AuthPassphrase": v3AuthPassphrase,
			"v3PrivProtocol":   v3PrivProtocol,
			"v3PrivPassphrase": v3PrivPassphrase,
		}
		require.NoError(t, encoder.Encode(input))

		expectedDBSnmpConf := snmp.ServerConfig{
			ID:                   1,
			Owner:                conf.GlobalConfig.GatewayName,
			LocalUDPAddress:      localUDPAddress,
			Community:            snmpCommunity,
			SNMPv3Only:           v3Only,
			SNMPv3Username:       v3Username,
			SNMPv3AuthProtocol:   v3AuthProtocol,
			SNMPv3AuthPassphrase: v3AuthPassphrase,
			SNMPv3PrivProtocol:   v3PrivProtocol,
			SNMPv3PrivPassphrase: v3PrivPassphrase,
		}
		expectedLoc := SNMPServerPath

		req := httptest.NewRequest(http.MethodPut, SNMPServerPath, &reqBody)
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code,
			`Then the response code should be "201 Created"`)
		assert.Empty(t, w.Body.String(), `Then the response body should be empty`)
		assert.Equal(t, expectedLoc, w.Header().Get("Location"),
			`Then the response location header should have been set correctly`)
		assert.Empty(t, w.Body.String(), `Then the response body should be empty`)

		var dbSnmpConfig snmp.ServerConfig

		require.NoError(t, db.Get(&dbSnmpConfig, "owner=?", conf.GlobalConfig.GatewayName).Run())
		assert.Equal(t, expectedDBSnmpConf, dbSnmpConfig,
			`Then the SNMP service config should have been inserted in the database`)

		assert.Equal(t, localUDPAddress, snmp.GlobalService.GetServerAddr(),
			`Then the SNMP server should be listening at the provided local address`)
	})
}

func TestUpdateSNMPServer(t *testing.T) {
	const (
		newLocalUDPAddress = "127.0.0.1:1611"
		newSnmpCommunity   = "public"
		newV3Only          = false
	)

	t.Run("When changing the existing SNMP server", func(t *testing.T) {
		logger := testhelpers.GetTestLogger(t)
		db := dbtest.TestDatabase(t)
		handle := setSnmpService(logger, db)

		oldDBSnmpConf := snmp.ServerConfig{
			LocalUDPAddress:      "127.0.0.1:1610",
			Community:            "waarp-gw",
			SNMPv3Only:           true,
			SNMPv3Username:       "toto",
			SNMPv3AuthProtocol:   snmp.AuthSHA,
			SNMPv3AuthPassphrase: "123456789",
			SNMPv3PrivProtocol:   snmp.PrivAES,
			SNMPv3PrivPassphrase: "987654321",
		}
		require.NoError(t, db.Insert(&oldDBSnmpConf).Run())
		setupSnmpService(t, db)

		reqBody := bytes.Buffer{}
		encoder := json.NewEncoder(&reqBody)

		input := map[string]any{
			"localUDPAddress": newLocalUDPAddress,
			"community":       newSnmpCommunity,
			"v3Only":          newV3Only,
		}
		require.NoError(t, encoder.Encode(input))

		expectedDBSnmpConf := snmp.ServerConfig{
			ID:                   oldDBSnmpConf.ID,
			Owner:                oldDBSnmpConf.Owner,
			LocalUDPAddress:      newLocalUDPAddress,
			Community:            newSnmpCommunity,
			SNMPv3Only:           newV3Only,
			SNMPv3Username:       oldDBSnmpConf.SNMPv3Username,
			SNMPv3AuthProtocol:   oldDBSnmpConf.SNMPv3AuthProtocol,
			SNMPv3AuthPassphrase: oldDBSnmpConf.SNMPv3AuthPassphrase,
			SNMPv3PrivProtocol:   oldDBSnmpConf.SNMPv3PrivProtocol,
			SNMPv3PrivPassphrase: oldDBSnmpConf.SNMPv3PrivPassphrase,
		}
		expectedLoc := SNMPServerPath

		req := httptest.NewRequest(http.MethodPut, SNMPServerPath, &reqBody)
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code,
			`Then the response code should be "201 Created"`)
		assert.Empty(t, w.Body.String(), `Then the response body should be empty`)
		assert.Equal(t, expectedLoc, w.Header().Get("Location"),
			`Then the response location header should have been set correctly`)
		assert.Empty(t, w.Body.String(), `Then the response body should be empty`)

		var dbSnmpConfig snmp.ServerConfig

		require.NoError(t, db.Get(&dbSnmpConfig, "owner=?", conf.GlobalConfig.GatewayName).Run())
		assert.Equal(t, expectedDBSnmpConf, dbSnmpConfig,
			`Then the SNMP service config should have been updated in the database`)

		assert.Equal(t, newLocalUDPAddress, snmp.GlobalService.GetServerAddr(),
			`Then the SNMP server should be listening at the newly provided local address`)
	})
}

func TestDeleteSnmpServer(t *testing.T) {
	t.Run("When deleting the existing SNMP server config", func(t *testing.T) {
		logger := testhelpers.GetTestLogger(t)
		db := dbtest.TestDatabase(t)
		handle := deleteSnmpService(logger, db)

		dbSnmpConf := snmp.ServerConfig{
			LocalUDPAddress:      "127.0.0.1:1610",
			Community:            "waarp-gw",
			SNMPv3Only:           false,
			SNMPv3Username:       "toto",
			SNMPv3AuthProtocol:   snmp.AuthSHA,
			SNMPv3AuthPassphrase: "123456789",
			SNMPv3PrivProtocol:   snmp.PrivAES,
			SNMPv3PrivPassphrase: "987654321",
		}
		require.NoError(t, db.Insert(&dbSnmpConf).Run())

		setupSnmpService(t, db)
		require.Equal(t, dbSnmpConf.LocalUDPAddress, snmp.GlobalService.GetServerAddr())

		req := httptest.NewRequest(http.MethodGet, SNMPServerPath, nil)
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code, `Then the response code should be "204 No Content"`)
		assert.Zero(t, w.Body.String(), `Then the response body should be blank`)

		var dbCheck snmp.ServerConfig
		var dbErr *database.NotFoundError

		require.ErrorAs(t, db.Get(&dbCheck, "true").Run(), &dbErr,
			`Then the SNMP server config should have been deleted from the database`)

		assert.Zero(t, snmp.GlobalService.GetServerAddr(),
			`Then the SNMP server should no longer be listening on the network`)
	})
}

package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	updateconfAPIPath = "/updateconf"
	exportconfAPIPath = "/exportconf"
)

func TestUpdateconfTargets(t *testing.T) {
	t.Parallel()

	// Setup
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	body := mkBody(t, map[string]any{
		"rules": []map[string]any{{
			"name":   "test-rule",
			"isSend": false,
			"path":   "/rule-path",
		}},
		"clients": []map[string]any{{
			"name":     "test-client",
			"protocol": testProto1,
		}},
	})

	// Request: only "rules" is targeted
	req := makeRequest(http.MethodPost, body, updateconfAPIPath, "")
	req.Header.Set(UpdateconfTargetsHeader, "rules")

	w := httptest.NewRecorder()
	updateconf(logger, db).ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusCreated, w.Code, `Then the response code should be "201 CREATED"`)

	// Check database: only the rule should have been imported
	var rules model.Rules
	require.NoError(t, db.Select(&rules).Run())
	assert.Len(t, rules, 1, `Then the rule should have been imported`)

	var clients model.Clients
	require.NoError(t, db.Select(&clients).Run())
	assert.Empty(t, clients, `Then the client should NOT have been imported`)
}

func TestUpdateconfDefaultTargets(t *testing.T) {
	t.Parallel()

	// Setup
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	body := mkBody(t, map[string]any{
		"rules": []map[string]any{{
			"name":   "test-rule",
			"isSend": false,
			"path":   "/rule-path",
		}},
		"clients": []map[string]any{{
			"name":     "test-client",
			"protocol": testProto1,
		}},
	})

	// Request: no Targets header, should default to "all"
	req := makeRequest(http.MethodPost, body, updateconfAPIPath, "")

	w := httptest.NewRecorder()
	updateconf(logger, db).ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusCreated, w.Code, `Then the response code should be "201 CREATED"`)

	// Check database: both the rule and the client should have been imported
	var rules model.Rules
	require.NoError(t, db.Select(&rules).Run())
	assert.Len(t, rules, 1, `Then the rule should have been imported`)

	var clients model.Clients
	require.NoError(t, db.Select(&clients).Run())
	require.Len(t, clients, 1, `Then the client should have been imported`)

	// Check services: since "Restart" was not set, the client should NOT have
	// been registered as a running service
	ok := services.Clients.Exists(clients[0])
	assert.False(t, ok, `Then the client should NOT have been started as a service`)
}

func TestUpdateconfDryRun(t *testing.T) {
	t.Parallel()

	// Setup
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	body := mkBody(t, map[string]any{
		"rules": []map[string]any{{
			"name":   "test-rule",
			"isSend": false,
			"path":   "/rule-path",
		}},
	})

	// Request
	req := makeRequest(http.MethodPost, body, updateconfAPIPath, "")
	req.Header.Set(UpdateconfDryRunHeader, "true")

	w := httptest.NewRecorder()
	updateconf(logger, db).ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusCreated, w.Code, `Then the response code should be "201 CREATED"`)

	// Check database: the import should have been rolled back
	var rules model.Rules
	require.NoError(t, db.Select(&rules).Run())
	assert.Empty(t, rules, `Then the rule should NOT have been imported`)
}

func TestUpdateconfReset(t *testing.T) {
	t.Parallel()

	// Setup
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	existing := &model.Rule{
		Name:   "existing-rule",
		IsSend: true,
		Path:   "/existing-path",
	}
	require.NoError(t, db.Insert(existing).Run())

	body := mkBody(t, map[string]any{
		"rules": []map[string]any{{
			"name":   "new-rule",
			"isSend": false,
			"path":   "/new-path",
		}},
	})

	// Request
	req := makeRequest(http.MethodPost, body, updateconfAPIPath, "")
	req.Header.Set(UpdateconfResetHeader, "true")

	w := httptest.NewRecorder()
	updateconf(logger, db).ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusCreated, w.Code, `Then the response code should be "201 CREATED"`)

	// Check database: the existing rule should have been deleted, and only the
	// new one should remain
	var rules model.Rules
	require.NoError(t, db.Select(&rules).Run())
	require.Len(t, rules, 1)
	assert.Equal(t, "new-rule", rules[0].Name, `Then only the imported rule should remain`)
}

func TestUpdateconfRestart(t *testing.T) {
	// Setup
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	const newClientName = "restart-client"

	body := mkBody(t, map[string]any{
		"clients": []map[string]any{{
			"name":     newClientName,
			"protocol": testProto1,
		}},
	})

	// Request
	req := makeRequest(http.MethodPost, body, updateconfAPIPath, "")
	req.Header.Set(UpdateconfTargetsHeader, "clients")
	req.Header.Set(UpdateconfRestartHeader, "true")

	w := httptest.NewRecorder()
	updateconf(logger, db).ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusCreated, w.Code, `Then the response code should be "201 CREATED"`)

	var dbClient model.Client
	require.NoError(t, db.Get(&dbClient, "name=?", newClientName).Run())

	t.Cleanup(func() {
		_ = services.Clients.Remove(context.Background(), &dbClient)
	})

	// Check that the client service has been started
	service, ok := services.Clients.Get(&dbClient)
	require.True(t, ok, `Then the client service should have been added to the running services`)

	state, _ := service.State()
	assert.Equal(t, utils.StateRunning, state, `Then the client service should be running`)
}

func TestExportconfTargets(t *testing.T) {
	t.Parallel()

	// Setup
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	rule := &model.Rule{
		Name:   "test-rule",
		IsSend: false,
		Path:   "/rule-path",
	}
	require.NoError(t, db.Insert(rule).Run())

	client := &model.Client{
		Name:     "test-client",
		Protocol: testProto1,
	}
	require.NoError(t, db.Insert(client).Run())

	// Request: only "rules" is targeted
	req := makeRequest(http.MethodGet, nil, exportconfAPIPath, "")
	req.Header.Set(UpdateconfTargetsHeader, "rules")

	w := httptest.NewRecorder()
	exportconf(logger, db).ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code, `Then the response code should be "200 OK"`)

	var data file.Data
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &data))

	require.Len(t, data.Rules, 1, `Then the rule should have been exported`)
	assert.Equal(t, rule.Name, data.Rules[0].Name)
	assert.Empty(t, data.Clients, `Then the client should NOT have been exported`)
}

func TestExportconfDefaultTargets(t *testing.T) {
	t.Parallel()

	// Setup
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	rule := &model.Rule{
		Name:   "test-rule",
		IsSend: false,
		Path:   "/rule-path",
	}
	require.NoError(t, db.Insert(rule).Run())

	client := &model.Client{
		Name:     "test-client",
		Protocol: testProto1,
	}
	require.NoError(t, db.Insert(client).Run())

	// Request: no Targets header, should default to "all"
	req := makeRequest(http.MethodGet, nil, exportconfAPIPath, "")

	w := httptest.NewRecorder()
	exportconf(logger, db).ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code, `Then the response code should be "200 OK"`)

	var data file.Data
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &data))

	require.Len(t, data.Rules, 1, `Then the rule should have been exported`)
	assert.Equal(t, rule.Name, data.Rules[0].Name)
	require.Len(t, data.Clients, 1, `Then the client should have been exported`)
	assert.Equal(t, client.Name, data.Clients[0].Name)
}

func TestExportconfJSONFormat(t *testing.T) {
	t.Parallel()

	// Setup
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	rule := &model.Rule{
		Name:   "test-rule",
		IsSend: false,
		Path:   "/rule-path",
	}
	require.NoError(t, db.Insert(rule).Run())

	// Request: no Accept header, should default to JSON
	req := makeRequest(http.MethodGet, nil, exportconfAPIPath, "")
	req.Header.Set(UpdateconfTargetsHeader, "rules")

	w := httptest.NewRecorder()
	exportconf(logger, db).ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code, `Then the response code should be "200 OK"`)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"),
		`Then the response should be in JSON format`)

	var data file.Data
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &data),
		`Then the response body should be valid JSON`)
	require.Len(t, data.Rules, 1, `Then the rule should have been exported`)
}

func TestExportconfYAMLFormat(t *testing.T) {
	t.Parallel()

	// Setup
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	rule := &model.Rule{
		Name:   "test-rule",
		IsSend: false,
		Path:   "/rule-path",
	}
	require.NoError(t, db.Insert(rule).Run())

	// Request
	req := makeRequest(http.MethodGet, nil, exportconfAPIPath, "")
	req.Header.Set(UpdateconfTargetsHeader, "rules")
	req.Header.Set("Accept", "application/yaml")

	w := httptest.NewRecorder()
	exportconf(logger, db).ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code, `Then the response code should be "200 OK"`)
	assert.Equal(t, "application/yaml", w.Header().Get("Content-Type"),
		`Then the response should be in YAML format`)

	var data file.Data
	require.NoError(t, yaml.Unmarshal(w.Body.Bytes(), &data),
		`Then the response body should be valid YAML`)
	require.Len(t, data.Rules, 1, `Then the rule should have been exported`)
}

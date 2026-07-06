package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/filewatcher"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	filewatchersAPIPath = "/filewatchers"
	filewatcherAPIPath  = "/filewatchers/{filewatcher}"

	// FIXME: remove and add partner reference once eager loading is implemented on accounts
	fwPartnerName = "test-partner"
)

func addFwPrereqs(tb testing.TB, db *database.DB) struct {
	*model.RemoteAgent
	*model.RemoteAccount
	*model.Client
	*model.Rule
} {
	tb.Helper()

	partner := &model.RemoteAgent{
		Name:     "test-partner",
		Protocol: testFwProto,
		Address:  types.Addr("localhost", 21),
	}
	require.NoError(tb, db.Insert(partner).Run())

	account := &model.RemoteAccount{
		RemoteAgentID: partner.ID,
		Login:         "test-account",
	}
	require.NoError(tb, db.Insert(account).Run())

	client := &model.Client{
		Name:     "test-client",
		Protocol: testFwProto,
	}
	require.NoError(tb, db.Insert(client).Run())

	rule := &model.Rule{
		Name:   "test-rule",
		IsSend: false,
		Path:   "/fw-rule-path",
	}
	require.NoError(tb, db.Insert(rule).Run())

	return struct {
		*model.RemoteAgent
		*model.RemoteAccount
		*model.Client
		*model.Rule
	}{
		RemoteAgent:   partner,
		RemoteAccount: account,
		Client:        client,
		Rule:          rule,
	}
}

var fwMux sync.Mutex

func setupFWTest(tb testing.TB) (*database.DB, *log.Logger, *model.FileWatcher) {
	tb.Helper()

	db := dbtest.TestDatabase(tb)
	logger := testhelpers.GetTestLogger(tb)
	prereqs := addFwPrereqs(tb, db)

	existing := &model.FileWatcher{
		Flow:          "old-flow",
		Interval:      time.Minute,
		Pattern:       "*.txt",
		RemoteAccount: *prereqs.RemoteAccount,
		Client:        *prereqs.Client,
		Rule:          *prereqs.Rule,
	}
	require.NoError(tb, db.Insert(existing).Run())

	fw := filewatcher.NewFilewatcher(db, existing)
	require.NoError(tb, fw.Start())

	fwMux.Lock()
	filewatcher.Filewatchers.Add(existing, fw)
	tb.Cleanup(func() {
		defer fwMux.Unlock()
		require.NoError(tb, filewatcher.Filewatchers.Remove(context.Background(), existing))
	})

	return db, logger, existing
}

func TestAddFilewatcher(t *testing.T) {
	t.Parallel()

	// Setup
	db, logger, existing := setupFWTest(t)

	const (
		newFWFlow     = "updated-flow"
		newFWPattern  = "*.txt"
		newFWInterval = "2m30s"
	)

	// Request
	body := mkBody(t, map[string]any{
		"flow":     newFWFlow,
		"pattern":  newFWPattern,
		"interval": newFWInterval,
		"partner":  fwPartnerName,
		"account":  existing.RemoteAccount.Login,
		"client":   existing.Client.Name,
		"rule":     existing.Rule.Name,
	})
	req := makeRequest(http.MethodPost, body, filewatchersAPIPath, "")
	w := httptest.NewRecorder()
	createFilewatcher(logger, db).ServeHTTP(w, req)
	expectedLoc, _ := replaceURLVar(filewatcherAPIPath, newFWFlow)

	// Check response
	assert.Equal(t, http.StatusCreated, w.Code, `Then the response code should be "201 CREATED"`)
	assert.Empty(t, w.Body.String(), `Then the response body should be empty`)
	assert.Equal(t, expectedLoc, w.Header().Get("Location"),
		`Then the response location header should have been set correctly`)

	// Check database
	var dbFW model.FileWatcher
	require.NoError(t, db.Get(&dbFW, "flow=?", newFWFlow).Run())
	assert.Equal(t, newFWFlow, dbFW.Flow)
	assert.Equal(t, newFWPattern, dbFW.Pattern)
	assert.Equal(t, newFWInterval, dbFW.Interval.String())
	assert.Equal(t, existing.RemoteAccount.ID, dbFW.RemoteAccountID)
	assert.Equal(t, existing.Client.ID, dbFW.ClientID)
	assert.Equal(t, existing.Rule.ID, dbFW.RuleID)

	// Check services
	service, ok := filewatcher.Filewatchers.Get(dbFW)
	require.True(t, ok)
	assert.Equal(t, newFWFlow, service.Name())
}

func TestGetFilewatcher(t *testing.T) {
	t.Parallel()

	// Setup
	db, logger, existing := setupFWTest(t)

	// Request
	req := makeRequest(http.MethodGet, nil, filewatcherAPIPath, existing.Flow)
	w := httptest.NewRecorder()
	getFilewatcher(logger, db).ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code, `Then the response code should be "200 OK"`)

	expected := marshal(t, map[string]any{
		"disabled":         existing.Disabled,
		"flow":             existing.Flow,
		"interval":         existing.Interval.String(),
		"pattern":          existing.Pattern,
		"noDuplicateCheck": existing.NoDuplicateCheck,
		"partner":          fwPartnerName,
		"account":          existing.RemoteAccount.Login,
		"client":           existing.Client.Name,
		"rule":             existing.Rule.Name,
	})
	assert.JSONEq(t, expected, w.Body.String(), `Then the filewatcher should be returned`)
}

func TestDeleteFilewatcher(t *testing.T) {
	t.Parallel()

	// Setup
	db, logger, existing := setupFWTest(t)

	// Request
	req := makeRequest(http.MethodDelete, nil, filewatcherAPIPath, existing.Flow)
	w := httptest.NewRecorder()
	deleteFilewatcher(logger, db).ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusNoContent, w.Code, `Then the response code should be "204 NO CONTENT"`)
	assert.Empty(t, w.Body.String(), `Then the response body should be empty`)

	// Check database
	var dbFW model.FileWatcher
	var dbErr *database.NotFoundError
	require.ErrorAs(t, db.Get(&dbFW, "id=?", existing.ID).Run(), &dbErr)

	// Check services
	require.False(t, filewatcher.Filewatchers.Exists(existing))
}

func TestUpdateFilewatcher(t *testing.T) {
	t.Parallel()

	// Setup
	db, logger, existing := setupFWTest(t)

	const (
		newFWFlow    = "updated-flow"
		newFWPattern = "*.csv"
	)

	// Request
	body := mkBody(t, map[string]any{
		"flow":    newFWFlow,
		"pattern": newFWPattern,
	})
	req := makeRequest(http.MethodPatch, body, filewatcherAPIPath, existing.Flow)
	w := httptest.NewRecorder()
	updateFilewatcher(logger, db).ServeHTTP(w, req)
	expectedLoc, _ := replaceURLVar(filewatcherAPIPath, newFWFlow)

	// Check response
	assert.Equal(t, http.StatusCreated, w.Code, `Then the response code should be "201 CREATED"`)
	assert.Empty(t, w.Body.String(), `Then the response body should be empty`)
	assert.Equal(t, expectedLoc, w.Header().Get("Location"),
		`Then the response location header should have been set correctly`)

	// Check database
	var dbFW model.FileWatcher
	require.NoError(t, db.Get(&dbFW, "flow=?", newFWFlow).Run())
	assert.Equal(t, newFWFlow, dbFW.Flow)
	assert.Equal(t, newFWPattern, dbFW.Pattern)

	// Check services
	service, ok := filewatcher.Filewatchers.Get(dbFW)
	require.True(t, ok)
	assert.Equal(t, newFWFlow, service.Name())
}

func TestListFilewatchers(t *testing.T) {
	t.Parallel()

	// Setup
	db, logger, existing1 := setupFWTest(t)
	existing2 := &model.FileWatcher{
		Flow:          "existing2",
		Interval:      30 * time.Second,
		Pattern:       "*.txt",
		RemoteAccount: existing1.RemoteAccount,
		Client:        existing1.Client,
		Rule:          existing1.Rule,
	}
	require.NoError(t, db.Insert(existing2).Run())

	// Request
	req := httptest.NewRequest(http.MethodGet, filewatchersAPIPath, nil)
	w := httptest.NewRecorder()
	listFilewatchers(logger, db).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, `Then the response code should be "200 OK"`)

	// Check response
	expected := marshal(t, map[string]any{
		"filewatchers": []map[string]any{{
			"disabled":         existing1.Disabled,
			"flow":             existing1.Flow,
			"interval":         existing1.Interval.String(),
			"pattern":          existing1.Pattern,
			"noDuplicateCheck": existing1.NoDuplicateCheck,
			"partner":          fwPartnerName,
			"account":          existing1.RemoteAccount.Login,
			"client":           existing1.Client.Name,
			"rule":             existing1.Rule.Name,
		}, {
			"disabled":         existing2.Disabled,
			"flow":             existing2.Flow,
			"interval":         existing2.Interval.String(),
			"pattern":          existing2.Pattern,
			"noDuplicateCheck": existing2.NoDuplicateCheck,
			"partner":          fwPartnerName,
			"account":          existing2.RemoteAccount.Login,
			"client":           existing2.Client.Name,
			"rule":             existing2.Rule.Name,
		}},
	})
	assert.JSONEq(t, expected, w.Body.String(),
		`Then the list of filewatchers should be returned`)
}

func TestFireFilewatcher(t *testing.T) {
	t.Parallel()

	const path = filewatcherAPIPath + "/fire"

	// Setup
	db, logger, existing := setupFWTest(t)
	testFile := &testFileInfo{
		name:    "test.txt",
		size:    123,
		mode:    0o740,
		modTime: time.Now(),
	}
	filewatcher.EnableTestProtocol(testFile)

	// Request
	req := makeRequest(http.MethodPut, nil, path, existing.Flow)
	w := httptest.NewRecorder()
	fireFilewatcher(logger, db).ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusAccepted, w.Code, `Then the response code should be "202 ACCEPTED"`)
	assert.Empty(t, w.Body.String(), `Then the response body should be empty`)

	// Check database
	var trans model.Transfers
	require.NoError(t, db.Select(&trans).Run())
	require.NotEmpty(t, trans)
	assert.Equal(t, types.StatusPlanned, trans[0].Status)
	assert.Equal(t, existing.ClientID, trans[0].ClientID.Int64)
	assert.Equal(t, existing.RuleID, trans[0].RuleID)
	assert.Equal(t, existing.RemoteAccountID, trans[0].RemoteAccountID.Int64)
	assert.Equal(t, testFile.name, trans[0].SrcFilename)
	assert.Equal(t, testFile.size, trans[0].Filesize)
}

func TestStartFilewatcher(t *testing.T) {
	t.Parallel()

	const path = filewatcherAPIPath + "/start"

	// Setup
	db, logger, existing := setupFWTest(t)
	stopped, err := filewatcher.Filewatchers.Stop(t.Context(), existing)
	require.NoError(t, err)
	require.True(t, stopped)

	// Request
	req := makeRequest(http.MethodPut, nil, path, existing.Flow)
	w := httptest.NewRecorder()
	startFilewatcher(logger, db).ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusAccepted, w.Code, `Then the response code should be "202 ACCEPTED"`)
	assert.Empty(t, w.Body.String(), `Then the response body should be empty`)

	// Check service
	service, ok := filewatcher.Filewatchers.Get(existing)
	require.True(t, ok)
	state, _ := service.State()
	assert.Equal(t, utils.StateRunning, state)
}

func TestStopFilewatcher(t *testing.T) {
	t.Parallel()

	const path = filewatcherAPIPath + "/stop"

	// Setup
	db, logger, existing := setupFWTest(t)

	// Request
	req := makeRequest(http.MethodPut, nil, path, existing.Flow)
	w := httptest.NewRecorder()
	stopFilewatcher(logger, db).ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusAccepted, w.Code, `Then the response code should be "202 ACCEPTED"`)
	assert.Empty(t, w.Body.String(), `Then the response body should be empty`)

	// Check service
	service, ok := filewatcher.Filewatchers.Get(existing)
	require.True(t, ok)
	state, _ := service.State()
	assert.Equal(t, utils.StateOffline, state)
}

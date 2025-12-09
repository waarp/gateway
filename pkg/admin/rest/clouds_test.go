package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestGetCloud(t *testing.T) {
	cloudType := fstest.MakeDummyBackend(t)
	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)
	handle := getCloud(logger, db)

	existing := &model.CloudInstance{
		Name:    "existing",
		Type:    cloudType,
		Key:     "key",
		Secret:  "secret",
		Options: map[string]string{"opt": "val"},
	}
	require.NoError(t, db.Insert(existing).Run())

	expectedJSON, jsonErr := json.Marshal(map[string]any{
		"name":    existing.Name,
		"type":    existing.Type,
		"key":     existing.Key,
		"options": existing.Options,
	})
	require.NoError(t, jsonErr)

	t.Run("Given a valid cloud name", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "", nil)
		require.NoError(t, err)

		r = mux.SetURLVars(r, map[string]string{"cloud": existing.Name})
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code,
			`Then the response code should be "200 OK"`)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"),
			`Then the response content type should be "application/json"`)
		assert.JSONEq(t, string(expectedJSON), w.Body.String(),
			`Then the JSON response body should be as expected`)
	})

	t.Run("Given an unknown cloud name", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "", nil)
		require.NoError(t, err)

		const name = "unknown"
		r = mux.SetURLVars(r, map[string]string{"cloud": name})

		w := httptest.NewRecorder()
		handle.ServeHTTP(w, r)

		assert.Equal(t, http.StatusNotFound, w.Code,
			`Then the response code should be "404 Not Found"`)
		assert.Equal(t,
			fmt.Sprintf("cloud %q not found\n", name),
			w.Body.String(),
			`Then the body should contain an error message`)
	})
}

func TestAddCloud(t *testing.T) {
	cloudType := fstest.MakeDummyBackend(t)

	t.Run("Given a valid cloud object", func(t *testing.T) {
		logger := testhelpers.GetTestLogger(t)
		db := dbtest.TestDatabase(t)
		handle := addCloud(logger, db)
		body := bytes.Buffer{}
		encoder := json.NewEncoder(&body)

		input := api.PostCloudReqObject{
			Name:    "input",
			Type:    cloudType,
			Key:     "app key",
			Secret:  "app secret",
			Options: map[string]string{"opt": "val"},
		}
		require.NoError(t, encoder.Encode(input))

		expectedCloud := model.CloudInstance{
			ID:      1,
			Owner:   conf.GlobalConfig.GatewayName,
			Name:    input.Name,
			Type:    input.Type,
			Key:     input.Key,
			Secret:  database.SecretText(input.Secret),
			Options: input.Options,
		}
		expectedLoc := path.Join("clouds", input.Name)

		r, err := http.NewRequest(http.MethodPost, "clouds", &body)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		handle.ServeHTTP(w, r)

		assert.Equal(t, http.StatusCreated, w.Code,
			`Then the response code should be "201 Created"`)
		assert.Empty(t, w.Body.String(), `Then the response body should be empty`)
		assert.Equal(t, expectedLoc, w.Header().Get("Location"),
			`Then the response location header should have been set correctly`)
		assert.Empty(t, w.Body.String(), `Then the response body should be empty`)

		var dbCloud model.CloudInstance

		require.NoError(t, db.Get(&dbCloud, "name=? AND owner=?", input.Name,
			conf.GlobalConfig.GatewayName).Run())
		assert.EqualExportedValues(t, expectedCloud, dbCloud,
			`Then the cloud instance should have been inserted in the database`)
	})

	t.Run("Given an invalid cloud object", func(t *testing.T) {
		logger := testhelpers.GetTestLogger(t)
		db := dbtest.TestDatabase(t)
		handle := addCloud(logger, db)
		body := bytes.NewBufferString("this is a string")

		r, err := http.NewRequest(http.MethodPost, "clouds", body)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		handle.ServeHTTP(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code,
			`Then the response code should be "400 Bad Request"`)
		assert.Contains(t, w.Body.String(),
			`malformed JSON object`,
			`Then the response body should contain an error message`)

		var dbClouds model.CloudInstances

		require.NoError(t, db.Select(&dbClouds).Run())
		assert.Empty(t, dbClouds, `Then there should be no cloud instances in the database`)
	})
}

func TestDeleteCloud(t *testing.T) {
	cloudType := fstest.MakeDummyBackend(t)

	setup := func(tb testing.TB) (*database.DB, http.Handler, *model.CloudInstance) {
		tb.Helper()

		logger := testhelpers.GetTestLogger(tb)
		db := dbtest.TestDatabase(tb)
		handle := deleteCloud(logger, db)

		ex := &model.CloudInstance{
			Name:    "existing",
			Type:    cloudType,
			Key:     "key",
			Secret:  "secret",
			Options: map[string]string{"opt": "val"},
		}
		require.NoError(tb, db.Insert(ex).Run())

		return db, handle, ex
	}

	t.Run("Given a valid cloud name", func(t *testing.T) {
		db, handle, existing := setup(t)

		r, err := http.NewRequest(http.MethodDelete, "", nil)
		require.NoError(t, err)

		r = mux.SetURLVars(r, map[string]string{"cloud": existing.Name})
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, r)

		assert.Equal(t, http.StatusNoContent, w.Code,
			`Then the response code should be "204 No Content"`)
		assert.Empty(t, w.Body.String(),
			`Then the response body should be empty`)

		var dbClouds model.CloudInstances

		require.NoError(t, db.Select(&dbClouds).Run())
		assert.Empty(t, dbClouds,
			`Then there should be no cloud instances left in the database`)
	})

	t.Run("Given an unknown cloud name", func(t *testing.T) {
		db, handle, ex := setup(t)

		r, err := http.NewRequest(http.MethodGet, "", nil)
		require.NoError(t, err)

		const name = "unknown"
		r = mux.SetURLVars(r, map[string]string{"cloud": name})

		w := httptest.NewRecorder()
		handle.ServeHTTP(w, r)

		assert.Equal(t, http.StatusNotFound, w.Code,
			`Then the response code should be "404 Not Found"`)
		assert.Equal(t,
			fmt.Sprintf("cloud %q not found\n", name),
			w.Body.String(),
			`Then the body should contain an error message`)

		var dbClouds model.CloudInstances

		require.NoError(t, db.Select(&dbClouds).Run())
		require.Len(t, dbClouds, 1)
		assert.EqualExportedValues(t, ex, dbClouds[0],
			`Then it should have left the database untouched`)
	})
}

func TestUpdateCloud(t *testing.T) {
	testUpdateReplaceCloud(t, false)
}

func TestReplaceCloud(t *testing.T) {
	testUpdateReplaceCloud(t, true)
}

func testUpdateReplaceCloud(t *testing.T, isReplace bool) {
	t.Helper()

	cloudType := fstest.MakeDummyBackend(t)
	input := api.PostCloudReqObject{
		Name:    "new_name",
		Type:    cloudType,
		Key:     "new_key",
		Options: map[string]string{"new_opt": "new_val"},
	}

	mkHandle := updateCloud
	method := http.MethodPatch

	if isReplace {
		mkHandle = replaceCloud
		method = http.MethodPut
	}

	mkExpected := func(old *model.CloudInstance) model.CloudInstance {
		newCloud := *old

		if isReplace || input.Name != "" {
			newCloud.Name = input.Name
		}

		if isReplace || input.Type != "" {
			newCloud.Type = input.Type
		}

		if isReplace || input.Key != "" {
			newCloud.Key = input.Key
		}

		if isReplace || input.Secret != "" {
			newCloud.Secret = database.SecretText(input.Secret)
		}

		if isReplace || input.Options != nil {
			newCloud.Options = input.Options
		}

		return newCloud
	}

	setup := func(tb testing.TB) (*database.DB, http.Handler, *model.CloudInstance) {
		tb.Helper()

		logger := testhelpers.GetTestLogger(tb)
		db := dbtest.TestDatabase(tb)
		handle := mkHandle(logger, db)

		ex := &model.CloudInstance{
			Name:    "existing",
			Type:    cloudType,
			Key:     "key",
			Secret:  "secret",
			Options: map[string]string{"opt": "val"},
		}
		require.NoError(tb, db.Insert(ex).Run())

		return db, handle, ex
	}

	t.Run("Given a valid cloud name", func(t *testing.T) {
		db, handle, old := setup(t)

		body := bytes.Buffer{}
		encoder := json.NewEncoder(&body)

		require.NoError(t, encoder.Encode(input))

		expectedCloud := mkExpected(old)
		expectedLoc := path.Join("clouds", input.Name)

		r, err := http.NewRequest(method, "clouds", &body)
		require.NoError(t, err)

		r = mux.SetURLVars(r, map[string]string{"cloud": old.Name})
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, r)

		assert.Equal(t, http.StatusCreated, w.Code,
			`Then the response code should be "201 Created"`)
		assert.Empty(t, w.Body.String(), `Then the response body should be empty`)
		assert.Equal(t, expectedLoc, w.Header().Get("Location"),
			`Then the response location header should have been set correctly`)
		assert.Empty(t, w.Body.String(), `Then the response body should be empty`)

		var dbCloud model.CloudInstance

		require.NoError(t, db.Get(&dbCloud, "id=?", old.ID).Run())
		assert.EqualExportedValues(t,
			expectedCloud,
			dbCloud,
			`Then the cloud instance should have been updated`)
	})
}

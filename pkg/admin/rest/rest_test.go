package rest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func mkBody(tb testing.TB, body map[string]any) io.Reader {
	tb.Helper()

	return strings.NewReader(marshal(tb, body))
}

type getBean[T any] interface {
	*T
	database.GetBean
}

func testAdd[T any, U getBean[T]](tb testing.TB, mkHandler handler,
	reqPath, elem string, reqBodyObject map[string]any, expectedDBObject U,
) {
	tb.Helper()

	logger := testhelpers.GetTestLogger(tb)
	db := dbtest.TestDatabase(tb)
	handlr := mkHandler(logger, db)

	expectedLoc := path.Join(reqPath, elem)
	reqBody := mkBody(tb, reqBodyObject)
	req := httptest.NewRequest(http.MethodPost, reqPath, reqBody)
	w := httptest.NewRecorder()

	handlr.ServeHTTP(w, req)

	assert.Equal(tb, http.StatusCreated, w.Code,
		`Then the response code should be "201 Created"`)
	assert.Empty(tb, w.Body.String(), `Then the response body should be empty`)
	assert.Equal(tb, expectedLoc, w.Header().Get("Location"),
		`Then the response location header should have been set correctly`)
	assert.Empty(tb, w.Body.String(), `Then the response body should be empty`)

	cop := *expectedDBObject
	actual := U(&cop)

	require.NoError(tb, db.Get(actual, "true").Run())
	assert.Equalf(tb, expectedDBObject, actual,
		`Then the %s should have been inserted in the database`, expectedDBObject.Appellation())
}

func testGet(tb testing.TB, mkHandler handler, reqPath, elem string,
	dbObject database.InsertBean, expectedResponse map[string]any,
) {
	tb.Helper()

	logger := testhelpers.GetTestLogger(tb)
	db := dbtest.TestDatabase(tb)
	handlr := mkHandler(logger, db)

	require.NoError(tb, db.Insert(dbObject).Run())

	req := makeRequest(http.MethodGet, nil, reqPath, elem)
	w := httptest.NewRecorder()
	handlr.ServeHTTP(w, req)

	expectedJSON := marshal(tb, expectedResponse)

	assert.Equal(tb, http.StatusOK, w.Code, `Then the response code should be "200 OK"`)
	assert.JSONEqf(tb, expectedJSON, w.Body.String(),
		`Then the %s should have been returned`, dbObject.Appellation())
}

func testDelete(tb testing.TB, mkHandler handler, reqPath, elem string,
	dbObject database.InsertBean,
) {
	tb.Helper()

	logger := testhelpers.GetTestLogger(tb)
	db := dbtest.TestDatabase(tb)
	handlr := mkHandler(logger, db)

	require.NoError(tb, db.Insert(dbObject).Run())

	req := makeRequest(http.MethodDelete, nil, reqPath, elem)
	w := httptest.NewRecorder()
	handlr.ServeHTTP(w, req)

	assert.Equal(tb, http.StatusNoContent, w.Code,
		`Then the response code should be "204 No Content"`)
	assert.Zero(tb, w.Body.String(), `Then the response body should be empty`)

	var nfErr *database.NotFoundError
	assert.ErrorAsf(tb, db.Get(dbObject, "true").Run(), &nfErr,
		`Then the %s should have been deleted`, dbObject.Appellation())
}

func testUpdate[T any, U getBean[T]](tb testing.TB, mkHandler handler,
	reqPath, elem, newElem string, dbObject database.InsertBean,
	reqBodyObject map[string]any, expectedDBObject U,
) {
	tb.Helper()

	logger := testhelpers.GetTestLogger(tb)
	db := dbtest.TestDatabase(tb)
	handlr := mkHandler(logger, db)

	require.NoError(tb, db.Insert(dbObject).Run())

	reqBody := mkBody(tb, reqBodyObject)
	req := makeRequest(http.MethodPatch, reqBody, reqPath, elem)
	w := httptest.NewRecorder()
	expectedLoc, _ := replaceURLVar(reqPath, newElem)

	handlr.ServeHTTP(w, req)

	assert.Equal(tb, http.StatusCreated, w.Code,
		`Then the response code should be "201 Created"`)
	assert.Empty(tb, w.Body.String(), `Then the response body should be empty`)
	assert.Equal(tb, expectedLoc, w.Header().Get("Location"),
		`Then the response location header should have been set correctly`)
	assert.Empty(tb, w.Body.String(), `Then the response body should be empty`)

	cop := *expectedDBObject
	actual := U(&cop)

	require.NoError(tb, db.Get(actual, "true").Run())
	assert.Equalf(tb, expectedDBObject, actual,
		`Then the %s database entry should have been updated`, expectedDBObject.Appellation())
}

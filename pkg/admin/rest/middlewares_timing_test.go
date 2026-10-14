package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The test changes utils.BcryptRounds, so it cannot run in parallel.
//
//nolint:paralleltest //changes a package variable
func TestAuthenticationTiming(t *testing.T) {
	const timingCost = 10

	db := dbtest.TestDatabase(t)
	t.Cleanup(utils.ChangeBcryptRounds(timingCost))

	hash, err := utils.HashPassword("sesame")
	require.NoError(t, err)
	require.NoError(t, db.Insert(&model.User{
		Username: "timing_user", PasswordHash: string(hash), Permissions: model.PermAll,
	}).Run())

	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := AuthenticationMiddleware(testhelpers.GetTestLogger(t), db).Middleware(ok)

	login := func(user, password string) (time.Duration, *httptest.ResponseRecorder) {
		best := time.Duration(1<<63 - 1)

		var rec *httptest.ResponseRecorder

		for range 3 {
			req := httptest.NewRequest(http.MethodGet, "/api/about", nil)
			req.SetBasicAuth(user, password)

			rec = httptest.NewRecorder()
			start := time.Now()

			handler.ServeHTTP(rec, req)

			if d := time.Since(start); d < best {
				best = d
			}
		}

		return best, rec
	}

	login("unknown_user", "warm-up")

	existing, recExisting := login("timing_user", "not_sesame")
	unknown, recUnknown := login("unknown_user", "not_sesame")

	assert.Equal(t, http.StatusUnauthorized, recExisting.Code)
	assert.Equal(t, http.StatusUnauthorized, recUnknown.Code)
	assert.Equal(t, recExisting.Body.String(), recUnknown.Body.String(), "same error for both")
	assert.GreaterOrEqual(t, unknown, existing/2,
		"unknown account: %v, existing account: %v", unknown, existing)
}

package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"github.com/stretchr/testify/assert"
)

func TestRuleTaskUnknownElement(t *testing.T) {
	t.Parallel()

	// SETUP
	db := dbtest.TestDatabase(t)
	logger := logtest.GetTestLogger(t)

	// REQUEST
	body := mkBody(t, map[string]any{
		"name":   "test_rule",
		"isSend": true,
		"preTasks": []map[string]any{{
			"type": tasks.Email,
			"args": map[string]string{
				"sender":     "unknown", // purposefully unknown sender
				"recipients": "foobar@example.com",
				"template":   "email_template",
			},
		}},
	})

	r := makeRequest(http.MethodPost, body, "/api/rules", "")
	w := httptest.NewRecorder()
	addRule(logger, db)(w, r)

	// CHECK RESPONSE
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

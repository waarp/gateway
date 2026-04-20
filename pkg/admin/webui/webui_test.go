//go:build manual_test

package webui

import (
	"net/http"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/webui/internal/session"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
)

func TestGUI(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := logtest.GetTestLogger(t)

	handler := &Handler{
		db:       db,
		logger:   logger,
		sessions: session.NewStore(),
		isDev:    true,
	}

	http.ListenAndServe("localhost:8090", handler)
}

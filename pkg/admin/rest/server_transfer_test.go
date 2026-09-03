package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreregisterTransfer(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := logtest.GetTestLogger(t)
	handler := preregisterServerTransfer(logger, db)

	// ########## SETUP ##########
	server := &model.LocalAgent{
		Name:     "test_server",
		Address:  types.Addr("", 12345),
		Protocol: testProto1,
	}
	require.NoError(t, db.Insert(server).Run())

	account := &model.LocalAccount{
		LocalAgent: *server,
		Login:      "test_account",
	}
	require.NoError(t, db.Insert(account).Run())

	rule := &model.Rule{Name: "test_rule", IsSend: true}
	require.NoError(t, db.Insert(rule).Run())

	const (
		file  = "file.test"
		info1 = "info1"
		info2 = "info2"
		val1  = true
		val2  = "value2"
	)

	dueDate := time.Date(2040, time.January, 1, 0, 0, 0, 0, time.Local)

	// ########## REQUEST ##########
	body := mkBody(t, map[string]any{
		"rule":         rule.Name,
		"server":       server.Name,
		"account":      account.Login,
		"isSend":       rule.IsSend,
		"file":         file,
		"dueDate":      dueDate,
		"transferInfo": map[string]any{info1: val1, info2: val2},
	})
	req := makeRequest(http.MethodPut, body, "/api/transfers", "")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// ########## CHECK RESPONSE ##########
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Empty(t, w.Body.String())
	assert.Equal(t, "/api/transfers/1", w.Header().Get("Location"))

	// ########## CHECK DATABASE ##########
	var check model.Transfer
	require.NoError(t, db.Get(&check, "id=?", 1).Eager().Run())

	assert.Equal(t, rule.ID, check.RuleID)
	assert.Equal(t, account.ID, check.LocalAccountID.Int64)
	assert.Equal(t, file, check.SrcFilename)
	gwtesting.Equal(t, dueDate, check.Start)
	assert.Subset(t, check.TransferInfo, map[string]any{info1: val1, info2: val2})
}

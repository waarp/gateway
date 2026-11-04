package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// setFixtures: a client, a partner with its account, two rules; a failed and
// a planned transfer of the first rule, a failed transfer of the second, and
// a cancelled transfer of the first rule in the history.
type setFixtures struct {
	db *database.DB

	orders, reports            *model.Rule
	failed, planned, otherFail *model.Transfer
	cancelled                  *model.HistoryEntry
}

func newSetFixtures(t *testing.T) *setFixtures {
	t.Helper()

	f := &setFixtures{db: dbtest.TestDatabase(t)}

	client := &model.Client{Name: "client", Protocol: testProto1}
	require.NoError(t, f.db.Insert(client).Run())

	partner := &model.RemoteAgent{Name: "partner", Protocol: testProto1, Address: types.Addr("localhost", 1)}
	require.NoError(t, f.db.Insert(partner).Run())

	account := &model.RemoteAccount{RemoteAgentID: partner.ID, Login: "login"}
	require.NoError(t, f.db.Insert(account).Run())

	f.orders = &model.Rule{Name: "orders", IsSend: true, Path: "/orders"}
	f.reports = &model.Rule{Name: "reports", IsSend: true, Path: "/reports"}
	require.NoError(t, f.db.Insert(f.orders).Run())
	require.NoError(t, f.db.Insert(f.reports).Run())

	transfer := func(rule *model.Rule, file string, status types.TransferStatus) *model.Transfer {
		trans := &model.Transfer{
			RuleID: rule.ID, ClientID: client.NullableID(), RemoteAccountID: account.NullableID(),
			SrcFilename: file, DestFilename: file, Status: status, Filesize: model.UnknownSize,
			Start: time.Date(2030, 1, 1, 1, 0, 0, 0, time.Local),
		}
		if status == types.StatusError {
			trans.ErrCode, trans.ErrDetails, trans.Step = types.TeDataTransfer, "connection lost", types.StepData
		}

		require.NoError(t, f.db.Insert(trans).Run())

		return trans
	}

	f.failed = transfer(f.orders, "order-1.edi", types.StatusError)
	f.planned = transfer(f.orders, "order-2.edi", types.StatusPlanned)
	f.otherFail = transfer(f.reports, "report-1.pdf", types.StatusError)

	f.cancelled = &model.HistoryEntry{
		Identifier: model.ID(1001), RemoteTransferID: "r-1001", IsSend: true, Rule: f.orders.Name,
		Client: client.Name, Account: account.Login, Agent: partner.Name, Protocol: testProto1,
		SrcFilename: "order-0.edi", DestFilename: "order-0.edi", Filesize: 1024, Progress: 1024,
		Start: time.Date(2029, 1, 1, 1, 0, 0, 0, time.Local), Stop: time.Date(2029, 1, 1, 2, 0, 0, 0, time.Local),
		Status: types.StatusCancelled,
	}
	require.NoError(t, f.db.Insert(f.cancelled).Run())

	return f
}

func (f *setFixtures) id(id int64) string { return strconv.FormatInt(id, 10) }

// password hashes the password of a test user (the hash helper of the
// package needs a goconvey context).
func password(t *testing.T, pwd string) string {
	t.Helper()

	hashed, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	require.NoError(t, err)

	return string(hashed)
}

// call serves the request with the handler and decodes the counted answer of
// a 200; the answer is nil otherwise.
func (f *setFixtures) call(t *testing.T, handler http.HandlerFunc, method, target string,
) (*httptest.ResponseRecorder, *api.OutTransferSet) {
	t.Helper()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(method, target, nil))

	if rec.Code != http.StatusOK {
		return rec, nil
	}

	var answer api.OutTransferSet
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &answer), rec.Body.String())

	return rec, &answer
}

// view reads a transfer as the monitoring sees it, live or in the history.
func (f *setFixtures) view(t *testing.T, id int64) *model.NormalizedTransferView {
	t.Helper()

	var view model.NormalizedTransferView
	require.NoError(t, f.db.Get(&view, "id=?", id).Run())

	return &view
}

// guestFailure adds a failed transfer of the first rule requested by another
// account, "guest", to another partner, "other": the requester and the
// requested then tell the transfers of the fixtures apart (#843).
func (f *setFixtures) guestFailure(t *testing.T) *model.Transfer {
	t.Helper()

	other := &model.RemoteAgent{Name: "other", Protocol: testProto1, Address: types.Addr("localhost", 2)}
	require.NoError(t, f.db.Insert(other).Run())

	guest := &model.RemoteAccount{RemoteAgentID: other.ID, Login: "guest"}
	require.NoError(t, f.db.Insert(guest).Run())

	var client model.Client
	require.NoError(t, f.db.Get(&client, "name=?", "client").Run())

	trans := &model.Transfer{
		RuleID: f.orders.ID, ClientID: client.NullableID(), RemoteAccountID: guest.NullableID(),
		SrcFilename: "order-9.edi", DestFilename: "order-9.edi", Status: types.StatusError,
		Filesize: model.UnknownSize, Start: time.Date(2030, 1, 1, 1, 0, 0, 0, time.Local),
		ErrCode: types.TeDataTransfer, ErrDetails: "connection lost", Step: types.StepData,
	}
	require.NoError(t, f.db.Insert(trans).Run())

	return trans
}

func TestResumeTransferSet(t *testing.T) {
	t.Parallel()

	logger := testhelpers.GetTestLogger(t)

	t.Run("Given the filters of the list", func(t *testing.T) {
		t.Parallel()

		f := newSetFixtures(t)
		rec, answer := f.call(t, resumeTransfers(logger, f.db), http.MethodPut, "/api/transfers/resume-all?rule=orders")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		assert.Equal(t, 1, answer.Done, "the failed transfer of the rule")
		assert.Equal(t, 2, answer.Refused)
		assert.Equal(t, []api.OutTransferRefusal{
			{ID: f.planned.ID, Reason: api.RefusalState},
			{ID: f.cancelled.ID, Reason: api.RefusalState},
		}, answer.Refusals, "neither a planned transfer nor a finished one of the rule resumes")
		assert.Empty(t, answer.Created, "a resume creates no transfer")

		assert.Equal(t, types.StatusPlanned, f.view(t, f.failed.ID).Status, "resumed: planned again")
		assert.Equal(t, types.StatusError, f.view(t, f.otherFail.ID).Status, "out of the filters: intact")
	})

	t.Run("Given the requested and the requester", func(t *testing.T) {
		t.Parallel()

		f := newSetFixtures(t)
		guest := f.guestFailure(t)
		handler := resumeTransfers(logger, f.db)

		rec, answer := f.call(t, handler, http.MethodPut, "/api/transfers/resume-all?rule=orders&requested=partner")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		assert.Equal(t, 1, answer.Done, "the failed transfer of the rule requested to the partner")
		assert.Equal(t, []api.OutTransferRefusal{
			{ID: f.planned.ID, Reason: api.RefusalState},
			{ID: f.cancelled.ID, Reason: api.RefusalState},
		}, answer.Refusals, "the other transfers of the rule requested to the partner, refused as before")
		assert.Equal(t, types.StatusPlanned, f.view(t, f.failed.ID).Status, "resumed")
		assert.Equal(t, types.StatusError, f.view(t, guest.ID).Status, "requested to another partner: intact")

		rec, answer = f.call(t, handler, http.MethodPut, "/api/transfers/resume-all?requester=guest")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		assert.Equal(t, 1, answer.Done, "the transfer requested by the account")
		assert.Equal(t, 0, answer.Refused)
		assert.Equal(t, types.StatusPlanned, f.view(t, guest.ID).Status, "resumed")
		assert.Equal(t, types.StatusError, f.view(t, f.otherFail.ID).Status, "requested by another account: intact")
	})

	t.Run("Given transfers left out", func(t *testing.T) {
		t.Parallel()

		f := newSetFixtures(t)
		rec, answer := f.call(t, resumeTransfers(logger, f.db), http.MethodPut,
			"/api/transfers/resume-all?rule=orders&except="+f.id(f.failed.ID))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		assert.Equal(t, 0, answer.Done)
		assert.Equal(t, []api.OutTransferRefusal{
			{ID: f.planned.ID, Reason: api.RefusalState},
			{ID: f.cancelled.ID, Reason: api.RefusalState},
		}, answer.Refusals)
		assert.Equal(t, types.StatusError, f.view(t, f.failed.ID).Status, "left out: intact")
	})

	t.Run("Given identifiers", func(t *testing.T) {
		t.Parallel()

		f := newSetFixtures(t)
		rec, answer := f.call(t, resumeTransfers(logger, f.db), http.MethodPut,
			"/api/transfers/resume-all?rule=orders&ids="+f.id(f.failed.ID)+"&ids="+f.id(f.otherFail.ID))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		assert.Equal(t, 1, answer.Done, "the given transfer of the rule")
		assert.Equal(t, 0, answer.Refused, "an identifier out of the filters is ignored")
		assert.Equal(t, types.StatusError, f.view(t, f.otherFail.ID).Status, "out of the filters: intact")
	})

	t.Run("Given no filter", func(t *testing.T) {
		t.Parallel()

		f := newSetFixtures(t)
		rec, answer := f.call(t, resumeTransfers(logger, f.db), http.MethodPut, "/api/transfers/resume-all")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		assert.Equal(t, 2, answer.Done, "every failed transfer")
		assert.Equal(t, []api.OutTransferRefusal{
			{ID: f.planned.ID, Reason: api.RefusalState},
			{ID: f.cancelled.ID, Reason: api.RefusalState},
		}, answer.Refusals, "the planned transfer and the finished one, in the order of the identifiers")
	})

	t.Run("Given bad parameters", func(t *testing.T) {
		t.Parallel()

		f := newSetFixtures(t)
		handler := resumeTransfers(logger, f.db)

		for _, target := range []string{
			"/api/transfers/resume-all?ids=abc",
			"/api/transfers/resume-all?except=-1",
			"/api/transfers/resume-all?ids=1&except=2",
			"/api/transfers/resume-all?start=yesterday",
		} {
			rec, _ := f.call(t, handler, http.MethodPut, target)
			assert.Equalf(t, http.StatusBadRequest, rec.Code, "%s: %s", target, rec.Body.String())
		}

		assert.Equal(t, types.StatusError, f.view(t, f.failed.ID).Status, "nothing was resumed")
	})
}

func TestRetryTransferSet(t *testing.T) {
	t.Parallel()

	logger := testhelpers.GetTestLogger(t)

	t.Run("Given a finished transfer and a date", func(t *testing.T) {
		t.Parallel()

		f := newSetFixtures(t)
		date := time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC)
		rec, answer := f.call(t, retryTransfers(logger, f.db), http.MethodPut,
			"/api/transfers/retry-all?status=CANCELLED&date="+date.Format(time.RFC3339))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		assert.Equal(t, 1, answer.Done)
		assert.Equal(t, 0, answer.Refused)
		require.Len(t, answer.Created, 1, "the new transfer")

		var created model.Transfer
		require.NoError(t, f.db.Get(&created, "id=?", answer.Created[0]).Run())
		assert.Equal(t, types.StatusPlanned, created.Status)
		assert.Equal(t, f.orders.ID, created.RuleID)
		assert.Equal(t, "order-0.edi", created.SrcFilename)
		assert.True(t, created.Start.Equal(date), "planned at the date given: %s", created.Start)
	})

	t.Run("Given the filters of the list", func(t *testing.T) {
		t.Parallel()

		f := newSetFixtures(t)
		rec, answer := f.call(t, retryTransfers(logger, f.db), http.MethodPut, "/api/transfers/retry-all?rule=orders")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		assert.Equal(t, 2, answer.Done, "the failed transfer and the cancelled one")
		assert.Equal(t, []api.OutTransferRefusal{{ID: f.planned.ID, Reason: api.RefusalState}}, answer.Refusals,
			"a planned transfer is not retried")
		assert.Len(t, answer.Created, 2)
	})

	t.Run("Given a bad date", func(t *testing.T) {
		t.Parallel()

		f := newSetFixtures(t)
		rec, _ := f.call(t, retryTransfers(logger, f.db), http.MethodPut, "/api/transfers/retry-all?date=tomorrow")
		assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	})
}

func TestCancelTransferSet(t *testing.T) {
	t.Parallel()

	logger := testhelpers.GetTestLogger(t)

	t.Run("Given the filters of the list", func(t *testing.T) {
		t.Parallel()

		f := newSetFixtures(t)
		rec, answer := f.call(t, cancelTransfers(logger, f.db), http.MethodDelete, "/api/transfers?rule=orders")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		assert.Equal(t, 2, answer.Done, "the failed and the planned transfers of the rule")
		assert.Equal(t, []api.OutTransferRefusal{{ID: f.cancelled.ID, Reason: api.RefusalState}}, answer.Refusals,
			"a finished transfer is not cancelled")

		for _, id := range []int64{f.failed.ID, f.planned.ID} {
			view := f.view(t, id)
			assert.False(t, view.IsTransfer, "gone to the history")
			assert.Equal(t, types.StatusCancelled, view.Status)
		}

		other := f.view(t, f.otherFail.ID)
		assert.True(t, other.IsTransfer, "out of the filters: intact")
		assert.Equal(t, types.StatusError, other.Status)
	})

	t.Run("Given a target and the filters", func(t *testing.T) {
		t.Parallel()

		f := newSetFixtures(t)
		rec, answer := f.call(t, cancelTransfers(logger, f.db), http.MethodDelete, "/api/transfers?target=planned&rule=orders")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		assert.Equal(t, 1, answer.Done, "the planned transfer of the rule")
		assert.Equal(t, 0, answer.Refused)
		assert.Equal(t, types.StatusError, f.view(t, f.failed.ID).Status, "the failed one is not of the target")
	})

	t.Run("Given the requested alone", func(t *testing.T) {
		t.Parallel()

		f := newSetFixtures(t)
		guest := f.guestFailure(t)
		rec, answer := f.call(t, cancelTransfers(logger, f.db), http.MethodDelete, "/api/transfers?requested=other")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		assert.Equal(t, 1, answer.Done, "the requested names a set: the transfer requested to that partner")
		assert.Equal(t, 0, answer.Refused)
		assert.False(t, f.view(t, guest.ID).IsTransfer, "cancelled, gone to the history")
		assert.True(t, f.view(t, f.failed.ID).IsTransfer, "requested to another partner: intact")
	})

	t.Run("Given a target alone", func(t *testing.T) {
		t.Parallel()

		f := newSetFixtures(t)
		rec, _ := f.call(t, cancelTransfers(logger, f.db), http.MethodDelete, "/api/transfers?target=error")

		assert.Equal(t, http.StatusAccepted, rec.Code, "the former behaviour, unchanged")
		assert.Equal(t, "Transfers canceled successfully", rec.Body.String())
		assert.False(t, f.view(t, f.failed.ID).IsTransfer)
		assert.False(t, f.view(t, f.otherFail.ID).IsTransfer)
		assert.Equal(t, types.StatusPlanned, f.view(t, f.planned.ID).Status)
	})

	t.Run("Given bad parameters", func(t *testing.T) {
		t.Parallel()

		f := newSetFixtures(t)
		handler := cancelTransfers(logger, f.db)

		rec, _ := f.call(t, handler, http.MethodDelete, "/api/transfers")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "missing 'target' parameter", "the former behaviour, unchanged")

		rec, _ = f.call(t, handler, http.MethodDelete, "/api/transfers?target=bogus&rule=orders")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), `unknown cancel target "bogus"`)

		assert.True(t, f.view(t, f.failed.ID).IsTransfer, "nothing was cancelled")
	})
}

// TestTransferSetRoutes: the grouped gestures are routes of the API that
// require the write right on the transfers, next to the unit routes.
func TestTransferSetRoutes(t *testing.T) {
	t.Parallel()

	f := newSetFixtures(t)

	writer := &model.User{
		Username: "writer", PasswordHash: password(t, "writer_pwd"),
		Permissions: model.PermTransfersRead | model.PermTransfersWrite,
	}
	require.NoError(t, f.db.Insert(writer).Run())

	reader := &model.User{Username: "reader", PasswordHash: password(t, "reader_pwd"), Permissions: model.PermTransfersRead}
	require.NoError(t, f.db.Insert(reader).Run())

	router := mux.NewRouter()
	MakeRESTHandler(testhelpers.GetTestLogger(t), f.db, router)

	serve := func(user, password, method, target string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, target, nil)
		req.SetBasicAuth(user, password)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		return rec
	}

	for _, route := range []struct{ method, target string }{
		{http.MethodPut, "/transfers/resume-all?rule=orders"},
		{http.MethodPut, "/transfers/retry-all?rule=orders"},
		{http.MethodDelete, "/transfers?rule=orders"},
	} {
		rec := serve("reader", "reader_pwd", route.method, route.target)
		assert.Equalf(t, http.StatusForbidden, rec.Code, "%s %s without the write right", route.method, route.target)
	}

	rec := serve("writer", "writer_pwd", http.MethodPut, "/transfers/resume-all?rule=orders")
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"done":1`)

	rec = serve("writer", "writer_pwd", http.MethodGet, "/transfers/"+f.id(f.planned.ID))
	assert.Equal(t, http.StatusOK, rec.Code, "the unit route is intact")
}

package rest

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/batch"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// The gestures of the transfer monitoring applied to a set of transfers at
// once (#793): PUT /api/transfers/resume-all, PUT /api/transfers/retry-all and
// DELETE /api/transfers with the filters of the list. The set is what GET
// /api/transfers lists with the same filters, minus the identifiers of
// "except", or the identifiers of "ids" that belong to the filters; the batch
// package applies the gesture, as the web interface does, and the answer
// counts what was done and what was left aside, with its reason.

// The parameters of the filters of the list of the transfers, and those of
// the identifiers of a set.
const (
	filterRuleParam      = "rule"
	filterStatusParam    = "status"
	filterRequesterParam = "requester"
	filterRequestedParam = "requested"
	filterFollowIDParam  = "followID"
	filterStartParam     = "start"
	setIDsParam          = "ids"
	setExceptParam       = "except"

	// The stop of the running transfers of a set is bounded by setCancelTimeout.
	setCancelTimeout = 10 * time.Second
)

// setParams are the parameters that name a set: the filters of the list and
// the identifiers.
func setParams() []string {
	return []string{
		filterRuleParam, filterStatusParam, filterRequesterParam, filterRequestedParam,
		filterFollowIDParam, filterStartParam, setIDsParam, setExceptParam,
	}
}

// namesSet says whether a request names a set of transfers; without any of
// these parameters, the grouped cancellation keeps its former behaviour.
func namesSet(form url.Values) bool {
	return slices.ContainsFunc(setParams(), func(name string) bool { return len(form[name]) > 0 })
}

// parseSetIDs reads the identifiers of a parameter; one that is not a
// positive integer is refused.
func parseSetIDs(name string, values []string) ([]int64, error) {
	ids := make([]int64, 0, len(values))

	for _, value := range values {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil || id <= 0 {
			return nil, badRequestf("%q is not a transfer identifier (parameter %q)", value, name)
		}

		ids = append(ids, id)
	}

	return ids, nil
}

// transferSet enumerates the set of a request: the transfers of the filters
// of the list (and of the given statuses), minus those of "except", or those
// of "ids" that belong to the filters; an identifier out of the filters is
// ignored. The identifiers come in ascending order.
func transferSet(r *http.Request, db *database.DB, statuses []types.TransferStatus) ([]int64, error) {
	if err := r.ParseForm(); err != nil {
		return nil, badRequestf("cannot parse form data: %v", err)
	}

	only, err := parseSetIDs(setIDsParam, r.Form[setIDsParam])
	if err != nil {
		return nil, err
	}

	except, err := parseSetIDs(setExceptParam, r.Form[setExceptParam])
	if err != nil {
		return nil, err
	}

	if len(only) > 0 && len(except) > 0 {
		return nil, badRequestf("the parameters %q and %q cannot be given together", setIDsParam, setExceptParam)
	}

	var transfers model.NormalizedTransfers

	query := db.Select(&transfers)
	if filterErr := transferFilters(r, query); filterErr != nil {
		return nil, filterErr
	}

	if len(statuses) > 0 {
		query.In("status", utils.AsAny(statuses)...)
	}

	if len(only) > 0 {
		query.In("id", utils.AsAny(only)...)
	}

	query.OrderBy("id", true)

	set, err := batch.IDsOf(query, &transfers)
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate the set of transfers: %w", err)
	}

	return slices.DeleteFunc(set, func(id int64) bool { return slices.Contains(except, id) }), nil
}

// setAnswer is the counted answer of a gesture applied to a set.
func setAnswer(result *batch.Result) *api.OutTransferSet {
	answer := &api.OutTransferSet{
		Done:     len(result.Done),
		Refused:  len(result.Refused),
		Refusals: make([]api.OutTransferRefusal, len(result.Refused)),
		Created:  result.New,
	}

	for i, refusal := range result.Refused {
		answer.Refusals[i] = api.OutTransferRefusal{ID: refusal.ID, Reason: string(refusal.Reason)}
	}

	return answer
}

// targetStatuses are the statuses a target of the grouped cancellation names.
func targetStatuses(target string) ([]types.TransferStatus, error) {
	switch target {
	case "":
		return nil, nil
	case "planned":
		return []types.TransferStatus{types.StatusPlanned}, nil
	case "running":
		return []types.TransferStatus{types.StatusRunning}, nil
	case "paused":
		return []types.TransferStatus{types.StatusPaused}, nil
	case "interrupted":
		return []types.TransferStatus{types.StatusInterrupted}, nil
	case "error":
		return []types.TransferStatus{types.StatusError}, nil
	case "all":
		return []types.TransferStatus{
			types.StatusPlanned, types.StatusRunning, types.StatusPaused,
			types.StatusInterrupted, types.StatusError,
		}, nil
	default:
		return nil, badRequestf("unknown cancel target %q", target)
	}
}

// resumeTransfers serves PUT /api/transfers/resume-all: the transfers of the
// set resume where they stopped, now; a transfer the model refuses is counted
// with its reason without stopping the set, an error of the base resumes
// nothing.
func resumeTransfers(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		set, err := transferSet(r, db, nil)
		if handleError(w, logger, err) {
			return
		}

		result, err := batch.ResumeSet(db, set, time.Now())
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, setAnswer(result)))
	}
}

// retryTransfers serves PUT /api/transfers/retry-all: a new transfer is made
// from each transfer of the set, planned at the given date (now by default),
// as the unit retry does.
func retryTransfers(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		set, err := transferSet(r, db, nil)
		if handleError(w, logger, err) {
			return
		}

		date := time.Now()

		if dateStr := r.FormValue("date"); dateStr != "" {
			if date, err = time.Parse(time.RFC3339Nano, dateStr); err != nil {
				handleError(w, logger, badRequestf("%q is not a valid date", dateStr))

				return
			}
		}

		result, err := batch.RetrySet(db, set, date)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, setAnswer(result)))
	}
}

// cancelTransferSet serves DELETE /api/transfers when the request names a
// set: its transfers are cancelled, the running ones stopped first; the
// target, when given, narrows the set to the statuses it names.
func cancelTransferSet(w http.ResponseWriter, r *http.Request, logger *log.Logger, db *database.DB) {
	statuses, err := targetStatuses(r.FormValue("target"))
	if handleError(w, logger, err) {
		return
	}

	set, err := transferSet(r, db, statuses)
	if handleError(w, logger, err) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), setCancelTimeout)
	defer cancel()

	result, err := batch.CancelSet(ctx, db, logger, set)
	if handleError(w, logger, err) {
		return
	}

	handleError(w, logger, writeJSON(w, setAnswer(result)))
}

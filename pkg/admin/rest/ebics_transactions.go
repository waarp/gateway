package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func getEbicsTransactionSegments(
	db database.ReadAccess,
	transactionID int64,
) ([]*api.OutEbicsTransactionSegment, error) {
	var segments model.EbicsTransactionSegments
	if err := db.Select(&segments).Owner().Where("ebics_transaction_id=?", transactionID).
		OrderBy("segment_number", true).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve EBICS transaction segments for transaction %d: %w", transactionID, err)
	}

	out := make([]*api.OutEbicsTransactionSegment, len(segments))
	for i, segment := range segments {
		out[i] = DBEbicsTransactionSegmentToREST(segment)
	}

	return out, nil
}

func getEbicsTransaction(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		transaction, err := getDBEbicsTransaction(r, db)
		if handleError(w, logger, err) {
			return
		}

		segments, err := getEbicsTransactionSegments(db, transaction.ID)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, map[string]any{
			"transaction": DBEbicsTransactionToREST(transaction),
			"segments":    segments,
		}))
	}
}

func listEbicsTransactionSegments(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		transaction, err := getDBEbicsTransaction(r, db)
		if handleError(w, logger, err) {
			return
		}

		segments, err := getEbicsTransactionSegments(db, transaction.ID)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, map[string]any{"segments": segments}))
	}
}

func getEbicsTransactionSegment(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		segment, err := getDBEbicsTransactionSegment(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, DBEbicsTransactionSegmentToREST(segment)))
	}
}

//nolint:dupl // list handlers stay explicit per EBICS resource
func listEbicsTransactions(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default":      order{col: "id", asc: false},
		"id+":          order{col: "id", asc: true},
		"id-":          order{col: "id", asc: false},
		"transaction+": order{col: "transaction_id", asc: true},
		"transaction-": order{col: "transaction_id", asc: false},
		"status+":      order{col: "status", asc: true},
		"status-":      order{col: "status", asc: false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var transactions model.EbicsTransactions

		query, err := parseSelectQuery(r, db, validSorting, &transactions)
		if handleError(w, logger, err) {
			return
		}

		query.Owner()
		if err = query.Run(); handleError(w, logger, err) {
			return
		}

		out := make([]*api.OutEbicsTransaction, len(transactions))
		for i, transaction := range transactions {
			out[i] = DBEbicsTransactionToREST(transaction)
		}

		handleError(w, logger, writeJSON(w, map[string]any{"transactions": out}))
	}
}

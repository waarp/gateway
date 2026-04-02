package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	ebicsmodule "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics"
)

func getEbicsOperation(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		operation, err := getDBEbicsOperation(r, db)
		if handleError(w, logger, err) {
			return
		}

		out, err := DBEbicsOperationDetailToREST(db, operation)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, out))
	}
}

func listEbicsOperations(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default":    order{col: "id", asc: false},
		"id+":        order{col: "id", asc: true},
		"id-":        order{col: "id", asc: false},
		"status+":    order{col: "status", asc: true},
		"status-":    order{col: "status", asc: false},
		"created+":   order{col: "created_at", asc: true},
		"created-":   order{col: "created_at", asc: false},
		"orderType+": order{col: "order_type", asc: true},
		"orderType-": order{col: "order_type", asc: false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var operations model.EbicsOperations

		query, err := parseSelectQuery(r, db, validSorting, &operations)
		if handleError(w, logger, err) {
			return
		}

		query.Owner()
		if err = query.Run(); handleError(w, logger, err) {
			return
		}

		out := make([]*api.OutEbicsOperation, len(operations))
		for i, operation := range operations {
			out[i], err = DBEbicsOperationToREST(operation)
			if handleError(w, logger, err) {
				return
			}
		}

		handleError(w, logger, writeJSON(w, map[string]any{"operations": out}))
	}
}

func executeEbicsReportingOperation(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request := &api.InEbicsReportingAction{}
		if err := readJSON(r, request); handleError(w, logger, err) {
			return
		}

		operation, err := ebicsmodule.ExecuteReportingAction(r.Context(), db, &ebicsmodule.ReportingActionInput{
			ClientID:          request.ClientID,
			EbicsSubscriberID: request.EbicsSubscriberID,
			OrderType:         request.OrderType,
			OrderID:           request.OrderID,
			Service:           restServiceRefToRuntime(request.Service),
			ServiceFilters:    restServiceRefsToRuntime(request.ServiceFilters),
			CompleteOrderData: request.CompleteOrderData,
			FetchLimit:        request.FetchLimit,
			FetchOffset:       request.FetchOffset,
			Metadata:          request.Metadata,
		})
		if handleError(w, logger, err) {
			return
		}

		out, err := DBEbicsOperationToREST(operation)
		if handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, fmt.Sprint(operation.ID)))
		w.WriteHeader(http.StatusCreated)
		handleError(w, logger, writeJSON(w, out))
	}
}

func executeEbicsSignatureOperation(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request := &api.InEbicsSignatureAction{}
		if err := readJSON(r, request); handleError(w, logger, err) {
			return
		}

		operation, err := ebicsmodule.ExecuteSignatureAction(r.Context(), db, &ebicsmodule.SignatureActionInput{
			ClientID:          request.ClientID,
			EbicsSubscriberID: request.EbicsSubscriberID,
			OrderType:         request.OrderType,
			OrderID:           request.OrderID,
			Service:           restServiceRefToRuntime(request.Service),
			OrderData:         request.OrderData,
			SignatureData:     request.SignatureData,
			Metadata:          request.Metadata,
		})
		if handleError(w, logger, err) {
			return
		}

		out, err := DBEbicsOperationToREST(operation)
		if handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, fmt.Sprint(operation.ID)))
		w.WriteHeader(http.StatusCreated)
		handleError(w, logger, writeJSON(w, out))
	}
}

func restServiceRefToRuntime(ref *api.InEbicsServiceRef) *ebicsmodule.ServiceRef {
	if ref == nil {
		return nil
	}

	return &ebicsmodule.ServiceRef{
		ServiceName:   ref.ServiceName,
		ServiceOption: ref.ServiceOption,
		Scope:         ref.Scope,
		MsgName:       ref.MsgName,
		ContainerType: ref.ContainerType,
	}
}

func restServiceRefsToRuntime(refs []*api.InEbicsServiceRef) []ebicsmodule.ServiceRef {
	if len(refs) == 0 {
		return nil
	}

	items := make([]ebicsmodule.ServiceRef, 0, len(refs))
	for _, ref := range refs {
		if ref == nil {
			continue
		}
		items = append(items, *restServiceRefToRuntime(ref))
	}

	return items
}

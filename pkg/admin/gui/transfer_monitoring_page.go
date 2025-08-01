package gui

import (
	"net/http"
	"strconv"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const LimitTransfer = 10

func listTransfer(db *database.DB, r *http.Request) ([]*model.NormalizedTransferView, FiltersPagination) {
	filter := FiltersPagination{
		Offset:          0,
		Limit:           LimitTransfer,
		OrderAsc:        false,
		DisableNext:     false,
		DisablePrevious: false,
	}

	urlParams := r.URL.Query()
	if urlParams.Get("orderAsc") == "true" {
		filter.OrderAsc = true
	} else if urlParams.Get("orderAsc") == "false" {
		filter.OrderAsc = false
	}

	if limitRes := urlParams.Get("limit"); limitRes != "" {
		if l, err := strconv.Atoi(limitRes); err == nil {
			filter.Limit = l
		}
	}

	if offsetRes := urlParams.Get("offset"); offsetRes != "" {
		if o, err := strconv.Atoi(offsetRes); err == nil {
			filter.Offset = o
		}
	}

	paginationPageTransfer(&filter, r)

	transfer := internal.StartTransferQuery(db, "start", filter.OrderAsc).Limit(filter.Limit, filter.Offset*filter.Limit)

	transfers, err := transfer.Run()
	if err != nil {
		return nil, filter
	}

	return transfers, filter
}

func paginationPageTransfer(filter *FiltersPagination, r *http.Request) {
	if r.URL.Query().Get("previous") == "true" && filter.Offset > 0 {
		filter.Offset--
	}

	if r.URL.Query().Get("next") == "true" {
		filter.Offset++
	}

	if filter.Offset == 0 {
		filter.DisablePrevious = true
	}
}

func transferMonitoringPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tabTranslated := //nolint:forcetypeassert //u
			pageTranslated("transfer_monitoring_page", userLanguage.(string)) //nolint:errcheck //u
		transferList, filter := listTransfer(db, r)

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Error("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		currentPage := filter.Offset + 1

		if r.URL.Query().Get("partial") == "true" {
			if err := transferMonitoringTemplate.ExecuteTemplate(w, "transfer_monitoring_tbody", map[string]any{
				"myPermission": myPermission,
				"tab":          tabTranslated,
				"username":     user.Username,
				"language":     userLanguage,
				"transfer":     transferList,
				"filter":       filter,
				"Request":      r,
				"currentPage":  currentPage,
			}); err == nil {
				return
			}
		}

		if err := transferMonitoringTemplate.ExecuteTemplate(w, "transfer_monitoring_page", map[string]any{
			"myPermission": myPermission,
			"tab":          tabTranslated,
			"username":     user.Username,
			"language":     userLanguage,
			"transfer":     transferList,
			"filter":       filter,
			"Request":      r,
			"currentPage":  currentPage,
		}); err != nil {
			logger.Error("render transfer_monitoring_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}

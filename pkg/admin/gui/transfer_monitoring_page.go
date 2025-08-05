package gui

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const LimitTransfer = 10

func addTransfer(db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	srcFilename := r.FormValue("transferFile")
	dstFilename := r.FormValue("transferOut")
	ruleName := r.FormValue("transferRule")
	partnerName := r.FormValue("transferPartner")
	accountName := r.FormValue("transferLogin")
	clientName := r.FormValue("transferClient")
	dateStr := r.FormValue("transferDate")
	transferInfosKeys := r.Form["transferInfosKeys[]"]
	transferInfosValues := r.Form["transferInfosValues[]"]

	rule, err := internal.GetRuleByName(db, ruleName)
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	account, err := internal.GetPartnerAccount(db, partnerName, accountName)
	if err != nil {
		return fmt.Errorf("failed to get remote account: %w", err)
	}

	client, err := internal.GetClient(db, clientName)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	var date time.Time
	if dateStr != "" {
		date, err = time.Parse("2006-01-02T15:04", dateStr)
		if err != nil {
			date = time.Now()
		}
	} else {
		date = time.Now()
	}

	transferInfos := make(map[string]any)

	for i := range transferInfosKeys {
		if i < len(transferInfosValues) {
			transferInfos[transferInfosKeys[i]] = transferInfosValues[i]
		}
	}

	err = internal.InsertNewTransfer(db, srcFilename, dstFilename, rule, account, client, date, transferInfos)
	if err != nil {
		return fmt.Errorf("add transfer failed: %w", err)
	}

	return nil
}

func pauseTransfer(db *database.DB, r *http.Request) error {
	transferID := r.FormValue("pauseTransferID")

	id, err := strconv.Atoi(transferID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	transfer, err := internal.GetNormalizedTransferView(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get transfer: %w", err)
	}

	err = internal.PauseTransfer(r.Context(), db, transfer)
	if err != nil {
		return fmt.Errorf("pause transfer failed: %w", err)
	}

	return nil
}

func resumeTransfer(db *database.DB, r *http.Request) error {
	transferID := r.FormValue("resumeTransferID")

	id, err := strconv.Atoi(transferID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	transfer, err := internal.GetNormalizedTransferView(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get transfer: %w", err)
	}

	err = internal.ResumeTransfer(db, transfer)
	if err != nil {
		return fmt.Errorf("resume transfer failed: %w", err)
	}

	return nil
}

func cancelTransfer(db *database.DB, r *http.Request) error {
	transferID := r.FormValue("cancelTransferID")

	id, err := strconv.Atoi(transferID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	transfer, err := internal.GetNormalizedTransferView(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get transfer: %w", err)
	}

	err = internal.CancelTransfer(r.Context(), db, transfer)
	if err != nil {
		return fmt.Errorf("cancel transfer failed: %w", err)
	}

	return nil
}

func rescheduleTransfer(db *database.DB, r *http.Request) error {
	transferID := r.FormValue("rescheduleTransferID")

	id, err := strconv.Atoi(transferID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	transfer, err := internal.GetNormalizedTransferView(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get transfer: %w", err)
	}

	dateStr := r.FormValue("rescheduleTransferDate")

	var date time.Time
	if dateStr != "" {
		date, err = time.Parse("2006-01-02T15:04", dateStr)
		if err != nil {
			date = time.Now()
		}
	} else {
		date = time.Now()
	}

	_, err = internal.ReprogramTransfer(db, transfer, date)
	if err != nil {
		return fmt.Errorf("reschedule transfer failed: %w", err)
	}

	return nil
}

func listTransfer(db *database.DB, r *http.Request) ([]*model.NormalizedTransferView, FiltersPagination) {
	filter := FiltersPagination{
		Offset:          0,
		Limit:           LimitTransfer,
		OrderAsc:        false,
		OrderBy:         "",
		DisableNext:     false,
		DisablePrevious: false,
	}

	urlParams := r.URL.Query()

	order := urlParams.Get("order")
	switch order {
	case "date-end":
		filter.OrderAsc = true
		filter.OrderBy = "start"
	case "date-start":
		filter.OrderAsc = false
		filter.OrderBy = "start"
	case "id-end":
		filter.OrderAsc = true
		filter.OrderBy = "id"
	case "id-start":
		filter.OrderAsc = false
		filter.OrderBy = "id"
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

	transfer := internal.StartTransferQuery(db, filter.OrderBy, filter.OrderAsc)

	totalTransfers, err := transfer.Count()
	if err != nil {
		return nil, filter
	}

	paginationPage(&filter, int(totalTransfers), r)
	transfer.Limit(filter.Limit, filter.Offset*filter.Limit)

	transfers, err := transfer.Run()
	if err != nil {
		return nil, filter
	}

	return transfers, filter
}

//nolint:funlen // unique method
func callMethodsTransferMonitoring(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
) (bool, string, string) {
	if r.Method == http.MethodPost && r.FormValue("transferFile") != "" {
		if newTransferErr := addTransfer(db, r); newTransferErr != nil {
			logger.Error("failed to add transfer: %v", newTransferErr)

			return false, newTransferErr.Error(), "addTransferModal"
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("pauseTransferID") != "" {
		if pauseTransferErr := pauseTransfer(db, r); pauseTransferErr != nil {
			logger.Error("transfer pause failed : %v", pauseTransferErr)

			return false, pauseTransferErr.Error(), ""
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("resumeTransferID") != "" {
		if resumeTransferErr := resumeTransfer(db, r); resumeTransferErr != nil {
			logger.Error("transfer resume failed : %v", resumeTransferErr)

			return false, resumeTransferErr.Error(), ""
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("cancelTransferID") != "" {
		if cancelTransferErr := cancelTransfer(db, r); cancelTransferErr != nil {
			logger.Error("transfer cancel failed : %v", cancelTransferErr)

			return false, cancelTransferErr.Error(), ""
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("rescheduleTransferID") != "" {
		if rescheduleTransferErr := rescheduleTransfer(db, r); rescheduleTransferErr != nil {
			logger.Error("reschesule cancel failed : %v", rescheduleTransferErr)

			return false, rescheduleTransferErr.Error(), ""
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", ""
	}

	return false, "", ""
}

func transferMonitoringPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tabTranslated := //nolint:forcetypeassert //u
			pageTranslated("transfer_monitoring_page", userLanguage.(string)) //nolint:errcheck //u
		transferList, filter := listTransfer(db, r)

		value, errMsg, modalOpen := callMethodsTransferMonitoring(logger, db, w, r)
		if value {
			return
		}

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
				"errMsg":       errMsg,
				"modalOpen":    modalOpen,
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
			"errMsg":       errMsg,
			"modalOpen":    modalOpen,
		}); err != nil {
			logger.Error("render transfer_monitoring_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}

package gui

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

const LimitTransfer = 10

type Status struct {
	DONE        bool
	ERROR       bool
	PLANNED     bool
	RUNNING     bool
	PAUSED      bool
	INTERRUPTED bool
	CANCELLED   bool //nolint:misspell // check the good spelling
}

func addTransfer(db *database.DB, r *http.Request) (int, error) {
	if err := r.ParseForm(); err != nil {
		return -1, fmt.Errorf("failed to parse form: %w", err)
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
		return -1, fmt.Errorf("failed to get rule: %w", err)
	}

	account, err := internal.GetPartnerAccount(db, partnerName, accountName)
	if err != nil {
		return -1, fmt.Errorf("failed to get remote account: %w", err)
	}

	client, err := internal.GetClient(db, clientName)
	if err != nil {
		return -1, fmt.Errorf("failed to get client: %w", err)
	}

	var date time.Time
	if dateStr != "" {
		date, err = time.ParseInLocation("2006-01-02T15:04", dateStr, time.Local)
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

	transfer, err := internal.InsertNewTransfer(db, srcFilename, dstFilename, rule, account, client, date, transferInfos)
	if err != nil {
		return -1, fmt.Errorf("add transfer failed: %w", err)
	}

	return int(transfer.ID), nil
}

func pauseTransfer(db *database.DB, r *http.Request) error {
	transferID := r.FormValue("pauseTransferID")

	id, err := strconv.ParseUint(transferID, 10, 64)
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

	id, err := strconv.ParseUint(transferID, 10, 64)
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

	id, err := strconv.ParseUint(transferID, 10, 64)
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

func rescheduleTransfer(db *database.DB, r *http.Request) (int, error) {
	transferID := r.FormValue("rescheduleTransferID")

	id, err := strconv.ParseUint(transferID, 10, 64)
	if err != nil {
		return -1, fmt.Errorf("failed to convert id to int: %w", err)
	}

	transfer, err := internal.GetNormalizedTransferView(db, int64(id))
	if err != nil {
		return -1, fmt.Errorf("failed to get transfer: %w", err)
	}

	dateStr := r.FormValue("rescheduleTransferDate")

	var date time.Time
	if dateStr != "" {
		date, err = time.ParseInLocation("2006-01-02T15:04", dateStr, time.Local)
		if err != nil {
			date = time.Now()
		}
	} else {
		date = time.Now()
	}

	newTransfer, err := internal.ReprogramTransfer(db, transfer, date)
	if err != nil {
		return -1, fmt.Errorf("reschedule transfer failed: %w", err)
	}

	return int(newTransfer.ID), nil
}

func filterTransfer(filter *Filters, statuses []string, r *http.Request) {
	urlParams := r.URL.Query()

	for _, s := range statuses {
		switch s {
		case "DONE":
			filter.Status.DONE = true
		case "ERROR":
			filter.Status.ERROR = true
		case "PLANNED":
			filter.Status.PLANNED = true
		case "RUNNING":
			filter.Status.RUNNING = true
		case "PAUSED":
			filter.Status.PAUSED = true
		case "INTERRUPTED":
			filter.Status.INTERRUPTED = true
		case "CANCELLED": //nolint:misspell // check the good spelling
			filter.Status.CANCELLED = true //nolint:misspell // check the good spelling
		}
	}

	filterRuleSend := urlParams.Get("filterRuleSend")
	if filterRuleSend != "" {
		filter.FilterRuleSend = filterRuleSend
	}

	filterRuleReceive := urlParams.Get("filterRuleReceive")
	if filterRuleReceive != "" {
		filter.FilterRuleReceive = filterRuleReceive
	}

	dateStartStr := urlParams.Get("dateStart")
	if dateStartStr != "" {
		filter.DateStart = dateStartStr
	}

	dateEndStr := urlParams.Get("dateEnd")
	if dateEndStr != "" {
		filter.DateEnd = dateEndStr
	}

	filterFilePattern := urlParams.Get("filterFilePattern")
	if filterFilePattern != "" {
		filter.FilterFilePattern = filterFilePattern
	}

	filterAgent := urlParams.Get("filterAgent")
	if filterAgent != "" {
		filter.FilterAgent = filterAgent
	}

	filterAccount := urlParams.Get("filterAccount")
	if filterAccount != "" {
		filter.FilterAccount = filterAccount
	}
}

//nolint:gocyclo,cyclop,funlen // all filter
func listTransfer(db *database.DB, r *http.Request) ([]*model.NormalizedTransferView, Filters) {
	filter := Filters{
		Offset:            0,
		Limit:             LimitTransfer,
		OrderAsc:          false,
		OrderBy:           "",
		DisableNext:       false,
		DisablePrevious:   false,
		Status:            Status{},
		FilterRuleSend:    "",
		FilterRuleReceive: "",
	}

	urlParams := r.URL.Query()
	statuses := urlParams["status"]

	filterTransfer(&filter, statuses, r)

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
		if l, err := strconv.ParseUint(limitRes, 10, 64); err == nil {
			filter.Limit = l
		}
	}

	if offsetRes := urlParams.Get("offset"); offsetRes != "" {
		if o, err := strconv.ParseUint(offsetRes, 10, 64); err == nil {
			filter.Offset = o
		}
	}
	var dateStart, dateEnd time.Time
	var err error

	if filter.DateStart != "" {
		dateStart, err = time.ParseInLocation("2006-01-02T15:04", filter.DateStart, time.Local)
		if err != nil {
			dateStart = time.Time{}
		}
	}

	if filter.DateEnd != "" {
		dateEnd, err = time.ParseInLocation("2006-01-02T15:04", filter.DateEnd, time.Local)
		if err != nil {
			dateEnd = time.Time{}
		}
	}

	transfer := internal.StartTransferQuery(db, filter.OrderBy, filter.OrderAsc)

	if len(statuses) > 0 {
		var statusEnums []types.TransferStatus
		for _, s := range statuses {
			statusEnums = append(statusEnums, types.TransferStatus(s))
		}

		transfer.Status(statusEnums...)
	}

	if filter.FilterRuleSend != "" {
		transfer.Rule(filter.FilterRuleSend, true)
	}

	if filter.FilterRuleReceive != "" {
		transfer.Rule(filter.FilterRuleReceive, false)
	}

	switch {
	case !dateStart.IsZero() && !dateEnd.IsZero():
		transfer.Date(dateStart, dateEnd)
	case !dateStart.IsZero():
		transfer.Date(dateStart, time.Now())
	case !dateEnd.IsZero():
		transfer.Date(time.Time{}, dateEnd)
	}

	if filter.FilterFilePattern != "" {
		transfer.FilePattern(filter.FilterFilePattern)
	}

	if filter.FilterAgent != "" && filter.FilterAccount != "" {
		transfer.Requester(filter.FilterAccount)
	} else if filter.FilterAgent != "" && filter.FilterAccount == "" {
		idx := strings.Index(filter.FilterAgent, ":")
		if idx != -1 && idx+1 < len(filter.FilterAgent) {
			agentName := filter.FilterAgent[idx+1:]
			transfer.Requested(agentName)
		}
	}

	totalTransfers, err := transfer.Count()
	if err != nil {
		return nil, filter
	}

	paginationPage(&filter, totalTransfers, r)
	transfer.Limit(int(filter.Limit), int(filter.Offset*filter.Limit))

	transfers, err := transfer.Run()
	if err != nil {
		return nil, filter
	}

	return transfers, filter
}

//nolint:funlen // unique method
func callMethodsTransferMonitoring(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
) (value bool, errMsg, modalOpen string) {
	if r.Method == http.MethodPost && r.FormValue("transferFile") != "" {
		var newTransferID int
		var newTransferErr error

		if newTransferID, newTransferErr = addTransfer(db, r); newTransferErr != nil {
			logger.Error("failed to add transfer: %v", newTransferErr)

			return false, newTransferErr.Error(), "addTransferModal"
		}

		http.Redirect(w, r, r.URL.Path+"?successAddTransfer="+strconv.Itoa(newTransferID), http.StatusSeeOther)

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
		var newTransferID int
		var rescheduleTransferErr error

		if newTransferID, rescheduleTransferErr = rescheduleTransfer(db, r); rescheduleTransferErr != nil {
			logger.Error("reschesule cancel failed : %v", rescheduleTransferErr)

			return false, rescheduleTransferErr.Error(), ""
		}

		http.Redirect(w, r, r.URL.Path+"?successReprogramTransfer="+strconv.Itoa(newTransferID), http.StatusSeeOther)

		return true, "", ""
	}

	return false, "", ""
}

func mapUtilsTemplate(db *database.DB, listPartners []*model.RemoteAgent,
	listServers []*model.LocalAgent,
) (map[string][]string, map[string][]string) {
	listAccountsNames := make(map[string][]string)

	for _, p := range listPartners {
		accounts, err := internal.ListPartnerAccounts(db, p.Name, "login", true, 0, 0)
		if err == nil {
			var accountNames []string
			for _, acc := range accounts {
				accountNames = append(accountNames, acc.Login)
			}
			listAccountsNames[p.Name] = accountNames
		}
	}

	listAgentsNames := make(map[string][]string)

	for _, p := range listPartners {
		if accs, err := internal.ListPartnerAccounts(db, p.Name, "login", true, 0, 0); len(accs) > 0 && err == nil {
			for _, a := range accs {
				listAgentsNames["partner:"+p.Name] = append(listAgentsNames["partner:"+p.Name], a.Login)
			}
		}
	}

	for _, s := range listServers {
		if accs, err := internal.ListServerAccounts(db, s.Name, "login", true, 0, 0); len(accs) > 0 && err == nil {
			for _, a := range accs {
				listAgentsNames["server:"+s.Name] = append(listAgentsNames["server:"+s.Name], a.Login)
			}
		}
	}

	return listAccountsNames, listAgentsNames
}

func listUtilsTemplate(logger *log.Logger, db *database.DB) ([]*model.RemoteAgent,
	[]*model.LocalAgent, []string, []string, []string, []string, []string,
) {
	listPartners, err := internal.ListPartners(db, "name", true, 0, 0)
	if err != nil {
		logger.Error("failed to get list partner: %v", err)
	}

	listServers, err := internal.ListServers(db, "name", true, 0, 0)
	if err != nil {
		logger.Error("failed to get list server: %v", err)
	}

	listClients, err := internal.ListClients(db, "name", true, 0, 0)
	if err != nil {
		logger.Error("failed to get list client: %v", err)
	}

	ruleSend, err := internal.ListRulesByDirection(db, "name", true, 0, 0, true)
	if err != nil {
		logger.Error("failed to get list sending rules: %v", err)
	}

	ruleReceive, err := internal.ListRulesByDirection(db, "name", true, 0, 0, false)
	if err != nil {
		logger.Error("failed to get list reception rules: %v", err)
	}

	var ruleSendNames []string
	for _, r := range ruleSend {
		ruleSendNames = append(ruleSendNames, r.Name)
	}

	var ruleReceiveNames []string
	for _, r := range ruleReceive {
		ruleReceiveNames = append(ruleReceiveNames, r.Name)
	}

	var listPartnersNames []string
	for _, r := range listPartners {
		listPartnersNames = append(listPartnersNames, r.Name)
	}

	var listServersNames []string
	for _, r := range listServers {
		listServersNames = append(listServersNames, r.Name)
	}

	var listClientsNames []string
	for _, r := range listClients {
		listClientsNames = append(listClientsNames, r.Name)
	}

	return listPartners, listServers, listPartnersNames, listServersNames,
		listClientsNames, ruleSendNames, ruleReceiveNames
}

//nolint:funlen // map template is too long
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

		successAddTransfer := r.URL.Query().Get("successAddTransfer")
		successReprogramTransfer := r.URL.Query().Get("successReprogramTransfer")

		listPartners, listServers, listPartnersNames, listServersNames, listClientsNames,
			ruleSendNames, ruleReceiveNames := listUtilsTemplate(logger, db)
		listAccountsNames, listAgentsNames := mapUtilsTemplate(db, listPartners, listServers)

		myPermission := model.MaskToPerms(user.Permissions)
		currentPage := filter.Offset + 1

		if r.URL.Query().Get("partial") == "true" {
			if err := transferMonitoringTemplate.ExecuteTemplate(w, "transfer_monitoring_tbody", map[string]any{
				"myPermission":             myPermission,
				"tab":                      tabTranslated,
				"username":                 user.Username,
				"language":                 userLanguage,
				"transfer":                 transferList,
				"filter":                   filter,
				"Request":                  r,
				"currentPage":              currentPage,
				"listPartners":             listPartnersNames,
				"listServers":              listServersNames,
				"listAccounts":             listAccountsNames,
				"listAgents":               listAgentsNames,
				"listClients":              listClientsNames,
				"ruleSend":                 ruleSendNames,
				"ruleReceive":              ruleReceiveNames,
				"successAddTransfer":       successAddTransfer,
				"successReprogramTransfer": successReprogramTransfer,
				"errMsg":                   errMsg,
				"modalOpen":                modalOpen,
			}); err == nil {
				return
			}
		}

		if err := transferMonitoringTemplate.ExecuteTemplate(w, "transfer_monitoring_page", map[string]any{
			"myPermission":             myPermission,
			"tab":                      tabTranslated,
			"username":                 user.Username,
			"language":                 userLanguage,
			"transfer":                 transferList,
			"filter":                   filter,
			"Request":                  r,
			"currentPage":              currentPage,
			"listPartners":             listPartnersNames,
			"listServers":              listServersNames,
			"listAccounts":             listAccountsNames,
			"listAgents":               listAgentsNames,
			"listClients":              listClientsNames,
			"ruleSend":                 ruleSendNames,
			"ruleReceive":              ruleReceiveNames,
			"successAddTransfer":       successAddTransfer,
			"successReprogramTransfer": successReprogramTransfer,
			"errMsg":                   errMsg,
			"modalOpen":                modalOpen,
		}); err != nil {
			logger.Error("render transfer_monitoring_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}

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

	var remainingTries int64
	if nbOfAttempts := r.FormValue("nbOfAttempts"); nbOfAttempts != "" {
		remainingTries, err = strconv.ParseInt(nbOfAttempts, 10, 8)
		if err != nil {
			return -1, fmt.Errorf("failed to parse remainingTries in int: %w", err)
		}
	}

	retryDelay := ""
	if h := r.FormValue("retryDelaypH"); h != "" {
		retryDelay += h + "h"
	}

	if m := r.FormValue("retryDelayM"); m != "" {
		retryDelay += m + "m"
	}

	if s := r.FormValue("timeoutIcapS"); s != "" {
		retryDelay += s + "s"
	}
	var retryDelayS int32

	if retryDelay != "" {
		firstRetryDelay, retryErr := time.ParseDuration(retryDelay)
		if retryErr == nil {
			retryDelayS = int32(firstRetryDelay.Seconds())
		}
	}

	var retryIncrementFloat float64
	if retryIncrementFactor := r.FormValue("retryIncrementFactor"); retryIncrementFactor != "" {
		retryIncrementFloat, err = strconv.ParseFloat(retryIncrementFactor, 32)
		if err != nil {
			return -1, fmt.Errorf("failed to parse retryIncrement in float: %w", err)
		}
	}

	transfer, err := internal.InsertNewTransfer(db, srcFilename, dstFilename, rule, account, client, date,
		int8(remainingTries), retryDelayS, float32(retryIncrementFloat), transferInfos)
	if err != nil {
		return -1, fmt.Errorf("add transfer failed: %w", err)
	}

	return int(transfer.ID), nil
}

func addRegisterTransfer(db *database.DB, r *http.Request) (int, error) {
	if err := r.ParseForm(); err != nil {
		return -1, fmt.Errorf("failed to parse form: %w", err)
	}

	filename := r.FormValue("preRegisterFile")
	ruleName := r.FormValue("preRegisterRule")
	serverName := r.FormValue("preRegisterServer")
	accountName := r.FormValue("preRegisterLogin")
	dateStr := r.FormValue("preRegisterDate")
	preRegisterInfosKeys := r.Form["preRegisterInfosKeys[]"]
	preRegisterInfosValues := r.Form["preRegisterInfosValues[]"]

	rule, err := internal.GetRuleByName(db, ruleName)
	if err != nil {
		return -1, fmt.Errorf("failed to get rule: %w", err)
	}

	account, err := internal.GetServerAccount(db, serverName, accountName)
	if err != nil {
		return -1, fmt.Errorf("failed to get local account: %w", err)
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

	for i := range preRegisterInfosKeys {
		if i < len(preRegisterInfosValues) {
			transferInfos[preRegisterInfosKeys[i]] = preRegisterInfosValues[i]
		}
	}

	register, registerErr := internal.RegisterNewTransfer(db, filename, rule, account, date, transferInfos)
	if registerErr != nil {
		return -1, fmt.Errorf("add register transfer failed: %w", registerErr)
	}

	return int(register.ID), nil
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

//nolint:gocyclo,cyclop,gocognit,funlen // all filter
func listTransfer(db *database.DB, r *http.Request) ([]*model.NormalizedTransferView, Filters) {
	defaultFilter := Filters{
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

	filter := defaultFilter
	if saved, ok := GetPageFilters(r, "transfer_monitoring_page"); ok {
		filter = saved
	}

	if r.URL.Query().Get("applyFilters") == True {
		filter = defaultFilter
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
) (value bool, errMsg, modalOpen string, modalElement map[string]any) {
	if r.Method == http.MethodPost && r.FormValue("transferFile") != "" {
		var newTransferID int
		var newTransferErr error

		if newTransferID, newTransferErr = addTransfer(db, r); newTransferErr != nil {
			logger.Errorf("failed to add transfer: %v", newTransferErr)
			modalElement = getFormValues(r)

			return false, newTransferErr.Error(), "addTransferModal", modalElement
		}

		http.Redirect(w, r, r.URL.Path+"?successAddTransfer="+strconv.Itoa(newTransferID), http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("preRegisterFile") != "" {
		var newRegisterID int
		var newRegisterErr error

		if newRegisterID, newRegisterErr = addRegisterTransfer(db, r); newRegisterErr != nil {
			logger.Errorf("failed to register transfer: %v", newRegisterErr)
			modalElement = getFormValues(r)

			return false, newRegisterErr.Error(), "preRegisterTransferModal", modalElement
		}

		http.Redirect(w, r, r.URL.Path+"?successAddRegister="+strconv.Itoa(newRegisterID), http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("pauseTransferID") != "" {
		if pauseTransferErr := pauseTransfer(db, r); pauseTransferErr != nil {
			logger.Errorf("transfer pause failed : %v", pauseTransferErr)

			return false, pauseTransferErr.Error(), "", nil
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("resumeTransferID") != "" {
		if resumeTransferErr := resumeTransfer(db, r); resumeTransferErr != nil {
			logger.Errorf("transfer resume failed : %v", resumeTransferErr)

			return false, resumeTransferErr.Error(), "", nil
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("cancelTransferID") != "" {
		if cancelTransferErr := cancelTransfer(db, r); cancelTransferErr != nil {
			logger.Errorf("transfer cancel failed : %v", cancelTransferErr)

			return false, cancelTransferErr.Error(), "", nil
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("rescheduleTransferID") != "" {
		var newTransferID int
		var rescheduleTransferErr error

		if newTransferID, rescheduleTransferErr = rescheduleTransfer(db, r); rescheduleTransferErr != nil {
			logger.Errorf("reschesule cancel failed : %v", rescheduleTransferErr)

			return false, rescheduleTransferErr.Error(), "", nil
		}

		http.Redirect(w, r, r.URL.Path+"?successReprogramTransfer="+strconv.Itoa(newTransferID), http.StatusSeeOther)

		return true, "", "", nil
	}

	return false, "", "", nil
}

func mapUtilsTemplate(db *database.DB, listPartners []*model.RemoteAgent, listServers []*model.LocalAgent,
) (listAccountsPartnerNames, listAccountsServerNames, listAgentsNames map[string][]string) {
	listAccountsPartnerNames = make(map[string][]string)

	for _, p := range listPartners {
		accounts, err := internal.ListPartnerAccounts(db, p.Name, "login", true, 0, 0)
		if err == nil {
			var accountNames []string
			for _, acc := range accounts {
				accountNames = append(accountNames, acc.Login)
			}
			listAccountsPartnerNames[p.Name] = accountNames
		}
	}

	listAccountsServerNames = make(map[string][]string)

	for _, p := range listServers {
		accounts, err := internal.ListServerAccounts(db, p.Name, "login", true, 0, 0)
		if err == nil {
			var accountNames []string
			for _, acc := range accounts {
				accountNames = append(accountNames, acc.Login)
			}
			listAccountsServerNames[p.Name] = accountNames
		}
	}

	listAgentsNames = make(map[string][]string)

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

	return listAccountsPartnerNames, listAccountsServerNames, listAgentsNames
}

func listUtilsTemplate(logger *log.Logger, db *database.DB) (
	listPartners []*model.RemoteAgent, listServers []*model.LocalAgent,
	listPartnersNames, listServersNames, listClientsNames, ruleSendNames, ruleReceiveNames []string,
) {
	listPartners, err := internal.ListPartners(db, "name", true, 0, 0)
	if err != nil {
		logger.Errorf("failed to get list partner: %v", err)
	}

	listServers, err = internal.ListServers(db, "name", true, 0, 0)
	if err != nil {
		logger.Errorf("failed to get list server: %v", err)
	}

	listClients, err := internal.ListClients(db, "name", true, 0, 0)
	if err != nil {
		logger.Errorf("failed to get list client: %v", err)
	}

	ruleSend, err := internal.ListRulesByDirection(db, "name", true, 0, 0, true)
	if err != nil {
		logger.Errorf("failed to get list sending rules: %v", err)
	}

	ruleReceive, err := internal.ListRulesByDirection(db, "name", true, 0, 0, false)
	if err != nil {
		logger.Errorf("failed to get list reception rules: %v", err)
	}

	for _, r := range ruleSend {
		ruleSendNames = append(ruleSendNames, r.Name)
	}

	for _, r := range ruleReceive {
		ruleReceiveNames = append(ruleReceiveNames, r.Name)
	}

	for _, r := range listPartners {
		listPartnersNames = append(listPartnersNames, r.Name)
	}

	for _, r := range listServers {
		listServersNames = append(listServersNames, r.Name)
	}

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

		if pageName := r.URL.Query().Get("clearFiltersPage"); pageName != "" {
			ClearPageFilters(r, pageName)
			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

			return
		}

		PersistPageFilters(r, "transfer_monitoring_page", &filter)

		value, errMsg, modalOpen, modalElement := callMethodsTransferMonitoring(logger, db, w, r)
		if value {
			return
		}

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Errorf("Internal error: %v", err)
		}

		successAddTransfer := r.URL.Query().Get("successAddTransfer")
		successAddRegister := r.URL.Query().Get("successAddRegister")
		successReprogramTransfer := r.URL.Query().Get("successReprogramTransfer")

		listPartners, listServers, listPartnersNames, listServersNames, listClientsNames,
			ruleSendNames, ruleReceiveNames := listUtilsTemplate(logger, db)
		listAccountsPartnerNames, listAccountsServerNames, listAgentsNames := mapUtilsTemplate(db, listPartners, listServers)

		myPermission := model.MaskToPerms(user.Permissions)
		currentPage := filter.Offset + 1

		data := map[string]any{
			"myPermission":             myPermission,
			"tab":                      tabTranslated,
			"username":                 user.Username,
			"language":                 userLanguage,
			"transfer":                 transferList,
			"filter":                   filter,
			"Request":                  r,
			"currentPage":              currentPage,
			"listPartners":             listPartnersNames,
			"listAccountsPartner":      listAccountsPartnerNames,
			"listServers":              listServersNames,
			"listAccountsServer":       listAccountsServerNames,
			"listAgents":               listAgentsNames,
			"listClients":              listClientsNames,
			"ruleSend":                 ruleSendNames,
			"ruleReceive":              ruleReceiveNames,
			"successAddTransfer":       successAddTransfer,
			"successAddRegister":       successAddRegister,
			"successReprogramTransfer": successReprogramTransfer,
			"errMsg":                   errMsg,
			"modalOpen":                modalOpen,
			"modalElement":             modalElement,
		}
		if r.URL.Query().Get("partial") == True {
			if tableErr := transferMonitoringTemplate.ExecuteTemplate(w, "transfer_monitoring_tbody", data); tableErr == nil {
				return
			}
		}

		if tmplErr := transferMonitoringTemplate.ExecuteTemplate(w, "transfer_monitoring_page", data); tmplErr != nil {
			logger.Errorf("render transfer_monitoring_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}

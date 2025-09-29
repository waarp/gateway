package gui

import (
	"encoding/json"
	"fmt"
	"net/http"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

func addRule(db *database.DB, r *http.Request) error {
	var newRule model.Rule

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	if newRuleName := r.FormValue("addRuleName"); newRuleName != "" {
		newRule.Name = newRuleName
	}

	newRule.IsSend = r.FormValue("addRuleIsSend") == True

	if addRuleComment := r.FormValue("addRuleComment"); addRuleComment != "" {
		newRule.Comment = addRuleComment
	}

	if addRulePath := r.FormValue("addRulePath"); addRulePath != "" {
		newRule.Path = addRulePath
	}

	if addRuleLocalDir := r.FormValue("addRuleLocalDir"); addRuleLocalDir != "" {
		newRule.LocalDir = addRuleLocalDir
	}

	if addRuleRemoteDir := r.FormValue("addRuleRemoteDir"); addRuleRemoteDir != "" {
		newRule.RemoteDir = addRuleRemoteDir
	}

	if addRuleTmpLocalRcvDir := r.FormValue("addRuleTmpLocalRcvDir"); addRuleTmpLocalRcvDir != "" {
		newRule.TmpLocalRcvDir = addRuleTmpLocalRcvDir
	}

	if err := internal.InsertRule(db, &newRule); err != nil {
		return fmt.Errorf("failed to add rule: %w", err)
	}

	return nil
}

func editRule(db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	ruleID := r.FormValue("editRuleID")

	id, err := internal.ParseUint[uint64](ruleID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	editRule, err := internal.GetRuleByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	if editRuleName := r.FormValue("editRuleName"); editRuleName != "" {
		editRule.Name = editRuleName
	}

	editRule.IsSend = r.FormValue("editRuleIsSend") == True

	editRule.Comment = r.FormValue("editRuleComment")

	editRule.Path = r.FormValue("editRulePath")

	editRule.LocalDir = r.FormValue("editRuleLocalDir")

	editRule.RemoteDir = r.FormValue("editRuleRemoteDir")

	editRule.TmpLocalRcvDir = r.FormValue("editRuleTmpLocalRcvDir")

	if upErr := internal.UpdateRule(db, editRule); upErr != nil {
		return fmt.Errorf("failed to edit rule: %w", upErr)
	}

	return nil
}

//nolint:dupl // no similar func (is method for rule)
func deleteRule(db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	ruleID := r.FormValue("deleteRule")

	id, err := internal.ParseUint[uint64](ruleID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	rule, err := internal.GetRuleByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	if dlErr := internal.DeleteRule(db, rule); dlErr != nil {
		return fmt.Errorf("failed to delete rule: %w", dlErr)
	}

	return nil
}

func listRule(db *database.DB, r *http.Request) ([]*model.Rule, Filters, string) {
	ruleFound := ""
	defaultFilter := Filters{
		Offset:          0,
		Limit:           DefaultLimitPagination,
		OrderAsc:        true,
		DisableNext:     false,
		DisablePrevious: false,
	}

	filter := defaultFilter
	if saved, ok := GetPageFilters(r, "transfer_rules_management_page"); ok {
		filter = saved
	}

	if r.URL.Query().Get("applyFilters") == True {
		filter = defaultFilter
	}

	urlParams := r.URL.Query()

	if urlParams.Get("orderAsc") != "" {
		filter.OrderAsc = urlParams.Get("orderAsc") == True
	}

	if limitRes := urlParams.Get("limit"); limitRes != "" {
		if l, err := internal.ParseUint[uint64](limitRes); err == nil {
			filter.Limit = l
		}
	}

	if offsetRes := urlParams.Get("offset"); offsetRes != "" {
		if o, err := internal.ParseUint[uint64](offsetRes); err == nil {
			filter.Offset = o
		}
	}

	rule, err := internal.ListRules(db, "name", true, 0, 0)
	if err != nil {
		return nil, Filters{}, ruleFound
	}

	if search := urlParams.Get("search"); search != "" && searchRule(search, rule) == nil {
		ruleFound = False
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		ruleFound = True

		return []*model.Rule{searchRule(search, rule)}, filter, ruleFound
	}

	paginationPage(&filter, uint64(len(rule)), r)

	rules, err := internal.ListRules(db, "name", filter.OrderAsc, int(filter.Limit), int(filter.Offset*filter.Limit))
	if err != nil {
		return nil, Filters{}, ruleFound
	}

	return rules, filter, ruleFound
}

func searchRule(ruleNameSearch string, listRuleSearch []*model.Rule,
) *model.Rule {
	for _, r := range listRuleSearch {
		if r.Name == ruleNameSearch {
			return r
		}
	}

	return nil
}

//nolint:dupl // no similar func
func autocompletionRulesFunc(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		prefix := r.URL.Query().Get("q")

		rules, err := internal.GetRulesLike(db, prefix)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)

			return
		}

		names := make([]string, len(rules))
		for i, u := range rules {
			names[i] = u.Name
		}

		w.Header().Set("Content-Type", "application/json")

		if jsonErr := json.NewEncoder(w).Encode(names); jsonErr != nil {
			http.Error(w, "error json", http.StatusInternalServerError)
		}
	}
}

//nolint:dupl // no similar func
func callMethodsRuleManagement(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
) (value bool, errMsg, modalOpen string, modalElement map[string]any) {
	if r.Method == http.MethodPost && r.FormValue("addRuleName") != "" {
		if newRuleErr := addRule(db, r); newRuleErr != nil {
			logger.Errorf("failed to add rule: %v", newRuleErr)
			modalElement = getFormValues(r)

			return false, newRuleErr.Error(), "addRuleModal", modalElement
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("deleteRule") != "" {
		deleteRuleErr := deleteRule(db, r)
		if deleteRuleErr != nil {
			logger.Errorf("failed to delete rule: %v", deleteRuleErr)

			return false, deleteRuleErr.Error(), "", nil
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("editRuleID") != "" {
		idEdit := r.FormValue("editRuleID")

		id, err := internal.ParseUint[uint64](idEdit)
		if err != nil {
			logger.Errorf("failed to convert id to int: %v", err)

			return false, "", "", nil
		}

		if editRuleErr := editRule(db, r); editRuleErr != nil {
			logger.Errorf("failed to edit rule: %v", editRuleErr)
			modalElement = getFormValues(r)

			return false, editRuleErr.Error(), fmt.Sprintf("editRuleModal_%d", id), modalElement
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", "", nil
	}

	return false, "", "", nil
}

func ruleManagementPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := locale.GetLanguage(r)
		tabTranslated := pageTranslated("transfer_rules_management_page", userLanguage)
		ruleList, filter, ruleFound := listRule(db, r)

		if pageName := r.URL.Query().Get("clearFiltersPage"); pageName != "" {
			ClearPageFilters(r, pageName)
			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

			return
		}

		PersistPageFilters(r, "transfer_rules_management_page", &filter)

		value, errMsg, modalOpen, modalElement := callMethodsRuleManagement(logger, db, w, r)
		if value {
			return
		}

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Errorf("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		currentPage := filter.Offset + 1

		if tmplErr := ruleManagementTemplate.ExecuteTemplate(w, "transfer_rules_management_page", map[string]any{
			"appName":      constants.AppName,
			"version":      version.Num,
			"compileDate":  version.Date,
			"revision":     version.Commit,
			"docLink":      constants.DocLink(userLanguage),
			"myPermission": myPermission,
			"tab":          tabTranslated,
			"username":     user.Username,
			"language":     userLanguage,
			"rule":         ruleList,
			"ruleFound":    ruleFound,
			"filter":       filter,
			"currentPage":  currentPage,
			"errMsg":       errMsg,
			"modalOpen":    modalOpen,
			"modalElement": modalElement,
		}); tmplErr != nil {
			logger.Errorf("render transfer_rules_management_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}

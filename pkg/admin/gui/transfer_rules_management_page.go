package gui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
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

	id, err := strconv.ParseUint(ruleID, 10, 64)
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

	if editRuleComment := r.FormValue("editRuleComment"); editRuleComment != "" {
		editRule.Comment = editRuleComment
	}

	if editRulePath := r.FormValue("editRulePath"); editRulePath != "" {
		editRule.Path = editRulePath
	}

	if editRuleLocalDir := r.FormValue("editRuleLocalDir"); editRuleLocalDir != "" {
		editRule.LocalDir = editRuleLocalDir
	}

	if editRuleRemoteDir := r.FormValue("editRuleRemoteDir"); editRuleRemoteDir != "" {
		editRule.RemoteDir = editRuleRemoteDir
	}

	if editRuleTmpLocalRcvDir := r.FormValue("editRuleTmpLocalRcvDir"); editRuleTmpLocalRcvDir != "" {
		editRule.TmpLocalRcvDir = editRuleTmpLocalRcvDir
	}

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

	id, err := strconv.ParseUint(ruleID, 10, 64)
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
	filter := Filters{
		Offset:          0,
		Limit:           DefaultLimitPagination,
		OrderAsc:        true,
		DisableNext:     false,
		DisablePrevious: false,
	}

	urlParams := r.URL.Query()

	filter.OrderAsc = urlParams.Get("orderAsc") == True

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
) (value bool, errMsg, modalOpen string) {
	if r.Method == http.MethodPost && r.FormValue("addRuleName") != "" {
		if newRuleErr := addRule(db, r); newRuleErr != nil {
			logger.Errorf("failed to add rule: %v", newRuleErr)

			return false, newRuleErr.Error(), "addRuleModal"
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("deleteRule") != "" {
		deleteRuleErr := deleteRule(db, r)
		if deleteRuleErr != nil {
			logger.Errorf("failed to delete rule: %v", deleteRuleErr)

			return false, deleteRuleErr.Error(), ""
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("editRuleID") != "" {
		idEdit := r.FormValue("editRuleID")

		id, err := strconv.ParseUint(idEdit, 10, 64)
		if err != nil {
			logger.Errorf("failed to convert id to int: %v", err)

			return false, "", ""
		}

		if editRuleErr := editRule(db, r); editRuleErr != nil {
			logger.Errorf("failed to edit rule: %v", editRuleErr)

			return false, editRuleErr.Error(), fmt.Sprintf("editRuleModal_%d", id)
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", ""
	}

	return false, "", ""
}

func ruleManagementPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tabTranslated := //nolint:forcetypeassert //u
			pageTranslated("transfer_rules_management_page", userLanguage.(string)) //nolint:errcheck //u
		ruleList, filter, ruleFound := listRule(db, r)

		value, errMsg, modalOpen := callMethodsRuleManagement(logger, db, w, r)
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
		}); tmplErr != nil {
			logger.Errorf("render transfer_rules_management_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}

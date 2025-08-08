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

	if addRuleIsSend := r.FormValue("addRuleIsSend"); addRuleIsSend == "true" {
		newRule.IsSend = true
	} else {
		newRule.IsSend = false
	}

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

	id, err := strconv.Atoi(ruleID)
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

	if editRuleIsSend := r.FormValue("editRuleIsSend"); editRuleIsSend == "true" {
		editRule.IsSend = true
	} else {
		editRule.IsSend = false
	}

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

	if err := internal.UpdateRule(db, editRule); err != nil {
		return fmt.Errorf("failed to edit rule: %w", err)
	}

	return nil
}

//nolint:dupl // no similar func (is method for rule)
func deleteRule(db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	ruleID := r.FormValue("deleteRule")

	id, err := strconv.Atoi(ruleID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	rule, err := internal.GetRuleByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	if err := internal.DeleteRule(db, rule); err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	return nil
}

func listRule(db *database.DB, r *http.Request) ([]*model.Rule, FiltersPagination, string) {
	ruleFound := ""
	filter := FiltersPagination{
		Offset:          0,
		Limit:           LimitPagination,
		OrderAsc:        true,
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

	rule, err := internal.ListRules(db, "name", true, 0, 0)
	if err != nil {
		return nil, FiltersPagination{}, ruleFound
	}

	if search := urlParams.Get("search"); search != "" && searchRule(search, rule) == nil {
		ruleFound = "false"
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		ruleFound = "true"

		return []*model.Rule{searchRule(search, rule)}, filter, ruleFound
	}

	paginationPage(&filter, len(rule), r)

	rules, err := internal.ListRules(db, "name", filter.OrderAsc, filter.Limit, filter.Offset*filter.Limit)
	if err != nil {
		return nil, FiltersPagination{}, ruleFound
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

		if err := json.NewEncoder(w).Encode(names); err != nil {
			http.Error(w, "error json", http.StatusInternalServerError)
		}
	}
}

//nolint:dupl // no similar func
func callMethodsRuleManagement(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
) (bool, string, string) {
	if r.Method == http.MethodPost && r.FormValue("addRuleName") != "" {
		if newRuleErr := addRule(db, r); newRuleErr != nil {
			logger.Error("failed to add rule: %v", newRuleErr)

			return false, newRuleErr.Error(), "addRuleModal"
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("deleteRule") != "" {
		deleteRuleErr := deleteRule(db, r)
		if deleteRuleErr != nil {
			logger.Error("failed to delete rule: %v", deleteRuleErr)

			return false, deleteRuleErr.Error(), ""
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("editRuleID") != "" {
		idEdit := r.FormValue("editRuleID")

		id, err := strconv.Atoi(idEdit)
		if err != nil {
			logger.Error("failed to convert id to int: %v", err)

			return false, "", ""
		}

		if editRuleErr := editRule(db, r); editRuleErr != nil {
			logger.Error("failed to edit rule: %v", editRuleErr)

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
			logger.Error("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		currentPage := filter.Offset + 1

		if err := ruleManagementTemplate.ExecuteTemplate(w, "transfer_rules_management_page", map[string]any{
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
		}); err != nil {
			logger.Error("render transfer_rules_management_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}

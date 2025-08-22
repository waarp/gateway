package gui

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

var (
	ErrServerAlreadyAuthorized        = errors.New("this server is already authorized")
	ErrPartnerAlreadyAuthorized       = errors.New("this partner is already authorized")
	ErrLocalAccountAlreadyAuthorized  = errors.New("this local account is already authorized")
	ErrRemoteAccountAlreadyAuthorized = errors.New("this remote account is already authorized")
)

//nolint:dupl // method for authorized servers
func addAuthorizedServers(ruleID int, db *database.DB, r *http.Request) error {
	var newAuthorizedServer *model.LocalAgent
	var err error

	if err = r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	if addAuthorizedServerName := r.FormValue("addAuthorizedServerName"); addAuthorizedServerName != "" {
		if newAuthorizedServer, err = internal.GetServer(db, addAuthorizedServerName); err != nil {
			return fmt.Errorf("failed to get server: %w", err)
		}
	}

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	err = internal.AddRuleAccess(db, rule, newAuthorizedServer)
	if err != nil {
		return fmt.Errorf("failed to authorize server: %w", err)
	}

	return nil
}

func editAuthorizedServers(ruleID int, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	oldName := r.FormValue("editAuthorizedServerOldName")
	editAuthorizedServerName := r.FormValue("editAuthorizedServerName")

	servers, err := internal.ListAuthorizedServers(db, rule)
	if err != nil {
		return nil
	}

	for _, s := range servers {
		if s.Name == editAuthorizedServerName {
			return ErrServerAlreadyAuthorized
		}
	}

	oldServer, err := internal.GetServer(db, oldName)
	if err != nil {
		return fmt.Errorf("failed to get old server: %w", err)
	}

	newServer, err := internal.GetServer(db, editAuthorizedServerName)
	if err != nil {
		return fmt.Errorf("failed to get new server: %w", err)
	}

	if dlErr := internal.DeleteRuleAccess(db, rule, oldServer); dlErr != nil {
		return fmt.Errorf("failed to remove old access: %w", dlErr)
	}

	if addErr := internal.AddRuleAccess(db, rule, newServer); addErr != nil {
		return fmt.Errorf("failed to add new access: %w", addErr)
	}

	return nil
}

//nolint:dupl // method for authorized servers
func deleteAuthorizedServers(ruleID int, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	errorTaskName := r.FormValue("deleteAuthorizedServer")

	server, err := internal.GetServer(db, errorTaskName)
	if err != nil {
		return fmt.Errorf("failed to get new server: %w", err)
	}

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	if err = internal.DeleteRuleAccess(db, rule, server); err != nil {
		return fmt.Errorf("failed to delete access: %w", err)
	}

	return nil
}

//nolint:dupl // method for authorized servers
func callMethodsAuthorizedServersRules(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
	ruleID int,
) (value bool, errMsg, modalOpen string) {
	if r.Method == http.MethodPost && r.FormValue("addAuthorizedServerName") != "" {
		addAuthorizedServerErr := addAuthorizedServers(ruleID, db, r)
		if addAuthorizedServerErr != nil {
			logger.Errorf("failed to add authorized server: %v", addAuthorizedServerErr)

			return false, addAuthorizedServerErr.Error(), "addAuthorizedServerModal"
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("editAuthorizedServerOldName") != "" {
		if editAuthorizedServerErr := editAuthorizedServers(ruleID, db, r); editAuthorizedServerErr != nil {
			logger.Errorf("failed to edit authorized server: %v", editAuthorizedServerErr)

			return false, editAuthorizedServerErr.Error(),
				"editAuthorizedServerModal_" + r.FormValue("editAuthorizedServerOldName")
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("deleteAuthorizedServer") != "" {
		if deleteAuthorizedServerErr := deleteAuthorizedServers(ruleID, db, r); deleteAuthorizedServerErr != nil {
			logger.Errorf("failed to delete authorized server: %v", deleteAuthorizedServerErr)

			return false, deleteAuthorizedServerErr.Error(), ""
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	return false, "", ""
}

//nolint:dupl // method for partners
func addAuthorizedPartners(ruleID int, db *database.DB, r *http.Request) error {
	var newAuthorizedPartner *model.RemoteAgent
	var err error

	if err = r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	if addAuthorizedPartnerName := r.FormValue("addAuthorizedPartnerName"); addAuthorizedPartnerName != "" {
		if newAuthorizedPartner, err = internal.GetPartner(db, addAuthorizedPartnerName); err != nil {
			return fmt.Errorf("failed to get partner: %w", err)
		}
	}

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	err = internal.AddRuleAccess(db, rule, newAuthorizedPartner)
	if err != nil {
		return fmt.Errorf("failed to authorize partner: %w", err)
	}

	return nil
}

func editAuthorizedPartners(ruleID int, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	oldName := r.FormValue("editAuthorizedPartnerOldName")
	editAuthorizedPartnerName := r.FormValue("editAuthorizedPartnerName")

	partners, err := internal.ListAuthorizedPartners(db, rule)
	if err != nil {
		return nil
	}

	for _, p := range partners {
		if p.Name == editAuthorizedPartnerName {
			return ErrPartnerAlreadyAuthorized
		}
	}

	oldPartner, err := internal.GetPartner(db, oldName)
	if err != nil {
		return fmt.Errorf("failed to get old partner: %w", err)
	}

	newPartner, err := internal.GetPartner(db, editAuthorizedPartnerName)
	if err != nil {
		return fmt.Errorf("failed to get new partner: %w", err)
	}

	if dlErr := internal.DeleteRuleAccess(db, rule, oldPartner); dlErr != nil {
		return fmt.Errorf("failed to remove old access: %w", dlErr)
	}

	if addErr := internal.AddRuleAccess(db, rule, newPartner); addErr != nil {
		return fmt.Errorf("failed to add new access: %w", addErr)
	}

	return nil
}

//nolint:dupl // method for authorized partners
func deleteAuthorizedPartners(ruleID int, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	errorTaskName := r.FormValue("deleteAuthorizedPartner")

	partner, err := internal.GetPartner(db, errorTaskName)
	if err != nil {
		return fmt.Errorf("failed to get new partner: %w", err)
	}

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	if err = internal.DeleteRuleAccess(db, rule, partner); err != nil {
		return fmt.Errorf("failed to delete access: %w", err)
	}

	return nil
}

//nolint:dupl // method for authorized partners
func callMethodsAuthorizedPartnersRules(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
	ruleID int,
) (value bool, errMsg, modalOpen string) {
	if r.Method == http.MethodPost && r.FormValue("addAuthorizedPartnerName") != "" {
		addAuthorizedPartnerErr := addAuthorizedPartners(ruleID, db, r)
		if addAuthorizedPartnerErr != nil {
			logger.Errorf("failed to add authorized partner: %v", addAuthorizedPartnerErr)

			return false, addAuthorizedPartnerErr.Error(), "addAuthorizedPartnerModal"
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("editAuthorizedPartnerOldName") != "" {
		if editAuthorizedPartnerErr := editAuthorizedPartners(ruleID, db, r); editAuthorizedPartnerErr != nil {
			logger.Errorf("failed to edit authorized partner: %v", editAuthorizedPartnerErr)

			return false, editAuthorizedPartnerErr.Error(),
				"editAuthorizedPartnerModal_" + r.FormValue("editAuthorizedPartnerOldName")
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("deleteAuthorizedPartner") != "" {
		if deleteAuthorizedPartnerErr := deleteAuthorizedPartners(ruleID, db, r); deleteAuthorizedPartnerErr != nil {
			logger.Errorf("failed to delete authorized partner: %v", deleteAuthorizedPartnerErr)

			return false, deleteAuthorizedPartnerErr.Error(), ""
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	return false, "", ""
}

func addAuthorizedLocalAccounts(ruleID int, db *database.DB, r *http.Request) error {
	var newAuthorizedLocalAccounts *model.LocalAccount
	var err error

	if err = r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	addAuthorizedLocalAccountsServerName := r.FormValue("addAuthorizedLocalAccountsServerName")

	if addAuthLocalAccName := r.FormValue("addAuthorizedLocalAccountsName"); addAuthLocalAccName != "" {
		if newAuthorizedLocalAccounts, err = internal.GetServerAccount(db,
			addAuthorizedLocalAccountsServerName, addAuthLocalAccName); err != nil {
			return fmt.Errorf("failed to get localAccount: %w", err)
		}
	}

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	err = internal.AddRuleAccess(db, rule, newAuthorizedLocalAccounts)
	if err != nil {
		return fmt.Errorf("failed to authorize localAccount: %w", err)
	}

	return nil
}

func editAuthorizedLocalAccounts(ruleID int, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	oldName := r.FormValue("editAuthorizedLocalAccountOldName")
	editAuthorizedLocalAccountName := r.FormValue("editAuthorizedLocalAccountsName")
	editAuthorizeServerName := r.FormValue("editAuthorizedLocalAccountsServerName")

	localAccounts, err := internal.ListAuthorizedLocalAccounts(db, rule)
	if err != nil {
		return nil
	}

	for _, la := range localAccounts {
		if la.Login == editAuthorizedLocalAccountName {
			return ErrLocalAccountAlreadyAuthorized
		}
	}

	oldLocalAccount, err := internal.GetServerAccount(db, editAuthorizeServerName, oldName)
	if err != nil {
		return fmt.Errorf("failed to get old localAccount: %w", err)
	}

	newLocalAccount, err := internal.GetServerAccount(db, editAuthorizeServerName, editAuthorizedLocalAccountName)
	if err != nil {
		return fmt.Errorf("failed to get new localAccount: %w", err)
	}

	if dlErr := internal.DeleteRuleAccess(db, rule, oldLocalAccount); dlErr != nil {
		return fmt.Errorf("failed to remove old access: %w", dlErr)
	}

	if addErr := internal.AddRuleAccess(db, rule, newLocalAccount); addErr != nil {
		return fmt.Errorf("failed to add new access: %w", addErr)
	}

	return nil
}

//nolint:dupl // method for local accounts
func deleteAuthorizedLocalAccounts(ruleID int, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	errorTaskID := r.FormValue("deleteAuthorizedLocalAccount")

	id, err := strconv.ParseUint(errorTaskID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to get id: %w", err)
	}

	localAccount, err := internal.GetServerAccountByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get new localAccount: %w", err)
	}

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	if err = internal.DeleteRuleAccess(db, rule, localAccount); err != nil {
		return fmt.Errorf("failed to delete access: %w", err)
	}

	return nil
}

//nolint:dupl // method for local accounts
func callMethodsAuthorizedLocalAccountsRules(logger *log.Logger, db *database.DB, w http.ResponseWriter,
	r *http.Request, ruleID int,
) (value bool, errMsg, modalOpen string) {
	if r.Method == http.MethodPost && r.FormValue("addAuthorizedLocalAccountsName") != "" {
		addAuthorizedLocalAccountsErr := addAuthorizedLocalAccounts(ruleID, db, r)
		if addAuthorizedLocalAccountsErr != nil {
			logger.Errorf("failed to add authorized localAccount: %v", addAuthorizedLocalAccountsErr)

			return false, addAuthorizedLocalAccountsErr.Error(), "addAuthorizedLocalAccountModal"
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("editAuthorizedLocalAccountOldName") != "" {
		if editAuthLocalAccErr := editAuthorizedLocalAccounts(ruleID, db, r); editAuthLocalAccErr != nil {
			logger.Errorf("failed to edit authorized localAccount: %v", editAuthLocalAccErr)

			return false, editAuthLocalAccErr.Error(),
				"editAuthorizedLocalAccountModal_" + r.FormValue("editAuthorizedLocalAccountOldName")
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("deleteAuthorizedLocalAccount") != "" {
		if deleteAuthLocalAccErr := deleteAuthorizedLocalAccounts(ruleID, db, r); deleteAuthLocalAccErr != nil {
			logger.Errorf("failed to delete authorized localAccount: %v", deleteAuthLocalAccErr)

			return false, deleteAuthLocalAccErr.Error(), ""
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	return false, "", ""
}

func addAuthorizedRemoteAccounts(ruleID int, db *database.DB, r *http.Request) error {
	var newAuthorizedRemoteAccounts *model.RemoteAccount
	var err error

	if err = r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	addAuthorizedRemoteAccountsPartnerName := r.FormValue("addAuthorizedRemoteAccountsPartnerName")

	if addAuthRemoteAccName := r.FormValue("addAuthorizedRemoteAccountsName"); addAuthRemoteAccName != "" {
		if newAuthorizedRemoteAccounts, err = internal.GetPartnerAccount(db,
			addAuthorizedRemoteAccountsPartnerName, addAuthRemoteAccName); err != nil {
			return fmt.Errorf("failed to get remoteAccount: %w", err)
		}
	}

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	err = internal.AddRuleAccess(db, rule, newAuthorizedRemoteAccounts)
	if err != nil {
		return fmt.Errorf("failed to authorize remoteAccount: %w", err)
	}

	return nil
}

func editAuthorizedRemoteAccounts(ruleID int, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	oldName := r.FormValue("editAuthorizedRemoteAccountOldName")
	editAuthorizedRemoteAccountName := r.FormValue("editAuthorizedRemoteAccountsName")
	editAuthorizePartnerName := r.FormValue("editAuthorizedRemoteAccountsPartnerName")

	remoteAccounts, err := internal.ListAuthorizedRemoteAccounts(db, rule)
	if err != nil {
		return nil
	}

	for _, ra := range remoteAccounts {
		if ra.Login == editAuthorizedRemoteAccountName {
			return ErrRemoteAccountAlreadyAuthorized
		}
	}

	oldRemoteAccount, err := internal.GetPartnerAccount(db, editAuthorizePartnerName, oldName)
	if err != nil {
		return fmt.Errorf("failed to get old remoteAccount: %w", err)
	}

	newRemoteAccount, err := internal.GetPartnerAccount(db, editAuthorizePartnerName, editAuthorizedRemoteAccountName)
	if err != nil {
		return fmt.Errorf("failed to get new remoteAccount: %w", err)
	}

	if dlErr := internal.DeleteRuleAccess(db, rule, oldRemoteAccount); dlErr != nil {
		return fmt.Errorf("failed to remove old access: %w", dlErr)
	}

	if addErr := internal.AddRuleAccess(db, rule, newRemoteAccount); addErr != nil {
		return fmt.Errorf("failed to add new access: %w", addErr)
	}

	return nil
}

//nolint:dupl // method for authorized remote accounts
func deleteAuthorizedRemoteAccounts(ruleID int, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	errorTaskID := r.FormValue("deleteAuthorizedRemoteAccount")

	id, err := strconv.ParseUint(errorTaskID, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to get id: %w", err)
	}

	remoteAccount, err := internal.GetPartnerAccountByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get new remoteAccount: %w", err)
	}

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	if err = internal.DeleteRuleAccess(db, rule, remoteAccount); err != nil {
		return fmt.Errorf("failed to delete access: %w", err)
	}

	return nil
}

//nolint:dupl // method for authorized remote accounts
func callMethodsAuthorizedRemoteAccountsRules(logger *log.Logger, db *database.DB, w http.ResponseWriter,
	r *http.Request, ruleID int,
) (value bool, errMsg, modalOpen string) {
	if r.Method == http.MethodPost && r.FormValue("addAuthorizedRemoteAccountsName") != "" {
		addAuthorizedRemoteAccountsErr := addAuthorizedRemoteAccounts(ruleID, db, r)
		if addAuthorizedRemoteAccountsErr != nil {
			logger.Errorf("failed to add authorized remoteAccount: %v", addAuthorizedRemoteAccountsErr)

			return false, addAuthorizedRemoteAccountsErr.Error(), "addAuthorizedRemoteAccountModal"
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("editAuthorizedRemoteAccountOldName") != "" {
		if editAuthRemoteAccErr := editAuthorizedRemoteAccounts(ruleID, db, r); editAuthRemoteAccErr != nil {
			logger.Errorf("failed to edit authorized remoteAccount: %v", editAuthRemoteAccErr)

			return false, editAuthRemoteAccErr.Error(),
				"editAuthorizedRemoteAccountModal_" + r.FormValue("editAuthorizedRemoteAccountOldName")
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("deleteAuthorizedRemoteAccount") != "" {
		if dlAuthRemoteAccErr := deleteAuthorizedRemoteAccounts(ruleID, db, r); dlAuthRemoteAccErr != nil {
			logger.Errorf("failed to delete authorized remoteAccount: %v", dlAuthRemoteAccErr)

			return false, dlAuthRemoteAccErr.Error(), ""
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	return false, "", ""
}

func listAuthorizedServers(db *database.DB, rule *model.Rule, listServers []*model.LocalAgent,
) ([]string, []*model.LocalAgent) {
	serverNames := make([]string, len(listServers))
	for i, s := range listServers {
		serverNames[i] = s.Name
	}

	authorizedServers, err := internal.ListAuthorizedServers(db, rule)
	if err != nil {
		return nil, nil
	}

	return serverNames, authorizedServers
}

func listAuthorizedPartners(db *database.DB, rule *model.Rule, listPartners []*model.RemoteAgent,
) ([]string, []*model.RemoteAgent) {
	partnerNames := make([]string, len(listPartners))
	for i, s := range listPartners {
		partnerNames[i] = s.Name
	}

	authorizedPartners, err := internal.ListAuthorizedPartners(db, rule)
	if err != nil {
		return nil, nil
	}

	return partnerNames, authorizedPartners
}

//nolint:dupl // method for authorized local accounts
func listAuthorizedLocalAccounts(db *database.DB, rule *model.Rule, listServers []*model.LocalAgent,
) (map[string][]string, []*model.LocalAccount) {
	listLocalAccounts := make(map[string][]string, len(listServers))

	for _, servers := range listServers {
		localAccounts, err := internal.ListServerAccounts(db, servers.Name, "login", true, 0, 0)
		if err != nil {
			return nil, nil
		}

		names := make([]string, len(localAccounts))
		for i, la := range localAccounts {
			names[i] = la.Login
		}
		listLocalAccounts[servers.Name] = names
	}

	authorizedLocalAccounts, err := internal.ListAuthorizedLocalAccounts(db, rule)
	if err != nil {
		return nil, nil
	}

	return listLocalAccounts, authorizedLocalAccounts
}

//nolint:dupl // method for remote accounts
func listAuthorizedRemoteAccounts(db *database.DB, rule *model.Rule, listPartners []*model.RemoteAgent,
) (map[string][]string, []*model.RemoteAccount) {
	listRemoteAccounts := make(map[string][]string, len(listPartners))

	for _, partners := range listPartners {
		remoteAccounts, err := internal.ListPartnerAccounts(db, partners.Name, "login", true, 0, 0)
		if err != nil {
			return nil, nil
		}

		names := make([]string, len(remoteAccounts))
		for i, ra := range remoteAccounts {
			names[i] = ra.Login
		}
		listRemoteAccounts[partners.Name] = names
	}

	authorizedRemoteAccounts, err := internal.ListAuthorizedRemoteAccounts(db, rule)
	if err != nil {
		return nil, nil
	}

	return listRemoteAccounts, authorizedRemoteAccounts
}

func callMethodsAllAuthorizedRules(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
	ruleID int,
) (handled bool, errMsg, modalOpen string) {
	if h, em, mo := callMethodsAuthorizedServersRules(logger, db, w, r, ruleID); h {
		return true, "", ""
	} else if em != "" {
		return false, em, mo
	}

	if h, em, mo := callMethodsAuthorizedPartnersRules(logger, db, w, r, ruleID); h {
		return true, "", ""
	} else if em != "" {
		return false, em, mo
	}

	if h, em, mo := callMethodsAuthorizedLocalAccountsRules(logger, db, w, r, ruleID); h {
		return true, "", ""
	} else if em != "" {
		return false, em, mo
	}

	if h, em, mo := callMethodsAuthorizedRemoteAccountsRules(logger, db, w, r, ruleID); h {
		return true, "", ""
	} else if em != "" {
		return false, em, mo
	}

	return false, "", ""
}

//nolint:funlen // is for one page
func managementUsageRightsRulesPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tTranslated := //nolint:forcetypeassert //u
			pageTranslated("management_usage_rights_rules_page", userLanguage.(string)) //nolint:errcheck //u

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Errorf("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		var rule *model.Rule
		var id uint64

		ruleID := r.URL.Query().Get("ruleID")
		if ruleID != "" {
			id, err = strconv.ParseUint(ruleID, 10, 64)
			if err != nil {
				logger.Errorf("failed to convert id to int: %v", err)
			}

			rule, err = internal.GetRuleByID(db, int64(id))
			if err != nil {
				logger.Errorf("failed to get id: %v", err)
			}
		}

		listServers, err := internal.ListServers(db, "name", true, 0, 0)
		if err != nil {
			return
		}
		serverNames, authorizedServers := listAuthorizedServers(db, rule, listServers)

		listPartners, err := internal.ListPartners(db, "name", true, 0, 0)
		if err != nil {
			return
		}
		partnerNames, authorizedPartners := listAuthorizedPartners(db, rule, listPartners)

		listLocalAccounts, authorizedLocalAccounts := listAuthorizedLocalAccounts(db, rule, listServers)

		listRemoteAccounts, authorizedRemoteAccounts := listAuthorizedRemoteAccounts(db, rule, listPartners)

		handled, errMsg, modalOpen := callMethodsAllAuthorizedRules(logger, db, w, r, int(rule.ID))
		if handled {
			return
		}

		managementUsageRightsRulesTemplate := template.Must(
			template.New("management_usage_rights_rules_page.html").
				Funcs(CombinedFuncMap(db)).
				ParseFS(webFS, index, header, sidebar, "front-end/html/management_usage_rights_rules_page.html"),
		)

		if tmplErr := managementUsageRightsRulesTemplate.ExecuteTemplate(w, "management_usage_rights_rules_page",
			map[string]any{
				"myPermission":             myPermission,
				"tab":                      tTranslated,
				"username":                 user.Username,
				"language":                 userLanguage,
				"rule":                     rule,
				"taskTypes":                TaskTypes,
				"authorizedServers":        authorizedServers,
				"listServers":              serverNames,
				"authorizedPartners":       authorizedPartners,
				"listPartners":             partnerNames,
				"authorizedLocalAccounts":  authorizedLocalAccounts,
				"listLocalAccounts":        listLocalAccounts,
				"authorizedRemoteAccounts": authorizedRemoteAccounts,
				"listRemoteAccounts":       listRemoteAccounts,
				"errMsg":                   errMsg,
				"modalOpen":                modalOpen,
				"hasRuleID":                true,
			}); tmplErr != nil {
			logger.Errorf("render management_usage_rights_rules_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}

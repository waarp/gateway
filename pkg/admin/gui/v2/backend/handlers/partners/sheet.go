package partners

// The sheet of a remote partner: next to the "Details" tab, the
// "Authentication" tab (what the gateway checks of the partner) and the
// "Remote accounts" tab (the identities the gateway presents to it); the
// sheet of a remote account (details, authentication). This file holds what
// these pages share: the owner of a means (the partner or one of its remote
// accounts) as the shared pages of the means see it, the tab strip, the
// breadcrumb, the localised "not found" page and the redirections.

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"code.waarp.fr/lib/gui"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/means"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	// Suffixes of the routes of the sheet, under /partners/{id} and under
	// /partners/{id}/accounts/{aid}.
	authSuffix     = "/auth"
	accountsSuffix = "/accounts"
	newSuffix      = means.NewSuffix
	editSuffix     = means.EditSuffix
	deleteSuffix   = means.DeleteSuffix

	// The tabs of the sheets.
	tabDetails  = "details"
	tabAuth     = "auth"
	tabAccounts = "accounts"

	// The kind of object the previous action concerned, carried by the
	// redirection next to its result: a means or an account; an action on
	// the partner itself carries none.
	kindParam   = liststate.KindParam
	kindCred    = means.KindCred
	kindAccount = "account"

	sortAuth = "auth"
)

//nolint:gochecknoglobals // reference data of the screen
var accountSortKeys = []string{sortName, sortAuth}

// The tab strip, the state of a tab list and the "not found" page come from
// the shared pages of the means.
type (
	tab          = means.Tab
	tabList      = liststate.TabList
	notFoundData = means.NotFoundData
)

// parseQuery reads the state of a tab list: its search, its sort, its page.
func parseQuery(v url.Values, sorts []string) *listQuery { return liststate.Parse(v, sorts, pageSizes) }

// redirectSheet sends to a page of the sheet (a tab, already carrying the
// list state) with the result of the action: on a means or an account, the
// kind says which; on the partner itself, the kind is empty.
func redirectSheet(w http.ResponseWriter, r *http.Request, target, kind, done, name string) {
	result := url.Values{doneParam: {done}, nameParam: {name}}
	if kind != "" {
		result.Set(kindParam, kind)
	}

	liststate.RedirectTo(w, r, target, result)
}

// owner is the holder of the authentication means a page concerns: the
// partner itself (what the gateway checks of it) or one of its remote
// accounts (what the gateway presents to it).
type owner struct {
	ptn entry
	acc *model.RemoteAccount
}

func (own owner) isAccount() bool { return own.acc != nil }

// table is the owner as the model sees it.
func (own owner) table() model.CredOwnerTable {
	if own.acc != nil {
		return own.acc
	}

	return own.ptn.agent
}

// authURL is the "Authentication" tab of the owner.
func (own owner) authURL() string {
	if own.acc != nil {
		return accountURL(own.ptn.agent.ID, own.acc.ID, authSuffix)
	}

	return partnerURL(own.ptn.agent.ID, authSuffix)
}

// allowed are the kinds of means the protocol of the partner accepts for the owner.
func (own owner) allowed() []string {
	if own.acc != nil {
		return means.RemoteAccountTypes(own.ptn.agent.Protocol)
	}

	return means.PartnerTypes(own.ptn.agent.Protocol)
}

// label names the owner in a sentence.
func (own owner) label(translate gui.Translator) string {
	if own.acc != nil {
		return translate("auth.owner.remote_account", "login", own.acc.Login, "name", own.ptn.agent.Name)
	}

	return translate("auth.owner.partner", "name", own.ptn.agent.Name)
}

// title is the title of the sheet of the owner.
func (own owner) title(translate gui.Translator) string {
	if own.acc != nil {
		return translate("remote_account.title", "login", own.acc.Login)
	}

	return translate("partners.detail.title", "name", own.ptn.agent.Name)
}

// texts are the sentences of the pages of the means that name the owner.
func (own owner) texts() means.Texts {
	t := means.Texts{
		Intro: "auth.title.partner", Empty: "auth.empty.partner", Delete: "auth.delete.partner",
		NoneTitle: "auth.none_for_protocol.partner.title", NoneText: "auth.none_for_protocol.partner.text",
		Back: "partners.back", Tabs: "partners.tabs",
	}

	if own.acc != nil {
		t.Intro, t.Empty, t.Delete = "auth.title.remote_account", "auth.empty.remote_account", "auth.delete.remote_account"
		t.Back = "partners.back_partner"
	}

	return t
}

// meansOwner describes the owner to the shared pages of the means: the
// partner holds what the gateway checks of it (the internal registry of the
// model), a remote account what the gateway presents (the external one).
func (h *Handlers) meansOwner(translate gui.Translator, own owner, carried string, user rights) *means.Owner {
	back := listURL(carried)
	if own.acc != nil {
		back = withRet(partnerURL(own.ptn.agent.ID, accountsSuffix), carried)
	}

	return &means.Owner{
		Table:    own.table(),
		Protocol: own.ptn.agent.Protocol,
		Verified: !own.isAccount(),
		Allowed:  own.allowed(),
		AuthURL:  own.authURL(),
		BackURL:  back,
		Carried:  carried,
		Title:    own.title(translate),
		Label:    own.label(translate),
		Texts:    own.texts(),
		Rights:   means.Rights{Write: user.Write, Delete: user.Delete},
		Tabs:     func(current string) ([]tab, error) { return h.tabsOf(translate, own, current, carried) },
		Crumbs:   func(last string) []gui.Crumb { return sheetCrumbs(translate, own, last, carried) },
		Chrome:   chrome(),
	}
}

// chrome is the screen of the partners around the pages of a sheet.
func chrome() means.Chrome {
	return means.Chrome{
		Section: constants.SidebarSectionConnection, Menu: constants.SidebarMenuPartners, Doc: docPage,
		Stylesheets: sheetStylesheets(),
	}
}

// sheetStylesheets are the stylesheets of the pages of the screen: the
// shared rules of a sheet, then the screen.
func sheetStylesheets() []string { return []string{protoStylesheet, means.Stylesheet, stylesheet} }

func accountURL(partnerID, accountID int64, suffix string) string {
	return partnerURL(partnerID, accountsSuffix+"/"+strconv.FormatInt(accountID, 10)+suffix)
}

// owner reads the {id} of the partner and, on the routes of an account, its
// {aid}; a missing one renders the localised "not found" page and returns
// false.
func (h *Handlers) owner(w http.ResponseWriter, r *http.Request) (owner, bool) {
	ptn, ok := h.partner(w, r)
	if !ok {
		return owner{}, false
	}

	raw, has := mux.Vars(r)["aid"]
	if !has {
		return owner{ptn: ptn}, true
	}

	id, parseErr := strconv.ParseInt(raw, 10, 64)

	var acc model.RemoteAccount

	err := h.db.Get(&acc, "id=? AND remote_agent_id=?", id, ptn.agent.ID).Run()
	if parseErr == nil && err == nil {
		return owner{ptn: ptn, acc: &acc}, true
	}

	if err != nil && !database.IsNotFound(err) {
		h.logger.Errorf("failed to retrieve the account %d of the partner %d: %v", id, ptn.agent.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return owner{}, false
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("remote_accounts.not_found.title")

	h.notFoundPage(w, r, sheetCrumbs(translate, owner{ptn: ptn}, title, carried), notFoundData{
		Title:     title,
		Text:      translate("remote_accounts.not_found.text"),
		Icon:      "users",
		BackLabel: translate("partners.back_partner"),
		BackURL:   withRet(partnerURL(ptn.agent.ID, accountsSuffix), carried),
	})

	return owner{}, false
}

// notFoundPage renders the localised "not found" page of the screen.
func (h *Handlers) notFoundPage(w http.ResponseWriter, r *http.Request, crumbs []gui.Crumb, data notFoundData) {
	h.means.NotFound(w, r, chrome(), crumbs, data)
}

// ---------------------------------------------------------------- tabs ----

// partnerTabs is the tab strip of a partner: its details, its authentication
// means and its remote accounts, counted.
func (h *Handlers) partnerTabs(translate gui.Translator, e entry, current, carried string) ([]tab, error) {
	creds, err := h.db.Count(&model.Credential{}).Where(e.agent.GetCredCond()).Run()
	if err != nil {
		return nil, fmt.Errorf("failed to count the authentication means of the partner: %w", err)
	}

	id := e.agent.ID

	return []tab{
		{Label: translate("partners.tab.details"), URL: withRet(partnerURL(id, ""), carried), Current: current == tabDetails},
		{
			Label: translate("partners.tab.auth"), URL: withRet(partnerURL(id, authSuffix), carried),
			Count: creds, HasCount: true, Current: current == tabAuth,
		},
		{
			Label: translate("partners.tab.accounts"), URL: withRet(partnerURL(id, accountsSuffix), carried),
			Count: uint64(e.accounts), HasCount: true, Current: current == tabAccounts, //nolint:gosec // a count of rows
		},
	}, nil
}

// newPartnerTabs is the tab strip of the form of a new partner: its details,
// then the means and the accounts it will have once created, greyed.
func newPartnerTabs(translate gui.Translator) []tab {
	return means.PendingTabs(translate, translate("partners.tab.details"),
		translate("partners.tab.auth"), translate("partners.tab.accounts"))
}

// accountTabs is the tab strip of a remote account: its details and its means.
func (h *Handlers) accountTabs(translate gui.Translator, own owner, current, carried string) ([]tab, error) {
	creds, err := h.db.Count(&model.Credential{}).Where(own.acc.GetCredCond()).Run()
	if err != nil {
		return nil, fmt.Errorf("failed to count the authentication means of the account: %w", err)
	}

	pid, aid := own.ptn.agent.ID, own.acc.ID

	return []tab{
		{
			Label: translate("remote_account.tab.details"), URL: withRet(accountURL(pid, aid, ""), carried),
			Current: current == tabDetails,
		},
		{
			Label: translate("remote_account.tab.auth"), URL: withRet(accountURL(pid, aid, authSuffix), carried),
			Count: creds, HasCount: true, Current: current == tabAuth,
		},
	}, nil
}

// tabsOf is the tab strip of a page of an owner: the account's on the sheet
// of an account, the partner's otherwise.
func (h *Handlers) tabsOf(translate gui.Translator, own owner, current, carried string) ([]tab, error) {
	if own.acc != nil {
		return h.accountTabs(translate, own, current, carried)
	}

	return h.partnerTabs(translate, own.ptn, current, carried)
}

// sheetCrumbs is the breadcrumb of a page of the sheet: the list (as the
// user left it), the partner, then, on the sheet of an account, the accounts
// tab and the account, then the action page when there is one. An item is a
// link when the page is deeper than it.
func sheetCrumbs(translate gui.Translator, own owner, last, carried string) []gui.Crumb {
	c := []gui.Crumb{
		{Label: translate("nav.home"), URL: common.WebuiURL("/home")},
		{Label: translate("nav.connections")},
		{Label: translate("partners.title"), URL: listURL(carried)},
	}

	partner := gui.Crumb{Label: own.ptn.agent.Name}
	if own.acc != nil || last != "" {
		partner.URL = withRet(partnerURL(own.ptn.agent.ID, ""), carried)
	}

	c = append(c, partner)

	if own.acc != nil {
		c = append(c, gui.Crumb{
			Label: translate("partners.tab.accounts"), URL: withRet(partnerURL(own.ptn.agent.ID, accountsSuffix), carried),
		})

		account := gui.Crumb{Label: own.acc.Login}
		if last != "" {
			account.URL = withRet(accountURL(own.ptn.agent.ID, own.acc.ID, ""), carried)
		}

		c = append(c, account)
	}

	if last != "" {
		c = append(c, gui.Crumb{Label: last})
	}

	return c
}

// sheetScreen is a page of the sheet: the chrome of the screen around the
// data, with the result of the previous action when the page is a tab.
func (h *Handlers) sheetScreen(translate gui.Translator, title string, own owner, last, carried string,
	status int, notices []gui.Notice,
) common.Screen {
	return common.Screen{
		Title:   title,
		Section: constants.SidebarSectionConnection,
		Menu:    constants.SidebarMenuPartners,
		Doc:     docPage,
		Crumbs:  sheetCrumbs(translate, own, last, carried),
		Status:  status,
		Notices: notices,

		Stylesheets: sheetStylesheets(),
	}
}

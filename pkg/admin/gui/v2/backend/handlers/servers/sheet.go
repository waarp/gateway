package servers

// The sheet of a local server (#709): next to the "Details" tab, the
// "Authentication" tab (the identity the server presents to the partners) and
// the "Local accounts" tab; the sheet of an account (details, authentication);
// the pages that add, show, edit and delete an authentication means or an
// account. This file holds what these pages share: the owner of a means (the
// server or one of its accounts) as the shared pages of the means see it,
// the tab strip, the breadcrumb, the localised "not found" page and the
// redirections.

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

	// Suffixes of the routes of the sheet, under /servers/{id} and under
	// /servers/{id}/accounts/{aid}.
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
	// the server itself carries none.
	kindParam   = liststate.KindParam
	kindCred    = means.KindCred
	kindAccount = "account"

	sortAuth = "auth"
)

//nolint:gochecknoglobals // reference data of the screen
var (
	accountSortKeys = []string{sortName, sortAuth}
	sheetDone       = []string{doneCreated, doneUpdated, doneDeleted}
)

// The tab strip, the state of a tab list and the "not found" page come from
// the shared pages of the means.
type (
	tab          = means.Tab
	tabList      = liststate.TabList
	notFoundData = means.NotFoundData
)

// owner is the holder of the authentication means a page concerns: the
// server itself (its identity) or one of its local accounts (what the
// partner presents).
type owner struct {
	srv entry
	acc *model.LocalAccount
}

func (own owner) isAccount() bool { return own.acc != nil }

// table is the owner as the model sees it.
func (own owner) table() model.CredOwnerTable {
	if own.acc != nil {
		return own.acc
	}

	return own.srv.agent
}

// authURL is the "Authentication" tab of the owner.
func (own owner) authURL() string {
	if own.acc != nil {
		return accountURL(own.srv.agent.ID, own.acc.ID, authSuffix)
	}

	return serverURL(own.srv.agent.ID, authSuffix)
}

// allowed are the kinds of means the protocol of the server accepts for the owner.
func (own owner) allowed() []string {
	if own.acc != nil {
		return means.LocalAccountTypes(own.srv.agent.Protocol)
	}

	return means.ServerTypes(own.srv.agent.Protocol)
}

// label names the owner in a sentence.
func (own owner) label(translate gui.Translator) string {
	if own.acc != nil {
		return translate("auth.owner.account", "login", own.acc.Login, "name", own.srv.agent.Name)
	}

	return translate("auth.owner.server", "name", own.srv.agent.Name)
}

// title is the title of the sheet of the owner.
func (own owner) title(translate gui.Translator) string {
	if own.acc != nil {
		return translate("account.title", "login", own.acc.Login)
	}

	return translate("servers.detail.title", "name", own.srv.agent.Name)
}

// texts are the sentences of the pages of the means that name the owner.
func (own owner) texts() means.Texts {
	t := means.Texts{
		Intro: "auth.title.server", Empty: "auth.empty.server", Delete: "auth.delete.server",
		NoneTitle: "auth.none_for_protocol.title", NoneText: "auth.none_for_protocol.text",
		Back: "servers.back", Tabs: "servers.tabs",
	}

	if own.acc != nil {
		t.Intro, t.Empty, t.Delete = "auth.title.account", "auth.empty.account", "auth.delete.account"
		t.Back = "servers.back_server"
	}

	return t
}

// meansOwner describes the owner to the shared pages of the means: a local
// account holds what the partner presents (the internal registry of the
// model), the server what it presents itself (the external one).
func (h *Handlers) meansOwner(translate gui.Translator, own owner, carried string, user rights) *means.Owner {
	back := listURL(carried)
	if own.acc != nil {
		back = withRet(serverURL(own.srv.agent.ID, accountsSuffix), carried)
	}

	return &means.Owner{
		Table:    own.table(),
		Protocol: own.srv.agent.Protocol,
		Verified: own.isAccount(),
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

// chrome is the screen of the servers around the pages of a sheet.
func chrome() means.Chrome {
	return means.Chrome{
		Section: constants.SidebarSectionConnection, Menu: constants.SidebarMenuServers, Doc: docPage,
		Stylesheets: sheetStylesheets(),
	}
}

// sheetStylesheets are the stylesheets of the pages of the screen: the
// shared rules of a sheet, then the screen.
func sheetStylesheets() []string { return []string{means.Stylesheet, stylesheet} }

func accountURL(serverID, accountID int64, suffix string) string {
	return serverURL(serverID, accountsSuffix+"/"+strconv.FormatInt(accountID, 10)+suffix)
}

// owner reads the {id} of the server and, on the routes of an account, its
// {aid}; a missing one renders the localised "not found" page and returns
// false.
func (h *Handlers) owner(w http.ResponseWriter, r *http.Request) (owner, bool) {
	srv, ok := h.server(w, r)
	if !ok {
		return owner{}, false
	}

	raw, has := mux.Vars(r)["aid"]
	if !has {
		return owner{srv: srv}, true
	}

	id, parseErr := strconv.ParseInt(raw, 10, 64)

	var acc model.LocalAccount

	err := h.db.Get(&acc, "id=? AND local_agent_id=?", id, srv.agent.ID).Run()
	if parseErr == nil && err == nil {
		return owner{srv: srv, acc: &acc}, true
	}

	if err != nil && !database.IsNotFound(err) {
		h.logger.Errorf("failed to retrieve the account %d of the server %d: %v", id, srv.agent.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return owner{}, false
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("accounts.not_found.title")

	h.notFoundPage(w, r, sheetCrumbs(translate, owner{srv: srv}, title, carried), notFoundData{
		Title:     title,
		Text:      translate("accounts.not_found.text"),
		Icon:      "users",
		BackLabel: translate("servers.back_server"),
		BackURL:   withRet(serverURL(srv.agent.ID, accountsSuffix), carried),
	})

	return owner{}, false
}

// notFoundPage renders the localised "not found" page of the screen.
func (h *Handlers) notFoundPage(w http.ResponseWriter, r *http.Request, crumbs []gui.Crumb, data notFoundData) {
	h.means.NotFound(w, r, chrome(), crumbs, data)
}

// ---------------------------------------------------------------- tabs ----

// serverTabs is the tab strip of a server: its details, its authentication
// means and its local accounts, counted.
func (h *Handlers) serverTabs(translate gui.Translator, e entry, current, carried string) ([]tab, error) {
	creds, err := h.db.Count(&model.Credential{}).Where(e.agent.GetCredCond()).Run()
	if err != nil {
		return nil, fmt.Errorf("failed to count the authentication means of the server: %w", err)
	}

	accounts, err := h.db.Count(&model.LocalAccount{}).Where("local_agent_id=?", e.agent.ID).Run()
	if err != nil {
		return nil, fmt.Errorf("failed to count the accounts of the server: %w", err)
	}

	id := e.agent.ID

	return []tab{
		{Label: translate("servers.tab.details"), URL: withRet(serverURL(id, ""), carried), Current: current == tabDetails},
		{
			Label: translate("servers.tab.auth"), URL: withRet(serverURL(id, authSuffix), carried),
			Count: creds, HasCount: true, Current: current == tabAuth,
		},
		{
			Label: translate("servers.tab.accounts"), URL: withRet(serverURL(id, accountsSuffix), carried),
			Count: accounts, HasCount: true, Current: current == tabAccounts,
		},
	}, nil
}

// accountTabs is the tab strip of an account: its details and its means.
func (h *Handlers) accountTabs(translate gui.Translator, own owner, current, carried string) ([]tab, error) {
	creds, err := h.db.Count(&model.Credential{}).Where(own.acc.GetCredCond()).Run()
	if err != nil {
		return nil, fmt.Errorf("failed to count the authentication means of the account: %w", err)
	}

	sid, aid := own.srv.agent.ID, own.acc.ID

	return []tab{
		{
			Label: translate("account.tab.details"), URL: withRet(accountURL(sid, aid, ""), carried),
			Current: current == tabDetails,
		},
		{
			Label: translate("account.tab.auth"), URL: withRet(accountURL(sid, aid, authSuffix), carried),
			Count: creds, HasCount: true, Current: current == tabAuth,
		},
	}, nil
}

// tabsOf is the tab strip of a page of an owner: the account's on the sheet
// of an account, the server's otherwise.
func (h *Handlers) tabsOf(translate gui.Translator, own owner, current, carried string) ([]tab, error) {
	if own.acc != nil {
		return h.accountTabs(translate, own, current, carried)
	}

	return h.serverTabs(translate, own.srv, current, carried)
}

// sheetCrumbs is the breadcrumb of a page of the sheet: the list (as the
// user left it), the server, then, on the sheet of an account, the accounts
// tab and the account, then the action page when there is one. An item is a
// link when the page is deeper than it.
func sheetCrumbs(translate gui.Translator, own owner, last, carried string) []gui.Crumb {
	c := []gui.Crumb{
		{Label: translate("nav.home"), URL: common.WebuiURL("/home")},
		{Label: translate("nav.connections")},
		{Label: translate("servers.title"), URL: listURL(carried)},
	}

	server := gui.Crumb{Label: own.srv.agent.Name}
	if own.acc != nil || last != "" {
		server.URL = withRet(serverURL(own.srv.agent.ID, ""), carried)
	}

	c = append(c, server)

	if own.acc != nil {
		c = append(c, gui.Crumb{
			Label: translate("servers.tab.accounts"), URL: withRet(serverURL(own.srv.agent.ID, accountsSuffix), carried),
		})

		account := gui.Crumb{Label: own.acc.Login}
		if last != "" {
			account.URL = withRet(accountURL(own.srv.agent.ID, own.acc.ID, ""), carried)
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
		Menu:    constants.SidebarMenuServers,
		Doc:     docPage,
		Crumbs:  sheetCrumbs(translate, own, last, carried),
		Status:  status,
		Notices: notices,

		Stylesheets: sheetStylesheets(),
	}
}

// redirectSheet sends to a page of the sheet (a tab, already carrying the
// list state) with the result of the action.
func redirectSheet(w http.ResponseWriter, r *http.Request, target, kind, done, name string) {
	liststate.RedirectTo(w, r, target, url.Values{kindParam: {kind}, doneParam: {done}, nameParam: {name}})
}

// counter is the set of keys of a count: "N objects", "N objects · n shown".
type counter = liststate.Counter

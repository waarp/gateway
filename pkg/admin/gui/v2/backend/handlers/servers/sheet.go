package servers

// The sheet of a local server (#709): next to the "Details" tab, the
// "Authentication" tab (the identity the server presents to the partners) and
// the "Local accounts" tab; the sheet of an account (details, authentication);
// the pages that add, show, edit and delete an authentication means or an
// account. This file holds what these pages share: the owner of a means (the
// server or one of its accounts), the tab strip, the breadcrumb, the state of
// a tab list, the localised "not found" page, the forms with files and the
// redirections.

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"code.waarp.fr/lib/gui"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (

	// Suffixes of the routes of the sheet, under /servers/{id} and under
	// /servers/{id}/accounts/{aid}.
	authSuffix     = "/auth"
	accountsSuffix = "/accounts"
	newSuffix      = "/new"
	editSuffix     = "/edit"
	deleteSuffix   = "/delete"

	// The tabs of the sheets.
	tabDetails  = "details"
	tabAuth     = "auth"
	tabAccounts = "accounts"

	// The kind of object the previous action concerned, carried by the
	// redirection next to its result: a means or an account; an action on
	// the server itself carries none.
	kindParam   = "kind"
	kindCred    = "cred"
	kindAccount = "account"

	sortType = "type"
	sortAuth = "auth"

	// A PEM file weighs a few kilobytes: the limit of an upload.
	maxUpload = 4 << 20
)

//nolint:gochecknoglobals // reference data of the screen
var (
	credSortKeys    = []string{sortName, sortType}
	accountSortKeys = []string{sortName, sortAuth}
	sheetDone       = []string{doneCreated, doneUpdated, doneDeleted}
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

// credURL is a page of one means of the owner.
func (own owner) credURL(id int64, suffix string) string {
	return own.authURL() + "/" + strconv.FormatInt(id, 10) + suffix
}

// allowed are the kinds of means the protocol of the server accepts for the owner.
func (own owner) allowed() []string {
	if own.acc != nil {
		return accountCredTypes(own.srv.agent.Protocol)
	}

	return serverCredTypes(own.srv.agent.Protocol)
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

// credential reads the {cid} of the route among the means of the owner; a
// missing one renders the localised "not found" page and returns false.
func (h *Handlers) credential(w http.ResponseWriter, r *http.Request, own owner) (*model.Credential, bool) {
	id, parseErr := strconv.ParseInt(mux.Vars(r)["cid"], 10, 64)

	var cred model.Credential

	err := h.db.Get(&cred, "id=?", id).And(own.table().GetCredCond()).Run()
	if parseErr == nil && err == nil {
		return &cred, true
	}

	if err != nil && !database.IsNotFound(err) {
		h.logger.Errorf("failed to retrieve the authentication means %d: %v", id, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return nil, false
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("auth.not_found.title")

	h.notFoundPage(w, r, sheetCrumbs(translate, own, title, carried), notFoundData{
		Title:     title,
		Text:      translate("auth.not_found.text"),
		Icon:      "key",
		BackLabel: translate("action.back"),
		BackURL:   withRet(own.authURL(), carried),
	})

	return nil, false
}

// notFoundData is a localised "not found" page: what is missing and the way
// back.
type notFoundData struct {
	Title     string
	Text      string
	Icon      string
	BackLabel string
	BackURL   string
}

func (h *Handlers) notFoundPage(w http.ResponseWriter, r *http.Request, crumbs []gui.Crumb, data notFoundData) {
	h.shell.Render(w, h.logger, r, common.Screen{
		Title:   data.Title,
		Section: constants.SidebarSectionConnection,
		Menu:    constants.SidebarMenuServers,
		Doc:     docPage,
		Crumbs:  crumbs,
		Status:  http.StatusNotFound,
	}, notFoundView(data))
}

// ---------------------------------------------------------------- tabs ----

// tab is an entry of the tab strip of a sheet: a link to a page, with the
// number of objects it lists when it is a list.
type tab struct {
	Label    string
	URL      string
	Count    uint64
	HasCount bool
	Current  bool
}

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

		Stylesheets: []string{stylesheet},
	}
}

// redirectSheet sends to a page of the sheet (a tab, already carrying the
// list state) with the result of the action.
func redirectSheet(w http.ResponseWriter, r *http.Request, target, kind, done, name string) {
	u, err := url.Parse(target)
	if err != nil {
		http.Redirect(w, r, target, http.StatusSeeOther)

		return
	}

	query := u.Query()
	query.Set(kindParam, kind)
	query.Set(doneParam, done)
	query.Set(nameParam, name)
	u.RawQuery = query.Encode()

	http.Redirect(w, r, u.String(), http.StatusSeeOther)
}

// ------------------------------------------------------------ tab list ----

// tabList is the state of the list of a tab: the toolbar form, the base of
// the sort links, the count, the rows per page and the list foot. The sheet
// carries the state of the servers list (?ret=) through every link and form.
type tabList struct {
	Action      string
	Base        string
	Q           string
	Names       []string
	Hidden      map[string]string
	Count       string
	Sort        string
	Dir         string
	Caption     string
	Pager       *pager
	PagerHidden map[string]string
	Sizes       []int
	Size        int
	Empty       bool
	NoMatch     bool
	NoMatchText string
	NoMatchURL  string
	NoMatchLink string
}

// newTabList builds the state of a tab list from its query.
func newTabList(base string, query *listQuery, carried string) tabList {
	hidden, pagerHidden := query.ToolbarHidden(), query.PagerHidden()
	if carried != "" {
		hidden[retParam], pagerHidden[retParam] = carried, carried
	}

	return tabList{
		Action:      base,
		Base:        withRet(liststate.WithQuery(base, query.FilterValues()), carried),
		Q:           query.Q,
		Hidden:      hidden,
		Sort:        query.Sort,
		Dir:         query.Dir,
		PagerHidden: pagerHidden,
		Size:        query.Size,
	}
}

// noMatch is the empty result of a search: the way out clears the search.
func (l *tabList) noMatch(translate gui.Translator, text, base, carried string) {
	l.NoMatch = true
	l.NoMatchText = text
	l.NoMatchURL, l.NoMatchLink = withRet(base, carried), translate("search.clear")
}

// foot fills the list foot: the rows per page (offered as soon as there are
// rows) and the pages.
func (l *tabList) foot(info string, b liststate.Bounds, query *listQuery, base, carried string) {
	l.Sizes = pageSizes
	l.Pager = &pager{Info: info, Page: b.Page, Pages: b.Pages}

	if b.Page > 1 {
		l.Pager.PrevURL = withRet(liststate.WithQuery(base, query.WithPage(b.Page-1)), carried)
	}

	if b.Page < b.Pages {
		l.Pager.NextURL = withRet(liststate.WithQuery(base, query.WithPage(b.Page+1)), carried)
	}
}

// counter is the set of keys of a count: "N objects", "N objects · n shown".
type counter = liststate.Counter

// ---------------------------------------------------------------- forms ----

// parseForm reads the posted form, multipart (with files) or not.
func parseForm(r *http.Request) error {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		if err := r.ParseMultipartForm(maxUpload); err != nil {
			return fmt.Errorf("failed to read the multipart form: %w", err)
		}

		return nil
	}

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to read the form: %w", err)
	}

	return nil
}

// fileOrText is the content of the file uploaded in a field or, without one,
// of the text field next to it where the same content can be pasted. Line
// ends are normalised and the content trimmed, so that a file written on
// any system is read the same way.
func fileOrText(r *http.Request, fileField, textField string) (string, error) {
	if r.MultipartForm != nil {
		file, _, err := r.FormFile(fileField)

		switch {
		case err == nil:
			defer file.Close() //nolint:errcheck // read only

			content, readErr := io.ReadAll(io.LimitReader(file, maxUpload))
			if readErr != nil {
				return "", fmt.Errorf("failed to read the uploaded file: %w", readErr)
			}

			if len(content) > 0 {
				return normalizeText(string(content)), nil
			}
		case !errors.Is(err, http.ErrMissingFile):
			return "", fmt.Errorf("failed to read the uploaded file: %w", err)
		}
	}

	return normalizeText(r.FormValue(textField)), nil
}

func normalizeText(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
}

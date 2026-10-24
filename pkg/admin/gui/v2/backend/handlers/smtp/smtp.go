// Package smtp serves the "SMTP credentials" screen of the web interface
// (Treatments): the list of the accounts the EMAIL task sends with (searched
// by sender, sorted and paged), with their server, their authentication and
// the rules that use them, the sheet of a credential in reading, and the
// pages that create, edit and delete one.
//
// A credential holds the sender address (the "From" of the e-mails), the
// address of the SMTP server, and an optional login and password; the
// password is a secret, never rendered back. An EMAIL task names its
// credential by the sender address, without a link in the database: the
// screen keeps the sender of a credential a rule uses, and replaces its
// deletion by the list of the rules to change first. Until this screen, the
// credentials were managed through the REST API and the command line only.
package smtp

import (
	"cmp"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"code.waarp.fr/lib/gui"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/ruleuse"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

const (
	// The stylesheet of the screen, under the web interface.
	stylesheet = "/static_v2/smtp/smtp.css"

	docPage  = "user/webui/smtp"
	pagePath = "/smtp"

	// The cookie that keeps the state of the list for the browser session.
	stateCookie = "webui_smtp_list"

	doneParam = liststate.DoneParam
	nameParam = liststate.NameParam
	dirDesc   = liststate.DirDesc

	sortSender = "sender"
	sortServer = "server"

	doneCreated = liststate.DoneCreated
	doneUpdated = liststate.DoneUpdated
	doneDeleted = liststate.DoneDeleted

	// The port of SMTP over TLS: the mail library opens TLS from the start
	// on it, and offers STARTTLS on any other.
	implicitTLSPort = 465

	// Prefixes of the catalogue keys chosen at run time.
	sortKeyPrefix  = "smtp.sort."
	toastKeyPrefix = "smtp.toast."

	// Kinds of a gui.Notice.
	stateOK = "ok"
)

//nolint:gochecknoglobals // reference data of the screen
var (
	sortKeys  = []string{sortSender, sortServer}
	doneKinds = []string{doneCreated, doneUpdated, doneDeleted}

	counter = liststate.Counter{
		One: "smtp.count_one", Many: "smtp.count", ShownOne: "smtp.count_shown_one", ShownMany: "smtp.count_shown",
	}

	listSpec = liststate.Spec{Path: pagePath, Cookie: stateCookie, Sorts: sortKeys, Sizes: liststate.DefaultSizes}

	// The EMAIL task names its credential by the sender address.
	emailParams = ruleuse.Params{tasks.Email: {"sender"}}
)

// Handlers serves the pages and the actions of the screen.
type Handlers struct {
	db     *database.DB
	logger *log.Logger
	shell  *common.Shell
}

func New(db *database.DB, logger *log.Logger) *Handlers {
	return &Handlers{db: db, logger: logger, shell: common.NewShell()}
}

// rights of the current user on the credentials: those of the rules, as the
// REST API of the credentials. Reading is implied by reaching the screen;
// writing and deleting are checked again by the router on each action.
type rights struct{ Write, Delete bool }

func rightsOf(r *http.Request) rights {
	perms := common.GetUser(r).Permissions

	return rights{
		Write:  perms.HasPermission(model.PermRulesWrite),
		Delete: perms.HasPermission(model.PermRulesDelete),
	}
}

// entry is a credential with the rules whose EMAIL tasks name it.
type entry struct {
	cred *model.SMTPCredential
	uses []ruleuse.Use
}

func entryOf(c *model.SMTPCredential, uses map[string][]ruleuse.Use) entry {
	return entry{cred: c, uses: uses[c.EmailAddress]}
}

// server is the address of the SMTP server as the list and the pages show it.
func (e *entry) server() string { return e.cred.ServerAddress.String() }

// incomplete tells whether the address of the server lacks its host or its
// port: the REST API accepts it, the task could not connect.
func (e *entry) incomplete() bool {
	return e.cred.ServerAddress.Host == "" || e.cred.ServerAddress.Port == 0
}

// auth is what the list says of the authentication: the login, or none.
func (e *entry) auth(translate gui.Translator) string {
	if e.cred.Login == "" {
		return translate("smtp.auth.none")
	}

	return e.cred.Login
}

// connection is the encryption the port implies.
func connection(translate gui.Translator, port uint16) string {
	if port == implicitTLSPort {
		return translate("smtp.tls.implicit")
	}

	return translate("smtp.tls.starttls")
}

type listQuery = liststate.Query

func parseListQuery(v url.Values) *listQuery { return listSpec.Parse(v) }

func ret(r *http.Request) string   { return listSpec.Ret(r) }
func listURL(ret string) string    { return listSpec.ListURL(ret) }
func withRet(u, ret string) string { return liststate.WithRet(u, ret) }

func credURL(id int64, suffix string) string {
	return common.WebuiURL(pagePath + "/" + strconv.FormatInt(id, 10) + suffix)
}

// uses reads the EMAIL tasks and their rules: for every sender, the rules
// that use it.
func (h *Handlers) uses() (map[string][]ruleuse.Use, error) {
	return ruleuse.Read(h.db, emailParams) //nolint:wrapcheck // logged by the caller with its context
}

// all reads the credentials, by sender, each with the rules that use it.
func (h *Handlers) all() ([]entry, error) {
	var list model.SMTPCredentials
	if err := h.db.Select(&list).OrderBy("email_address", true).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	uses, err := h.uses()
	if err != nil {
		return nil, err
	}

	entries := make([]entry, 0, len(list))
	for _, c := range list {
		entries = append(entries, entryOf(c, uses))
	}

	return entries, nil
}

type row struct {
	ID      int64
	Sender  string
	URL     string
	Server  string
	Auth    string
	NoAuth  bool
	Uses    string
	Actions []gui.Action
}

type listData struct {
	AddURL      string
	Action      string
	Base        string
	Q           string
	Names       []string
	Sort        string
	Dir         string
	Hidden      map[string]string
	Count       string
	Rows        []row
	Caption     string
	Pager       *liststate.Pager
	PagerHidden map[string]string
	Sizes       []int
	Size        int
	Empty       bool
	NoMatch     bool
	NoMatchText string
	NoMatchURL  string
	NoMatchLink string
	Error       bool
	RetryURL    string
}

// List serves GET /webui/smtp.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	fragment := liststate.IsListFragment(r)
	if !fragment && listSpec.Restore(w, r) {
		return
	}

	translate := h.shell.Translator(r)
	query := parseListQuery(r.URL.Query())
	screen := h.screen(translate, translate("smtp.title"), "", 0)
	screen.Crumbs = crumbs(translate, "", "")
	screen.Notices = notices(translate, r.URL.Query())
	screen.Head = gui.LiveAssets()

	entries, err := h.all()
	if err != nil {
		h.logger.Errorf("failed to list the SMTP credentials: %v", err)

		if fragment {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}

		screen.Status = http.StatusInternalServerError
		h.shell.Render(w, h.logger, r, screen,
			listView(listData{Error: true, RetryURL: listSpec.ListURLOf(query.WithPage(query.Page))}))

		return
	}

	data := buildList(translate, entries, query, rightsOf(r))
	if fragment {
		h.shell.RenderFragment(w, h.logger, r, listFragment(data))

		return
	}

	h.shell.Render(w, h.logger, r, screen, listView(data))
}

// buildList computes the list page: search (sender prefix), sort, page,
// toolbar state, row actions according to the rights.
func buildList(translate gui.Translator, all []entry, query *listQuery, user rights) listData {
	data := listData{
		Action: listSpec.ListURLOf(nil),
		Base:   listSpec.ListURLOf(query.FilterValues()),
		Q:      query.Q,
		Sort:   query.Sort,
		Dir:    query.Dir,
		Size:   query.Size,
		Hidden: query.ToolbarHidden(),
	}

	if user.Write {
		data.AddURL = withRet(common.WebuiURL(pagePath+"/new"), query.Encode(query.Page))
	}

	names := make([]string, 0, len(all))
	kept := make([]entry, 0, len(all))

	for i := range all {
		e := &all[i]
		names = append(names, e.cred.EmailAddress)

		if query.Search.Matches(e.cred.EmailAddress) {
			kept = append(kept, *e)
		}
	}

	slices.Sort(names)
	data.Names = names
	sortEntries(kept, query.Sort, query.Dir)
	data.Count = counter.Text(translate, len(all), len(kept))

	switch {
	case len(all) == 0:
		data.Empty = true

		return data
	case len(kept) == 0:
		withoutQ := *query
		withoutQ.Q = ""
		data.NoMatch = true
		data.NoMatchText = translate("smtp.none.title", "query", query.Q)
		data.NoMatchURL, data.NoMatchLink = listSpec.ExplicitListURL(withoutQ.Values()), translate("search.clear")

		return data
	}

	b := query.Bounds(len(kept))
	carried := query.Encode(b.Page)
	data.Caption = translate("smtp.caption",
		"page", b.Page, "pages", b.Pages, "sort", translate(sortKeyPrefix+query.Sort))

	for i := range kept[b.From:b.To] {
		e := &kept[b.From+i]
		data.Rows = append(data.Rows, row{
			ID:      e.cred.ID,
			Sender:  e.cred.EmailAddress,
			URL:     withRet(credURL(e.cred.ID, ""), carried),
			Server:  e.server(),
			Auth:    e.auth(translate),
			NoAuth:  e.cred.Login == "",
			Uses:    ruleuse.Count(translate, len(e.uses)),
			Actions: actions(translate, e, user, false, carried),
		})
	}

	data.Sizes = liststate.DefaultSizes
	data.PagerHidden = query.PagerHidden()
	data.Pager = listSpec.Pager(query, translate("smtp.pager", "from", b.From+1, "to", b.To, "total", len(kept)), b)

	return data
}

// sortEntries sorts by the asked column; ties keep the sender order
// (regardless of case).
func sortEntries(entries []entry, key, dir string) {
	bySender := func(a, b *entry) int {
		return strings.Compare(strings.ToLower(a.cred.EmailAddress), strings.ToLower(b.cred.EmailAddress))
	}

	slices.SortStableFunc(entries, func(left, right entry) int {
		order := bySender(&left, &right)
		if key == sortServer {
			order = strings.Compare(strings.ToLower(left.server()), strings.ToLower(right.server()))
		}

		if dir == dirDesc {
			order = -order
		}

		return cmp.Or(order, bySender(&left, &right))
	})
}

// actions is the "⋯" menu of a credential, in the list (with "View the
// sheet" and "Edit") and on the sheet (without: the buttons are there).
func actions(translate gui.Translator, e *entry, user rights, onSheet bool, carried string) []gui.Action {
	id := e.cred.ID

	var acts []gui.Action

	if !onSheet {
		acts = append(acts, gui.Action{
			Icon: "eye", Label: translate("smtp.action.detail"), URL: withRet(credURL(id, ""), carried),
		})

		if user.Write {
			acts = append(acts, gui.Action{
				Icon: "pen", Label: translate("smtp.action.edit"), URL: withRet(credURL(id, "/edit"), carried),
			})
		}
	}

	if user.Delete {
		acts = append(acts, gui.Action{
			Icon: "trash", Label: translate("smtp.action.delete"), URL: withRet(credURL(id, "/delete"), carried),
			Danger: true,
		})
	}

	return acts
}

// crumbs is the breadcrumb: the list alone, or the list (as the user left
// it) then the page.
func crumbs(translate gui.Translator, last, carried string) []gui.Crumb {
	c := []gui.Crumb{
		{Label: translate("nav.home"), URL: common.WebuiURL("/home")},
		{Label: translate("nav.treatments")},
	}

	if last == "" {
		return append(c, gui.Crumb{Label: translate("smtp.title")})
	}

	return append(c, gui.Crumb{Label: translate("smtp.title"), URL: listURL(carried)}, gui.Crumb{Label: last})
}

// notices turns the result of the previous action into a notification.
func notices(translate gui.Translator, values url.Values) []gui.Notice {
	if done := values.Get(doneParam); slices.Contains(doneKinds, done) {
		return []gui.Notice{{Kind: stateOK, Title: translate(toastKeyPrefix+done, "name", values.Get(nameParam))}}
	}

	return nil
}

// screen is a page of the screen: the chrome around the data.
func (h *Handlers) screen(translate gui.Translator, title, carried string, status int) common.Screen {
	return common.Screen{
		Title:       title,
		Section:     constants.SidebarSectionTreatments,
		Menu:        constants.SidebarMenuSMTPCreds,
		Doc:         docPage,
		Crumbs:      crumbs(translate, title, carried),
		Status:      status,
		Stylesheets: []string{stylesheet, ruleuse.Stylesheet},
	}
}

// -------------------------------------------------------------- pages ----

type notFoundData struct {
	Title     string
	Text      string
	BackLabel string
	BackURL   string
}

// credential reads the {id} of the route with the rules that use it; a
// missing credential renders the localised "not found" page.
func (h *Handlers) credential(w http.ResponseWriter, r *http.Request) (entry, bool) {
	id, parseErr := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

	var c model.SMTPCredential

	err := h.db.Get(&c, "id=?", id).Run()
	if parseErr == nil && err == nil {
		var uses map[string][]ruleuse.Use
		if uses, err = h.uses(); err == nil {
			return entryOf(&c, uses), true
		}
	}

	if err != nil && !database.IsNotFound(err) {
		h.logger.Errorf("failed to retrieve the SMTP credential %d: %v", id, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return entry{}, false
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("smtp.not_found.title")

	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, http.StatusNotFound), notFoundView(notFoundData{
		Title:     title,
		Text:      translate("smtp.not_found.text"),
		BackLabel: translate("smtp.back"),
		BackURL:   listURL(carried),
	}))

	return entry{}, false
}

type detailData struct {
	Sender      string
	Host        string
	Port        string
	Connection  string
	Incomplete  bool
	Login       string
	PasswordSet bool
	Uses        []ruleuse.Row
	BackURL     string
	EditURL     string
	Actions     []gui.Action
}

// Detail serves GET /webui/smtp/{id}: the credential in reading, never its
// password, with the rules whose EMAIL tasks name it.
func (h *Handlers) Detail(w http.ResponseWriter, r *http.Request) {
	item, ok := h.credential(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	user := rightsOf(r)
	addr := item.cred.ServerAddress

	data := detailData{
		Sender:      item.cred.EmailAddress,
		Host:        addr.Host,
		Incomplete:  item.incomplete(),
		Login:       item.cred.Login,
		PasswordSet: item.cred.Password != "",
		Uses:        ruleuse.Rows(translate, item.uses),
		BackURL:     listURL(carried),
		Actions:     actions(translate, &item, user, true, carried),
	}

	if addr.Port != 0 {
		data.Port = strconv.Itoa(int(addr.Port))
		data.Connection = connection(translate, addr.Port)
	}

	if user.Write {
		data.EditURL = withRet(credURL(item.cred.ID, "/edit"), carried)
	}

	title := translate("smtp.detail.title", "name", item.cred.EmailAddress)
	screen := h.screen(translate, title, carried, 0)
	screen.Notices = notices(translate, r.URL.Query())
	h.shell.Render(w, h.logger, r, screen, detailView(data))
}

type deleteData struct {
	Title     string
	Sender    string
	Server    string
	Uses      []ruleuse.Row
	Used      bool
	Rules     string // how many rules use it
	CancelURL string
	Action    string
}

// DeleteForm serves GET /webui/smtp/{id}/delete: the confirmation page; for
// a credential a rule uses, the page that lists the rules to change first.
func (h *Handlers) DeleteForm(w http.ResponseWriter, r *http.Request) {
	item, ok := h.credential(w, r)
	if !ok {
		return
	}

	h.renderDelete(w, r, &item, 0)
}

func (h *Handlers) renderDelete(w http.ResponseWriter, r *http.Request, item *entry, status int) {
	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("smtp.delete.title", "name", item.cred.EmailAddress)

	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, status), deleteView(deleteData{
		Title:     title,
		Sender:    item.cred.EmailAddress,
		Server:    item.server(),
		Uses:      ruleuse.Rows(translate, item.uses),
		Used:      len(item.uses) > 0,
		Rules:     ruleuse.Count(translate, len(item.uses)),
		CancelURL: withRet(credURL(item.cred.ID, ""), carried),
		Action:    withRet(credURL(item.cred.ID, "/delete"), carried),
	}))
}

// Delete serves POST /webui/smtp/{id}/delete. A credential a rule uses is
// not deleted (its EMAIL tasks would fail at the next transfer, the product
// refusing nothing): the page lists the rules again (409).
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	item, ok := h.credential(w, r)
	if !ok {
		return
	}

	if len(item.uses) > 0 {
		h.renderDelete(w, r, &item, http.StatusConflict)

		return
	}

	if err := h.db.Delete(item.cred).Run(); err != nil {
		h.logger.Errorf("failed to delete the SMTP credential %q: %v", item.cred.EmailAddress, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	listSpec.RedirectList(w, r, ret(r), url.Values{doneParam: {doneDeleted}, nameParam: {item.cred.EmailAddress}})
}

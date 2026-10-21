// Package emailtemplates serves the "Email templates" screen of the web
// interface (Treatments): the list of the messages the EMAIL task sends
// (searched by name, sorted and paged), with their subject, format,
// attachments and the rules that use them, the sheet of a template with a
// preview of the message, and the pages that create, edit and delete one.
//
// A template holds a subject, a body in a format (plain text or HTML) and the
// paths of attachments, all of which accept the substitution markers of the
// tasks. An EMAIL task names its template by its name, without a link in the
// database: the screen keeps the name of a template a rule uses, and replaces
// its deletion by the list of the rules to change first. Until this screen,
// the templates were managed through the REST API and the command line only.
package emailtemplates

import (
	"cmp"
	"mime"
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
	// The stylesheet and the script of the screen, under the web interface.
	stylesheet = "/static_v2/email_templates/email-templates.css"
	script     = "/static_v2/email_templates/email-templates.js"

	docPage  = "user/webui/modeles"
	pagePath = "/email-templates"

	// The cookie that keeps the state of the list for the browser session.
	stateCookie = "webui_email_templates_list"

	doneParam = liststate.DoneParam
	nameParam = liststate.NameParam
	dirDesc   = liststate.DirDesc

	sortName    = "name"
	sortSubject = "subject"

	doneCreated = liststate.DoneCreated
	doneUpdated = liststate.DoneUpdated
	doneDeleted = liststate.DoneDeleted

	// The two formats the screen edits; a template may hold another MIME
	// type, set through the REST API, shown as it is.
	mimeText = "text/plain"
	mimeHTML = "text/html"

	// Prefixes of the catalogue keys chosen at run time.
	sortKeyPrefix  = "email_templates.sort."
	toastKeyPrefix = "email_templates.toast."

	// Kinds of a gui.Notice.
	stateOK = "ok"
)

//nolint:gochecknoglobals // reference data of the screen
var (
	sortKeys  = []string{sortName, sortSubject}
	doneKinds = []string{doneCreated, doneUpdated, doneDeleted}

	counter = liststate.Counter{
		One: "email_templates.count_one", Many: "email_templates.count",
		ShownOne: "email_templates.count_shown_one", ShownMany: "email_templates.count_shown",
	}

	listSpec = liststate.Spec{Path: pagePath, Cookie: stateCookie, Sorts: sortKeys, Sizes: liststate.DefaultSizes}

	// The EMAIL task names its template by its name.
	emailParams = ruleuse.Params{tasks.Email: {"template"}}
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

// rights of the current user on the templates: those of the rules, as the
// REST API of the templates. Reading is implied by reaching the screen;
// writing and deleting are checked again by the router on each action.
type rights struct{ Write, Delete bool }

func rightsOf(r *http.Request) rights {
	perms := common.GetUser(r).Permissions

	return rights{
		Write:  perms.HasPermission(model.PermRulesWrite),
		Delete: perms.HasPermission(model.PermRulesDelete),
	}
}

// entry is a template with the rules whose EMAIL tasks name it.
type entry struct {
	tpl  *model.EmailTemplate
	uses []ruleuse.Use
}

func entryOf(t *model.EmailTemplate, uses map[string][]ruleuse.Use) entry {
	return entry{tpl: t, uses: uses[t.Name]}
}

// mediaType is the MIME type of a template without its parameters
// ("text/html; charset=utf-8" is "text/html"), or as it is if it does not
// parse.
func mediaType(mimeType string) string {
	if parsed, _, err := mime.ParseMediaType(mimeType); err == nil {
		return parsed
	}

	return mimeType
}

// isHTML tells whether the body of a template is HTML.
func isHTML(mimeType string) bool { return mediaType(mimeType) == mimeHTML }

// formatLabel is the format of a template as the screen names it: "Text",
// "HTML", or the MIME type the REST API set, as it is.
func formatLabel(translate gui.Translator, mimeType string) string {
	switch mediaType(mimeType) {
	case mimeText, "":
		return translate("email_templates.format.text")
	case mimeHTML:
		return translate("email_templates.format.html")
	default:
		return mimeType
	}
}

// attachmentsCount says in a cell how many attachments a template has.
func attachmentsCount(translate gui.Translator, n int) string {
	switch n {
	case 0:
		return translate("email_templates.attachments.none")
	case 1:
		return translate("email_templates.attachments.one")
	default:
		return translate("email_templates.attachments.many", "n", n)
	}
}

type listQuery = liststate.Query

func parseListQuery(v url.Values) *listQuery { return listSpec.Parse(v) }

func ret(r *http.Request) string   { return listSpec.Ret(r) }
func listURL(ret string) string    { return listSpec.ListURL(ret) }
func withRet(u, ret string) string { return liststate.WithRet(u, ret) }

func templateURL(id int64, suffix string) string {
	return common.WebuiURL(pagePath + "/" + strconv.FormatInt(id, 10) + suffix)
}

// uses reads the EMAIL tasks and their rules: for every template, the rules
// that use it.
func (h *Handlers) uses() (map[string][]ruleuse.Use, error) {
	return ruleuse.Read(h.db, emailParams) //nolint:wrapcheck // logged by the caller with its context
}

// all reads the templates, by name, each with the rules that use it.
func (h *Handlers) all() ([]entry, error) {
	var list model.EmailTemplates
	if err := h.db.Select(&list).OrderBy("name", true).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	uses, err := h.uses()
	if err != nil {
		return nil, err
	}

	entries := make([]entry, 0, len(list))
	for _, t := range list {
		entries = append(entries, entryOf(t, uses))
	}

	return entries, nil
}

type row struct {
	ID          int64
	Name        string
	URL         string
	Subject     string
	Format      string
	Attachments string
	Uses        string
	Actions     []gui.Action
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

// List serves GET /webui/email-templates.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	if listSpec.Restore(w, r) {
		return
	}

	translate := h.shell.Translator(r)
	query := parseListQuery(r.URL.Query())
	screen := h.screen(translate, translate("email_templates.title"), "", 0)
	screen.Crumbs = crumbs(translate, "", "")
	screen.Notices = notices(translate, r.URL.Query())

	entries, err := h.all()
	if err != nil {
		h.logger.Errorf("failed to list the email templates: %v", err)

		screen.Status = http.StatusInternalServerError
		h.shell.Render(w, h.logger, r, screen,
			listView(listData{Error: true, RetryURL: listSpec.ListURLOf(query.WithPage(query.Page))}))

		return
	}

	h.shell.Render(w, h.logger, r, screen, listView(buildList(translate, entries, query, rightsOf(r))))
}

// buildList computes the list page: search (name prefix), sort, page,
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
	needle := strings.ToLower(query.Q)

	for i := range all {
		e := &all[i]
		names = append(names, e.tpl.Name)

		if needle == "" || strings.HasPrefix(strings.ToLower(e.tpl.Name), needle) {
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
		data.NoMatchText = translate("email_templates.none.title", "query", query.Q)
		data.NoMatchURL, data.NoMatchLink = listSpec.ExplicitListURL(withoutQ.Values()), translate("search.clear")

		return data
	}

	b := query.Bounds(len(kept))
	carried := query.Encode(b.Page)
	data.Caption = translate("email_templates.caption",
		"page", b.Page, "pages", b.Pages, "sort", translate(sortKeyPrefix+query.Sort))

	for i := range kept[b.From:b.To] {
		e := &kept[b.From+i]
		data.Rows = append(data.Rows, row{
			ID:          e.tpl.ID,
			Name:        e.tpl.Name,
			URL:         withRet(templateURL(e.tpl.ID, ""), carried),
			Subject:     e.tpl.Subject,
			Format:      formatLabel(translate, e.tpl.MIMEType),
			Attachments: attachmentsCount(translate, len(e.tpl.Attachments)),
			Uses:        ruleuse.Count(translate, len(e.uses)),
			Actions:     actions(translate, e, user, false, carried),
		})
	}

	data.Sizes = liststate.DefaultSizes
	data.PagerHidden = query.PagerHidden()
	data.Pager = listSpec.Pager(query,
		translate("email_templates.pager", "from", b.From+1, "to", b.To, "total", len(kept)), b)

	return data
}

// sortEntries sorts by the asked column; ties keep the name order
// (regardless of case).
func sortEntries(entries []entry, key, dir string) {
	byName := func(a, b *entry) int {
		return strings.Compare(strings.ToLower(a.tpl.Name), strings.ToLower(b.tpl.Name))
	}

	slices.SortStableFunc(entries, func(left, right entry) int {
		order := byName(&left, &right)
		if key == sortSubject {
			order = strings.Compare(strings.ToLower(left.tpl.Subject), strings.ToLower(right.tpl.Subject))
		}

		if dir == dirDesc {
			order = -order
		}

		return cmp.Or(order, byName(&left, &right))
	})
}

// actions is the "⋯" menu of a template, in the list (with "View the sheet"
// and "Edit") and on the sheet (without: the buttons are there).
func actions(translate gui.Translator, e *entry, user rights, onSheet bool, carried string) []gui.Action {
	id := e.tpl.ID

	var acts []gui.Action

	if !onSheet {
		acts = append(acts, gui.Action{
			Icon: "eye", Label: translate("email_templates.action.detail"), URL: withRet(templateURL(id, ""), carried),
		})

		if user.Write {
			acts = append(acts, gui.Action{
				Icon: "pen", Label: translate("email_templates.action.edit"), URL: withRet(templateURL(id, "/edit"), carried),
			})
		}
	}

	if user.Delete {
		acts = append(acts, gui.Action{
			Icon: "trash", Label: translate("email_templates.action.delete"), URL: withRet(templateURL(id, "/delete"), carried),
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
		return append(c, gui.Crumb{Label: translate("email_templates.title")})
	}

	return append(c, gui.Crumb{Label: translate("email_templates.title"), URL: listURL(carried)}, gui.Crumb{Label: last})
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
		Menu:        constants.SidebarMenuEmailTemplates,
		Doc:         docPage,
		Crumbs:      crumbs(translate, title, carried),
		Status:      status,
		Stylesheets: []string{stylesheet},
	}
}

// -------------------------------------------------------------- pages ----

type notFoundData struct {
	Title     string
	Text      string
	BackLabel string
	BackURL   string
}

// template reads the {id} of the route with the rules that use it; a
// missing template renders the localised "not found" page.
func (h *Handlers) template(w http.ResponseWriter, r *http.Request) (entry, bool) {
	id, parseErr := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

	var t model.EmailTemplate

	err := h.db.Get(&t, "id=?", id).Run()
	if parseErr == nil && err == nil {
		var uses map[string][]ruleuse.Use
		if uses, err = h.uses(); err == nil {
			return entryOf(&t, uses), true
		}
	}

	if err != nil && !database.IsNotFound(err) {
		h.logger.Errorf("failed to retrieve the email template %d: %v", id, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return entry{}, false
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("email_templates.not_found.title")

	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, http.StatusNotFound), notFoundView(notFoundData{
		Title:     title,
		Text:      translate("email_templates.not_found.text"),
		BackLabel: translate("email_templates.back"),
		BackURL:   listURL(carried),
	}))

	return entry{}, false
}

type detailData struct {
	Name        string
	Subject     string
	Format      string
	Attachments []string
	Uses        []ruleuse.Row
	Preview     *previewData
	BackURL     string
	EditURL     string
	Actions     []gui.Action
}

// Detail serves GET /webui/email-templates/{id}: the template in reading,
// with a preview of the message (?values= chooses the values of the
// markers, ?mark=1 highlights them) and the rules whose EMAIL tasks name it.
func (h *Handlers) Detail(w http.ResponseWriter, r *http.Request) {
	item, ok := h.template(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	user := rightsOf(r)

	data := detailData{
		Name:        item.tpl.Name,
		Subject:     item.tpl.Subject,
		Format:      formatLabel(translate, item.tpl.MIMEType),
		Attachments: item.tpl.Attachments,
		Uses:        ruleuse.Rows(translate, item.uses),
		BackURL:     listURL(carried),
		Actions:     actions(translate, &item, user, true, carried),
	}

	query := r.URL.Query()
	data.Preview = h.preview(translate, item.tpl, item.uses, query.Get(valuesParam), query.Get(markParam) != "")
	data.Preview.Action, data.Preview.Ret = templateURL(item.tpl.ID, ""), carried

	if user.Write {
		data.EditURL = withRet(templateURL(item.tpl.ID, "/edit"), carried)
	}

	title := translate("email_templates.detail.title", "name", item.tpl.Name)
	screen := h.screen(translate, title, carried, 0)
	screen.Notices = notices(translate, query)
	h.shell.Render(w, h.logger, r, screen, detailView(data))
}

type deleteData struct {
	Title     string
	Name      string
	Subject   string
	Uses      []ruleuse.Row
	Used      bool
	Rules     string // how many rules use it
	CancelURL string
	Action    string
}

// DeleteForm serves GET /webui/email-templates/{id}/delete: the
// confirmation page; for a template a rule uses, the page that lists the
// rules to change first.
func (h *Handlers) DeleteForm(w http.ResponseWriter, r *http.Request) {
	item, ok := h.template(w, r)
	if !ok {
		return
	}

	h.renderDelete(w, r, &item, 0)
}

func (h *Handlers) renderDelete(w http.ResponseWriter, r *http.Request, item *entry, status int) {
	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("email_templates.delete.title", "name", item.tpl.Name)

	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, status), deleteView(deleteData{
		Title:     title,
		Name:      item.tpl.Name,
		Subject:   item.tpl.Subject,
		Uses:      ruleuse.Rows(translate, item.uses),
		Used:      len(item.uses) > 0,
		Rules:     ruleuse.Count(translate, len(item.uses)),
		CancelURL: withRet(templateURL(item.tpl.ID, ""), carried),
		Action:    withRet(templateURL(item.tpl.ID, "/delete"), carried),
	}))
}

// Delete serves POST /webui/email-templates/{id}/delete. A template a rule
// uses is not deleted (its EMAIL tasks would fail at the next transfer, the
// product refusing nothing): the page lists the rules again (409).
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	item, ok := h.template(w, r)
	if !ok {
		return
	}

	if len(item.uses) > 0 {
		h.renderDelete(w, r, &item, http.StatusConflict)

		return
	}

	if err := h.db.Delete(item.tpl).Run(); err != nil {
		h.logger.Errorf("failed to delete the email template %q: %v", item.tpl.Name, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	listSpec.RedirectList(w, r, ret(r), url.Values{doneParam: {doneDeleted}, nameParam: {item.tpl.Name}})
}

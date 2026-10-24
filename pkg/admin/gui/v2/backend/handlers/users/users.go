// Package users serves the "Users" screen of the web interface
// (Administration): the list of the access accounts of the gateway with their
// rights by domain (searched, filtered, sorted and paged), the sheet of a
// user in reading, and the pages that create, edit and delete one.
//
// A user is an access account to the administration of the gateway: the same
// login and password serve the web interface, the REST API and the command
// line. It holds a login, a password kept hashed and its rights by domain,
// nothing else. Two rules of the product frame the actions: the last
// administrator (the last user holding the write right on the users) cannot
// be deleted, and, as the REST API does, a user cannot delete their own
// account.
package users

import (
	"cmp"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"code.waarp.fr/lib/gui"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/rights"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	// The stylesheet of the screen, under the web interface.
	stylesheet = "/static_v2/users/users.css"

	docPage  = "user/webui/utilisateurs"
	pagePath = "/users"

	// The "My account" page: where a user lands after saving their own
	// account without the read right on the users (the sheet would be
	// refused).
	accountPath = "/account"

	// The cookie that keeps the state of the list for the browser session, so
	// that coming back through the sidebar finds the list as it was left.
	stateCookie = "webui_users_list"

	// The parameters of the list state and of the result of the previous
	// action, shared by the list screens.
	retParam  = liststate.RetParam
	doneParam = liststate.DoneParam
	nameParam = liststate.NameParam
	dirDesc   = liststate.DirDesc

	sortName = "name" // the other sort keys are the domains of the rights

	doneCreated = liststate.DoneCreated
	doneUpdated = liststate.DoneUpdated
	doneDeleted = liststate.DoneDeleted
	// The result "My account" is told of: the user just saved their own
	// account.
	doneAccount = "account"

	// The filters of the list: a domain of the rights and a right held in it.
	domainParam = "domain"
	rightParam  = "right"
	rightRead   = "read"
	rightWrite  = "write"
	rightDelete = "delete"
	rightNone   = "none"

	// Why a deletion is refused: the signed-in user's own account, the last
	// administrator.
	blockedSelf      = "self"
	blockedLastAdmin = "last_admin"

	// Prefixes of the catalogue keys chosen at run time: the column and the
	// sort name of a domain ("users.col.servers", "users.sort.servers"),
	// the levels of the right filter ("users.filter.level.read"), the cause
	// of a refused deletion ("users.delete.blocked.self"), the result of an
	// action ("users.toast.created").
	colKeyPrefix     = "users.col."
	sortKeyPrefix    = "users.sort."
	levelKeyPrefix   = "users.filter.level."
	blockedKeyPrefix = "users.delete.blocked."
	toastKeyPrefix   = "users.toast."

	// Kinds of a gui.Notice.
	stateOK = "ok"
)

//nolint:gochecknoglobals // reference data of the screen
var (
	sortKeys    = sortKeysOf()
	rightLevels = []string{rightRead, rightWrite, rightDelete, rightNone}
	doneKinds   = []string{doneCreated, doneUpdated, doneDeleted}

	usersCounter = liststate.Counter{
		One: "users.count_one", Many: "users.count",
		ShownOne: "users.count_shown_one", ShownMany: "users.count_shown",
	}

	// The list of this screen (listSpec): its page, its session cookie, its
	// sort keys (the login, then the domains), its rows per page and the
	// filters by domain and right.
	listSpec = liststate.Spec{
		Path: pagePath, Cookie: stateCookie, Sorts: sortKeys, Sizes: liststate.DefaultSizes,
		ParseFilters: parseFilter, FilterParams: []string{domainParam, rightParam},
	}
)

// sortKeysOf lists the sort keys: the login first (the default), then one
// per domain of the rights, in the order of the columns.
func sortKeysOf() []string {
	keys := []string{sortName}
	for _, d := range rights.Domains {
		keys = append(keys, d.Key)
	}

	return keys
}

// domainOf finds the domain of a sort key or a filter; false for the login
// or an unknown key.
func domainOf(key string) (rights.Domain, bool) {
	for _, d := range rights.Domains {
		if d.Key == key {
			return d, true
		}
	}

	return rights.Domain{}, false
}

// Handlers serves the pages and the actions of the screen.
type Handlers struct {
	db     *database.DB
	logger *log.Logger
	shell  *common.Shell
}

func New(db *database.DB, logger *log.Logger) *Handlers {
	return &Handlers{db: db, logger: logger, shell: common.NewShell()}
}

// caller is the signed-in user and what they may do on the users: reading is
// implied by reaching the screen; writing and deleting are checked again by
// the router on each action. The templates hide what the rights forbid.
type caller struct {
	user       *model.User
	write, del bool
}

func callerOf(r *http.Request) caller {
	user := common.GetUser(r)

	return caller{
		user:  user,
		write: user.HasPermission(model.PermUsersWrite),
		del:   user.HasPermission(model.PermUsersDelete),
	}
}

// is tells whether the user is the signed-in one.
func (c caller) is(user *model.User) bool { return c.user.ID == user.ID }

// choice is an option of a select of the filters.
type choice = protoform.Choice

// --------------------------------------------------------------- list ----

// filter keeps the users holding a right in a domain. The domain alone keeps
// the users with any right in it; the right alone, those holding it in any
// domain; "none", those without any right in the domain (in every domain
// when none is chosen).
type filter struct {
	Domain string
	Right  string
}

// parseFilter reads the filters from a query, dropping what is unknown.
func parseFilter(v url.Values) liststate.Filters {
	f := &filter{}

	if _, ok := domainOf(v.Get(domainParam)); ok {
		f.Domain = v.Get(domainParam)
	}

	if slices.Contains(rightLevels, v.Get(rightParam)) {
		f.Right = v.Get(rightParam)
	}

	return f
}

func (f *filter) Values() url.Values {
	v := url.Values{}
	if f.Domain != "" {
		v.Set(domainParam, f.Domain)
	}

	if f.Right != "" {
		v.Set(rightParam, f.Right)
	}

	return v
}

func (f *filter) Count() int {
	n := 0
	if f.Domain != "" {
		n++
	}

	if f.Right != "" {
		n++
	}

	return n
}

// domains are the domains the filter looks at: the chosen one, or all.
func (f *filter) domains() []rights.Domain {
	if d, ok := domainOf(f.Domain); ok {
		return []rights.Domain{d}
	}

	return rights.Domains
}

// Matches tells whether a user passes the filter.
func (f *filter) Matches(perms model.PermsMask) bool {
	if f.Count() == 0 {
		return true
	}

	if f.Right == rightNone {
		return !slices.ContainsFunc(f.domains(), func(d rights.Domain) bool { return d.Count(perms) > 0 })
	}

	return slices.ContainsFunc(f.domains(), func(d rights.Domain) bool { return holds(d, perms, f.Right) })
}

// holds tells whether the user holds the right in the domain; any right
// when none is asked.
func holds(d rights.Domain, perms model.PermsMask, right string) bool {
	switch right {
	case rightRead:
		return perms.HasPermission(d.Read)
	case rightWrite:
		return perms.HasPermission(d.Write)
	case rightDelete:
		return d.Delete != 0 && perms.HasPermission(d.Delete)
	default:
		return d.Count(perms) > 0
	}
}

// domainChoices are the options of the domain select: every domain, or the
// chosen one.
func (f *filter) domainChoices(translate gui.Translator) []choice {
	choices := []choice{{Value: "", Label: translate("users.filter.any_domain"), Selected: f.Domain == ""}}
	for _, d := range rights.Domains {
		choices = append(choices, choice{Value: d.Key, Label: translate(d.LabelKey()), Selected: f.Domain == d.Key})
	}

	return choices
}

// rightChoices are the options of the right select: every right, or the
// chosen level.
func (f *filter) rightChoices(translate gui.Translator) []choice {
	choices := []choice{{Value: "", Label: translate("users.filter.any_right"), Selected: f.Right == ""}}
	for _, level := range rightLevels {
		choices = append(choices, choice{Value: level, Label: translate(levelKeyPrefix + level), Selected: f.Right == level})
	}

	return choices
}

// listQuery is the state of the list asked by the URL, with the filters.
type listQuery = liststate.Query

func parseListQuery(v url.Values) *listQuery { return listSpec.Parse(v) }

// filterOf is the filter of a list query.
func filterOf(q *listQuery) *filter {
	if f, ok := q.Filters.(*filter); ok {
		return f
	}

	return &filter{}
}

// The list state carried by a request, the list as the user left it, and a
// URL carrying the state: the shared mechanics, on the spec of this list.
func ret(r *http.Request) string   { return listSpec.Ret(r) }
func listURL(ret string) string    { return listSpec.ListURL(ret) }
func withRet(u, ret string) string { return liststate.WithRet(u, ret) }

func userURL(id int64, suffix string) string {
	return common.WebuiURL(pagePath + "/" + strconv.FormatInt(id, 10) + suffix)
}

// column is a domain column of the list.
type column struct {
	Key   string
	Label string
}

func columns(translate gui.Translator) []column {
	cols := make([]column, 0, len(rights.Domains))
	for _, d := range rights.Domains {
		cols = append(cols, column{Key: d.Key, Label: translate(colKeyPrefix + d.Key)})
	}

	return cols
}

type row struct {
	ID      int64
	Name    string
	URL     string
	Self    bool     // the signed-in user: marked "You"
	Letters []string // the notation of the product, one per domain ("rwd")
	Actions []gui.Action
}

type listData struct {
	AddURL       string // empty without the write right: no coloured action at all
	Action       string // toolbar form action: the bare list
	Base         string // list with search and filters: base of the sort links
	Q            string
	Names        []string
	Domains      []choice
	Rights       []choice
	FiltersOpen  bool
	FiltersCount int
	ResetURL     string
	Sort         string
	Dir          string
	Hidden       map[string]string // carried by the toolbar form
	Count        string
	Columns      []column
	Rows         []row
	Caption      string
	Pager        *liststate.Pager
	PagerHidden  map[string]string
	Sizes        []int // rows-per-page choices, offered as soon as there are rows
	Size         int
	Empty        bool // no user at all
	NoMatch      bool // users exist, none matches
	NoMatchText  string
	NoMatchURL   string
	NoMatchLink  string
	Error        bool // the users could not be read
	RetryURL     string
}

// List serves GET /webui/users.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	fragment := liststate.IsListFragment(r)
	if !fragment && listSpec.Restore(w, r) {
		return
	}

	translate := h.shell.Translator(r)
	query := parseListQuery(r.URL.Query())
	screen := h.screen(translate, translate("users.title"), "", 0)
	screen.Crumbs = crumbs(translate, "", "")
	screen.Notices = notices(translate, r.URL.Query())
	screen.Head = gui.LiveAssets()

	users, err := internal.ListUsers(h.db, "username", true, 0, 0)
	if err != nil {
		h.logger.Errorf("failed to list the users: %v", err)

		if fragment {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}

		screen.Status = http.StatusInternalServerError
		h.shell.Render(w, h.logger, r, screen,
			listView(listData{Error: true, RetryURL: listSpec.ListURLOf(query.WithPage(query.Page))}))

		return
	}

	data := buildList(translate, users, query, callerOf(r))
	if fragment {
		h.shell.RenderFragment(w, h.logger, r, listFragment(data))

		return
	}

	h.shell.Render(w, h.logger, r, screen, listView(data))
}

// buildList computes the list page: filter (login prefix, domain and
// right), sort, page, toolbar state, row actions according to the rights.
func buildList(translate gui.Translator, all []*model.User, query *listQuery, c caller) listData {
	active := filterOf(query)
	data := listData{
		Action:       listSpec.ListURLOf(nil),
		Base:         listSpec.ListURLOf(query.FilterValues()),
		Q:            query.Q,
		Sort:         query.Sort,
		Dir:          query.Dir,
		Size:         query.Size,
		ResetURL:     listSpec.ResetURL(),
		Hidden:       query.ToolbarHidden(),
		Domains:      active.domainChoices(translate),
		Rights:       active.rightChoices(translate),
		FiltersCount: active.Count(),
		FiltersOpen:  active.Count() > 0,
		Columns:      columns(translate),
	}

	if c.write {
		data.AddURL = withRet(common.WebuiURL(pagePath+"/new"), query.Encode(query.Page))
	}

	names := make([]string, 0, len(all))
	kept := make([]*model.User, 0, len(all))
	needle := strings.ToLower(query.Q)

	for _, user := range all {
		names = append(names, user.Username)

		if needle != "" && !strings.HasPrefix(strings.ToLower(user.Username), needle) {
			continue
		}

		if !active.Matches(user.Permissions) {
			continue
		}

		kept = append(kept, user)
	}

	slices.Sort(names)
	data.Names = names
	sortUsers(kept, query.Sort, query.Dir)

	data.Count = usersCounter.Text(translate, len(all), len(kept))

	switch {
	case len(all) == 0:
		data.Empty = true

		return data
	case len(kept) == 0:
		data.NoMatch = true

		if query.Q != "" {
			withoutQ := *query
			withoutQ.Q = ""
			data.NoMatchText = translate("users.none.title", "query", query.Q)
			data.NoMatchURL, data.NoMatchLink = listSpec.ExplicitListURL(withoutQ.Values()), translate("search.clear")
		} else {
			data.NoMatchText = translate("users.none.filters")
			data.NoMatchURL, data.NoMatchLink = listSpec.ResetURL(), translate("filters.reset")
		}

		return data
	}

	b := query.Bounds(len(kept))
	carried := query.Encode(b.Page)
	data.Caption = translate("users.caption",
		"page", b.Page, "pages", b.Pages, "sort", translate(sortKeyPrefix+query.Sort))

	for _, user := range kept[b.From:b.To] {
		data.Rows = append(data.Rows, row{
			ID:      user.ID,
			Name:    user.Username,
			URL:     withRet(userURL(user.ID, ""), carried),
			Self:    c.is(user),
			Letters: letters(user.Permissions),
			Actions: actions(translate, user, c, false, carried),
		})
	}

	// The rows per page are offered as soon as there are rows.
	data.Sizes = liststate.DefaultSizes
	data.PagerHidden = query.PagerHidden()
	data.Pager = listSpec.Pager(query, translate("users.pager", "from", b.From+1, "to", b.To, "total", len(kept)), b)

	return data
}

// letters is the notation of the product for every domain, in the order of
// the columns.
func letters(perms model.PermsMask) []string {
	out := make([]string, 0, len(rights.Domains))
	for _, d := range rights.Domains {
		out = append(out, d.Letters(perms))
	}

	return out
}

// sortUsers sorts by the asked column: the login (regardless of case), or a
// domain from the fewest rights to the most; ties keep the login order.
func sortUsers(users []*model.User, key, dir string) {
	byName := func(a, b *model.User) int {
		return strings.Compare(strings.ToLower(a.Username), strings.ToLower(b.Username))
	}

	slices.SortStableFunc(users, func(a, b *model.User) int {
		order := byName(a, b)

		if d, ok := domainOf(key); ok {
			order = cmp.Or(cmp.Compare(d.Count(a.Permissions), d.Count(b.Permissions)),
				strings.Compare(d.Letters(a.Permissions), d.Letters(b.Permissions)))
		}

		if dir == dirDesc {
			order = -order
		}

		return cmp.Or(order, byName(a, b))
	})
}

// actions is the "⋯" menu of a user, in the list (with "View the sheet" and
// "Edit") and on the sheet (without: the buttons are there). Entries follow
// the rights; the destructive one comes last, and never on the signed-in
// user's own row. Every URL carries the list state.
func actions(translate gui.Translator, user *model.User, c caller, onSheet bool, carried string) []gui.Action {
	var acts []gui.Action

	if !onSheet {
		acts = append(acts, gui.Action{
			Icon: "eye", Label: translate("users.action.detail"), URL: withRet(userURL(user.ID, ""), carried),
		})

		if c.write {
			acts = append(acts, gui.Action{
				Icon: "pen", Label: translate("users.action.edit"), URL: withRet(userURL(user.ID, "/edit"), carried),
			})
		}
	}

	if c.del && !c.is(user) {
		acts = append(acts, gui.Action{
			Icon:   "trash",
			Label:  translate("users.action.delete"),
			URL:    withRet(userURL(user.ID, "/delete"), carried),
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
		{Label: translate("nav.administration")},
	}

	if last == "" {
		return append(c, gui.Crumb{Label: translate("users.title")})
	}

	return append(c,
		gui.Crumb{Label: translate("users.title"), URL: listURL(carried)},
		gui.Crumb{Label: last})
}

// notices turns the result of the previous action, carried by the
// redirection, into a notification.
func notices(translate gui.Translator, values url.Values) []gui.Notice {
	if done := values.Get(doneParam); slices.Contains(doneKinds, done) {
		return []gui.Notice{{Kind: stateOK, Title: translate(toastKeyPrefix+done, "name", values.Get(nameParam))}}
	}

	return nil
}

// -------------------------------------------------------------- pages ----

// notFoundData is the localised "not found" page.
type notFoundData struct {
	Title     string
	Text      string
	BackLabel string
	BackURL   string
}

// user reads the {id} of the route; a missing user renders the localised
// "not found" page and returns false.
func (h *Handlers) user(w http.ResponseWriter, r *http.Request) (*model.User, bool) {
	id, parseErr := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

	user, err := internal.GetUserByID(h.db, id)
	if parseErr == nil && err == nil {
		return user, true
	}

	if err != nil && !database.IsNotFound(err) {
		h.logger.Errorf("failed to retrieve the user %d: %v", id, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return nil, false
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("users.not_found.title")

	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, http.StatusNotFound), notFoundView(notFoundData{
		Title:     title,
		Text:      translate("users.not_found.text"),
		BackLabel: translate("users.back"),
		BackURL:   listURL(carried),
	}))

	return nil, false
}

// screen is a page of the screen: the chrome around the data. The list
// itself replaces the breadcrumb and the notices.
func (h *Handlers) screen(translate gui.Translator, title, carried string, status int) common.Screen {
	return common.Screen{
		Title:       title,
		Section:     constants.SidebarSectionAdministration,
		Menu:        constants.SidebarMenuUsers,
		Doc:         docPage,
		Crumbs:      crumbs(translate, title, carried),
		Status:      status,
		Stylesheets: []string{stylesheet, rights.Stylesheet},
	}
}

// lastAdmin tells whether the user is the last one able to manage the users
// (the last holder of the write right on them): the model refuses to delete
// them.
func (h *Handlers) lastAdmin(user *model.User) (bool, error) {
	if !user.HasPermission(model.PermUsersWrite) {
		return false, nil
	}

	admins, err := model.CountAdmins(h.db)
	if err != nil {
		return false, err //nolint:wrapcheck // logged by the caller with its context
	}

	return admins <= 1, nil
}

type detailData struct {
	Name      string
	Self      bool
	LastAdmin bool
	BackURL   string
	EditURL   string
	Actions   []gui.Action
	Caption   string
	Rights    []rights.Row
}

// Detail serves GET /webui/users/{id}: the user in reading, with their
// rights by domain.
func (h *Handlers) Detail(w http.ResponseWriter, r *http.Request) {
	user, ok := h.user(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	c := callerOf(r)

	last, err := h.lastAdmin(user)
	if err != nil {
		h.logger.Errorf("failed to count the administrators: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	data := detailData{
		Name:      user.Username,
		Self:      c.is(user),
		LastAdmin: last,
		BackURL:   listURL(carried),
		Actions:   actions(translate, user, c, true, carried),
		Caption:   translate("users.detail.rights_caption", "name", user.Username),
		Rights:    rights.Rows(translate, user.Permissions),
	}

	if c.write {
		data.EditURL = withRet(userURL(user.ID, "/edit"), carried)
	}

	title := translate("users.detail.title", "name", user.Username)
	screen := h.screen(translate, title, carried, 0)
	screen.Notices = notices(translate, r.URL.Query())
	h.shell.Render(w, h.logger, r, screen, detailView(data))
}

type deleteData struct {
	Name        string
	Caption     string
	Rights      []rights.Row
	CancelURL   string
	Action      string
	Blocked     bool   // the deletion is refused
	BlockedText string // why: the signed-in user's own account, the last administrator
}

// blocked tells why the deletion of the user is refused, or nothing: the
// signed-in user's own account (the rule of the REST API), the last
// administrator (the rule of the model).
func (h *Handlers) blocked(c caller, user *model.User) (string, error) {
	if c.is(user) {
		return blockedSelf, nil
	}

	last, err := h.lastAdmin(user)
	if err != nil {
		return "", err
	}

	if last {
		return blockedLastAdmin, nil
	}

	return "", nil
}

func (h *Handlers) deletePage(w http.ResponseWriter, r *http.Request, user *model.User, blocked string, status int) {
	translate := h.shell.Translator(r)
	carried := ret(r)

	data := deleteData{
		Name:      user.Username,
		Caption:   translate("users.detail.rights_caption", "name", user.Username),
		Rights:    rights.Rows(translate, user.Permissions),
		CancelURL: withRet(userURL(user.ID, ""), carried),
		Action:    withRet(userURL(user.ID, "/delete"), carried),
		Blocked:   blocked != "",
	}

	if blocked != "" {
		data.BlockedText = translate(blockedKeyPrefix + blocked)
	}

	title := translate("users.delete.title", "name", user.Username)
	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, status), deleteView(data))
}

// DeleteForm serves GET /webui/users/{id}/delete: the confirmation page, or
// the refusal.
func (h *Handlers) DeleteForm(w http.ResponseWriter, r *http.Request) {
	user, ok := h.user(w, r)
	if !ok {
		return
	}

	blocked, err := h.blocked(callerOf(r), user)
	if err != nil {
		h.logger.Errorf("failed to count the administrators: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	h.deletePage(w, r, user, blocked, 0)
}

// Delete serves POST /webui/users/{id}/delete. The model checks the last
// administrator again: a refusal in between renders the same page.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	user, ok := h.user(w, r)
	if !ok {
		return
	}

	blocked, err := h.blocked(callerOf(r), user)
	if err != nil {
		h.logger.Errorf("failed to count the administrators: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	if blocked != "" {
		h.deletePage(w, r, user, blocked, http.StatusUnprocessableEntity)

		return
	}

	switch deleteErr := internal.DeleteUser(h.db, user); {
	case deleteErr == nil:
		listSpec.RedirectList(w, r, ret(r), url.Values{doneParam: {doneDeleted}, nameParam: {user.Username}})
	case database.IsValidationError(deleteErr):
		h.deletePage(w, r, user, blockedLastAdmin, http.StatusUnprocessableEntity)
	default:
		h.logger.Errorf("failed to delete the user %q: %v", user.Username, deleteErr)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

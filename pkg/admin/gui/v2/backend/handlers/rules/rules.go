// Package rules serves the "Transfer rules" screen of the web interface
// (Treatments): the list of the rules (searched by name or path, filtered by
// direction, sorted and paged) with their path, the number of tasks of each
// chain and their access, the sheet of a rule, and the pages that create,
// edit and delete one, with its Tasks tab (the chains of tasks) and its
// Access tab (who may use it).
//
// A rule is named by its name and its direction together. The screen does
// what the historic page did, and avoids what it can of the product's known
// behaviours: the direction stays in reading once the rule exists (changing
// it would move its tasks and its access along without a word), a slash is
// refused in a name (the REST API cannot address such a rule), and the
// deletion of a rule that transfers or file watchers still name is replaced
// by what holds it (the product refuses it, without saying why for a file
// watcher).
package rules

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
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/means"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	// The stylesheet of the screen, under the web interface.
	stylesheet = "/static_v2/rules/rules.css"

	docPage  = "user/webui/regles"
	pagePath = "/rules"

	// The transfer monitoring.
	transfersPage = "/transfers"

	// The tabs of the sheet.
	tabDetails = "details"
	tabTasks   = "tasks"
	tabAccess  = "access"

	// The cookie that keeps the state of the list for the browser session.
	stateCookie = "webui_rules_list"

	doneParam = liststate.DoneParam
	nameParam = liststate.NameParam
	dirDesc   = liststate.DirDesc

	sortName      = "name"
	sortDirection = "direction"
	sortPath      = "path"

	doneCreated = liststate.DoneCreated
	doneUpdated = liststate.DoneUpdated
	doneDeleted = liststate.DoneDeleted

	// The filter of the list, and the directions as the address and the
	// transfer monitoring name them ("dir" is the order of the sort).
	directionParam = "direction"
	dirSend        = "send"
	dirReceive     = "receive"

	// Prefixes of the catalogue keys chosen at run time.
	sortKeyPrefix      = "rules.sort."
	toastKeyPrefix     = "rules.toast."
	directionKeyPrefix = "rules.direction."
)

//nolint:gochecknoglobals // reference data of the screen
var (
	sortKeys  = []string{sortName, sortDirection, sortPath}
	doneKinds = []string{
		doneCreated, doneUpdated, doneDeleted, doneAccessAdded, doneAccessRemoved, doneAccessOpened,
		doneTaskAdded, doneTaskUpdated, doneTaskDeleted, doneTaskMoved, doneTasksOrdered,
	}
	directions = []string{dirSend, dirReceive}

	counter = liststate.Counter{
		One: "rules.count_one", Many: "rules.count", ShownOne: "rules.count_shown_one", ShownMany: "rules.count_shown",
	}

	listSpec = liststate.Spec{
		Path: pagePath, Cookie: stateCookie, Sorts: sortKeys, Sizes: liststate.DefaultSizes,
		ParseFilters: parseFilter, FilterParams: []string{directionParam},
	}
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

// rights of the current user on the rules, and whether the transfer
// monitoring is open to them (the link to the transfers of a rule).
type rights struct{ Write, Delete, Transfers bool }

func rightsOf(r *http.Request) rights {
	perms := common.GetUser(r).Permissions

	return rights{
		Write:     perms.HasPermission(model.PermRulesWrite),
		Delete:    perms.HasPermission(model.PermRulesDelete),
		Transfers: perms.HasPermission(model.PermTransfersRead),
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

// directionOf names the direction of a rule as the address does.
func directionOf(isSend bool) string {
	if isSend {
		return dirSend
	}

	return dirReceive
}

// directionIcon is the icon of a direction, the one of the transfer
// monitoring.
func directionIcon(isSend bool) string {
	if isSend {
		return "upload"
	}

	return "download"
}

// entry is a rule with the number of tasks of each chain and of its access
// entries.
type entry struct {
	rule            *model.Rule
	pre, post, errs int
	access          int
}

func (e *entry) tasks() int { return e.pre + e.post + e.errs }

func (e *entry) direction(translate gui.Translator) string {
	return translate(directionKeyPrefix + directionOf(e.rule.IsSend))
}

// tasksText is the number of tasks of each chain: before, after, on error.
func (e *entry) tasksText() string {
	return itoa(e.pre) + " · " + itoa(e.post) + " · " + itoa(e.errs)
}

func (e *entry) tasksTip(translate gui.Translator) string {
	return translate("rules.tasks_tip", "pre", e.pre, "post", e.post, "error", e.errs)
}

// accessText says whether the rule is open to all, or how many servers,
// partners and accounts it is kept for.
func (e *entry) accessText(translate gui.Translator) string {
	if e.access == 0 {
		return translate("rules.access.open")
	}

	return translate("rules.access.restricted", "n", e.access)
}

// filter is the state of the filter of the list: the directions.
type filter struct{ Directions []string }

func parseFilter(v url.Values) liststate.Filters {
	f := &filter{}

	for _, dir := range v[directionParam] {
		if slices.Contains(directions, dir) && !slices.Contains(f.Directions, dir) {
			f.Directions = append(f.Directions, dir)
		}
	}

	return f
}

func (f *filter) Values() url.Values {
	v := url.Values{}
	for _, dir := range f.Directions {
		v.Add(directionParam, dir)
	}

	return v
}

func (f *filter) Count() int { return len(f.Directions) }

type box struct {
	Name    string
	Value   string
	Label   string
	Checked bool
}

func (f *filter) boxes(translate gui.Translator) []box {
	out := make([]box, 0, len(directions))
	for _, dir := range directions {
		out = append(out, box{
			Name: directionParam, Value: dir, Label: translate(directionKeyPrefix + dir),
			Checked: slices.Contains(f.Directions, dir),
		})
	}

	return out
}

type listQuery = liststate.Query

func parseListQuery(v url.Values) *listQuery { return listSpec.Parse(v) }

func filterOf(q *listQuery) *filter {
	if f, ok := q.Filters.(*filter); ok {
		return f
	}

	return &filter{}
}

func ret(r *http.Request) string   { return listSpec.Ret(r) }
func listURL(ret string) string    { return listSpec.ListURL(ret) }
func withRet(u, ret string) string { return liststate.WithRet(u, ret) }

func ruleURL(id int64, suffix string) string {
	return common.WebuiURL(pagePath + "/" + strconv.FormatInt(id, 10) + suffix)
}

// transfersURL is the transfer monitoring filtered on the rule.
func transfersURL(rule *model.Rule) string {
	v := url.Values{"rule": {rule.Name}, directionParam: {directionOf(rule.IsSend)}}

	return common.WebuiURL(transfersPage) + "?" + v.Encode()
}

// counts reads the tasks and the access entries of the rules: for every rule
// ID, its entry without the rule.
func (h *Handlers) counts() (map[int64]*entry, error) {
	var (
		tasks  model.Tasks
		access model.RuleAccesses
	)

	if err := h.db.Select(&tasks).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	if err := h.db.Select(&access).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	out := map[int64]*entry{}
	of := func(id int64) *entry {
		if out[id] == nil {
			out[id] = &entry{}
		}

		return out[id]
	}

	for _, task := range tasks {
		e := of(task.RuleID)

		switch task.Chain {
		case model.ChainPre:
			e.pre++
		case model.ChainPost:
			e.post++
		case model.ChainError:
			e.errs++
		}
	}

	for _, a := range access {
		of(a.RuleID).access++
	}

	return out, nil
}

func entryOf(rule *model.Rule, counts map[int64]*entry) entry {
	e := entry{rule: rule}
	if c := counts[rule.ID]; c != nil {
		e.pre, e.post, e.errs, e.access = c.pre, c.post, c.errs, c.access
	}

	return e
}

// all reads the rules, each with its counts (the list sorts them).
func (h *Handlers) all() ([]entry, error) {
	var list model.Rules
	if err := h.db.Select(&list).OrderBy("name", true).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	counts, err := h.counts()
	if err != nil {
		return nil, err
	}

	entries := make([]entry, 0, len(list))
	for _, rule := range list {
		entries = append(entries, entryOf(rule, counts))
	}

	return entries, nil
}

type row struct {
	ID         int64
	Name       string
	URL        string
	Direction  string
	DirIcon    string
	Path       string
	Tasks      string
	TasksTip   string
	Access     string
	Restricted bool
	Comment    string
	Actions    []gui.Action
}

type listData struct {
	AddURL       string
	Action       string
	Base         string
	Q            string
	Names        []string
	Directions   []box
	FiltersOpen  bool
	FiltersCount int
	ResetURL     string
	Sort         string
	Dir          string
	Hidden       map[string]string
	Count        string
	Rows         []row
	Caption      string
	Pager        *liststate.Pager
	PagerHidden  map[string]string
	Sizes        []int
	Size         int
	Empty        bool
	NoMatch      bool
	NoMatchText  string
	NoMatchURL   string
	NoMatchLink  string
	Error        bool
	RetryURL     string
}

// List serves GET /webui/rules.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	fragment := liststate.IsListFragment(r)
	if listSpec.Restore(w, r) {
		return
	}

	translate := h.shell.Translator(r)
	query := parseListQuery(r.URL.Query())
	screen := h.screen(translate, translate("rules.title"), "", 0)
	screen.Crumbs = crumbs(translate, "", "")
	screen.Notices = notices(translate, r.URL.Query())
	screen.Head = gui.LiveAssets()

	entries, err := h.all()
	if err != nil {
		h.logger.Errorf("failed to list the transfer rules: %v", err)

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

// buildList computes the list page: search, filter, sort, page, toolbar
// state, row actions according to the rights.
func buildList(translate gui.Translator, all []entry, query *listQuery, user rights) listData {
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
		Directions:   active.boxes(translate),
		FiltersCount: active.Count(),
		FiltersOpen:  active.Count() > 0,
	}

	if user.Write {
		data.AddURL = withRet(common.WebuiURL(pagePath+"/new"), query.Encode(query.Page))
	}

	names := make([]string, 0, len(all))
	kept := make([]entry, 0, len(all))

	for i := range all {
		e := &all[i]
		if !slices.Contains(names, e.rule.Name) {
			names = append(names, e.rule.Name)
		}

		if !query.Search.Matches(e.rule.Name) && !query.Search.Matches(e.rule.Path) {
			continue
		}

		if len(active.Directions) > 0 && !slices.Contains(active.Directions, directionOf(e.rule.IsSend)) {
			continue
		}

		kept = append(kept, *e)
	}

	slices.Sort(names)
	data.Names = names
	sortEntries(translate, kept, query.Sort, query.Dir)
	data.Count = counter.Text(translate, len(all), len(kept))

	switch {
	case len(all) == 0:
		data.Empty = true

		return data
	case len(kept) == 0:
		data.NoMatch = true

		if query.Q != "" {
			withoutQ := *query
			withoutQ.Q = ""
			data.NoMatchText = translate("rules.none.title", "query", query.Q)
			data.NoMatchURL, data.NoMatchLink = listSpec.ExplicitListURL(withoutQ.Values()), translate("search.clear")
		} else {
			data.NoMatchText = translate("rules.none.filters")
			data.NoMatchURL, data.NoMatchLink = listSpec.ResetURL(), translate("filters.reset")
		}

		return data
	}

	b := query.Bounds(len(kept))
	carried := query.Encode(b.Page)
	data.Caption = translate("rules.caption",
		"page", b.Page, "pages", b.Pages, "sort", translate(sortKeyPrefix+query.Sort))

	for i := range kept[b.From:b.To] {
		e := &kept[b.From+i]
		data.Rows = append(data.Rows, row{
			ID:         e.rule.ID,
			Name:       e.rule.Name,
			URL:        withRet(ruleURL(e.rule.ID, ""), carried),
			Direction:  e.direction(translate),
			DirIcon:    directionIcon(e.rule.IsSend),
			Path:       e.rule.Path,
			Tasks:      e.tasksText(),
			TasksTip:   e.tasksTip(translate),
			Access:     e.accessText(translate),
			Restricted: e.access > 0,
			Comment:    e.rule.Comment,
			Actions:    actions(translate, e, user, false, carried),
		})
	}

	data.Sizes = liststate.DefaultSizes
	data.PagerHidden = query.PagerHidden()
	data.Pager = listSpec.Pager(query, translate("rules.pager", "from", b.From+1, "to", b.To, "total", len(kept)), b)

	return data
}

// sortEntries sorts by the asked column; ties keep the name order
// (regardless of case), then the sending rule before the receiving one.
func sortEntries(translate gui.Translator, entries []entry, key, dir string) {
	byName := func(a, b *entry) int {
		return cmp.Or(strings.Compare(strings.ToLower(a.rule.Name), strings.ToLower(b.rule.Name)),
			strings.Compare(directionOf(a.rule.IsSend), directionOf(b.rule.IsSend))*-1)
	}

	slices.SortStableFunc(entries, func(left, right entry) int {
		order := byName(&left, &right)

		switch key {
		case sortDirection:
			order = strings.Compare(left.direction(translate), right.direction(translate))
		case sortPath:
			order = strings.Compare(strings.ToLower(left.rule.Path), strings.ToLower(right.rule.Path))
		}

		if dir == dirDesc {
			order = -order
		}

		return cmp.Or(order, byName(&left, &right))
	})
}

// actions: the menu of a row, or of the sheet (no "view" nor "edit" there,
// the sheet has its button).
func actions(translate gui.Translator, item *entry, user rights, onSheet bool, carried string) []gui.Action {
	var acts []gui.Action

	if !onSheet {
		acts = append(acts, gui.Action{
			Icon: "eye", Label: translate("rules.action.detail"), URL: withRet(ruleURL(item.rule.ID, ""), carried),
		})

		if user.Write {
			acts = append(acts, gui.Action{
				Icon: "pen", Label: translate("rules.action.edit"), URL: withRet(ruleURL(item.rule.ID, "/edit"), carried),
			})
		}

		acts = append(acts,
			gui.Action{
				Icon: "list", Label: translate("rules.action.tasks"), URL: withRet(tasksTabURL(item.rule.ID), carried),
			},
			gui.Action{
				Icon: "shield", Label: translate("rules.action.access"), URL: withRet(accessURL(item.rule.ID, ""), carried),
			},
		)
	}

	if user.Transfers {
		acts = append(acts, gui.Action{
			Icon: "clock", Label: translate("rules.action.transfers"), URL: transfersURL(item.rule),
		})
	}

	if user.Delete {
		acts = append(acts, gui.Action{
			Icon: "trash", Label: translate("rules.action.delete"), URL: withRet(ruleURL(item.rule.ID, "/delete"), carried),
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
		return append(c, gui.Crumb{Label: translate("rules.title")})
	}

	return append(c, gui.Crumb{Label: translate("rules.title"), URL: listURL(carried)}, gui.Crumb{Label: last})
}

// notices turns the result of the previous action into a notification; a
// change refused because its chain changed is an error, which stays.
func notices(translate gui.Translator, values url.Values) []gui.Notice {
	if done := values.Get(doneParam); done == doneTaskStale {
		return []gui.Notice{{
			Kind: "ko", Title: translate(toastKeyPrefix + done), Text: translate(toastKeyPrefix + done + "_text"),
		}}
	} else if slices.Contains(doneKinds, done) {
		return []gui.Notice{{Kind: "ok", Title: translate(toastKeyPrefix+done, "name", values.Get(nameParam))}}
	}

	return nil
}

// screen is a page of the screen: the chrome around the data.
func (h *Handlers) screen(translate gui.Translator, title, carried string, status int) common.Screen {
	return common.Screen{
		Title:       title,
		Section:     constants.SidebarSectionTreatments,
		Menu:        constants.SidebarMenuRules,
		Doc:         docPage,
		Crumbs:      crumbs(translate, title, carried),
		Status:      status,
		Stylesheets: []string{stylesheet},
	}
}

// -------------------------------------------------------------- pages ----

// load reads the {id} of the route with its counts; a missing rule renders
// the localised "not found" page.
func (h *Handlers) load(w http.ResponseWriter, r *http.Request) (entry, bool) {
	id, parseErr := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

	var rule model.Rule

	err := h.db.Get(&rule, "id=?", id).Run()
	if parseErr == nil && err == nil {
		var counts map[int64]*entry
		if counts, err = h.counts(); err == nil {
			return entryOf(&rule, counts), true
		}
	}

	if err != nil && !database.IsNotFound(err) {
		h.logger.Errorf("failed to retrieve the transfer rule %d: %v", id, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return entry{}, false
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("rules.not_found.title")

	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, http.StatusNotFound),
		means.NotFoundView(means.NotFoundData{
			Title:     title,
			Text:      translate("rules.not_found.text"),
			Icon:      "list",
			BackLabel: translate("rules.back"),
			BackURL:   listURL(carried),
		}))

	return entry{}, false
}

// tabs is the tab strip of the sheet: Details, Tasks and Access; current is
// the tab shown.
func tabs(translate gui.Translator, e *entry, current string) []means.Tab {
	tasks, access := uint64(e.tasks()), uint64(e.access) //nolint:gosec // counts

	return []means.Tab{
		{Label: translate("rules.tab.details"), URL: ruleURL(e.rule.ID, ""), Current: current == tabDetails},
		{
			Label: translate("rules.tab.tasks"), URL: tasksTabURL(e.rule.ID), Count: tasks, HasCount: true,
			Current: current == tabTasks,
		},
		{
			Label: translate("rules.tab.access"), URL: accessURL(e.rule.ID, ""), Count: access, HasCount: true,
			Current: current == tabAccess,
		},
	}
}

// folder is a directory of the rule in reading: its value, or what the
// gateway uses when it is empty.
type folder struct {
	Label string
	Value string
	Empty string
}

type detailData struct {
	Title      string
	Name       string
	Direction  string
	DirIcon    string
	Path       string
	PathHint   string
	Comment    string
	Folders    []folder
	Access     string
	Restricted bool
	Tabs       []means.Tab
	BackURL    string
	EditURL    string
	Actions    []gui.Action
}

// ruleTitle is the title of the sheet of a rule, which says its direction
// (Driss, 2026-10-23: the direction was not visible enough).
func ruleTitle(translate gui.Translator, rule *model.Rule) string {
	if rule.IsSend {
		return translate("rules.detail.title_send", "name", rule.Name)
	}

	return translate("rules.detail.title_receive", "name", rule.Name)
}

// folders are the three directories of the rule, each with what an empty one
// means.
func folders(translate gui.Translator, rule *model.Rule) []folder {
	return []folder{
		{Label: translate("rules.dir.local"), Value: rule.LocalDir, Empty: translate("rules.dir.local_empty")},
		{Label: translate("rules.dir.remote"), Value: rule.RemoteDir, Empty: translate("rules.dir.remote_empty")},
		{Label: translate("rules.dir.tmp"), Value: rule.TmpLocalRcvDir, Empty: translate("rules.dir.tmp_empty")},
	}
}

// Detail serves GET /webui/rules/{id}: the rule in reading, tab Details.
func (h *Handlers) Detail(w http.ResponseWriter, r *http.Request) {
	item, ok := h.load(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	user := rightsOf(r)

	data := detailData{
		Title:      ruleTitle(translate, item.rule),
		Name:       item.rule.Name,
		Direction:  item.direction(translate),
		DirIcon:    directionIcon(item.rule.IsSend),
		Path:       item.rule.Path,
		PathHint:   translate("rules.detail.path_hint"),
		Comment:    item.rule.Comment,
		Folders:    folders(translate, item.rule),
		Access:     item.accessText(translate),
		Restricted: item.access > 0,
		Tabs:       tabs(translate, &item, tabDetails),
		BackURL:    listURL(carried),
		Actions:    actions(translate, &item, user, true, carried),
	}

	if user.Write {
		data.EditURL = withRet(ruleURL(item.rule.ID, "/edit"), carried)
	}

	screen := h.screen(translate, data.Title, carried, 0)
	screen.Notices = notices(translate, r.URL.Query())
	h.shell.Render(w, h.logger, r, screen, detailView(data))
}

// holds is what keeps a rule from being deleted: the transfers that still
// name it (whatever their state) and the file watchers that use it.
type holds struct {
	Transfers int
	Watchers  []string
}

func (b *holds) any() bool { return b.Transfers > 0 || len(b.Watchers) > 0 }

func (h *Handlers) holdsOf(rule *model.Rule) (holds, error) {
	n, err := h.db.Count(&model.Transfer{}).Where("rule_id=?", rule.ID).Run()
	if err != nil {
		return holds{}, err //nolint:wrapcheck // logged by the caller with its context
	}

	var watchers model.FileWatchers
	if wErr := h.db.Select(&watchers).Where("rule_id=?", rule.ID).Run(); wErr != nil {
		return holds{}, wErr //nolint:wrapcheck // logged by the caller with its context
	}

	b := holds{Transfers: int(n)} //nolint:gosec // a count of transfers
	for _, fw := range watchers {
		b.Watchers = append(b.Watchers, fw.Flow)
	}

	slices.Sort(b.Watchers)

	return b, nil
}

type deleteData struct {
	Title        string
	Name         string
	Direction    string
	Path         string
	Gone         string // what goes with the rule: its tasks and its access entries
	Held         bool
	Transfers    string
	TransfersURL string
	Watchers     []string
	Refused      string
	CancelURL    string
	Action       string
}

// DeleteForm serves GET /webui/rules/{id}/delete: the confirmation page; for
// a rule that transfers or file watchers name, the page that says what to
// change first.
func (h *Handlers) DeleteForm(w http.ResponseWriter, r *http.Request) {
	item, ok := h.load(w, r)
	if !ok {
		return
	}

	held, err := h.holdsOf(item.rule)
	if err != nil {
		h.logger.Errorf("failed to read what uses the transfer rule %q: %v", item.rule.Name, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	h.renderDelete(w, r, &item, held, "", 0)
}

func (h *Handlers) renderDelete(w http.ResponseWriter, r *http.Request, item *entry, held holds, refused string,
	status int,
) {
	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("rules.delete.title", "name", item.rule.Name)

	data := deleteData{
		Title:     title,
		Name:      item.rule.Name,
		Direction: item.direction(translate),
		Path:      item.rule.Path,
		Gone:      translate("rules.delete.gone", "tasks", item.tasks(), "access", item.access),
		Held:      held.any(),
		Watchers:  held.Watchers,
		Refused:   refused,
		CancelURL: withRet(ruleURL(item.rule.ID, ""), carried),
		Action:    withRet(ruleURL(item.rule.ID, "/delete"), carried),
	}

	if held.Transfers > 0 {
		data.Transfers = translate("rules.delete.transfers", "n", held.Transfers)
		if rightsOf(r).Transfers {
			data.TransfersURL = transfersURL(item.rule)
		}
	}

	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, status), deleteView(data))
}

// Delete serves POST /webui/rules/{id}/delete. A rule that transfers or file
// watchers name is not deleted (the product refuses it): the page says what
// to change first (409). Its tasks and its access entries go with it.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	item, ok := h.load(w, r)
	if !ok {
		return
	}

	held, err := h.holdsOf(item.rule)
	if err != nil {
		h.logger.Errorf("failed to read what uses the transfer rule %q: %v", item.rule.Name, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	if held.any() {
		h.renderDelete(w, r, &item, held, "", http.StatusConflict)

		return
	}

	if delErr := h.db.Delete(item.rule).Run(); delErr != nil {
		if common.Refused(delErr) {
			h.renderDelete(w, r, &item, held, delErr.Error(), http.StatusUnprocessableEntity)

			return
		}

		h.logger.Errorf("failed to delete the transfer rule %q: %v", item.rule.Name, delErr)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	listSpec.RedirectList(w, r, ret(r), url.Values{doneParam: {doneDeleted}, nameParam: {item.rule.Name}})
}

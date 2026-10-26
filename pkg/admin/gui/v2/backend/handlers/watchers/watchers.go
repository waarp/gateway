// Package watchers serves the "Partner watchers" screen of the web interface
// (Treatments): the list of the watchers (searched, filtered by partner and
// by state, sorted and paged), each with the state of its service, the pages
// that create, edit and delete one, and the actions that start or stop a
// watcher or run it once.
//
// A watcher polls a folder of a remote partner at a regular interval, with a
// remote account and a local client whose protocol lists files (SFTP, FTP,
// R66, WebDAV), and creates a receive transfer, under a receive rule, for
// every file that matches its pattern; the files already transferred are
// skipped unless asked. Each watcher is a service of the gateway, started
// with it unless disabled. The state of the service lives in the registry of
// the process, not in the database nor in the REST API: the screen reads it
// there. Until this screen, the watchers were managed through the REST API
// and the command line only; the rights are the API's, those of the servers.
package watchers

import (
	"cmp"
	"context"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/filewatcher"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	// The stylesheet of the screen, under the web interface.
	stylesheet = "/static_v2/watchers/watchers.css"

	docPage  = "user/webui/surveillances"
	pagePath = "/watchers"

	// The cookie that keeps the state of the list for the browser session.
	stateCookie = "webui_watchers_list"

	doneParam   = liststate.DoneParam
	failedParam = liststate.FailedParam
	nameParam   = liststate.NameParam
	reasonParam = liststate.ReasonParam
	dirDesc     = liststate.DirDesc
	// The parameter that carries how many transfers a run created.
	countParam = "count"

	sortName    = "name"
	sortPartner = "partner"
	sortRule    = "rule"

	doneCreated = liststate.DoneCreated
	doneUpdated = liststate.DoneUpdated
	doneDeleted = liststate.DoneDeleted
	doneStarted = "started"
	doneStopped = "stopped"
	doneFired   = "fired"
	failedFire  = "fire"

	// The filters of the list: the partners, the states.
	partnerParam = "partner"
	stateParam   = "state"

	// The states of a watcher, as the filter and the status name them.
	stateRunning = "running"
	stateStopped = "stopped"
	stateError   = "error"
	stateOK      = "ok"
	stateOff     = "off"
	stateKO      = "ko"

	// Prefixes of the catalogue keys chosen at run time.
	sortKeyPrefix  = "watchers.sort."
	toastKeyPrefix = "watchers.toast."
	stateKeyPrefix = "watchers.state."
)

//nolint:gochecknoglobals // reference data of the screen
var (
	sortKeys  = []string{sortName, sortPartner, sortRule}
	doneKinds = []string{doneCreated, doneUpdated, doneDeleted, doneStarted, doneStopped}
	states    = []string{stateRunning, stateStopped, stateError}

	counter = liststate.Counter{
		One: "watchers.count_one", Many: "watchers.count",
		ShownOne: "watchers.count_shown_one", ShownMany: "watchers.count_shown",
	}

	listSpec = liststate.Spec{
		Path: pagePath, Cookie: stateCookie, Sorts: sortKeys, Sizes: liststate.DefaultSizes,
		ParseFilters: parseFilter, FilterParams: []string{partnerParam, stateParam},
	}
)

// registry is what the screen asks of the services of the watchers: the
// registry of the gateway, or a fake in the tests. A watcher without a
// service yet (created by another node, or before this process started it)
// gets one, stopped, when it is first asked about.
type registry interface {
	// State is the state of the service of the watcher and, in error, why.
	State(fw *model.FileWatcher) (code, reason string)
	// Start starts the service, or starts it again on its new configuration.
	Start(ctx context.Context, fw *model.FileWatcher) error
	// Stop stops the service; true when it was running.
	Stop(ctx context.Context, fw *model.FileWatcher) (bool, error)
	// Remove stops the service and forgets it.
	Remove(ctx context.Context, fw *model.FileWatcher) error
	// Fire runs the watcher once, now.
	Fire(fw *model.FileWatcher) error
}

// gatewayRegistry is the registry of the gateway (filewatcher.Filewatchers).
type gatewayRegistry struct{ db *database.DB }

func (g gatewayRegistry) service(fw *model.FileWatcher) *filewatcher.Service {
	if svc, ok := filewatcher.Filewatchers.Get(fw); ok {
		return svc
	}

	svc := filewatcher.NewFilewatcher(g.db, fw)
	filewatcher.Filewatchers.Add(fw, svc)

	return svc
}

func (g gatewayRegistry) State(fw *model.FileWatcher) (code, reason string) {
	state, reason := g.service(fw).State()

	return state.String(), reason
}

func (g gatewayRegistry) Start(ctx context.Context, fw *model.FileWatcher) error {
	g.service(fw)

	return filewatcher.Filewatchers.Restart(ctx, fw) //nolint:wrapcheck // shown as is in the notification
}

func (g gatewayRegistry) Stop(ctx context.Context, fw *model.FileWatcher) (bool, error) {
	g.service(fw)

	return filewatcher.Filewatchers.Stop(ctx, fw) //nolint:wrapcheck // shown as is in the notification
}

func (g gatewayRegistry) Remove(ctx context.Context, fw *model.FileWatcher) error {
	g.service(fw)

	return filewatcher.Filewatchers.Remove(ctx, fw) //nolint:wrapcheck // logged by the caller
}

func (g gatewayRegistry) Fire(fw *model.FileWatcher) error {
	return g.service(fw).FireOnce(fw) //nolint:wrapcheck // shown as is in the notification
}

// Handlers serves the pages and the actions of the screen.
type Handlers struct {
	db       *database.DB
	logger   *log.Logger
	shell    *common.Shell
	services registry
}

func New(db *database.DB, logger *log.Logger) *Handlers {
	return &Handlers{db: db, logger: logger, shell: common.NewShell(), services: gatewayRegistry{db}}
}

// rights of the current user on the watchers: those of the servers, as the
// REST API asks them.
type rights struct{ Write, Delete bool }

func rightsOf(r *http.Request) rights {
	perms := common.GetUser(r).Permissions

	return rights{
		Write:  perms.HasPermission(model.PermServersWrite),
		Delete: perms.HasPermission(model.PermServersDelete),
	}
}

// entry is a watcher with its partner and the state of its service.
type entry struct {
	fw      *model.FileWatcher
	partner *model.RemoteAgent
	state   string // one of states
	reason  string // in error, why
}

func (e *entry) partnerName() string {
	if e.partner == nil {
		return ""
	}

	return e.partner.Name
}

// stateOf names the state of a service for the filter and the status.
func stateOf(code string) string {
	switch code {
	case utils.StateRunning.String():
		return stateRunning
	case utils.StateError.String():
		return stateError
	default:
		return stateStopped
	}
}

// status is the state as the kit shows it: shape, colour and label.
func status(translate gui.Translator, state string) gui.Status {
	switch state {
	case stateRunning:
		return gui.Status{State: stateOK, Label: translate(stateKeyPrefix + stateRunning)}
	case stateError:
		return gui.Status{State: stateKO, Label: translate(stateKeyPrefix + stateError)}
	default:
		return gui.Status{State: stateOff, Label: translate(stateKeyPrefix + stateStopped)}
	}
}

// filter is the state of the filters of the list: the partners, the states.
type filter struct {
	Partners []string
	States   []string
}

func parseFilter(v url.Values) liststate.Filters {
	f := &filter{}

	for _, name := range v[partnerParam] {
		if name = strings.TrimSpace(name); name != "" && !slices.Contains(f.Partners, name) {
			f.Partners = append(f.Partners, name)
		}
	}

	for _, state := range v[stateParam] {
		if slices.Contains(states, state) && !slices.Contains(f.States, state) {
			f.States = append(f.States, state)
		}
	}

	return f
}

func (f *filter) Values() url.Values {
	v := url.Values{}
	for _, name := range f.Partners {
		v.Add(partnerParam, name)
	}

	for _, state := range f.States {
		v.Add(stateParam, state)
	}

	return v
}

func (f *filter) Count() int { return len(f.Partners) + len(f.States) }

func (f *filter) keeps(e *entry) bool {
	if len(f.Partners) > 0 && !slices.Contains(f.Partners, e.partnerName()) {
		return false
	}

	return len(f.States) == 0 || slices.Contains(f.States, e.state)
}

// box is a checkbox of a filter.
type box struct {
	Name    string
	Value   string
	Label   string
	Checked bool
}

// partnerBoxes offers the partners that have a watcher.
func (f *filter) partnerBoxes(all []entry) []box {
	names := make([]string, 0, len(all))
	for i := range all {
		if name := all[i].partnerName(); name != "" && !slices.Contains(names, name) {
			names = append(names, name)
		}
	}

	slices.Sort(names)

	out := make([]box, 0, len(names))
	for _, name := range names {
		out = append(out, box{Name: partnerParam, Value: name, Label: name, Checked: slices.Contains(f.Partners, name)})
	}

	return out
}

func (f *filter) stateBoxes(translate gui.Translator) []box {
	out := make([]box, 0, len(states))
	for _, state := range states {
		out = append(out, box{
			Name: stateParam, Value: state, Label: translate(stateKeyPrefix + state), Checked: slices.Contains(f.States, state),
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

func watcherURL(id int64, suffix string) string {
	return common.WebuiURL(pagePath + "/" + strconv.FormatInt(id, 10) + suffix)
}

// all reads the watchers, by name, each with its partner and the state of
// its service.
func (h *Handlers) all() ([]entry, error) {
	var list model.FileWatchers
	// Eager: the account, the client and the rule come with the watcher.
	if err := h.db.Select(&list).OrderBy("flow", true).Eager().Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	partners, err := h.partners()
	if err != nil {
		return nil, err
	}

	entries := make([]entry, 0, len(list))
	for _, fw := range list {
		entries = append(entries, h.entryOf(fw, partners))
	}

	return entries, nil
}

// partners reads the remote partners, by identifier.
func (h *Handlers) partners() (map[int64]*model.RemoteAgent, error) {
	var list model.RemoteAgents
	if err := h.db.Select(&list).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	byID := make(map[int64]*model.RemoteAgent, len(list))
	for _, p := range list {
		byID[p.ID] = p
	}

	return byID, nil
}

func (h *Handlers) entryOf(fw *model.FileWatcher, partners map[int64]*model.RemoteAgent) entry {
	code, reason := h.services.State(fw)

	return entry{fw: fw, partner: partners[fw.RemoteAccount.RemoteAgentID], state: stateOf(code), reason: reason}
}

// intervalText says an interval in days, hours and minutes ("1 h 15 min").
func intervalText(translate gui.Translator, d time.Duration) string {
	if d <= 0 {
		return translate("watchers.interval.never")
	}

	const day = 24 * time.Hour

	parts := make([]string, 0, 3) //nolint:mnd // days, hours, minutes

	if days := int(d / day); days > 0 {
		parts = append(parts, translate("watchers.interval.days", "n", days))
		d -= time.Duration(days) * day
	}

	if hours := int(d / time.Hour); hours > 0 {
		parts = append(parts, translate("watchers.interval.hours", "n", hours))
		d -= time.Duration(hours) * time.Hour
	}

	if minutes := int(d / time.Minute); minutes > 0 || len(parts) == 0 {
		parts = append(parts, translate("watchers.interval.minutes", "n", minutes))
	}

	return strings.Join(parts, " ")
}

type row struct {
	ID       int64
	Name     string
	URL      string
	Partner  string
	Account  string
	Client   string
	Rule     string
	Pattern  string
	Interval string
	State    gui.Status
	Reason   string
	Actions  []gui.Action
}

type listData struct {
	AddURL       string
	Action       string
	Base         string
	Q            string
	Names        []string
	Partners     []box
	States       []box
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

// List serves GET /webui/watchers.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	fragment := liststate.IsListFragment(r)
	if !fragment && listSpec.Restore(w, r) {
		return
	}

	translate := h.shell.Translator(r)
	query := parseListQuery(r.URL.Query())
	screen := h.screen(translate, translate("watchers.title"), "", 0)
	screen.Crumbs = crumbs(translate, "", "")
	screen.Notices = notices(translate, r.URL.Query())
	screen.Head = gui.LiveAssets()

	entries, err := h.all()
	if err != nil {
		h.logger.Errorf("failed to list the partner watchers: %v", err)

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

// buildList computes the list page: search (the name), filters (partner,
// state), sort, page, toolbar state, row actions according to the rights.
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
		Partners:     active.partnerBoxes(all),
		States:       active.stateBoxes(translate),
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
		names = append(names, e.fw.Flow)

		if !query.Search.Matches(e.fw.Flow) || !active.keeps(e) {
			continue
		}

		kept = append(kept, *e)
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
		data.NoMatch = true

		if query.Q != "" {
			withoutQ := *query
			withoutQ.Q = ""
			data.NoMatchText = translate("watchers.none.title", "query", query.Q)
			data.NoMatchURL, data.NoMatchLink = listSpec.ExplicitListURL(withoutQ.Values()), translate("search.clear")
		} else {
			data.NoMatchText = translate("watchers.none.filters")
			data.NoMatchURL, data.NoMatchLink = listSpec.ResetURL(), translate("filters.reset")
		}

		return data
	}

	bounds := query.Bounds(len(kept))
	carried := query.Encode(bounds.Page)
	data.Caption = translate("watchers.caption",
		"page", bounds.Page, "pages", bounds.Pages, "sort", translate(sortKeyPrefix+query.Sort))

	for i := range kept[bounds.From:bounds.To] {
		e := &kept[bounds.From+i]
		line := row{
			ID:       e.fw.ID,
			Name:     e.fw.Flow,
			Partner:  e.partnerName(),
			Account:  e.fw.RemoteAccount.Login,
			Client:   e.fw.Client.Name,
			Rule:     e.fw.Rule.Name,
			Pattern:  e.fw.Pattern,
			Interval: intervalText(translate, e.fw.Interval),
			State:    status(translate, e.state),
			Reason:   e.reason,
			Actions:  actions(translate, e, user, false, carried),
			URL:      withRet(watcherURL(e.fw.ID, ""), carried),
		}

		data.Rows = append(data.Rows, line)
	}

	data.Sizes = liststate.DefaultSizes
	data.PagerHidden = query.PagerHidden()
	info := translate("watchers.pager", "from", bounds.From+1, "to", bounds.To, "total", len(kept))
	data.Pager = listSpec.Pager(query, info, bounds)

	return data
}

// sortEntries sorts by the asked column; ties keep the name order
// (regardless of case).
func sortEntries(entries []entry, key, dir string) {
	byName := func(a, b *entry) int {
		return strings.Compare(strings.ToLower(a.fw.Flow), strings.ToLower(b.fw.Flow))
	}

	slices.SortStableFunc(entries, func(left, right entry) int {
		var order int

		switch key {
		case sortPartner:
			order = cmp.Or(strings.Compare(left.partnerName(), right.partnerName()),
				strings.Compare(left.fw.RemoteAccount.Login, right.fw.RemoteAccount.Login))
		case sortRule:
			order = strings.Compare(left.fw.Rule.Name, right.fw.Rule.Name)
		default:
			order = byName(&left, &right)
		}

		if dir == dirDesc {
			order = -order
		}

		return cmp.Or(order, byName(&left, &right))
	})
}

// actions: the menu of a row, or of the sheet: the sheet (from a row),
// start or stop (according to the state), run once, edit (from a row: the
// sheet has its button), delete.
func actions(translate gui.Translator, item *entry, user rights, onSheet bool, carried string) []gui.Action {
	var acts []gui.Action

	if !onSheet {
		acts = append(acts, gui.Action{
			Icon: "eye", Label: translate("watchers.action.detail"), URL: withRet(watcherURL(item.fw.ID, ""), carried),
		})
	}

	if user.Write {
		if item.state == stateRunning {
			acts = append(acts, gui.Action{
				Icon: "stop", Label: translate("watchers.action.stop"), Action: withRet(watcherURL(item.fw.ID, "/stop"), carried),
			})
		} else {
			acts = append(acts, gui.Action{
				Icon: "play", Label: translate("watchers.action.start"), Action: withRet(watcherURL(item.fw.ID, "/start"), carried),
			})
		}

		acts = append(acts, gui.Action{
			Icon: "rotate", Label: translate("watchers.action.fire"), Action: withRet(watcherURL(item.fw.ID, "/fire"), carried),
		})

		if !onSheet {
			acts = append(acts, gui.Action{
				Icon: "pen", Label: translate("watchers.action.edit"), URL: withRet(watcherURL(item.fw.ID, "/edit"), carried),
			})
		}
	}

	if user.Delete {
		acts = append(acts, gui.Action{
			Icon: "trash", Label: translate("watchers.action.delete"), URL: withRet(watcherURL(item.fw.ID, "/delete"), carried),
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
		return append(c, gui.Crumb{Label: translate("watchers.title")})
	}

	return append(c, gui.Crumb{Label: translate("watchers.title"), URL: listURL(carried)}, gui.Crumb{Label: last})
}

// notices turns the result of the previous action into a notification: a
// watcher created, edited, deleted, started or stopped; a run with how many
// transfers it created, or its failure with the message of the product.
func notices(translate gui.Translator, values url.Values) []gui.Notice {
	name := values.Get(nameParam)

	switch done := values.Get(doneParam); {
	case slices.Contains(doneKinds, done):
		return []gui.Notice{{Kind: stateOK, Title: translate(toastKeyPrefix+done, "name", name)}}
	case done == doneFired:
		count, err := strconv.Atoi(values.Get(countParam))
		if err != nil {
			count = 0
		}

		switch count {
		case 0:
			return []gui.Notice{{Kind: stateOK, Title: translate("watchers.toast.fired_none", "name", name)}}
		case 1:
			return []gui.Notice{{Kind: stateOK, Title: translate("watchers.toast.fired_one", "name", name)}}
		default:
			return []gui.Notice{{Kind: stateOK, Title: translate("watchers.toast.fired", "name", name, "count", count)}}
		}
	}

	if values.Get(failedParam) == failedFire {
		return []gui.Notice{{
			Kind: stateKO, Title: translate("watchers.toast.fire_failed", "name", name), Text: values.Get(reasonParam),
		}}
	}

	return nil
}

// screen is a page of the screen: the chrome around the data.
func (h *Handlers) screen(translate gui.Translator, title, carried string, status int) common.Screen {
	return common.Screen{
		Title:       title,
		Section:     constants.SidebarSectionTreatments,
		Menu:        constants.SidebarMenuWatchers,
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

// watcher reads the {id} of the route, with its partner and its state; a
// missing watcher renders the localised "not found" page.
func (h *Handlers) watcher(w http.ResponseWriter, r *http.Request) (entry, bool) {
	id, parseErr := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

	var fw model.FileWatcher

	err := h.db.Get(&fw, "id=?", id).Eager().Run()
	if parseErr == nil && err == nil {
		var partners map[int64]*model.RemoteAgent
		if partners, err = h.partners(); err == nil {
			return h.entryOf(&fw, partners), true
		}
	}

	if err != nil && !database.IsNotFound(err) {
		h.logger.Errorf("failed to retrieve the partner watcher %d: %v", id, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return entry{}, false
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("watchers.not_found.title")

	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, http.StatusNotFound), notFoundView(notFoundData{
		Title:     title,
		Text:      translate("watchers.not_found.text"),
		BackLabel: translate("watchers.back"),
		BackURL:   listURL(carried),
	}))

	return entry{}, false
}

type deleteData struct {
	Title     string
	Name      string
	Partner   string
	Rule      string
	Interval  string
	Pattern   string
	CancelURL string
	Action    string
}

// DeleteForm serves GET /webui/watchers/{id}/delete: the confirmation page.
func (h *Handlers) DeleteForm(w http.ResponseWriter, r *http.Request) {
	item, ok := h.watcher(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("watchers.delete.title", "name", item.fw.Flow)

	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, 0), deleteView(deleteData{
		Title:     title,
		Name:      item.fw.Flow,
		Partner:   item.partnerName() + " · " + item.fw.RemoteAccount.Login,
		Rule:      item.fw.Rule.Name,
		Interval:  intervalText(translate, item.fw.Interval),
		Pattern:   item.fw.Pattern,
		CancelURL: listURL(carried),
		Action:    withRet(watcherURL(item.fw.ID, "/delete"), carried),
	}))
}

// Delete serves POST /webui/watchers/{id}/delete: the watcher goes, its
// service with it, as the REST API does.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	item, ok := h.watcher(w, r)
	if !ok {
		return
	}

	if err := h.db.Delete(item.fw).Run(); err != nil {
		h.logger.Errorf("failed to delete the partner watcher %q: %v", item.fw.Flow, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	if err := h.services.Remove(r.Context(), item.fw); err != nil {
		h.logger.Warningf("failed to stop the service of the deleted partner watcher %q: %v", item.fw.Flow, err)
	}

	listSpec.RedirectList(w, r, ret(r), url.Values{doneParam: {doneDeleted}, nameParam: {item.fw.Flow}})
}

// Start serves POST /webui/watchers/{id}/start: the service starts, or
// starts again on its configuration.
func (h *Handlers) Start(w http.ResponseWriter, r *http.Request) {
	item, ok := h.watcher(w, r)
	if !ok {
		return
	}

	if err := h.services.Start(r.Context(), item.fw); err != nil {
		h.logger.Errorf("failed to start the partner watcher %q: %v", item.fw.Flow, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	listSpec.RedirectList(w, r, ret(r), url.Values{doneParam: {doneStarted}, nameParam: {item.fw.Flow}})
}

// Stop serves POST /webui/watchers/{id}/stop.
func (h *Handlers) Stop(w http.ResponseWriter, r *http.Request) {
	item, ok := h.watcher(w, r)
	if !ok {
		return
	}

	if _, err := h.services.Stop(r.Context(), item.fw); err != nil {
		h.logger.Errorf("failed to stop the partner watcher %q: %v", item.fw.Flow, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	listSpec.RedirectList(w, r, ret(r), url.Values{doneParam: {doneStopped}, nameParam: {item.fw.Flow}})
}

// Fire serves POST /webui/watchers/{id}/fire: the watcher runs once, now;
// the notification says how many transfers the run created, or why it
// failed, as the product says it (a partner that does not answer, a folder
// that cannot be listed).
func (h *Handlers) Fire(w http.ResponseWriter, r *http.Request) {
	item, ok := h.watcher(w, r)
	if !ok {
		return
	}

	before, err := h.transfersOf(item.fw)
	if err == nil {
		err = h.services.Fire(item.fw)
	}

	if err != nil {
		h.logger.Warningf("the run of the partner watcher %q failed: %v", item.fw.Flow, err)
		listSpec.RedirectList(w, r, ret(r),
			url.Values{failedParam: {failedFire}, nameParam: {item.fw.Flow}, reasonParam: {err.Error()}})

		return
	}

	after, err := h.transfersOf(item.fw)
	if err != nil {
		h.logger.Errorf("failed to count the transfers of the partner watcher %q: %v", item.fw.Flow, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	listSpec.RedirectList(w, r, ret(r), url.Values{
		doneParam: {doneFired}, nameParam: {item.fw.Flow}, countParam: {strconv.FormatUint(after-before, 10)},
	})
}

// transfersOf counts the transfers of the rule and the account of the
// watcher: what a run adds to.
func (h *Handlers) transfersOf(fw *model.FileWatcher) (uint64, error) {
	//nolint:wrapcheck // logged by the caller
	return h.db.Count(&model.Transfer{}).Where("rule_id=? AND remote_account_id=?", fw.RuleID, fw.RemoteAccountID).Run()
}

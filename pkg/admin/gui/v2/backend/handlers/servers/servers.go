// Package servers serves the "Local servers" screen of the web interface: the
// list of the local servers of the gateway (searched, filtered, sorted and
// paged), the details of a server, the pages that create, edit and delete one,
// and the actions that start and stop one.
//
// It is the first screen of the reworked interface that writes: it installs
// the Post/Redirect/Get model (a successful action redirects to the list, as
// the user left it, with a notification), the form rendered again with its
// errors (422) and the confirmation page of a deletion.
package servers

import (
	"cmp"
	"context"
	"fmt"
	"maps"
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
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	listTemplateFile     = "servers/servers.gohtml"
	detailTemplateFile   = "servers/detail.gohtml"
	formTemplateFile     = "servers/form.gohtml"
	deleteTemplateFile   = "servers/delete.gohtml"
	notFoundTemplateFile = "servers/not-found.gohtml"
	docPage              = "user/webui/serveurs-locaux"
	pagePath             = "/servers"

	// Historic pages the "⋯" menu leads to until the server sheet exists
	// (next screen), and the page of the running transfers.
	authPath      = "/server_authentication"
	accountsPath  = "/local_account_management"
	transfersPath = "/transfer_monitoring"
	serverIDParam = "serverID"

	// The state of the list: search, protocols, sort, page and rows per page.
	// The action pages carry it (retParam) so that coming back keeps it.
	queryParam    = "q"
	protocolParam = "protocol"
	sortParam     = "sort"
	dirParam      = "dir"
	pageParam     = "page"
	sizeParam     = "size"
	retParam      = "ret"
	// resetParam makes a request explicit without any criterion: it clears the
	// state kept for the session ("Reset", "Clear the search").
	resetParam = "reset"

	// stateCookie keeps the state of the list for the browser session, so that
	// coming back through the sidebar finds the list as it was left (Driss,
	// 2026-10-19). An explicit address always wins over it.
	stateCookie = "webui_servers_list"

	// The result of the previous action, carried by the redirection.
	doneParam   = "done"
	failedParam = "failed"
	nameParam   = "name"
	reasonParam = "reason"

	sortName     = "name"
	sortState    = "state"
	sortProtocol = "protocol"
	sortAddress  = "address"
	dirAsc       = "asc"
	dirDesc      = "desc"

	doneCreated = "created"
	doneUpdated = "updated"
	doneDeleted = "deleted"
	doneStarted = "started"
	doneStopped = "stopped"
	failedStart = "start"
	failedStop  = "stop"

	// States of a gui.Status: the shape and colour of a status.
	stateOK      = "ok"
	stateKO      = "ko"
	stateOffline = "off"
	stateUnknown = "warn"

	// Prefixes of the catalogue keys chosen at run time: the name of a sorted
	// column ("servers.sort.name") and the result of an action
	// ("servers.toast.created").
	sortKeyPrefix  = "servers.sort."
	toastKeyPrefix = "servers.toast."
)

//nolint:gochecknoglobals // reference data of the screen
var (
	listParams = []string{queryParam, protocolParam, sortParam, dirParam, pageParam, sizeParam, resetParam}
	sortKeys   = []string{sortName, sortState, sortProtocol, sortAddress}
	pageSizes  = []int{10, 25, 50, 100} // rows per page, the first one by default
	doneKinds  = []string{doneCreated, doneUpdated, doneDeleted, doneStarted, doneStopped}
)

// Control drives the service of a server. On the gateway it is backed by the
// service lists (GatewayControl); the tests use stubs.
type Control struct {
	Status func(server *model.LocalAgent) (utils.StateCode, string)
	Add    func(db *database.DB, server *model.LocalAgent) error
	Remove func(ctx context.Context, db *database.DB, server *model.LocalAgent) error
	Start  func(ctx context.Context, server *model.LocalAgent) error
	Stop   func(ctx context.Context, server *model.LocalAgent) error
}

// GatewayControl is the Control of the running gateway.
func GatewayControl() Control {
	return Control{
		Status: internal.GetServerStatus,
		Add:    internal.AddServer,
		Remove: internal.RemoveServer,
		Start:  internal.RestartServer,
		Stop:   internal.StopServer,
	}
}

// Handlers serves the pages and the actions of the screen. Each page has its
// own shell, as each one defines its own "page" template.
type Handlers struct {
	db     *database.DB
	logger *log.Logger
	ctl    Control

	list, detail, form, del, notFound *common.Shell
}

func New(db *database.DB, logger *log.Logger, ctl Control) *Handlers {
	return &Handlers{
		db:       db,
		logger:   logger,
		ctl:      ctl,
		list:     common.NewShell(listTemplateFile),
		detail:   common.NewShell(detailTemplateFile),
		form:     common.NewShell(formTemplateFile),
		del:      common.NewShell(deleteTemplateFile),
		notFound: common.NewShell(notFoundTemplateFile),
	}
}

// rights of the current user on the servers: reading is implied by reaching
// the screen; writing and deleting are checked again by the router on each
// action. The templates hide what the rights forbid.
type rights struct{ Write, Delete bool }

func rightsOf(r *http.Request) rights {
	perms := common.GetUser(r).Permissions

	return rights{
		Write:  perms.HasPermission(model.PermServersWrite),
		Delete: perms.HasPermission(model.PermServersDelete),
	}
}

// kv is a row of a key/value list.
type kv struct {
	Label string
	Value string
	Mono  bool
}

// entry is a local server with its protocol and its running state.
type entry struct {
	agent  *model.LocalAgent
	proto  protocol
	state  utils.StateCode
	reason string
}

func (h *Handlers) entryOf(agent *model.LocalAgent) entry {
	state, reason := h.ctl.Status(agent)

	return entry{agent: agent, proto: protocolOf(agent.Protocol), state: state, reason: reason}
}

func (e entry) running() bool { return e.state == utils.StateRunning }

// address shows the listening address; a server without a host listens on
// every interface, shown as "*:port".
func (e entry) address() string {
	if e.agent.Address.Host == "" {
		return "*:" + strconv.FormatUint(uint64(e.agent.Address.Port), 10)
	}

	return e.agent.Address.String()
}

// status turns the state code of a server into a shape, a colour and a label;
// a code outside the three known ones reads "unknown".
func (e entry) status(translate gui.Translator) gui.Status {
	switch e.state {
	case utils.StateRunning:
		return gui.Status{State: stateOK, Label: translate("state.running")}
	case utils.StateOffline:
		return gui.Status{State: stateOffline, Label: translate("state.offline")}
	case utils.StateError:
		return gui.Status{State: stateKO, Label: translate("state.error")}
	default:
		return gui.Status{State: stateUnknown, Label: translate("state.unknown")}
	}
}

// stateRank orders the states from the healthiest to the worst, as the
// service status screen does.
func (e entry) stateRank() int {
	order := []utils.StateCode{utils.StateRunning, utils.StateOffline, utils.StateError}
	if rank := slices.Index(order, e.state); rank >= 0 {
		return rank
	}

	return len(order)
}

// --------------------------------------------------------------- list ----

// listQuery is the state of the list asked by the URL.
type listQuery struct {
	Q         string
	Protocols []string
	Sort      string
	Dir       string
	Page      int
	Size      int
}

// parseListQuery reads ?q=, ?protocol= (repeated or comma separated),
// ?sort=, ?dir=, ?page= and ?size=. Anything unknown falls back to the
// default; the page is bounded later, once the count is known.
func parseListQuery(v url.Values) listQuery {
	query := listQuery{Q: strings.TrimSpace(v.Get(queryParam)), Sort: sortName, Dir: dirAsc, Page: 1, Size: pageSizes[0]}

	for _, raw := range v[protocolParam] {
		for key := range strings.SplitSeq(raw, ",") {
			if key = strings.TrimSpace(key); key != "" && isProtocol(key) && !slices.Contains(query.Protocols, key) {
				query.Protocols = append(query.Protocols, key)
			}
		}
	}

	if slices.Contains(sortKeys, v.Get(sortParam)) {
		query.Sort = v.Get(sortParam)
	}

	if v.Get(dirParam) == dirDesc {
		query.Dir = dirDesc
	}

	if page, err := strconv.Atoi(v.Get(pageParam)); err == nil && page > 1 {
		query.Page = page
	}

	if size, err := strconv.Atoi(v.Get(sizeParam)); err == nil && slices.Contains(pageSizes, size) {
		query.Size = size
	}

	return query
}

// filterValues carries the search and the filters only: the base of the
// sort links.
func (q listQuery) filterValues() url.Values {
	v := url.Values{}
	if q.Q != "" {
		v.Set(queryParam, q.Q)
	}

	if len(q.Protocols) > 0 {
		v.Set(protocolParam, strings.Join(q.Protocols, ","))
	}

	return v
}

// values carries the search, the filters, the sort and the rows per page
// when they differ from the default; the page is added by withPage.
func (q listQuery) values() url.Values {
	v := q.filterValues()
	if q.Sort != sortName || q.Dir != dirAsc {
		v.Set(sortParam, q.Sort)
		v.Set(dirParam, q.Dir)
	}

	if q.Size != pageSizes[0] {
		v.Set(sizeParam, strconv.Itoa(q.Size))
	}

	return v
}

func (q listQuery) withPage(page int) url.Values {
	v := q.values()
	if page > 1 {
		v.Set(pageParam, strconv.Itoa(page))
	}

	return v
}

// encode is the list state carried by the action pages (?ret=).
func (q listQuery) encode(page int) string { return q.withPage(page).Encode() }

// toolbarHidden is what the toolbar form (search and filters) must carry so
// that a new search keeps the sort and the rows per page.
func (q listQuery) toolbarHidden() map[string]string {
	hidden := map[string]string{}

	for _, key := range []string{sortParam, dirParam, sizeParam} {
		if value := q.values().Get(key); value != "" {
			hidden[key] = value
		}
	}

	return hidden
}

// pagerHidden is what the list foot form must carry besides the page and the
// rows per page it sets itself.
func (q listQuery) pagerHidden() map[string]string {
	hidden := map[string]string{}

	for key, values := range q.values() {
		if key != sizeParam {
			hidden[key] = values[0]
		}
	}

	return hidden
}

// retOf reads a carried list state; unknown or invalid parameters are
// dropped, so that a forged value cannot lead anywhere else than the list.
func retOf(ret string) listQuery {
	values, err := url.ParseQuery(ret)
	if err != nil {
		return parseListQuery(nil)
	}

	return parseListQuery(values)
}

// ret is the list state carried by the request (?ret=), cleaned.
func ret(r *http.Request) string {
	q := retOf(r.URL.Query().Get(retParam))

	return q.encode(q.Page)
}

func listURLOf(v url.Values) string {
	if encoded := v.Encode(); encoded != "" {
		return common.WebuiURL(pagePath) + "?" + encoded
	}

	return common.WebuiURL(pagePath)
}

// listURL is the list as the user left it: the "Back to the list" links and
// the redirections after an action.
func listURL(ret string) string {
	q := retOf(ret)

	return listURLOf(q.withPage(q.Page))
}

// resetURL is the list without any criterion, said explicitly so that the
// state kept for the session is cleared instead of restored.
func resetURL() string { return listURLOf(url.Values{resetParam: {"1"}}) }

// explicitListURL is the list with the given criteria, or the explicit reset
// when there is none left.
func explicitListURL(v url.Values) string {
	if len(v) == 0 {
		return resetURL()
	}

	return listURLOf(v)
}

// hasListParams tells whether the request says anything about the state of
// the list, be it a criterion or the explicit reset.
func hasListParams(params url.Values) bool {
	return slices.ContainsFunc(listParams, params.Has)
}

// keptState reads the state kept for the session; a default or unreadable
// state is no state.
func keptState(r *http.Request) (listQuery, bool) {
	cookie, err := r.Cookie(stateCookie)
	if err != nil {
		return listQuery{}, false
	}

	state, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return listQuery{}, false
	}

	kept := retOf(state)

	return kept, kept.encode(kept.Page) != ""
}

// keepState stores the state of the list for the browser session, or forgets
// it when the state is the default one.
func keepState(w http.ResponseWriter, query listQuery) {
	cookie := &http.Cookie{
		Name:     stateCookie,
		Value:    url.QueryEscape(query.encode(query.Page)),
		Path:     common.WebuiURL(pagePath),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	if cookie.Value == "" {
		cookie.MaxAge = -1
	}

	http.SetCookie(w, cookie)
}

func serverURL(id int64, suffix string) string {
	return common.WebuiURL(pagePath + "/" + strconv.FormatInt(id, 10) + suffix)
}

// withRet appends the carried list state to a URL.
func withRet(u, ret string) string {
	if ret == "" {
		return u
	}

	sep := "?"
	if strings.Contains(u, "?") {
		sep = "&"
	}

	return u + sep + retParam + "=" + url.QueryEscape(ret)
}

func historicURL(path string, id int64) string {
	return common.WebuiURL(path) + "?" + serverIDParam + "=" + strconv.FormatInt(id, 10)
}

type filterBox struct {
	Key     string
	Label   string
	Checked bool
}

type row struct {
	ID       int64
	Name     string
	Status   gui.Status
	Reason   string // raw reason given by the service, never translated
	Protocol string
	Address  string
	Actions  []gui.Action
}

// pager is the list foot, with the fields of "gui/pager".
type pager struct {
	Info    string
	Page    int
	Pages   int
	PrevURL string
	NextURL string
}

type listData struct {
	AddURL       string // empty without the write right: no coloured action at all
	Action       string // toolbar form action: the bare list
	Base         string // list with search and filters: base of the sort links
	Q            string
	Names        []string
	Filters      []filterBox
	FiltersOpen  bool
	FiltersCount int
	ResetURL     string
	Sort         string
	Dir          string
	Hidden       map[string]string // carried by the toolbar form
	Count        string
	Rows         []row
	Caption      string
	Pager        *pager
	PagerHidden  map[string]string
	Sizes        []int // rows-per-page choices, offered as soon as there are rows
	Size         int
	Empty        bool // no server at all
	NoMatch      bool // servers exist, none matches
	NoMatchText  string
	NoMatchURL   string
	NoMatchLink  string
	Error        bool // the servers could not be read
	RetryURL     string
}

// List serves GET /webui/servers.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	if !hasListParams(params) {
		if kept, ok := keptState(r); ok {
			target := kept.withPage(kept.Page)
			for _, name := range []string{doneParam, failedParam, nameParam, reasonParam} {
				if params.Has(name) {
					target.Set(name, params.Get(name))
				}
			}

			http.Redirect(w, r, listURLOf(target), http.StatusSeeOther)

			return
		}
	}

	translate := h.list.Translator(r)
	query := parseListQuery(params)

	if hasListParams(params) {
		keepState(w, query)
	}
	screen := common.Screen{
		Title:   translate("servers.title"),
		Section: constants.SidebarSectionConnection,
		Menu:    constants.SidebarMenuServers,
		Doc:     docPage,
		Crumbs:  crumbs(translate, "", ""),
		Notices: notices(translate, r.URL.Query()),
	}

	agents, err := internal.ListServers(h.db, "name", true, 0, 0)
	if err != nil {
		h.logger.Errorf("failed to list the servers: %v", err)

		screen.Status = http.StatusInternalServerError
		screen.Data = listData{Error: true, RetryURL: listURLOf(query.withPage(query.Page))}
		h.list.Render(w, h.logger, r, screen)

		return
	}

	entries := make([]entry, 0, len(agents))
	for _, agent := range agents {
		entries = append(entries, h.entryOf(agent))
	}

	screen.Data = buildList(translate, entries, query, rightsOf(r))
	h.list.Render(w, h.logger, r, screen)
}

// buildList computes the list page: filter (name prefix, protocols), sort,
// page, toolbar state, row actions according to the rights.
//
//nolint:funlen // one pass over the list, in the order of the page
func buildList(translate gui.Translator, all []entry, query listQuery, user rights) listData {
	data := listData{
		Action:   listURLOf(nil),
		Base:     listURLOf(query.filterValues()),
		Q:        query.Q,
		Sort:     query.Sort,
		Dir:      query.Dir,
		Size:     query.Size,
		ResetURL: resetURL(),
		Hidden:   query.toolbarHidden(),
	}

	if user.Write {
		data.AddURL = withRet(common.WebuiURL(pagePath+"/new"), query.encode(query.Page))
	}

	for _, proto := range protocolList() {
		data.Filters = append(data.Filters,
			filterBox{Key: proto.Key, Label: proto.Label, Checked: slices.Contains(query.Protocols, proto.Key)})
	}

	data.FiltersOpen = len(query.Protocols) > 0
	data.FiltersCount = len(query.Protocols)

	names := make([]string, 0, len(all))
	kept := make([]entry, 0, len(all))
	needle := strings.ToLower(query.Q)

	for _, e := range all {
		names = append(names, e.agent.Name)

		if needle != "" && !strings.HasPrefix(strings.ToLower(e.agent.Name), needle) {
			continue
		}

		if len(query.Protocols) > 0 && !slices.Contains(query.Protocols, e.agent.Protocol) {
			continue
		}

		kept = append(kept, e)
	}

	slices.Sort(names)
	data.Names = names
	sortEntries(kept, query.Sort, query.Dir)

	data.Count = countText(translate, len(all), len(kept))

	switch {
	case len(all) == 0:
		data.Empty = true

		return data
	case len(kept) == 0:
		data.NoMatch = true

		if query.Q != "" {
			withoutQ := query
			withoutQ.Q = ""
			data.NoMatchText = translate("servers.none.title", "query", query.Q)
			data.NoMatchURL, data.NoMatchLink = explicitListURL(withoutQ.values()), translate("search.clear")
		} else {
			data.NoMatchText = translate("servers.none.filters")
			data.NoMatchURL, data.NoMatchLink = resetURL(), translate("filters.reset")
		}

		return data
	}

	pages := (len(kept) + query.Size - 1) / query.Size
	page := min(max(query.Page, 1), pages)
	from, to := (page-1)*query.Size, min(page*query.Size, len(kept))
	carried := query.encode(page)
	data.Caption = translate("servers.caption", "page", page, "pages", pages, "sort", translate(sortKeyPrefix+query.Sort))

	for _, e := range kept[from:to] {
		data.Rows = append(data.Rows, row{
			ID:       e.agent.ID,
			Name:     e.agent.Name,
			Status:   e.status(translate),
			Reason:   e.reason,
			Protocol: e.proto.Label,
			Address:  e.address(),
			Actions:  actions(translate, e, user, false, carried),
		})
	}

	// The rows per page are offered as soon as there are rows (Driss, 2026-10-19).
	data.Sizes = pageSizes

	if pages > 1 || data.Sizes != nil {
		data.PagerHidden = query.pagerHidden()
		data.Pager = &pager{
			Info:  translate("servers.pager", "from", from+1, "to", to, "total", len(kept)),
			Page:  page,
			Pages: pages,
		}

		if page > 1 {
			data.Pager.PrevURL = listURLOf(query.withPage(page - 1))
		}

		if page < pages {
			data.Pager.NextURL = listURLOf(query.withPage(page + 1))
		}
	}

	return data
}

func countText(translate gui.Translator, total, shown int) string {
	switch {
	case shown == total && total == 1:
		return translate("servers.count_one")
	case shown == total:
		return translate("servers.count", "total", total)
	case shown == 1:
		return translate("servers.count_shown_one", "total", total)
	default:
		return translate("servers.count_shown", "total", total, "shown", shown)
	}
}

// sortEntries sorts by the asked column; ties keep the name order.
func sortEntries(entries []entry, key, dir string) {
	byName := func(a, b entry) int {
		return strings.Compare(strings.ToLower(a.agent.Name), strings.ToLower(b.agent.Name))
	}

	slices.SortStableFunc(entries, func(a, b entry) int {
		var order int

		switch key {
		case sortState:
			order = cmp.Compare(a.stateRank(), b.stateRank())
		case sortProtocol:
			order = strings.Compare(a.proto.Label, b.proto.Label)
		case sortAddress:
			order = cmp.Or(strings.Compare(a.agent.Address.Host, b.agent.Address.Host),
				cmp.Compare(a.agent.Address.Port, b.agent.Address.Port))
		default:
			order = byName(a, b)
		}

		if dir == dirDesc {
			order = -order
		}

		return cmp.Or(order, byName(a, b))
	})
}

// actions is the "⋯" menu of a server, in the list (with "View details")
// and on the details page (without). Entries follow the rights; the
// destructive one comes last. Every URL carries the list state.
func actions(translate gui.Translator, e entry, user rights, onDetail bool, carried string) []gui.Action {
	agentID := e.agent.ID

	var acts []gui.Action
	if !onDetail {
		acts = append(acts, gui.Action{
			Icon: "eye", Label: translate("servers.action.detail"), URL: withRet(serverURL(agentID, ""), carried),
		})
	}

	if user.Write {
		if e.running() {
			acts = append(acts, gui.Action{
				Icon: "stop", Label: translate("servers.action.stop"), Action: withRet(serverURL(agentID, "/stop"), carried),
			})
		} else {
			acts = append(acts, gui.Action{
				Icon: "play", Label: translate("servers.action.start"), Action: withRet(serverURL(agentID, "/start"), carried),
			})
		}
	}

	acts = append(acts,
		gui.Action{Icon: "key", Label: translate("servers.action.auth"), URL: historicURL(authPath, agentID)},
		gui.Action{Icon: "users", Label: translate("servers.action.accounts"), URL: historicURL(accountsPath, agentID)})

	if user.Write && !onDetail {
		acts = append(acts, gui.Action{
			Icon: "pen", Label: translate("servers.action.edit"), URL: withRet(serverURL(agentID, "/edit"), carried),
		})
	}

	if user.Delete {
		acts = append(acts, gui.Action{
			Icon:   "trash",
			Label:  translate("servers.action.delete"),
			URL:    withRet(serverURL(agentID, "/delete"), carried),
			Danger: true,
		})
	}

	return acts
}

// crumbs is the breadcrumb: the list alone, or the list (as the user left
// it) then the action page.
func crumbs(translate gui.Translator, last, carried string) []gui.Crumb {
	c := []gui.Crumb{
		{Label: translate("nav.home"), URL: common.WebuiURL("/home")},
		{Label: translate("nav.connections")},
	}

	if last == "" {
		return append(c, gui.Crumb{Label: translate("servers.title")})
	}

	return append(c,
		gui.Crumb{Label: translate("servers.title"), URL: listURL(carried)},
		gui.Crumb{Label: last})
}

// notices turns the result of the previous action, carried by the
// redirection, into notifications.
func notices(translate gui.Translator, v url.Values) []gui.Notice {
	name := v.Get(nameParam)

	if done := v.Get(doneParam); slices.Contains(doneKinds, done) {
		return []gui.Notice{{Kind: stateOK, Title: translate(toastKeyPrefix+done, "name", name)}}
	}

	switch v.Get(failedParam) {
	case failedStart:
		return []gui.Notice{{
			Kind: stateKO, Title: translate("servers.toast.start_failed", "name", name), Text: v.Get(reasonParam),
		}}
	case failedStop:
		return []gui.Notice{{
			Kind: stateKO, Title: translate("servers.toast.stop_failed", "name", name), Text: v.Get(reasonParam),
		}}
	default:
		return nil
	}
}

// redirectList sends back to the list as the user left it, with the result
// of the action.
func redirectList(w http.ResponseWriter, r *http.Request, carried string, result url.Values) {
	q := retOf(carried)
	v := q.withPage(q.Page)

	maps.Copy(v, result)

	http.Redirect(w, r, listURLOf(v), http.StatusSeeOther)
}

// -------------------------------------------------------------- pages ----

// server reads the {id} of the route; a missing server renders the
// localised "not found" page and returns false.
func (h *Handlers) server(w http.ResponseWriter, r *http.Request) (entry, bool) {
	id, parseErr := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

	agent, err := internal.GetServerByID(h.db, id)
	if parseErr == nil && err == nil {
		return h.entryOf(agent), true
	}

	if err != nil && !database.IsNotFound(err) {
		h.logger.Errorf("failed to retrieve the server %d: %v", id, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return entry{}, false
	}

	translate := h.notFound.Translator(r)
	carried := ret(r)

	h.notFound.Render(w, h.logger, r, common.Screen{
		Title:   translate("servers.not_found.title"),
		Section: constants.SidebarSectionConnection,
		Menu:    constants.SidebarMenuServers,
		Doc:     docPage,
		Crumbs:  crumbs(translate, translate("servers.not_found.title"), carried),
		Status:  http.StatusNotFound,
		Data:    map[string]string{"BackURL": listURL(carried)},
	})

	return entry{}, false
}

// screen is a page of a server: the chrome of the screen around the data.
func (h *Handlers) screen(translate gui.Translator, title, carried string, status int, data any) common.Screen {
	return common.Screen{
		Title:   title,
		Section: constants.SidebarSectionConnection,
		Menu:    constants.SidebarMenuServers,
		Doc:     docPage,
		Crumbs:  crumbs(translate, title, carried),
		Status:  status,
		Data:    data,
	}
}

type detailData struct {
	Name     string
	Status   gui.Status
	Reason   string
	Protocol string
	BackURL  string
	EditURL  string
	Actions  []gui.Action
	Identity []kv
	Dirs     []kv
	Options  []kv
}

// Detail serves GET /webui/servers/{id}: the server in reading.
func (h *Handlers) Detail(w http.ResponseWriter, r *http.Request) {
	srv, ok := h.server(w, r)
	if !ok {
		return
	}

	translate := h.detail.Translator(r)
	carried := ret(r)
	user := rightsOf(r)

	accounts, err := h.db.Count(&model.LocalAccount{}).Where("local_agent_id=?", srv.agent.ID).Run()
	if err != nil {
		h.logger.Errorf("failed to count the accounts of the server %d: %v", srv.agent.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	autostart := translate("servers.detail.autostart_yes")
	if srv.agent.Disabled {
		autostart = translate("servers.detail.autostart_no")
	}

	data := detailData{
		Name:     srv.agent.Name,
		Status:   srv.status(translate),
		Reason:   srv.reason,
		Protocol: srv.proto.Label,
		BackURL:  listURL(carried),
		Actions:  actions(translate, srv, user, true, carried),
		Identity: []kv{
			{Label: translate("servers.detail.protocol"), Value: srv.proto.Label},
			{Label: translate("servers.detail.address"), Value: srv.address(), Mono: true},
			{Label: translate("servers.detail.autostart"), Value: autostart},
			{Label: translate("servers.detail.accounts"), Value: strconv.FormatUint(accounts, 10)},
		},
		Dirs: []kv{
			{Label: translate("servers.dirs.root"), Value: srv.agent.RootDir, Mono: true},
			{Label: translate("servers.dirs.receive"), Value: srv.agent.ReceiveDir, Mono: true},
			{Label: translate("servers.dirs.tmp"), Value: srv.agent.TmpReceiveDir, Mono: true},
			{Label: translate("servers.dirs.send"), Value: srv.agent.SendDir, Mono: true},
		},
		Options: optionRows(translate, srv.proto, srv.agent.ProtoConfig),
	}

	if user.Write {
		data.EditURL = withRet(serverURL(srv.agent.ID, "/edit"), carried)
	}

	title := translate("servers.detail.title", "name", srv.agent.Name)
	h.detail.Render(w, h.logger, r, h.screen(translate, title, carried, 0, data))
}

type deleteData struct {
	Name         string
	Status       gui.Status
	Running      bool
	Protocol     string
	Address      string
	Accounts     string
	CancelURL    string
	Action       string
	TransfersURL string
	Blocked      bool // running transfers use one of its accounts: the model refuses
}

// blocked tells whether the model refuses the deletion (running transfers
// use an account of the server), from the model's own rule.
func (h *Handlers) blocked(e entry) (bool, error) {
	err := e.agent.BeforeDelete(h.db)
	if err == nil {
		return false, nil
	}

	if database.IsValidationError(err) {
		return true, nil
	}

	return false, fmt.Errorf("failed to check the transfers of the server: %w", err)
}

func (h *Handlers) deletePage(w http.ResponseWriter, r *http.Request, e entry, isBlocked bool, status int) {
	translate := h.del.Translator(r)
	carried := ret(r)

	accounts, err := h.db.Count(&model.LocalAccount{}).Where("local_agent_id=?", e.agent.ID).Run()
	if err != nil {
		h.logger.Errorf("failed to count the accounts of the server %d: %v", e.agent.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	data := deleteData{
		Name:         e.agent.Name,
		Status:       e.status(translate),
		Running:      e.running(),
		Protocol:     e.proto.Label,
		Address:      e.address(),
		Accounts:     strconv.FormatUint(accounts, 10),
		CancelURL:    withRet(serverURL(e.agent.ID, ""), carried),
		Action:       withRet(serverURL(e.agent.ID, "/delete"), carried),
		TransfersURL: common.WebuiURL(transfersPath),
		Blocked:      isBlocked,
	}

	title := translate("servers.delete.title", "name", e.agent.Name)
	h.del.Render(w, h.logger, r, h.screen(translate, title, carried, status, data))
}

// DeleteForm serves GET /webui/servers/{id}/delete: the confirmation page,
// or the refusal when the server is in use.
func (h *Handlers) DeleteForm(w http.ResponseWriter, r *http.Request) {
	e, ok := h.server(w, r)
	if !ok {
		return
	}

	isBlocked, err := h.blocked(e)
	if err != nil {
		h.logger.Errorf("failed to check the transfers of the server %d: %v", e.agent.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	h.deletePage(w, r, e, isBlocked, 0)
}

// Delete serves POST /webui/servers/{id}/delete.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	e, ok := h.server(w, r)
	if !ok {
		return
	}

	if err := h.ctl.Remove(r.Context(), h.db, e.agent); err != nil {
		if database.IsValidationError(err) {
			h.deletePage(w, r, e, true, http.StatusUnprocessableEntity)

			return
		}

		h.logger.Errorf("failed to delete the server %d: %v", e.agent.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	redirectList(w, r, ret(r), url.Values{doneParam: {doneDeleted}, nameParam: {e.agent.Name}})
}

// Start serves POST /webui/servers/{id}/start. A server that does not reach
// the running state sends back the reason its service gives.
func (h *Handlers) Start(w http.ResponseWriter, r *http.Request) {
	e, ok := h.server(w, r)
	if !ok {
		return
	}

	result := url.Values{nameParam: {e.agent.Name}}

	if !e.running() {
		if err := h.ctl.Start(r.Context(), e.agent); err != nil {
			h.logger.Warningf("failed to start the server %q: %v", e.agent.Name, err)
		}
	}

	if state, reason := h.ctl.Status(e.agent); state == utils.StateRunning {
		result.Set(doneParam, doneStarted)
	} else {
		result.Set(failedParam, failedStart)
		result.Set(reasonParam, reason)
	}

	redirectList(w, r, ret(r), result)
}

// Stop serves POST /webui/servers/{id}/stop.
func (h *Handlers) Stop(w http.ResponseWriter, r *http.Request) {
	e, ok := h.server(w, r)
	if !ok {
		return
	}

	result := url.Values{nameParam: {e.agent.Name}}

	if err := h.ctl.Stop(r.Context(), e.agent); err != nil {
		h.logger.Warningf("failed to stop the server %q: %v", e.agent.Name, err)

		_, reason := h.ctl.Status(e.agent)
		result.Set(failedParam, failedStop)
		result.Set(reasonParam, reason)
	} else {
		result.Set(doneParam, doneStopped)
	}

	redirectList(w, r, ret(r), result)
}

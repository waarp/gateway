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
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	// The stylesheets of the screen and of the shared protocol options,
	// under the web interface.
	stylesheet      = "/static_v2/servers/servers.css"
	protoStylesheet = "/static_v2/proto/proto.css"
	docPage         = "user/webui/serveurs-locaux"
	pagePath        = "/servers"

	// The historic page of the running transfers, the way out of a refused
	// deletion.
	transfersPath = "/transfer_monitoring"

	// The cookie that keeps the state of the list for the browser session, so
	// that coming back through the sidebar finds the list as it was left
	// (Driss, 2026-10-19). An explicit address always wins over it.
	stateCookie = "webui_servers_list"

	// The parameters of the list state and of the result of the previous
	// action, shared by the list screens.
	retParam    = liststate.RetParam
	doneParam   = liststate.DoneParam
	failedParam = liststate.FailedParam
	nameParam   = liststate.NameParam
	reasonParam = liststate.ReasonParam
	dirAsc      = liststate.DirAsc
	dirDesc     = liststate.DirDesc

	sortName     = "name"
	sortState    = "state"
	sortProtocol = "protocol"
	sortAddress  = "address"

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
	sortKeys  = []string{sortName, sortState, sortProtocol, sortAddress}
	pageSizes = []int{10, 25, 50, 100} // rows per page, the first one by default
	doneKinds = []string{doneCreated, doneUpdated, doneDeleted, doneStarted, doneStopped}

	serversCounter = counter{
		One: "servers.count_one", Many: "servers.count",
		ShownOne: "servers.count_shown_one", ShownMany: "servers.count_shown",
	}

	// The list of this screen (listSpec): its page, its session cookie, its
	// sort keys, its rows per page and the protocol filters.
	listSpec = liststate.Spec{
		Path: pagePath, Cookie: stateCookie, Sorts: sortKeys, Sizes: pageSizes,
		ParseFilters: func(v url.Values) liststate.Filters { return protoform.ParseFilter(v) },
		FilterParams: protoform.FilterParams,
	}
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
// own shell, as each one defines its own "page" template; the pages of the
// sheet share their partials (tab strip, fields).
type Handlers struct {
	db     *database.DB
	logger *log.Logger
	ctl    Control

	shell *common.Shell
}

func New(db *database.DB, logger *log.Logger, ctl Control) *Handlers {
	return &Handlers{db: db, logger: logger, ctl: ctl, shell: common.NewShell()}
}

func transfersURL() string { return common.WebuiURL(transfersPath) }

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
type kv = protoform.KV

// choice is an option of a select, selected or not.
type choice = protoform.Choice

// entry is a local server with its protocol and its running state.
type entry struct {
	agent  *model.LocalAgent
	proto  *protoform.Protocol
	state  utils.StateCode
	reason string
}

func (h *Handlers) entryOf(agent *model.LocalAgent) entry {
	state, reason := h.ctl.Status(agent)

	return entry{agent: agent, proto: protoform.ProtocolOf(agent.Protocol), state: state, reason: reason}
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

// listQuery is the state of the list asked by the URL, with the protocol
// filters (families, TLS); the tab lists of the sheet share it without them.
type listQuery = liststate.Query

// parseListQuery reads the state of the servers list.
func parseListQuery(v url.Values) *listQuery { return listSpec.Parse(v) }

// parseQuery reads the state of a tab list, with its own sort keys.
func parseQuery(v url.Values, sorts []string) *listQuery { return liststate.Parse(v, sorts, pageSizes) }

// protoFilter is the protocol filter of a list query.
func protoFilter(q *listQuery) *protoform.Filter {
	if f, ok := q.Filters.(*protoform.Filter); ok {
		return f
	}

	return &protoform.Filter{}
}

// The list state carried by a request, the list as the user left it, and a
// URL carrying the state: the shared mechanics, on the spec of this list.
func ret(r *http.Request) string   { return listSpec.Ret(r) }
func listURL(ret string) string    { return listSpec.ListURL(ret) }
func withRet(u, ret string) string { return liststate.WithRet(u, ret) }

// redirectList sends back to the list as the user left it, with the result
// of the action.
func redirectList(w http.ResponseWriter, r *http.Request, carried string, result url.Values) {
	listSpec.RedirectList(w, r, carried, result)
}

func serverURL(id int64, suffix string) string {
	return common.WebuiURL(pagePath + "/" + strconv.FormatInt(id, 10) + suffix)
}

// filterBox is one family of the filters panel.
type (
	filterBox = protoform.FilterBox
	row       struct {
		ID       int64
		Name     string
		Status   gui.Status
		Reason   string // raw reason given by the service, never translated
		Protocol string // the family
		TLS      bool   // shown as a badge next to the family
		Address  string
		Actions  []gui.Action
	}
)

// pager is the list foot, with the fields of "gui/pager".
type pager = liststate.Pager

type listData struct {
	AddURL       string // empty without the write right: no coloured action at all
	Action       string // toolbar form action: the bare list
	Base         string // list with search and filters: base of the sort links
	Q            string
	Names        []string
	Filters      []filterBox
	TLSFilter    []protoform.Choice // with or without TLS, or both
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
	if listSpec.Restore(w, r) {
		return
	}

	translate := h.shell.Translator(r)
	query := parseListQuery(r.URL.Query())
	screen := common.Screen{
		Title:   translate("servers.title"),
		Section: constants.SidebarSectionConnection,
		Menu:    constants.SidebarMenuServers,
		Doc:     docPage,
		Crumbs:  crumbs(translate, "", ""),
		Notices: notices(translate, r.URL.Query()),

		Stylesheets: []string{stylesheet},
	}

	agents, err := internal.ListServers(h.db, "name", true, 0, 0)
	if err != nil {
		h.logger.Errorf("failed to list the servers: %v", err)

		screen.Status = http.StatusInternalServerError
		h.shell.Render(w, h.logger, r, screen,
			listView(listData{Error: true, RetryURL: listSpec.ListURLOf(query.WithPage(query.Page))}))

		return
	}

	entries := make([]entry, 0, len(agents))
	for _, agent := range agents {
		entries = append(entries, h.entryOf(agent))
	}

	h.shell.Render(w, h.logger, r, screen, listView(buildList(translate, entries, query, rightsOf(r))))
}

// buildList computes the list page: filter (name prefix, protocols), sort,
// page, toolbar state, row actions according to the rights.
//
//nolint:funlen // one pass over the list, in the order of the page
func buildList(translate gui.Translator, all []entry, query *listQuery, user rights) listData {
	filter := protoFilter(query)
	data := listData{
		Action:       listSpec.ListURLOf(nil),
		Base:         listSpec.ListURLOf(query.FilterValues()),
		Q:            query.Q,
		Sort:         query.Sort,
		Dir:          query.Dir,
		Size:         query.Size,
		ResetURL:     listSpec.ResetURL(),
		Hidden:       query.ToolbarHidden(),
		Filters:      filter.Boxes(),
		TLSFilter:    filter.TLSChoices(translate),
		FiltersCount: filter.Count(),
		FiltersOpen:  filter.Count() > 0,
	}

	if user.Write {
		data.AddURL = withRet(common.WebuiURL(pagePath+"/new"), query.Encode(query.Page))
	}

	names := make([]string, 0, len(all))
	kept := make([]entry, 0, len(all))
	needle := strings.ToLower(query.Q)

	for _, e := range all {
		names = append(names, e.agent.Name)

		if needle != "" && !strings.HasPrefix(strings.ToLower(e.agent.Name), needle) {
			continue
		}

		if !filter.Matches(e.proto) {
			continue
		}

		kept = append(kept, e)
	}

	slices.Sort(names)
	data.Names = names
	sortEntries(kept, query.Sort, query.Dir)

	data.Count = serversCounter.Text(translate, len(all), len(kept))

	switch {
	case len(all) == 0:
		data.Empty = true

		return data
	case len(kept) == 0:
		data.NoMatch = true

		if query.Q != "" {
			withoutQ := *query
			withoutQ.Q = ""
			data.NoMatchText = translate("servers.none.title", "query", query.Q)
			data.NoMatchURL, data.NoMatchLink = listSpec.ExplicitListURL(withoutQ.Values()), translate("search.clear")
		} else {
			data.NoMatchText = translate("servers.none.filters")
			data.NoMatchURL, data.NoMatchLink = listSpec.ResetURL(), translate("filters.reset")
		}

		return data
	}

	b := query.Bounds(len(kept))
	carried := query.Encode(b.Page)
	data.Caption = translate("servers.caption",
		"page", b.Page, "pages", b.Pages, "sort", translate(sortKeyPrefix+query.Sort))

	for _, e := range kept[b.From:b.To] {
		data.Rows = append(data.Rows, row{
			ID:       e.agent.ID,
			Name:     e.agent.Name,
			Status:   e.status(translate),
			Reason:   e.reason,
			Protocol: e.proto.Family.Label,
			TLS:      e.proto.TLS,
			Address:  e.address(),
			Actions:  actions(translate, e, user, false, carried),
		})
	}

	// The rows per page are offered as soon as there are rows (Driss, 2026-10-19).
	data.Sizes = pageSizes
	data.PagerHidden = query.PagerHidden()
	data.Pager = listSpec.Pager(query, translate("servers.pager", "from", b.From+1, "to", b.To, "total", len(kept)), b)

	return data
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
			order = cmp.Or(strings.Compare(a.proto.Family.Label, b.proto.Family.Label), boolCmp(a.proto.TLS, b.proto.TLS))
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

// actions is the "⋯" menu of a server, in the list (with "View details", the
// tabs of the sheet and "Edit") and on the details page (without: the tabs
// and the buttons are there). Entries follow the rights; the destructive one
// comes last. Every URL carries the list state.
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

	if !onDetail {
		acts = append(acts,
			gui.Action{
				Icon: "key", Label: translate("servers.action.auth"), URL: withRet(serverURL(agentID, authSuffix), carried),
			},
			gui.Action{
				Icon:  "users",
				Label: translate("servers.action.accounts"),
				URL:   withRet(serverURL(agentID, accountsSuffix), carried),
			})

		if user.Write {
			acts = append(acts, gui.Action{
				Icon: "pen", Label: translate("servers.action.edit"), URL: withRet(serverURL(agentID, "/edit"), carried),
			})
		}
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
// redirection, into notifications: on a server, on one of its means or on
// one of its accounts.
func notices(translate gui.Translator, values url.Values) []gui.Notice {
	name := values.Get(nameParam)
	done := values.Get(doneParam)

	switch values.Get(kindParam) {
	case kindCred:
		if slices.Contains(sheetDone, done) {
			return []gui.Notice{{Kind: stateOK, Title: translate("auth.toast."+done, "name", name)}}
		}

		return nil
	case kindAccount:
		if slices.Contains(sheetDone, done) {
			return []gui.Notice{{Kind: stateOK, Title: translate("accounts.toast."+done, "login", name)}}
		}

		return nil
	}

	if slices.Contains(doneKinds, done) {
		return []gui.Notice{{Kind: stateOK, Title: translate(toastKeyPrefix+done, "name", name)}}
	}

	switch values.Get(failedParam) {
	case failedStart:
		return []gui.Notice{{
			Kind: stateKO, Title: translate("servers.toast.start_failed", "name", name), Text: values.Get(reasonParam),
		}}
	case failedStop:
		return []gui.Notice{{
			Kind: stateKO, Title: translate("servers.toast.stop_failed", "name", name), Text: values.Get(reasonParam),
		}}
	default:
		return nil
	}
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

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("servers.not_found.title")

	h.notFoundPage(w, r, crumbs(translate, title, carried), notFoundData{
		Title:     title,
		Text:      translate("servers.not_found.text"),
		Icon:      "server",
		BackLabel: translate("servers.back"),
		BackURL:   listURL(carried),
	})

	return entry{}, false
}

// screen is a page of a server: the chrome of the screen around the data.
func (h *Handlers) screen(translate gui.Translator, title, carried string, status int) common.Screen {
	return common.Screen{
		Title:   title,
		Section: constants.SidebarSectionConnection,
		Menu:    constants.SidebarMenuServers,
		Doc:     docPage,
		Crumbs:  crumbs(translate, title, carried),
		Status:  status,

		Stylesheets: []string{stylesheet},
	}
}

// formStylesheets are the stylesheets of the form: the shared protocol
// options, then the screen.
func formStylesheets() []string { return []string{protoStylesheet, stylesheet} }

type detailData struct {
	Name     string
	Status   gui.Status
	Reason   string
	Protocol string
	Tabs     []tab
	BackURL  string
	EditURL  string
	Actions  []gui.Action
	Identity []kv
	Dirs     []kv
	Options  []kv
}

// Detail serves GET /webui/servers/{id}: the "Details" tab of the sheet,
// the server in reading.
func (h *Handlers) Detail(w http.ResponseWriter, r *http.Request) {
	srv, ok := h.server(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	user := rightsOf(r)

	tabs, err := h.serverTabs(translate, srv, tabDetails, carried)
	if err != nil {
		h.logger.Errorf("%v", err)
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
		Protocol: srv.proto.Family.Label,
		Tabs:     tabs,
		BackURL:  listURL(carried),
		Actions:  actions(translate, srv, user, true, carried),
		Identity: []kv{
			{Label: translate("servers.detail.protocol"), Value: protoform.Describe(translate, srv.proto)},
			{Label: translate("servers.detail.address"), Value: srv.address(), Mono: true},
			{Label: translate("servers.detail.autostart"), Value: autostart},
			// The last tab counts the accounts.
			{Label: translate("servers.detail.accounts"), Value: strconv.FormatUint(tabs[len(tabs)-1].Count, 10)},
		},
		Dirs: []kv{
			{Label: translate("servers.dirs.root"), Value: srv.agent.RootDir, Mono: true},
			{Label: translate("servers.dirs.receive"), Value: srv.agent.ReceiveDir, Mono: true},
			{Label: translate("servers.dirs.tmp"), Value: srv.agent.TmpReceiveDir, Mono: true},
			{Label: translate("servers.dirs.send"), Value: srv.agent.SendDir, Mono: true},
		},
		Options: protoform.Rows(translate, protoform.RoleServer, srv.proto, srv.agent.ProtoConfig),
	}

	if user.Write {
		data.EditURL = withRet(serverURL(srv.agent.ID, "/edit"), carried)
	}

	own := owner{srv: srv}
	screen := h.sheetScreen(translate, own.title(translate), own, "", carried, 0, notices(translate, r.URL.Query()))
	h.shell.Render(w, h.logger, r, screen, detailView(data))
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
	translate := h.shell.Translator(r)
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
		Protocol:     protoform.Describe(translate, e.proto),
		Address:      e.address(),
		Accounts:     strconv.FormatUint(accounts, 10),
		CancelURL:    withRet(serverURL(e.agent.ID, ""), carried),
		Action:       withRet(serverURL(e.agent.ID, "/delete"), carried),
		TransfersURL: transfersURL(),
		Blocked:      isBlocked,
	}

	title := translate("servers.delete.title", "name", e.agent.Name)
	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, status), deleteView(data))
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

	var startErr error
	if !e.running() {
		if startErr = h.ctl.Start(r.Context(), e.agent); startErr != nil {
			h.logger.Warningf("failed to start the server %q: %v", e.agent.Name, startErr)
		}
	}

	if state, reason := h.ctl.Status(e.agent); state == utils.StateRunning {
		result.Set(doneParam, doneStarted)
	} else {
		result.Set(failedParam, failedStart)
		result.Set(reasonParam, liststate.FailureReason(reason, startErr))
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
		result.Set(reasonParam, liststate.FailureReason(reason, err))
	} else {
		result.Set(doneParam, doneStopped)
	}

	redirectList(w, r, ret(r), result)
}

// boolCmp orders false before true.
func boolCmp(a, b bool) int {
	switch {
	case a == b:
		return 0
	case b:
		return -1
	default:
		return 1
	}
}

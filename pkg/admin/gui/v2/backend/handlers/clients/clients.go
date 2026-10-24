// Package clients serves the "Local clients" screen of the web interface: the
// list of the local clients of the gateway (searched, filtered, sorted and
// paged), the details of a client (with the partners it can reach), the
// pages that create, edit and delete one, and the actions that start and
// stop one.
//
// A local client is the outgoing side of the gateway for one protocol: it
// connects to the partners. It has no accounts nor authentication means of
// its own (the remote accounts of the partners carry them), so the screen
// has no sheet with tabs; it carries the automatic retries of the transfers.
package clients

import (
	"cmp"
	"context"
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
	stylesheet      = "/static_v2/clients/clients.css"
	protoStylesheet = "/static_v2/proto/proto.css"
	script          = "/static_v2/clients/clients.js" // the retries drawn live, on the form

	docPage  = "user/webui/clients-locaux"
	pagePath = "/clients"

	// The sheet of a remote partner (#714).
	partnersPath = "/partners"

	// The cookie that keeps the state of the list for the browser session, so
	// that coming back through the sidebar finds the list as it was left.
	stateCookie = "webui_clients_list"

	// The parameters of the list state and of the result of the previous
	// action, shared by the list screens.
	retParam    = liststate.RetParam
	doneParam   = liststate.DoneParam
	failedParam = liststate.FailedParam
	nameParam   = liststate.NameParam
	reasonParam = liststate.ReasonParam
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
	// column ("clients.sort.name") and the result of an action
	// ("clients.toast.created").
	sortKeyPrefix  = "clients.sort."
	toastKeyPrefix = "clients.toast."

	secondsPerMinute = 60
	secondsPerHour   = 3600
)

//nolint:gochecknoglobals // reference data of the screen
var (
	sortKeys  = []string{sortName, sortState, sortProtocol, sortAddress}
	pageSizes = []int{10, 25, 50, 100} // rows per page, the first one by default
	doneKinds = []string{doneCreated, doneUpdated, doneDeleted, doneStarted, doneStopped}

	clientsCounter = liststate.Counter{
		One: "clients.count_one", Many: "clients.count",
		ShownOne: "clients.count_shown_one", ShownMany: "clients.count_shown",
	}

	// The list of this screen (listSpec): its page, its session cookie, its
	// sort keys, its rows per page and the protocol filters.
	listSpec = liststate.Spec{
		Path: pagePath, Cookie: stateCookie, Sorts: sortKeys, Sizes: pageSizes,
		ParseFilters: func(v url.Values) liststate.Filters { return protoform.ParseFilter(v) },
		FilterParams: protoform.FilterParams,
	}
)

// Control drives the service of a client. On the gateway it is backed by the
// service lists (GatewayControl); the tests use stubs.
type Control struct {
	Status func(client *model.Client) (utils.StateCode, string)
	Add    func(db *database.DB, client *model.Client) error
	Remove func(ctx context.Context, db *database.DB, client *model.Client) error
	Start  func(ctx context.Context, client *model.Client) error
	Stop   func(ctx context.Context, client *model.Client) error
}

// GatewayControl is the Control of the running gateway.
func GatewayControl() Control {
	return Control{
		Status: internal.GetClientStatus,
		Add:    internal.AddClient,
		Remove: internal.RemoveClient,
		Start:  internal.RestartClient,
		Stop:   internal.StopClient,
	}
}

// Handlers serves the pages and the actions of the screen. Each page has its
// own shell, as each one defines its own "page" template.
type Handlers struct {
	db     *database.DB
	logger *log.Logger
	ctl    Control

	shell *common.Shell
}

func New(db *database.DB, logger *log.Logger, ctl Control) *Handlers {
	return &Handlers{
		db:     db,
		logger: logger,
		ctl:    ctl,
		shell:  common.NewShell(),
	}
}

// rights of the current user on the clients, which follow the rights on the
// servers (the product has no right of its own for the clients): reading is
// implied by reaching the screen; writing and deleting are checked again by
// the router on each action. The templates hide what the rights forbid.
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

// entry is a local client with its protocol and its running state.
type entry struct {
	client *model.Client
	proto  *protoform.Protocol
	state  utils.StateCode
	reason string
}

func (h *Handlers) entryOf(client *model.Client) entry {
	state, reason := h.ctl.Status(client)

	return entry{client: client, proto: protoform.ProtocolOf(client.Protocol), state: state, reason: reason}
}

func (e *entry) running() bool { return e.state == utils.StateRunning }

// address is the local address the client connects from, "" when the
// system chooses it (shown as "Automatic").
func (e *entry) address() string { return e.client.LocalAddress.String() }

// status turns the state code of a client into a shape, a colour and a label;
// a code outside the three known ones reads "unknown".
func (e *entry) status(translate gui.Translator) gui.Status {
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
func (e *entry) stateRank() int {
	order := []utils.StateCode{utils.StateRunning, utils.StateOffline, utils.StateError}
	if rank := slices.Index(order, e.state); rank >= 0 {
		return rank
	}

	return len(order)
}

// --------------------------------------------------------------- list ----

// listQuery is the state of the list asked by the URL, with the protocol
// filters (families, TLS).
type listQuery = liststate.Query

func parseListQuery(v url.Values) *listQuery { return listSpec.Parse(v) }

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

func clientURL(id int64, suffix string) string {
	return common.WebuiURL(pagePath + "/" + strconv.FormatInt(id, 10) + suffix)
}

// partnerURL is the sheet of a remote partner.
func partnerURL(id int64) string {
	return common.WebuiURL(partnersPath + "/" + strconv.FormatInt(id, 10))
}

type row struct {
	ID       int64
	Name     string
	URL      string
	Status   gui.Status
	Reason   string // raw reason given by the service, never translated
	Protocol string // the family
	TLS      bool   // shown as a badge next to the family
	Address  string // "" when the system chooses it
	Actions  []gui.Action
}

type listData struct {
	AddURL       string // empty without the write right: no coloured action at all
	Action       string // toolbar form action: the bare list
	Base         string // list with search and filters: base of the sort links
	Q            string
	Names        []string
	Filters      []protoform.FilterBox
	TLSFilter    []protoform.Choice
	FiltersOpen  bool
	FiltersCount int
	ResetURL     string
	Sort         string
	Dir          string
	Hidden       map[string]string // carried by the toolbar form
	Count        string
	Rows         []row
	Caption      string
	Pager        *liststate.Pager
	PagerHidden  map[string]string
	Sizes        []int // rows-per-page choices, offered as soon as there are rows
	Size         int
	Empty        bool // no client at all
	NoMatch      bool // clients exist, none matches
	NoMatchText  string
	NoMatchURL   string
	NoMatchLink  string
	Error        bool // the clients could not be read
	RetryURL     string
}

// List serves GET /webui/clients.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	fragment := liststate.IsListFragment(r)
	if !fragment && listSpec.Restore(w, r) {
		return
	}

	translate := h.shell.Translator(r)
	query := parseListQuery(r.URL.Query())
	screen := common.Screen{
		Title:   translate("clients.title"),
		Section: constants.SidebarSectionConnection,
		Menu:    constants.SidebarMenuClients,
		Doc:     docPage,
		Crumbs:  crumbs(translate, "", ""),
		Notices: notices(translate, r.URL.Query()),
		Head:    gui.LiveAssets(),

		Stylesheets: []string{protoStylesheet, stylesheet},
	}

	clients, err := internal.ListClients(h.db, "name", true, 0, 0)
	if err != nil {
		h.logger.Errorf("failed to list the clients: %v", err)

		if fragment {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}

		screen.Status = http.StatusInternalServerError
		h.shell.Render(w, h.logger, r, screen,
			listView(listData{Error: true, RetryURL: listSpec.ListURLOf(query.WithPage(query.Page))}))

		return
	}

	entries := make([]entry, 0, len(clients))
	for _, client := range clients {
		entries = append(entries, h.entryOf(client))
	}

	data := buildList(translate, entries, query, rightsOf(r))
	if fragment {
		h.shell.RenderFragment(w, h.logger, r, listFragment(data))

		return
	}

	h.shell.Render(w, h.logger, r, screen, listView(data))
}

// buildList computes the list page: filter (name prefix, protocols), sort,
// page, toolbar state, row actions according to the rights.
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

	for i := range all {
		e := &all[i]
		names = append(names, e.client.Name)

		if needle != "" && !strings.HasPrefix(strings.ToLower(e.client.Name), needle) {
			continue
		}

		if !filter.Matches(e.proto) {
			continue
		}

		kept = append(kept, *e)
	}

	slices.Sort(names)
	data.Names = names
	sortEntries(kept, query.Sort, query.Dir)

	data.Count = clientsCounter.Text(translate, len(all), len(kept))

	switch {
	case len(all) == 0:
		data.Empty = true

		return data
	case len(kept) == 0:
		data.NoMatch = true

		if query.Q != "" {
			withoutQ := *query
			withoutQ.Q = ""
			data.NoMatchText = translate("clients.none.title", "query", query.Q)
			data.NoMatchURL, data.NoMatchLink = listSpec.ExplicitListURL(withoutQ.Values()), translate("search.clear")
		} else {
			data.NoMatchText = translate("clients.none.filters")
			data.NoMatchURL, data.NoMatchLink = listSpec.ResetURL(), translate("filters.reset")
		}

		return data
	}

	b := query.Bounds(len(kept))
	carried := query.Encode(b.Page)
	data.Caption = translate("clients.caption",
		"page", b.Page, "pages", b.Pages, "sort", translate(sortKeyPrefix+query.Sort))

	for i := range kept[b.From:b.To] {
		e := &kept[b.From+i]
		data.Rows = append(data.Rows, row{
			ID:       e.client.ID,
			Name:     e.client.Name,
			URL:      withRet(clientURL(e.client.ID, ""), carried),
			Status:   e.status(translate),
			Reason:   e.reason,
			Protocol: e.proto.Family.Label,
			TLS:      e.proto.TLS,
			Address:  e.address(),
			Actions:  actions(translate, e, user, false, carried),
		})
	}

	// The rows per page are offered as soon as there are rows.
	data.Sizes = pageSizes
	data.PagerHidden = query.PagerHidden()
	data.Pager = listSpec.Pager(query, translate("clients.pager", "from", b.From+1, "to", b.To, "total", len(kept)), b)

	return data
}

// sortEntries sorts by the asked column; ties keep the name order. An
// automatic address sorts before the set ones.
func sortEntries(entries []entry, key, dir string) {
	byName := func(a, b *entry) int {
		return strings.Compare(strings.ToLower(a.client.Name), strings.ToLower(b.client.Name))
	}

	slices.SortStableFunc(entries, func(a, b entry) int {
		var order int

		switch key {
		case sortState:
			order = cmp.Compare(a.stateRank(), b.stateRank())
		case sortProtocol:
			order = cmp.Or(strings.Compare(a.proto.Family.Label, b.proto.Family.Label), boolCmp(a.proto.TLS, b.proto.TLS))
		case sortAddress:
			order = cmp.Or(strings.Compare(a.client.LocalAddress.Host, b.client.LocalAddress.Host),
				cmp.Compare(a.client.LocalAddress.Port, b.client.LocalAddress.Port))
		default:
			order = byName(&a, &b)
		}

		if dir == dirDesc {
			order = -order
		}

		return cmp.Or(order, byName(&a, &b))
	})
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

// actions is the "⋯" menu of a client, in the list (with "View details" and
// "Edit") and on the details page (without: the buttons are there). Entries
// follow the rights; the destructive one comes last. Every URL carries the
// list state.
func actions(translate gui.Translator, e *entry, user rights, onDetail bool, carried string) []gui.Action {
	clientID := e.client.ID

	var acts []gui.Action
	if !onDetail {
		acts = append(acts, gui.Action{
			Icon: "eye", Label: translate("clients.action.detail"), URL: withRet(clientURL(clientID, ""), carried),
		})
	}

	if user.Write {
		if e.running() {
			acts = append(acts, gui.Action{
				Icon: "stop", Label: translate("clients.action.stop"), Action: withRet(clientURL(clientID, "/stop"), carried),
			})
		} else {
			acts = append(acts, gui.Action{
				Icon: "play", Label: translate("clients.action.start"), Action: withRet(clientURL(clientID, "/start"), carried),
			})
		}

		if !onDetail {
			acts = append(acts, gui.Action{
				Icon: "pen", Label: translate("clients.action.edit"), URL: withRet(clientURL(clientID, "/edit"), carried),
			})
		}
	}

	if user.Delete {
		acts = append(acts, gui.Action{
			Icon:   "trash",
			Label:  translate("clients.action.delete"),
			URL:    withRet(clientURL(clientID, "/delete"), carried),
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
		return append(c, gui.Crumb{Label: translate("clients.title")})
	}

	return append(c,
		gui.Crumb{Label: translate("clients.title"), URL: listURL(carried)},
		gui.Crumb{Label: last})
}

// notices turns the result of the previous action, carried by the
// redirection, into notifications.
func notices(translate gui.Translator, values url.Values) []gui.Notice {
	name := values.Get(nameParam)

	if done := values.Get(doneParam); slices.Contains(doneKinds, done) {
		return []gui.Notice{{Kind: stateOK, Title: translate(toastKeyPrefix+done, "name", name)}}
	}

	switch values.Get(failedParam) {
	case failedStart:
		return []gui.Notice{{
			Kind: stateKO, Title: translate("clients.toast.start_failed", "name", name), Text: values.Get(reasonParam),
		}}
	case failedStop:
		return []gui.Notice{{
			Kind: stateKO, Title: translate("clients.toast.stop_failed", "name", name), Text: values.Get(reasonParam),
		}}
	default:
		return nil
	}
}

// -------------------------------------------------------------- pages ----

// notFoundData is the localised "not found" page.
type notFoundData struct {
	Title     string
	Text      string
	BackLabel string
	BackURL   string
}

// client reads the {id} of the route; a missing client renders the
// localised "not found" page and returns false.
func (h *Handlers) client(w http.ResponseWriter, r *http.Request) (entry, bool) {
	id, parseErr := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

	client, err := internal.GetClientByID(h.db, id)
	if parseErr == nil && err == nil {
		return h.entryOf(client), true
	}

	if err != nil && !database.IsNotFound(err) {
		h.logger.Errorf("failed to retrieve the client %d: %v", id, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return entry{}, false
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("clients.not_found.title")

	h.shell.Render(w, h.logger, r, common.Screen{
		Title:   title,
		Section: constants.SidebarSectionConnection,
		Menu:    constants.SidebarMenuClients,
		Doc:     docPage,
		Crumbs:  crumbs(translate, title, carried),
		Status:  http.StatusNotFound,
	}, notFoundView(notFoundData{
		Title:     title,
		Text:      translate("clients.not_found.text"),
		BackLabel: translate("clients.back"),
		BackURL:   listURL(carried),
	}))

	return entry{}, false
}

// screen is a page of a client: the chrome of the screen around the data.
func (h *Handlers) screen(translate gui.Translator, title, carried string, status int) common.Screen {
	return common.Screen{
		Title:   title,
		Section: constants.SidebarSectionConnection,
		Menu:    constants.SidebarMenuClients,
		Doc:     docPage,
		Crumbs:  crumbs(translate, title, carried),
		Status:  status,

		Stylesheets: []string{protoStylesheet, stylesheet},
	}
}

// formStylesheets are the stylesheets of the form: the shared protocol
// options, then the screen.
func formStylesheets() []string { return []string{protoStylesheet, stylesheet} }

// formScripts is the script of the form: the retries drawn live.
func formScripts() []string { return []string{script} }

// partnerRow is a remote partner the client can reach: a partner of its
// protocol (decision of 2026-10-19: the two screens stay apart, linked at
// the screen).
type partnerRow struct {
	Name    string
	Address string
	URL     string
}

type detailData struct {
	Name          string
	Status        gui.Status
	Reason        string
	Protocol      string // the family, for the title of the options card
	BackURL       string
	EditURL       string
	Actions       []gui.Action
	Identity      []kv
	Retries       []kv
	RetrySentence string // how long the retries take, in one sentence
	NoRetries     bool   // no automatic retry: the card says so instead of the rows
	Options       []kv
	Partners      []partnerRow
}

// Detail serves GET /webui/clients/{id}: the client in reading, with the
// partners it can reach.
func (h *Handlers) Detail(w http.ResponseWriter, r *http.Request) {
	cli, ok := h.client(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	user := rightsOf(r)

	partners, err := internal.ListPartners(h.db, "name", true, 0, 0, cli.client.Protocol)
	if err != nil {
		h.logger.Errorf("failed to list the partners of the client %d: %v", cli.client.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	autostart := translate("clients.detail.autostart_yes")
	if cli.client.Disabled {
		autostart = translate("clients.detail.autostart_no")
	}

	address := kv{Label: translate("clients.detail.address"), Value: translate("clients.address_auto")}
	if addr := cli.address(); addr != "" {
		address = kv{Label: translate("clients.detail.address"), Value: addr, Mono: true}
	}

	data := detailData{
		Name:     cli.client.Name,
		Status:   cli.status(translate),
		Reason:   cli.reason,
		Protocol: cli.proto.Family.Label,
		BackURL:  listURL(carried),
		Actions:  actions(translate, &cli, user, true, carried),
		Identity: []kv{
			{Label: translate("clients.detail.protocol"), Value: protoform.Describe(translate, cli.proto)},
			address,
			{Label: translate("clients.detail.autostart"), Value: autostart},
		},
		Options: protoform.Rows(translate, protoform.RoleClient, cli.proto, cli.client.ProtoConfig),
	}

	if cli.client.NbOfAttempts == 0 {
		data.NoRetries = true
	} else {
		data.Retries = retryRows(translate, cli.client)
		data.RetrySentence = retrySentence(translate,
			cli.client.NbOfAttempts, cli.client.FirstRetryDelay, cli.client.RetryIncrementFactor)
	}

	for _, partner := range partners {
		data.Partners = append(data.Partners,
			partnerRow{Name: partner.Name, Address: partner.Address.String(), URL: partnerURL(partner.ID)})
	}

	if user.Write {
		data.EditURL = withRet(clientURL(cli.client.ID, "/edit"), carried)
	}

	title := translate("clients.detail.title", "name", cli.client.Name)
	screen := h.screen(translate, title, carried, 0)
	screen.Notices = notices(translate, r.URL.Query())
	h.shell.Render(w, h.logger, r, screen, detailView(data))
}

// retryRows describes the automatic retries of the transfers of a client:
// the attempts, the delay before the first retry and the growth factor.
func retryRows(translate gui.Translator, client *model.Client) []kv {
	return []kv{
		{Label: translate("clients.retries.attempts"), Value: strconv.Itoa(int(client.NbOfAttempts))},
		{Label: translate("clients.retries.delay"), Value: formatDelay(translate, client.FirstRetryDelay)},
		{Label: translate("clients.retries.factor"), Value: formatFactor(client.RetryIncrementFactor)},
	}
}

// formatDelay writes a delay in seconds as hours, minutes and seconds, the
// zero parts left out; a zero delay reads "none".
func formatDelay(translate gui.Translator, seconds int32) string {
	if seconds <= 0 {
		return translate("clients.retries.delay_none")
	}

	hours, minutes, secs := splitDelay(seconds)

	var parts []string
	if hours > 0 {
		parts = append(parts, translate("clients.retries.delay_hours", "n", hours))
	}

	if minutes > 0 {
		parts = append(parts, translate("clients.retries.delay_minutes", "n", minutes))
	}

	if secs > 0 {
		parts = append(parts, translate("clients.retries.delay_seconds", "n", secs))
	}

	return strings.Join(parts, " ")
}

// splitDelay splits a delay in seconds into hours, minutes and seconds.
func splitDelay(seconds int32) (hours, minutes, secs int32) {
	return seconds / secondsPerHour, (seconds % secondsPerHour) / secondsPerMinute, seconds % secondsPerMinute
}

// formatFactor writes the growth factor of the delay; the product applies
// 1 when none is stored with a delay.
func formatFactor(factor float32) string {
	if factor == 0 {
		return "1"
	}

	return strconv.FormatFloat(float64(factor), 'f', -1, 32)
}

type deleteData struct {
	Name      string
	Status    gui.Status
	Running   bool
	Protocol  string
	Address   string // "" when the system chooses it
	CancelURL string
	Action    string
	Blocked   bool // folder watchers use the client: the database refuses
}

// blocked tells whether the deletion is refused: the folder watchers of the
// gateway name their client with a foreign key that forbids it.
func (h *Handlers) blocked(e *entry) (bool, error) {
	watchers, err := h.db.Count(&model.FileWatcher{}).Where("client_id=?", e.client.ID).Run()
	if err != nil {
		return false, err //nolint:wrapcheck // logged by the caller with its context
	}

	return watchers > 0, nil
}

func (h *Handlers) deletePage(w http.ResponseWriter, r *http.Request, e *entry, isBlocked bool, status int) {
	translate := h.shell.Translator(r)
	carried := ret(r)

	data := deleteData{
		Name:      e.client.Name,
		Status:    e.status(translate),
		Running:   e.running(),
		Protocol:  protoform.Describe(translate, e.proto),
		Address:   e.address(),
		CancelURL: withRet(clientURL(e.client.ID, ""), carried),
		Action:    withRet(clientURL(e.client.ID, "/delete"), carried),
		Blocked:   isBlocked,
	}

	title := translate("clients.delete.title", "name", e.client.Name)
	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, status), deleteView(data))
}

// DeleteForm serves GET /webui/clients/{id}/delete: the confirmation page,
// or the refusal when folder watchers use the client.
func (h *Handlers) DeleteForm(w http.ResponseWriter, r *http.Request) {
	e, ok := h.client(w, r)
	if !ok {
		return
	}

	isBlocked, err := h.blocked(&e)
	if err != nil {
		h.logger.Errorf("failed to count the folder watchers of the client %d: %v", e.client.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	h.deletePage(w, r, &e, isBlocked, 0)
}

// Delete serves POST /webui/clients/{id}/delete.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	cli, ok := h.client(w, r)
	if !ok {
		return
	}

	isBlocked, err := h.blocked(&cli)
	if err != nil {
		h.logger.Errorf("failed to count the folder watchers of the client %d: %v", cli.client.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	if isBlocked {
		h.deletePage(w, r, &cli, true, http.StatusUnprocessableEntity)

		return
	}

	if removeErr := h.ctl.Remove(r.Context(), h.db, cli.client); removeErr != nil {
		h.logger.Errorf("failed to delete the client %d: %v", cli.client.ID, removeErr)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	listSpec.RedirectList(w, r, ret(r), url.Values{doneParam: {doneDeleted}, nameParam: {cli.client.Name}})
}

// Start serves POST /webui/clients/{id}/start. A client that does not reach
// the running state sends back the reason its service gives.
func (h *Handlers) Start(w http.ResponseWriter, r *http.Request) {
	e, ok := h.client(w, r)
	if !ok {
		return
	}

	result := url.Values{nameParam: {e.client.Name}}

	var startErr error
	if !e.running() {
		if startErr = h.ctl.Start(r.Context(), e.client); startErr != nil {
			h.logger.Warningf("failed to start the client %q: %v", e.client.Name, startErr)
		}
	}

	if state, reason := h.ctl.Status(e.client); state == utils.StateRunning {
		result.Set(doneParam, doneStarted)
	} else {
		result.Set(failedParam, failedStart)
		result.Set(reasonParam, liststate.FailureReason(reason, startErr))
	}

	listSpec.RedirectList(w, r, ret(r), result)
}

// Stop serves POST /webui/clients/{id}/stop: stopping a client interrupts
// its running transfers.
func (h *Handlers) Stop(w http.ResponseWriter, r *http.Request) {
	e, ok := h.client(w, r)
	if !ok {
		return
	}

	result := url.Values{nameParam: {e.client.Name}}

	if err := h.ctl.Stop(r.Context(), e.client); err != nil {
		h.logger.Warningf("failed to stop the client %q: %v", e.client.Name, err)

		_, reason := h.ctl.Status(e.client)
		result.Set(failedParam, failedStop)
		result.Set(reasonParam, liststate.FailureReason(reason, err))
	} else {
		result.Set(doneParam, doneStopped)
	}

	listSpec.RedirectList(w, r, ret(r), result)
}

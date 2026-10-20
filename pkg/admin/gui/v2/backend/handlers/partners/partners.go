// Package partners serves the "Remote partners" screen of the web interface:
// the list of the remote partners of the gateway (searched, filtered, sorted
// and paged), the sheet of a partner (its details, the means the gateway
// checks of it, its remote accounts), the pages that create, edit and delete
// one.
//
// A remote partner is a third-party server the gateway connects to, through
// the local client of its protocol, to deposit or retrieve files. It is not a
// service: nothing to start or stop. It mirrors the sheet of a local server:
// the server presents its identity and receives accounts, the partner is
// checked and receives the identities the gateway presents to it.
package partners

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
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/means"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	// The stylesheets of the screen and of the shared protocol options,
	// under the web interface.
	stylesheet      = "/static_v2/partners/partners.css"
	protoStylesheet = "/static_v2/proto/proto.css"
	docPage         = "user/webui/partenaires-distants"
	pagePath        = "/partners"

	// The historic pages the sheet leads to: the running transfers (the way
	// out of a refused deletion) and the transfer rules (where the accesses
	// of a partner are managed).
	transfersPath = "/transfer_monitoring"
	rulesPath     = "/transfer_rules_management"

	// The cookie that keeps the state of the list for the browser session.
	stateCookie = "webui_partners_list"

	// The parameters of the list state and of the result of the previous
	// action, shared by the list screens.
	retParam  = liststate.RetParam
	doneParam = liststate.DoneParam
	nameParam = liststate.NameParam
	dirDesc   = liststate.DirDesc

	sortName     = "name"
	sortProtocol = "protocol"
	sortAddress  = "address"
	sortAccounts = "accounts"

	doneCreated = liststate.DoneCreated
	doneUpdated = liststate.DoneUpdated
	doneDeleted = liststate.DoneDeleted

	// States of a gui.Status: the shape and colour of a status.
	stateOK   = "ok"
	stateKO   = "ko"
	stateWarn = "warn"

	// Prefixes of the catalogue keys chosen at run time: the name of a sorted
	// column ("partners.sort.name") and the result of an action
	// ("partners.toast.created").
	sortKeyPrefix  = "partners.sort."
	toastKeyPrefix = "partners.toast."
)

//nolint:gochecknoglobals // reference data of the screen
var (
	sortKeys  = []string{sortName, sortProtocol, sortAddress, sortAccounts}
	pageSizes = liststate.DefaultSizes
	doneKinds = []string{doneCreated, doneUpdated, doneDeleted}

	partnersCounter = liststate.Counter{
		One: "partners.count_one", Many: "partners.count",
		ShownOne: "partners.count_shown_one", ShownMany: "partners.count_shown",
	}

	// The list of this screen (listSpec): its page, its session cookie, its
	// sort keys, its rows per page and the protocol filters.
	listSpec = liststate.Spec{
		Path: pagePath, Cookie: stateCookie, Sorts: sortKeys, Sizes: pageSizes,
		ParseFilters: func(v url.Values) liststate.Filters { return protoform.ParseFilter(v) },
		FilterParams: protoform.FilterParams,
	}
)

// Handlers serves the pages and the actions of the screen.
type Handlers struct {
	db     *database.DB
	logger *log.Logger
	shell  *common.Shell
	// means serves the pages of the authentication means of a partner and
	// of its remote accounts, shared with the other sheets.
	means means.Pages
}

func New(db *database.DB, logger *log.Logger) *Handlers {
	shell := common.NewShell()

	return &Handlers{db: db, logger: logger, shell: shell, means: means.Pages{DB: db, Logger: logger, Shell: shell}}
}

func transfersURL() string { return common.WebuiURL(transfersPath) }
func rulesURL() string     { return common.WebuiURL(rulesPath) }

// rights of the current user on the partners: reading is implied by reaching
// the screen; writing and deleting are checked again by the router on each
// action. The templates hide what the rights forbid.
type rights struct{ Write, Delete bool }

func rightsOf(r *http.Request) rights {
	perms := common.GetUser(r).Permissions

	return rights{
		Write:  perms.HasPermission(model.PermPartnersWrite),
		Delete: perms.HasPermission(model.PermPartnersDelete),
	}
}

// kv is a row of a key/value list.
type kv = protoform.KV

// entry is a remote partner with its protocol and the number of its remote
// accounts.
type entry struct {
	agent    *model.RemoteAgent
	proto    *protoform.Protocol
	accounts int
}

func entryOf(agent *model.RemoteAgent, accounts int) entry {
	return entry{agent: agent, proto: protoform.ProtocolOf(agent.Protocol), accounts: accounts}
}

func (e entry) address() string { return e.agent.Address.String() }

// accountsText says how many remote accounts the partner has, or that it has
// none (a partner without any account cannot be reached by a transfer).
func (e entry) accountsText(translate gui.Translator) string {
	if e.accounts == 0 {
		return translate("partners.accounts_none")
	}

	return strconv.Itoa(e.accounts)
}

// accountCounts counts the remote accounts of every partner in one query.
func (h *Handlers) accountCounts() (map[int64]int, error) {
	var accounts model.RemoteAccounts
	if err := h.db.Select(&accounts).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller
	}

	counts := map[int64]int{}
	for _, acc := range accounts {
		counts[acc.RemoteAgentID]++
	}

	return counts, nil
}

// --------------------------------------------------------------- list ----

// listQuery is the state of the list asked by the URL, with the protocol
// filters (families, TLS); the tab lists of the sheet share it without them.
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

// redirectList sends back to the list as the user left it, with the result
// of the action.
func redirectList(w http.ResponseWriter, r *http.Request, carried string, result url.Values) {
	listSpec.RedirectList(w, r, carried, result)
}

func partnerURL(id int64, suffix string) string {
	return common.WebuiURL(pagePath + "/" + strconv.FormatInt(id, 10) + suffix)
}

type (
	filterBox = protoform.FilterBox
	row       struct {
		ID         int64
		Name       string
		Protocol   string // the family
		TLS        bool   // shown as a badge next to the family
		Address    string
		Accounts   string
		NoAccounts bool // the partner cannot be reached: the warning of the kit
		URL        string
		Actions    []gui.Action
	}
)

type pager = liststate.Pager

type listData struct {
	AddURL       string // empty without the write right: no coloured action at all
	Action       string // toolbar form action: the bare list
	Base         string // list with search and filters: base of the sort links
	Q            string
	Names        []string
	Filters      []filterBox
	TLSFilter    []protoform.Choice
	FiltersOpen  bool
	FiltersCount int
	ResetURL     string
	Sort         string
	Dir          string
	Hidden       map[string]string
	Count        string
	Rows         []row
	Caption      string
	Pager        *pager
	PagerHidden  map[string]string
	Sizes        []int
	Size         int
	Empty        bool // no partner at all
	NoMatch      bool // partners exist, none matches
	NoMatchText  string
	NoMatchURL   string
	NoMatchLink  string
	Error        bool // the partners could not be read
	RetryURL     string
}

// List serves GET /webui/partners.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	if listSpec.Restore(w, r) {
		return
	}

	translate := h.shell.Translator(r)
	query := parseListQuery(r.URL.Query())
	screen := common.Screen{
		Title:   translate("partners.title"),
		Section: constants.SidebarSectionConnection,
		Menu:    constants.SidebarMenuPartners,
		Doc:     docPage,
		Crumbs:  crumbs(translate, "", ""),
		Notices: notices(translate, r.URL.Query()),

		Stylesheets: sheetStylesheets(),
	}

	var agents model.RemoteAgents

	err := h.db.Select(&agents).OrderBy("name", true).Run()
	counts := map[int64]int{}

	if err == nil {
		counts, err = h.accountCounts()
	}

	if err != nil {
		h.logger.Errorf("failed to list the partners: %v", err)

		screen.Status = http.StatusInternalServerError
		h.shell.Render(w, h.logger, r, screen,
			listView(listData{Error: true, RetryURL: listSpec.ListURLOf(query.WithPage(query.Page))}))

		return
	}

	entries := make([]entry, 0, len(agents))
	for _, agent := range agents {
		entries = append(entries, entryOf(agent, counts[agent.ID]))
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

	data.Count = partnersCounter.Text(translate, len(all), len(kept))

	switch {
	case len(all) == 0:
		data.Empty = true

		return data
	case len(kept) == 0:
		data.NoMatch = true

		if query.Q != "" {
			withoutQ := *query
			withoutQ.Q = ""
			data.NoMatchText = translate("partners.none.title", "query", query.Q)
			data.NoMatchURL, data.NoMatchLink = listSpec.ExplicitListURL(withoutQ.Values()), translate("search.clear")
		} else {
			data.NoMatchText = translate("partners.none.filters")
			data.NoMatchURL, data.NoMatchLink = listSpec.ResetURL(), translate("filters.reset")
		}

		return data
	}

	b := query.Bounds(len(kept))
	carried := query.Encode(b.Page)
	data.Caption = translate("partners.caption",
		"page", b.Page, "pages", b.Pages, "sort", translate(sortKeyPrefix+query.Sort))

	for _, e := range kept[b.From:b.To] {
		data.Rows = append(data.Rows, row{
			ID:         e.agent.ID,
			Name:       e.agent.Name,
			Protocol:   e.proto.Family.Label,
			TLS:        e.proto.TLS,
			Address:    e.address(),
			Accounts:   e.accountsText(translate),
			NoAccounts: e.accounts == 0,
			URL:        withRet(partnerURL(e.agent.ID, ""), carried),
			Actions:    actions(translate, e, user, false, carried),
		})
	}

	data.Sizes = pageSizes
	data.PagerHidden = query.PagerHidden()
	data.Pager = listSpec.Pager(query, translate("partners.pager", "from", b.From+1, "to", b.To, "total", len(kept)), b)

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
		case sortProtocol:
			order = cmp.Or(strings.Compare(a.proto.Family.Label, b.proto.Family.Label), boolCmp(a.proto.TLS, b.proto.TLS))
		case sortAddress:
			order = cmp.Or(strings.Compare(a.agent.Address.Host, b.agent.Address.Host),
				cmp.Compare(a.agent.Address.Port, b.agent.Address.Port))
		case sortAccounts:
			order = cmp.Compare(a.accounts, b.accounts)
		default:
			order = byName(a, b)
		}

		if dir == dirDesc {
			order = -order
		}

		return cmp.Or(order, byName(a, b))
	})
}

// actions is the "⋯" menu of a partner, in the list (with "View the sheet"
// and "Edit") and on the sheet (the buttons are there). Entries follow the
// rights; the destructive one comes last. Every URL carries the list state.
func actions(translate gui.Translator, e entry, user rights, onDetail bool, carried string) []gui.Action {
	id := e.agent.ID

	var acts []gui.Action
	if !onDetail {
		acts = append(acts, gui.Action{
			Icon: "eye", Label: translate("partners.action.detail"), URL: withRet(partnerURL(id, ""), carried),
		})

		if user.Write {
			acts = append(acts, gui.Action{
				Icon: "pen", Label: translate("partners.action.edit"), URL: withRet(partnerURL(id, editSuffix), carried),
			})
		}
	}

	if user.Delete {
		acts = append(acts, gui.Action{
			Icon:   "trash",
			Label:  translate("partners.action.delete"),
			URL:    withRet(partnerURL(id, deleteSuffix), carried),
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
		return append(c, gui.Crumb{Label: translate("partners.title")})
	}

	return append(c,
		gui.Crumb{Label: translate("partners.title"), URL: listURL(carried)},
		gui.Crumb{Label: last})
}

// notices turns the result of the previous action, carried by the
// redirection, into notifications: on a partner, on one of its means or on
// one of its accounts.
func notices(translate gui.Translator, values url.Values) []gui.Notice {
	name := values.Get(nameParam)
	done := values.Get(doneParam)

	switch values.Get(kindParam) {
	case kindCred:
		if slices.Contains(doneKinds, done) {
			return []gui.Notice{{Kind: stateOK, Title: translate("auth.toast."+done, "name", name)}}
		}

		return nil
	case kindAccount:
		if slices.Contains(doneKinds, done) {
			return []gui.Notice{{Kind: stateOK, Title: translate("remote_accounts.toast."+done, "login", name)}}
		}

		return nil
	}

	if slices.Contains(doneKinds, done) {
		return []gui.Notice{{Kind: stateOK, Title: translate(toastKeyPrefix+done, "name", name)}}
	}

	return nil
}

// -------------------------------------------------------------- pages ----

// partner reads the {id} of the route; a missing partner renders the
// localised "not found" page and returns false.
func (h *Handlers) partner(w http.ResponseWriter, r *http.Request) (entry, bool) {
	id, parseErr := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

	agent, err := internal.GetPartnerByID(h.db, id)
	if parseErr == nil && err == nil {
		accounts, countErr := h.db.Count(&model.RemoteAccount{}).Where("remote_agent_id=?", agent.ID).Run()
		if countErr != nil {
			h.logger.Errorf("failed to count the accounts of the partner %d: %v", id, countErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return entry{}, false
		}

		return entryOf(agent, int(accounts)), true //nolint:gosec // a count of rows
	}

	if err != nil && !database.IsNotFound(err) {
		h.logger.Errorf("failed to retrieve the partner %d: %v", id, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return entry{}, false
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("partners.not_found.title")

	h.notFoundPage(w, r, crumbs(translate, title, carried), notFoundData{
		Title:     title,
		Text:      translate("partners.not_found.text"),
		Icon:      "globe",
		BackLabel: translate("partners.back"),
		BackURL:   listURL(carried),
	})

	return entry{}, false
}

// screen is an action page of a partner: the chrome of the screen around
// the data.
func (h *Handlers) screen(translate gui.Translator, title, carried string, status int) common.Screen {
	return common.Screen{
		Title:   title,
		Section: constants.SidebarSectionConnection,
		Menu:    constants.SidebarMenuPartners,
		Doc:     docPage,
		Crumbs:  crumbs(translate, title, carried),
		Status:  status,

		Stylesheets: sheetStylesheets(),
	}
}

// formStylesheets are the stylesheets of the form: the shared protocol
// options, the shared rules of a sheet, then the screen.
func formStylesheets() []string { return []string{protoStylesheet, means.Stylesheet, stylesheet} }

// clientCard says which local client the transfers to the partner go
// through, from the rule of the product: the one client of the protocol, or
// one of several the transfer must name, or none yet (the gateway creates
// one, named after the protocol, on the first transfer).
type clientCard struct {
	Text   string
	Links  []gui.Link // the clients of the protocol, linked to their sheet
	AddURL string     // "Add a client", when there is none and the user may
}

// clientCardOf builds the card from the clients of the protocol of the partner.
func clientCardOf(translate gui.Translator, e entry, clients []*model.Client, canAdd bool) clientCard {
	family := e.proto.Family.Label
	card := clientCard{}

	for _, c := range clients {
		card.Links = append(card.Links, gui.Link{
			Label: c.Name, URL: common.WebuiURL("/clients/" + strconv.FormatInt(c.ID, 10)),
		})
	}

	switch len(clients) {
	case 0:
		card.Text = translate("partners.client.none", "protocol", family, "name", e.agent.Protocol)
		if canAdd {
			card.AddURL = common.WebuiURL("/clients/new")
		}
	case 1:
		card.Text = translate("partners.client.one", "name", clients[0].Name)
	default:
		card.Text = translate("partners.client.many", "n", len(clients), "protocol", family)
	}

	return card
}

// rulesCard lists the transfer rules the partner may use: every rule, or
// the ones that name it in their accesses; the accesses are managed on the
// rules screen.
type rulesCard struct {
	All   bool
	Rules []string
	URL   string
}

func rulesCardOf(translate gui.Translator, rules []*model.Rule, total uint64) rulesCard {
	card := rulesCard{All: uint64(len(rules)) == total, URL: rulesURL()}

	for _, rule := range rules {
		direction := translate("partners.detail.rule_receive")
		if rule.IsSend {
			direction = translate("partners.detail.rule_send")
		}

		card.Rules = append(card.Rules, rule.Name+" ("+direction+")")
	}

	return card
}

type detailData struct {
	Name     string
	Protocol string // the family, for the title of the options card
	Tabs     []tab
	BackURL  string
	EditURL  string
	Actions  []gui.Action
	Identity []kv
	Client   clientCard
	Rules    rulesCard
	Options  []kv
}

// Detail serves GET /webui/partners/{id}: the "Details" tab of the sheet.
func (h *Handlers) Detail(w http.ResponseWriter, r *http.Request) {
	ptn, ok := h.partner(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	user := rightsOf(r)

	var clients model.Clients
	if err := h.db.Select(&clients).Where("protocol=?", ptn.agent.Protocol).OrderBy("name", true).Run(); err != nil {
		h.logger.Errorf("failed to list the clients of the protocol %q: %v", ptn.agent.Protocol, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	rules, err := ptn.agent.GetAuthorizedRules(h.db)
	if err != nil {
		h.logger.Errorf("failed to list the rules of the partner %d: %v", ptn.agent.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	total, err := h.db.Count(&model.Rule{}).Run()
	if err != nil {
		h.logger.Errorf("failed to count the rules: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	own := owner{ptn: ptn}

	tabs, err := h.tabsOf(translate, own, tabDetails, carried)
	if err != nil {
		h.logger.Errorf("%v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	canAddClient := common.GetUser(r).Permissions.HasPermission(model.PermServersWrite)
	data := detailData{
		Name:     ptn.agent.Name,
		Protocol: ptn.proto.Family.Label,
		Tabs:     tabs,
		BackURL:  listURL(carried),
		Actions:  actions(translate, ptn, user, true, carried),
		Identity: []kv{
			{Label: translate("partners.detail.protocol"), Value: protoform.Describe(translate, ptn.proto)},
			{Label: translate("partners.detail.address"), Value: ptn.address(), Mono: true},
			{Label: translate("partners.detail.accounts"), Value: ptn.accountsText(translate)},
		},
		Client:  clientCardOf(translate, ptn, clients, canAddClient),
		Rules:   rulesCardOf(translate, rules, total),
		Options: protoform.Rows(translate, protoform.RolePartner, ptn.proto, ptn.agent.ProtoConfig),
	}

	if user.Write {
		data.EditURL = withRet(partnerURL(ptn.agent.ID, editSuffix), carried)
	}

	screen := h.sheetScreen(translate, own.title(translate), own, "", carried, 0, notices(translate, r.URL.Query()))
	h.shell.Render(w, h.logger, r, screen, detailView(data))
}

// ------------------------------------------------------------- delete ----

type deleteData struct {
	Name         string
	Protocol     string
	Address      string
	Accounts     int
	CancelURL    string
	Action       string
	TransfersURL string
	// The model refuses the deletion: running transfers use one of its
	// accounts (Transfers), or a folder watcher does (Watchers).
	Transfers bool
	Watchers  bool
}

func (d deleteData) blocked() bool { return d.Transfers || d.Watchers }

// blockers tells what refuses the deletion of the partner, from the model's
// own rule (running transfers) and from the folder watchers that name one
// of its accounts (the foreign key restricts the deletion).
func (h *Handlers) blockers(e entry) (transfers, watchers bool, err error) {
	if delErr := e.agent.BeforeDelete(h.db); delErr != nil {
		if !database.IsValidationError(delErr) {
			return false, false, delErr //nolint:wrapcheck // logged by the caller
		}

		transfers = true
	}

	n, err := h.db.Count(&model.FileWatcher{}).Where(
		"remote_account_id IN (SELECT id FROM "+model.TableRemAccounts+" WHERE remote_agent_id=?)", e.agent.ID,
	).Run()
	if err != nil {
		return false, false, err //nolint:wrapcheck // logged by the caller
	}

	return transfers, n > 0, nil
}

func (h *Handlers) deletePage(w http.ResponseWriter, r *http.Request, e entry, transfers, watchers bool, status int) {
	translate := h.shell.Translator(r)
	carried := ret(r)

	data := deleteData{
		Name:         e.agent.Name,
		Protocol:     protoform.Describe(translate, e.proto),
		Address:      e.address(),
		Accounts:     e.accounts,
		CancelURL:    withRet(partnerURL(e.agent.ID, ""), carried),
		Action:       withRet(partnerURL(e.agent.ID, deleteSuffix), carried),
		TransfersURL: transfersURL(),
		Transfers:    transfers,
		Watchers:     watchers,
	}

	title := translate("partners.delete.title", "name", e.agent.Name)
	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, status), deleteView(data))
}

// DeleteForm serves GET /webui/partners/{id}/delete: the confirmation page,
// or the refusal when the partner is in use.
func (h *Handlers) DeleteForm(w http.ResponseWriter, r *http.Request) {
	e, ok := h.partner(w, r)
	if !ok {
		return
	}

	transfers, watchers, err := h.blockers(e)
	if err != nil {
		h.logger.Errorf("failed to check the use of the partner %d: %v", e.agent.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	h.deletePage(w, r, e, transfers, watchers, 0)
}

// Delete serves POST /webui/partners/{id}/delete. The accounts, the means
// and the rule accesses of the partner go with it.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	e, ok := h.partner(w, r)
	if !ok {
		return
	}

	transfers, watchers, err := h.blockers(e)
	if err == nil && !transfers && !watchers {
		err = internal.DeletePartner(h.db, e.agent)
	}

	switch {
	case err == nil && !transfers && !watchers:
		redirectList(w, r, ret(r), url.Values{doneParam: {doneDeleted}, nameParam: {e.agent.Name}})
	case err == nil || database.IsValidationError(err):
		h.deletePage(w, r, e, transfers || database.IsValidationError(err), watchers, http.StatusUnprocessableEntity)
	default:
		h.logger.Errorf("failed to delete the partner %d: %v", e.agent.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
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

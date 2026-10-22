package transfers

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// The parameters of the filters of the list: the state (the row of the
// states) and the criteria of the filters panel.
const (
	paramState    = "state"
	paramRule     = "rule"
	paramDir      = "direction" // "dir" is the order of the sort
	paramPartner  = "partner"
	paramAccount  = "account"
	paramProtocol = "protocol"
	paramFrom     = "from"
	paramTo       = "to"

	dirSend    = "send"
	dirReceive = "receive"

	// The layout of a datetime-local field, in the time zone of the
	// gateway.
	localLayout = "2006-01-02T15:04"
)

//nolint:gochecknoglobals // reference data of the screen
var filterParams = []string{
	paramState, paramRule, paramDir, paramPartner, paramAccount, paramProtocol, paramFrom, paramTo,
}

// filters is the part of the state of the list that belongs to the screen.
type filters struct {
	State    string // key of the row of the states, "" for every state
	Rule     string
	Dir      string // dirSend, dirReceive or ""
	Partner  string // the remote partner or the local server
	Account  string
	Protocol string // a protocol family of the gateway, which covers its TLS variant
	From     string // datetime-local values (localLayout)
	To       string
}

// parseFilters reads the filters of a query, dropping what is unknown or
// invalid.
func parseFilters(v url.Values) *filters {
	crit := &filters{
		Rule:    strings.TrimSpace(v.Get(paramRule)),
		Partner: strings.TrimSpace(v.Get(paramPartner)),
		Account: strings.TrimSpace(v.Get(paramAccount)),
	}

	if _, ok := specOfKey(v.Get(paramState)); ok {
		crit.State = v.Get(paramState)
	}

	if dir := v.Get(paramDir); dir == dirSend || dir == dirReceive {
		crit.Dir = dir
	}

	crit.Protocol = protoform.FamilyKeyOf(v.Get(paramProtocol))

	if _, ok := parseLocal(v.Get(paramFrom)); ok {
		crit.From = v.Get(paramFrom)
	}

	if _, ok := parseLocal(v.Get(paramTo)); ok {
		crit.To = v.Get(paramTo)
	}

	return crit
}

// parseLocal reads a datetime-local value in the time zone of the gateway.
func parseLocal(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}

	trans, err := time.ParseInLocation(localLayout, value, time.Local)

	return trans, err == nil
}

// criteria are the filters with their parameter, in the order of the
// parameters.
func (f *filters) criteria() map[string]string {
	return map[string]string{
		paramState: f.State, paramRule: f.Rule, paramDir: f.Dir, paramPartner: f.Partner,
		paramAccount: f.Account, paramProtocol: f.Protocol, paramFrom: f.From, paramTo: f.To,
	}
}

// Values encodes the filters, nothing when none is set.
func (f *filters) Values() url.Values {
	v := url.Values{}

	for name, value := range f.criteria() {
		if value != "" {
			v.Set(name, value)
		}
	}

	return v
}

// Count is the number of the criteria of the filters panel; the state has
// its own row.
func (f *filters) Count() int {
	n := 0

	for name, value := range f.criteria() {
		if value != "" && name != paramState {
			n++
		}
	}

	return n
}

// filtersOf is the filters of a list query.
func filtersOf(q *liststate.Query) *filters {
	if crit, ok := q.Filters.(*filters); ok {
		return crit
	}

	return &filters{}
}

// selectQuery is the query of the transfers the list state asks for, sorted,
// in every state or in one.
func (h *Handlers) selectQuery(q *liststate.Query, crit *filters, status types.TransferStatus,
) *internal.TransferListQuery {
	query := internal.StartTransferQuery(h.db, sortExpression(q.Sort), q.Dir == liststate.DirAsc)

	if q.Q != "" {
		query.Search(q.Q)
	}

	if status != "" {
		query.Status(status)
	}

	if crit.Rule != "" {
		query.RuleName(crit.Rule)
	}

	if crit.Dir != "" {
		query.Direction(crit.Dir == dirSend)
	}

	if crit.Partner != "" {
		query.Requested(crit.Partner)
	}

	if crit.Account != "" {
		query.Requester(crit.Account)
	}

	if crit.Protocol != "" {
		query.Protocols(familyProtocols(crit.Protocol)...)
	}

	if from, ok := parseLocal(crit.From); ok {
		query.StartedFrom(from)
	}

	// The end of a period includes its last minute.
	if to, ok := parseLocal(crit.To); ok {
		query.StartedUntil(to.Add(time.Minute - time.Nanosecond))
	}

	return query
}

// familyProtocols are the protocol keys of a family: its plain and its TLS
// variants, those the gateway registers.
func familyProtocols(family string) []string {
	var keys []string

	for _, f := range protoform.Families() {
		if f.Key == family {
			keys = slices.DeleteFunc([]string{f.Plain, f.TLS}, func(key string) bool { return key == "" })
		}
	}

	return keys
}

// countStates counts the transfers the search and the filters match, in
// each state (the state filter apart): the row of the states.
func (h *Handlers) countStates(q *liststate.Query, crit *filters) (map[string]uint64, error) {
	counts := map[string]uint64{}

	for _, spec := range stateSpecs() {
		n, err := h.selectQuery(q, crit, spec.status).Count()
		if err != nil {
			return nil, fmt.Errorf("failed to count the transfers %s: %w", spec.status, err)
		}

		counts[spec.key] = n
	}

	return counts, nil
}

// stateItem is a state of the row of the states: its label, its count, the
// list in that state.
type stateItem struct {
	Key     string
	Label   string
	Hint    string // what the state means, as a tooltip
	Count   string
	Alert   bool // failed transfers: the label carries the shape of the error status
	URL     string
	Current bool
}

// row is a transfer of the list, on one line (Driss, 2026-10-22): what does
// not fit goes to the tooltips.
type row struct {
	ID         int64
	File       string // the file where it goes (destination), cut at its start when too long
	FileTitle  string // the same, whole
	Name       string // the name of the file, for the label of the actions
	URL        string
	Status     gui.Status
	StatusTip  string // the cause and the message of an error
	DirIcon    string // the direction, a cloud when the gateway is the server
	DirTip     string // the direction and the role of the gateway
	Rule       string
	RuleTip    string // the rule, the partner and the account (the partner folds on a narrow list)
	Partner    string
	PartnerTip string // the partner and the account
	Progress   progressView
	Size       string // the size of the file once known
	SizeTitle  string // the size to the byte
	Start      moment
	End        moment // the end, or the next attempt (Next)
	Next       bool
	Actions    []gui.Action
}

// moment is a date in a cell: its date and its time to the second, the
// sentence as a tooltip.
type moment struct {
	Text  string
	Title string
}

// progressView is the progress of a transfer: a bar and a percentage when the
// size is known, the bytes otherwise; the size under it.
type progressView struct {
	Bar   bool
	Pct   int
	Label string
	Size  string
	None  bool // nothing transferred and nothing to say: "—"
}

type listData struct {
	RefreshedAt    string
	RefreshSeconds int
	FragmentURL    string
	AddURL         string // empty without the write right
	PreregisterURL string

	States []stateItem

	Action       string // toolbar form action: the bare list
	Base         string // list with search and filters: base of the sort links
	Q            string
	Hidden       map[string]string // carried by the toolbar form
	Filters      filterView
	FiltersOpen  bool
	FiltersCount int
	ResetURL     string
	Sort         string
	Dir          string
	Count        string
	Rows         []row
	Caption      string
	Pager        *liststate.Pager
	PagerHidden  map[string]string
	TopHidden    map[string]string // the foot at the top carries the rows per page too
	Sizes        []int
	Size         int
	Empty        bool // no transfer at all
	NoMatch      bool // transfers exist, none matches
	NoMatchText  string
	NoMatchURL   string
	NoMatchLink  string
	Error        bool // the transfers could not be read
	RetryURL     string
}

// filterView is the filters panel: the current criteria and the choices.
type filterView struct {
	Rule     string
	Rules    []string
	Dir      string
	Partner  string
	Partners []string // the remote partners
	Servers  []string // the local servers
	Account  string
	Accounts []accountGroup
	// A partner or an account of the address that the lists do not hold (a
	// partner deleted since) shows alone, chosen: the filter in use stays seen.
	PartnerExtra string
	AccountExtra string
	AccountGroup int // the group that carries the chosen account: a login under several partners is chosen once
	Protocol     string
	Protocols    []gui.Choice
	From         string
	To           string
}

// accountGroup is the accounts of a partner or of a server, in the filters.
type accountGroup struct {
	Agent  string
	Logins []string
}

// List serves GET /webui/transfers; with ?fragment=list, the refreshed part
// alone (the row of the states, the count, the table and its foot).
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	fragment := r.URL.Query().Get(constants.FragmentParam) == listFragment
	if !fragment && listSpec.Restore(w, r) {
		return
	}

	translate := h.shell.Translator(r)
	query := listSpec.Parse(r.URL.Query())
	data := h.buildList(translate, query, canWrite(r))

	if fragment {
		// The state the script asks for (a state, a sort or a page loaded in
		// place) is the one to find again when coming back to the screen.
		if listSpec.HasParams(r.URL.Query()) {
			listSpec.KeepState(w, query)
		}

		h.shell.RenderFragment(w, h.logger, r, listBlock(data))

		return
	}

	screen := h.screen(translate("transfers.title"), crumbs(translate, "", ""), 0)
	screen.Notices = notices(translate, r.URL.Query())
	screen.Scripts = []string{script}

	if data.Error {
		screen.Status = http.StatusInternalServerError
	}

	h.shell.Render(w, h.logger, r, screen, listView(data))
}

// buildList reads and lays out one page of the list: the counts of the
// states, the page of transfers, the toolbar state, the row actions.
func (h *Handlers) buildList(translate gui.Translator, query *liststate.Query, write bool) listData {
	crit := filtersOf(query)
	carried := query.Encode(query.Page)
	data := listData{
		RefreshedAt:    time.Now().Format(time.TimeOnly),
		RefreshSeconds: int(refreshPeriod.Seconds()),
		FragmentURL:    listSpec.ListURLOf(query.WithPage(query.Page)),
		Action:         listSpec.ListURLOf(nil),
		Base:           listSpec.ListURLOf(query.FilterValues()),
		Q:              query.Q,
		Sort:           query.Sort,
		Dir:            query.Dir,
		Size:           query.Size,
		ResetURL:       listSpec.ResetURL(),
		Hidden:         query.ToolbarHidden(),
		Filters:        h.filterView(crit),
		FiltersCount:   crit.Count(),
		FiltersOpen:    crit.Count() > 0,
	}

	// The state has its own row: the toolbar form carries it.
	if crit.State != "" {
		data.Hidden[paramState] = crit.State
	}

	if write {
		data.AddURL = withRet(common.WebuiURL(pagePath+"/new"), carried)
		data.PreregisterURL = withRet(common.WebuiURL(pagePath+"/preregister"), carried)
	}

	counts, err := h.countStates(query, crit)
	if err != nil {
		h.logger.Errorf("failed to count the transfers: %v", err)

		data.Error, data.RetryURL = true, data.FragmentURL

		return data
	}

	var all, shown uint64
	for _, n := range counts {
		all += n
	}

	shown = all
	if crit.State != "" {
		shown = counts[crit.State]
	}

	data.States = stateItems(translate, query, counts, all)
	data.Count = countText(translate, shown)

	if shown == 0 {
		h.noRows(translate, &data, query, crit)

		return data
	}

	bounds := query.Bounds(int(shown))

	spec, _ := specOfKey(crit.State)

	transfers, err := h.selectQuery(query, crit, spec.status).Limit(query.Size, bounds.From).Run()
	if err != nil {
		h.logger.Errorf("failed to list the transfers: %v", err)

		data.Error, data.RetryURL = true, data.FragmentURL

		return data
	}

	carried = query.Encode(bounds.Page)
	data.Caption = translate("transfers.caption",
		"page", bounds.Page, "pages", bounds.Pages, "sort", translate(sortKeyPrefix+query.Sort))

	for _, trans := range transfers {
		data.Rows = append(data.Rows, rowOf(translate, trans, write, carried))
	}

	data.Sizes = pageSizes
	data.PagerHidden = query.PagerHidden()
	data.TopHidden = query.PagerHidden()

	// The foot at the top has no rows-per-page choice: it carries the one in
	// use.
	if size := query.Values().Get(liststate.SizeParam); size != "" {
		data.TopHidden[liststate.SizeParam] = size
	}

	data.Pager = listSpec.Pager(query, translate("transfers.pager",
		"from", groupDigits(translate, uint64(bounds.From+1)), "to", groupDigits(translate, uint64(bounds.To)),
		"total", groupDigits(translate, shown)), bounds)

	return data
}

// noRows fills the empty states: no transfer at all, or none matching the
// search, the state or the filters, with the way out.
func (h *Handlers) noRows(translate gui.Translator, data *listData, query *liststate.Query, crit *filters) {
	switch {
	case query.Q == "" && crit.Count() == 0 && crit.State == "":
		data.Empty = true
	case query.Q != "":
		withoutQ := *query
		withoutQ.Q = ""
		data.NoMatch = true
		data.NoMatchText = translate("transfers.none.query", "query", query.Q)
		data.NoMatchURL, data.NoMatchLink = listSpec.ExplicitListURL(withoutQ.Values()), translate("search.clear")
	case crit.Count() == 0:
		data.NoMatch = true
		data.NoMatchText = translate("transfers.none.state", "state", translate(statesKeyPrefix+crit.State))
		data.NoMatchURL, data.NoMatchLink = stateURL(query, ""), translate("transfers.none.all_states")
	default:
		data.NoMatch = true
		data.NoMatchText = translate("transfers.none.filters")
		data.NoMatchURL, data.NoMatchLink = listSpec.ResetURL(), translate("filters.reset")
	}
}

// stateItems is the row of the states: every state, then each one, with its
// count; a click shows the list in that state from its first page, the
// search, the filters and the sort kept.
func stateItems(translate gui.Translator, query *liststate.Query, counts map[string]uint64, all uint64) []stateItem {
	current := filtersOf(query).State
	items := []stateItem{{
		Label: translate("transfers.states.all"), Hint: translate(stateHintPrefix + "all"),
		Count: groupDigits(translate, all), URL: stateURL(query, ""), Current: current == "",
	}}

	for _, spec := range stateSpecs() {
		items = append(items, stateItem{
			Key: spec.key, Label: translate(statesKeyPrefix + spec.key), Hint: translate(stateHintPrefix + spec.key),
			Count: groupDigits(translate, counts[spec.key]), URL: stateURL(query, spec.key), Current: current == spec.key,
			Alert: spec.shape == stateKO && counts[spec.key] > 0,
		})
	}

	return items
}

// stateURL is the list in a state ("" for every state), from its first
// page.
func stateURL(query *liststate.Query, key string) string {
	v := query.Values()
	if key == "" {
		v.Del(paramState)
	} else {
		v.Set(paramState, key)
	}

	return listSpec.ExplicitListURL(v)
}

// countText is the count of the list.
func countText(translate gui.Translator, n uint64) string {
	if n == 1 {
		return translate("transfers.count_one")
	}

	return translate("transfers.count", "total", groupDigits(translate, n))
}

// filterView reads the choices of the filters panel: the names of the rules,
// of the partners and of the servers, the protocols of the gateway (option A
// of the screen sheet: offered to whoever reads the transfers, which show
// these names anyway).
func (h *Handlers) filterView(crit *filters) filterView {
	view := filterView{
		Rule: crit.Rule, Dir: crit.Dir, Partner: crit.Partner, Account: crit.Account, Protocol: crit.Protocol,
		From: crit.From, To: crit.To,
	}

	if rules, err := internal.ListRules(h.db, "name", true, 0, 0); err == nil {
		for _, rule := range rules {
			if !slices.Contains(view.Rules, rule.Name) {
				view.Rules = append(view.Rules, rule.Name)
			}
		}
	} else {
		h.logger.Warningf("failed to list the rules for the filters: %v", err)
	}

	view.Partners, view.Servers, view.Accounts = h.agentChoices(crit.Partner)
	view.AccountGroup = firstGroupOf(view.Accounts, crit.Account)

	known := slices.Contains(view.Partners, crit.Partner) || slices.Contains(view.Servers, crit.Partner)
	if crit.Partner != "" && !known {
		view.PartnerExtra = crit.Partner
	}

	if crit.Account != "" && view.AccountGroup < 0 {
		view.AccountExtra = crit.Account
	}

	// The families, each covering its TLS variant, as the filters of the
	// servers and the clients.
	for _, family := range protoform.Families() {
		view.Protocols = append(view.Protocols, gui.Choice{Value: family.Key, Label: family.Label})
	}

	return view
}

// agentChoices are the remote partners and the local servers of the filters,
// in order, and their accounts: those of the partner or of the server chosen
// alone when there is one, the filters applying as soon as they change
// (Driss, 2026-10-22: the partner and the account chosen in lists, as the
// rule and the direction).
func (h *Handlers) agentChoices(chosen string) (partners, servers []string, accounts []accountGroup) {
	if list, err := internal.ListPartners(h.db, "name", true, 0, 0); err == nil {
		for _, partner := range list {
			partners = append(partners, partner.Name)

			if chosen != "" && chosen != partner.Name {
				continue
			}

			if logins, accErr := internal.ListPartnerAccounts(h.db, partner.Name, "login", true, 0, 0); accErr == nil {
				accounts = appendAccounts(accounts, partner.Name, logins, func(acc *model.RemoteAccount) string {
					return acc.Login
				})
			}
		}
	} else {
		h.logger.Warningf("failed to list the partners for the filters: %v", err)
	}

	if list, err := internal.ListServers(h.db, "name", true, 0, 0); err == nil {
		for _, server := range list {
			servers = append(servers, server.Name)

			if chosen != "" && chosen != server.Name {
				continue
			}

			if logins, accErr := internal.ListServerAccounts(h.db, server.Name, "login", true, 0, 0); accErr == nil {
				accounts = appendAccounts(accounts, server.Name, logins, func(acc *model.LocalAccount) string {
					return acc.Login
				})
			}
		}
	} else {
		h.logger.Warningf("failed to list the servers for the filters: %v", err)
	}

	return partners, servers, accounts
}

// firstGroupOf is the index of the first group that holds a login, -1 for
// none: a login under several partners is chosen once.
func firstGroupOf(groups []accountGroup, login string) int {
	return slices.IndexFunc(groups, func(group accountGroup) bool { return slices.Contains(group.Logins, login) })
}

// appendAccounts adds the accounts of a partner or of a server to the
// filters, when it has some.
func appendAccounts[A any](groups []accountGroup, agent string, accounts []A, login func(A) string) []accountGroup {
	if len(accounts) == 0 {
		return groups
	}

	group := accountGroup{Agent: agent}
	for _, acc := range accounts {
		group.Logins = append(group.Logins, login(acc))
	}

	return append(groups, group)
}

// rowOf lays out a transfer of the list, in the order of the columns of the
// historic screen (Driss, 2026-10-22): the direction as an icon, the
// number, the state, the file, the rule, the partner, the start, the end,
// the progress and the size. The state is strong, its label in the colour
// of the state; a failed transfer says "Failed" alone, its cause and message
// in the tooltip (Driss, 2026-10-22: no part of the error under the state).
func rowOf(translate gui.Translator, trans *model.NormalizedTransferView, write bool, carried string) row {
	// The direction and the role as one icon, in the words of the historic
	// screen in the tooltip ("Sending from the server to the client").
	dir, icon, role := "receive", "download", "client"
	if trans.IsSend {
		dir, icon = "send", "upload"
	}

	if trans.IsServer {
		role, icon = "server", "cloud-"+icon
	}

	// The protocol in the tooltip of the partner, with the account (Driss,
	// 2026-10-22), not in that of the file.
	partnerTip := translate("transfers.partner_tip", "partner", trans.Agent, "account", trans.Account)
	if label := protocolLabel(trans.Protocol); label != "" {
		partnerTip += " · " + label
	}
	progress := progressOf(translate, trans)

	line := row{
		ID:         trans.ID,
		File:       destination(trans),
		FileTitle:  destination(trans),
		Name:       fileName(trans),
		URL:        withRet(transferURL(trans.ID, ""), carried),
		Status:     statusOf(translate, trans.Status),
		DirIcon:    icon,
		DirTip:     translate("transfers.dir_tip." + role + "_" + dir),
		Rule:       trans.Rule,
		RuleTip:    trans.Rule + " · " + partnerTip,
		Partner:    trans.Agent,
		PartnerTip: partnerTip,
		Progress:   progress,
		Size:       progress.Size,
		Start:      momentOf(translate, trans.Start),
		Actions:    rowActions(translate, trans, write, carried),
	}
	line.Status.Strong = true

	if line.Size != "" {
		line.SizeTitle = exactSize(translate, trans.Filesize)
	}

	if trans.Status == types.StatusError {
		line.StatusTip = errorLabel(translate, trans.ErrCode)

		if trans.ErrDetails != "" {
			line.StatusTip = translate("transfers.error_tip", "cause", line.StatusTip, "message", trans.ErrDetails)
		}
	}

	// The due date of a preregistered transfer, its start being the
	// connection of the partner.
	if trans.Status == types.StatusAvailable {
		line.Start.Title = translate("transfers.due_title", "date", line.Start.Title)
	}

	switch {
	case !trans.Stop.IsZero() && !trans.IsTransfer:
		line.End = momentOf(translate, trans.Stop)
	case trans.Status == types.StatusError && !trans.NextRetry.IsZero() && trans.RemainingTries > 0:
		line.End, line.Next = momentOf(translate, trans.NextRetry), true
		line.End.Title = translate("transfers.next_title", "date", line.End.Title)
	}

	return line
}

// momentOf writes a moment in a cell, in the time zone of the gateway: the
// same format on every line, the date and the time to the second (Driss,
// 2026-10-22: a time without its seconds may mislead); the sentence as a
// tooltip.
func momentOf(translate gui.Translator, at time.Time) moment {
	if at.IsZero() {
		return moment{}
	}

	clock, date := timeAndDate(translate, at)

	return moment{Text: translate("transfers.date_cell", "date", date, "time", clock), Title: dateTime(translate, at)}
}

// progressOf is the progress of a transfer.
func progressOf(translate gui.Translator, trans *model.NormalizedTransferView) progressView {
	// The size once known: a transfer that has not started yet may hold 0.
	view := progressView{}
	if trans.Filesize > 0 || (trans.Filesize == 0 && trans.Status == types.StatusDone) {
		view.Size = formatSize(translate, trans.Filesize)
	}

	switch {
	case trans.Filesize > 0:
		view.Bar = true
		view.Pct = int(min(max(trans.Progress*100/trans.Filesize, 0), 100)) //nolint:mnd // a percentage
		view.Label = translate("transfers.percent", "pct", view.Pct)
	case trans.Progress > 0:
		view.Label = formatSize(translate, trans.Progress)
	case trans.Status == types.StatusDone:
		view.Bar, view.Pct = true, 100
		view.Label = translate("transfers.percent", "pct", view.Pct)
	default:
		view.None = true
	}

	return view
}

// timeAndDate is a moment in the time zone of the gateway: its time and its
// date; "" for none.
func timeAndDate(translate gui.Translator, moment time.Time) (clock, date string) {
	if moment.IsZero() {
		return "", ""
	}

	local := moment.Local()

	return local.Format(time.TimeOnly), local.Format(translate("transfers.date"))
}

// dateTime is a moment in the time zone of the gateway, in one text.
func dateTime(translate gui.Translator, moment time.Time) string {
	if moment.IsZero() {
		return ""
	}

	clock, date := timeAndDate(translate, moment)

	return translate("transfers.date_time", "date", date, "time", clock)
}

// groupDigits writes a count with the thousands separator of the language.
func groupDigits(translate gui.Translator, n uint64) string {
	digits := strconv.FormatUint(n, 10)
	sep := translate("transfers.thousands")

	var out strings.Builder

	for i, c := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out.WriteString(sep)
		}

		out.WriteRune(c)
	}

	return out.String()
}

// exactSize is the size of a file to the byte, its digits grouped: the
// tooltip of the size in the list.
func exactSize(translate gui.Translator, n int64) string {
	if n < 2 { //nolint:mnd // 0 and 1 are singular in French
		return translate("transfers.size.byte", "n", n)
	}

	return translate("transfers.size.bytes", "n", groupDigits(translate, uint64(n)))
}

// formatSize writes a size in bytes, then in the largest unit that keeps it
// above one, with one decimal under ten.
func formatSize(translate gui.Translator, n int64) string {
	switch {
	case n < 2: //nolint:mnd // 0 and 1 are singular in French
		return translate("transfers.size.byte", "n", n)
	case n < 1024: //nolint:mnd // one kibibyte
		return translate("transfers.size.bytes", "n", n)
	}

	value := float64(n)
	units := []string{"kb", "mb", "gb", "tb"}
	unit := ""

	for _, u := range units {
		value /= 1024
		unit = u

		if value < 1024 { //nolint:mnd // one more unit
			break
		}
	}

	text := strconv.FormatFloat(value, 'f', 0, 64)
	if value < 10 { //nolint:mnd // one decimal under ten
		text = strings.Replace(strconv.FormatFloat(value, 'f', 1, 64), ".", translate("transfers.decimal"), 1)
	}

	return translate("transfers.size."+unit, "n", text)
}

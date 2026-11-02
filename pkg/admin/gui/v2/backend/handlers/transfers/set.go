package transfers

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/batch"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/format"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
)

// The grouped actions of the list (#792): the transfers the filters define,
// minus the rows the operator unchecks, or the rows checked alone, resumed,
// retried or cancelled at once, with the row selection of the kit (0.21.0).
// The resumption acts in one click when the set still counts what the page
// showed, and asks otherwise; the retry (a date) and the cancellation
// (destructive, D10) go through their page.
const (
	// The form of the selection bar, which the row checkboxes name.
	selectionID = "sel-transfers"

	// The parameter that carries how many transfers the page showed in the
	// selection: the server acts at once when the set still counts that many,
	// and asks otherwise. Without it (no JavaScript), the server asks.
	expectedParam = "expected"

	// How long the running transfers of a set may take to stop.
	setTimeout = 10 * time.Second

	// The results of the grouped actions, carried by the redirection (see
	// Notices).
	setResumed   = "set_resumed"
	setRetried   = "set_retried"
	setCancelled = "set_cancelled"
	setEmpty     = "set_empty"

	// Prefixes of the catalogue keys of the set pages and notifications.
	setKeyPrefix   = "transfers.set."
	asideKeyPrefix = "transfers.set.aside." // what a gesture leaves aside, by reason
)

// errBadForm is a request whose form cannot be read.
var errBadForm = errors.New("the form cannot be read")

// selection is the set a request names: the state of the list it was made on
// (the filters, the sort), what the form sent (the whole filter minus
// exceptions, or the rows checked), the identifiers of the set and the
// exceptions that belong to the filter, named on the pages.
type selection struct {
	query    *liststate.Query
	crit     *filters
	state    gui.SelectionState
	ids      []int64
	excepted []int64
}

// readSelection reads the selection of a request (a GET or a POST) and
// enumerates its transfers: those of the filters, minus the exceptions, or
// the rows checked that belong to the filters; an identifier out of the
// filters is ignored.
func (h *Handlers) readSelection(r *http.Request) (*selection, error) {
	if err := r.ParseForm(); err != nil {
		return nil, errBadForm
	}

	sel := &selection{query: listSpec.Parse(r.Form), state: gui.ParseSelection(r.Form)}
	sel.crit = filtersOf(sel.query)

	checked := parseIDs(sel.state.IDs)
	if sel.state.Empty() || (!sel.state.All && len(checked) == 0) {
		return sel, nil
	}

	// The identifiers alone, by identifier: a DISTINCT cannot order by a
	// column it does not read.
	byID := *sel.query
	byID.Sort, byID.Dir = sortID, liststate.DirAsc
	spec, _ := specOfKey(sel.crit.State)
	query := h.selectQuery(&byID, sel.crit, spec.status)

	if !sel.state.All {
		query.Among(checked...)
	}

	all, err := query.IDs()
	if err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller
	}

	if !sel.state.All {
		sel.ids = all

		return sel, nil
	}

	except := parseIDs(sel.state.Except)

	for _, id := range all {
		if slices.Contains(except, id) {
			sel.excepted = append(sel.excepted, id)
		} else {
			sel.ids = append(sel.ids, id)
		}
	}

	return sel, nil
}

// parseIDs reads identifiers, dropping what is not one.
func parseIDs(values []string) []int64 {
	ids := make([]int64, 0, len(values))

	for _, value := range values {
		if id, err := strconv.ParseInt(value, 10, 64); err == nil && id > 0 {
			ids = append(ids, id)
		}
	}

	return ids
}

// carried is the state of the list the selection was made on, for the way
// back.
func (s *selection) carried() string { return s.query.Encode(s.query.Page) }

// hidden are the fields a set page sends again: the state of the list, the
// selection as the form sent it, and the count of the set as the page says
// it (expected), so that the action runs at once when the set holds.
func (s *selection) hidden() []liststate.Hidden {
	fields := liststate.SortedHidden(flatten(s.query.WithPage(s.query.Page)))

	if s.state.All {
		fields = append(fields, liststate.Hidden{Name: "all", Value: "1"})

		for _, id := range s.excepted {
			fields = append(fields, liststate.Hidden{Name: "except", Value: strconv.FormatInt(id, 10)})
		}
	} else {
		for _, id := range s.ids {
			fields = append(fields, liststate.Hidden{Name: "ids", Value: strconv.FormatInt(id, 10)})
		}
	}

	return append(fields, liststate.Hidden{Name: expectedParam, Value: strconv.Itoa(len(s.ids))})
}

// flatten keeps the first value of each parameter.
func flatten(values url.Values) map[string]string {
	flat := make(map[string]string, len(values))
	for name, list := range values {
		flat[name] = list[0]
	}

	return flat
}

// selectionProps is the selection bar of the list: the plural name of the
// objects, the total of the filter (the live list keeps it up to date), the
// current state of the list as hidden fields and the three gestures: the
// resumption posts, the retry and the cancellation open their page.
func selectionProps(translate gui.Translator, query *liststate.Query, total int) *gui.SelectionProps {
	return &gui.SelectionProps{
		ID: selectionID, Objects: translate(setKeyPrefix + "objects"), Total: total, Hidden: flatten(query.Values()),
		Actions: []gui.SelectionAction{
			{Label: translate(setKeyPrefix + "resume"), Action: common.WebuiURL(pagePath + "/resume")},
			{Label: translate(setKeyPrefix + "retry"), Action: common.WebuiURL(pagePath + "/retry"), Get: true},
			{Label: translate(setKeyPrefix + "cancel"), Action: common.WebuiURL(pagePath + "/cancel"), Get: true},
		},
	}
}

// expectedOf reads the count the page showed, if the request carries one.
func expectedOf(r *http.Request) (int, bool) {
	n, err := strconv.Atoi(r.Form.Get(expectedParam))

	return n, err == nil && n >= 0
}

// ResumeFailedForm is the form another screen (the dashboard, #818) posts to
// resume every failed transfer at once, coming back to it: the address of the
// grouped resumption and its fields (the failed state, the whole filter, the
// count of failed transfers the screen showed).
func ResumeFailedForm(expected uint64) (action string, fields []liststate.Hidden) {
	return actionURLFrom(string(batch.Resume), fromDashboard), []liststate.Hidden{
		{Name: paramState, Value: StateError},
		{Name: "all", Value: "1"},
		{Name: expectedParam, Value: strconv.FormatUint(expected, 10)},
	}
}

// actionURLFrom is the address of a gesture on a set, which keeps where the
// gesture comes back to ("" for the list).
func actionURLFrom(name, from string) string {
	action := common.WebuiURL(pagePath + "/" + name)
	if from != "" {
		action = liststate.WithQuery(action, url.Values{fromParam: {from}})
	}

	return action
}

// fromDashboardSet tells whether a gesture on a set was asked from the
// dashboard, where it comes back. The parameter is read on the address alone:
// the filters of the list name their period "from" too, in the form.
func fromDashboardSet(r *http.Request) bool { return r.URL.Query().Get(fromParam) == fromDashboard }

// originOf is where a gesture on a set was asked from, as actionURLFrom
// carries it: the dashboard, or "" for the list.
func originOf(r *http.Request) string {
	if fromDashboardSet(r) {
		return fromDashboard
	}

	return ""
}

// backOf is where a gesture on the selection comes back to: the dashboard
// when it was asked from there, else the list as the user left it.
func backOf(r *http.Request, sel *selection) string {
	if fromDashboardSet(r) {
		return common.WebuiURL(constants.DashboardPath)
	}

	return listURL(sel.carried())
}

// redirectBack sends back where the gesture was asked, with its result.
func redirectBack(w http.ResponseWriter, r *http.Request, sel *selection, values url.Values) {
	if fromDashboardSet(r) {
		liststate.RedirectTo(w, r, common.WebuiURL(constants.DashboardPath), values)

		return
	}

	listSpec.RedirectList(w, r, sel.carried(), values)
}

// ------------------------------------------------------------- pages ----

// setData is a page of a grouped action: the set in clear (the filters, the
// exceptions, the count), what the gesture will do and leave aside, the form
// that sends the set again.
type setData struct {
	Title     string
	Text      string // what the gesture does
	Changed   string // the set changed since the page was shown
	Action    string
	CancelURL string
	Hidden    []liststate.Hidden
	Rows      []kv // the set in clear
	Will      kv   // the transfers the gesture will act on
	Running   kv   // the running transfers a cancellation stops; empty when none
	Aside     []kv // what the gesture leaves aside, by reason
	Date      *field
	Danger    bool
	Label     string
}

// renderSet renders the page of a gesture on the selection: the confirmation
// of the resumption (when the set changed, or when the page did not say its
// count), the retry page, the confirmation of the cancellation.
//
//nolint:funlen // the page, laid out field by field
func (h *Handlers) renderSet(w http.ResponseWriter, r *http.Request, sel *selection, gesture batch.Gesture,
	data *setData, status int,
) {
	translate := h.shell.Translator(r)
	preview, err := batch.PreviewOf(h.db, gesture, sel.ids)
	if err != nil {
		h.logger.Errorf("failed to count the transfers of the set: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	name := string(gesture)
	data.Title = countedText(translate, setKeyPrefix+name+"_title", len(sel.ids))
	data.Text = translate(setKeyPrefix + name + "_text")
	data.Action = actionURLFrom(name, originOf(r))
	data.CancelURL = backOf(r, sel)
	data.Hidden = sel.hidden()
	data.Rows = h.setRows(translate, sel)
	data.Will = kv{Label: translate(setKeyPrefix + "will_" + name), Value: digits(translate, preview.Accepted)}
	data.Label = translate(setKeyPrefix + name)

	if gesture == batch.Cancel {
		data.Danger = true

		if preview.Running > 0 {
			data.Running = kv{Label: translate(setKeyPrefix + "running"), Value: digits(translate, preview.Running)}
		}
	}

	for _, reason := range batch.Reasons() {
		if n := preview.Refused[reason]; n > 0 {
			data.Aside = append(data.Aside, kv{Label: translate(asideKeyPrefix + string(reason)), Value: digits(translate, n)})
		}
	}

	if expected, ok := expectedOf(r); ok && expected != len(sel.ids) {
		data.Changed = translate(setKeyPrefix+"changed", "n", len(sel.ids), "expected", expected)
	}

	screen := h.screen(data.Title, crumbs(translate, data.Title, sel.carried()), status)
	h.shell.Render(w, h.logger, r, screen, setView(data))
}

// countedText is the text of a key with a count, the singular key (_one)
// for one.
func countedText(translate gui.Translator, key string, n int) string {
	if n == 1 {
		return translate(key + "_one")
	}

	return translate(key, "n", digits(translate, n))
}

// digits writes a count, its digits grouped.
func digits(translate gui.Translator, n int) string {
	return format.GroupDigits(translate, uint64(max(n, 0)))
}

// setRows says the set in clear: the state, the search and the criteria of
// the filters in use, the exceptions, the count.
func (h *Handlers) setRows(translate gui.Translator, sel *selection) []kv {
	crit := sel.crit
	state := translate("transfers.states.all")

	if crit.State != "" {
		state = translate(statesKeyPrefix + crit.State)
	}

	rows := []kv{{Label: translate("transfers.col.state"), Value: state}}
	add := func(key, value string) {
		if value != "" {
			rows = append(rows, kv{Label: translate(key), Value: value})
		}
	}

	add(setKeyPrefix+"search", sel.query.Q)
	add("transfers.filter.rule", crit.Rule)

	if crit.Dir != "" {
		add("transfers.filter.direction", translate("transfers.dir."+crit.Dir))
	}

	add("transfers.filter.partner", crit.Partner)
	add("transfers.filter.account", crit.Account)
	add("transfers.filter.client", crit.Client)
	add("transfers.filter.protocol", familyLabel(crit.Protocol))
	add(setKeyPrefix+"period", periodText(translate, crit))

	if len(sel.excepted) > 0 {
		numbers := make([]string, len(sel.excepted))
		for i, id := range sel.excepted {
			numbers[i] = translate("transfers.detail.number", "id", id)
		}

		add(setKeyPrefix+"except", strings.Join(numbers, ", "))
	}

	return append(rows, kv{Label: translate(setKeyPrefix + "count"), Value: countText(translate, uint64(len(sel.ids)))})
}

// familyLabel is the name of a protocol family, "" for none.
func familyLabel(key string) string {
	for _, family := range protoform.Families() {
		if family.Key == key {
			return family.Label
		}
	}

	return ""
}

// periodText says the period of the start of the transfers, "" when the
// filters set none.
func periodText(translate gui.Translator, crit *filters) string {
	from, hasFrom := parseLocal(crit.From)
	to, hasTo := parseLocal(crit.To)

	switch {
	case hasFrom && hasTo:
		return translate(setKeyPrefix+"period_both", "from", dateTime(translate, from), "to", dateTime(translate, to))
	case hasFrom:
		return translate(setKeyPrefix+"period_from", "from", dateTime(translate, from))
	case hasTo:
		return translate(setKeyPrefix+"period_to", "to", dateTime(translate, to))
	default:
		return ""
	}
}

// ----------------------------------------------------------- actions ----

// selectionOf reads the selection of the request; an empty one sends back to
// the list, which says so, and a failure answers; the caller goes on when ok.
func (h *Handlers) selectionOf(w http.ResponseWriter, r *http.Request) (*selection, bool) {
	sel, err := h.readSelection(r)

	switch {
	case errors.Is(err, errBadForm):
		http.Error(w, "Bad request", http.StatusBadRequest)

		return nil, false
	case err != nil:
		h.logger.Errorf("failed to read the transfers of the selection: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return nil, false
	case len(sel.ids) == 0:
		redirectBack(w, r, sel, url.Values{liststate.DoneParam: {setEmpty}})

		return nil, false
	}

	return sel, true
}

// doneSet sends back to the list as the user left it, with the result of the
// gesture: how many transfers it was applied to, how many it left aside and
// why.
func doneSet(w http.ResponseWriter, r *http.Request, sel *selection, done string, result *batch.Result) {
	values := url.Values{liststate.DoneParam: {done}, liststate.NameParam: {strconv.Itoa(len(result.Done))}}

	if len(result.Refused) > 0 {
		values.Set(liststate.FailedParam, strconv.Itoa(len(result.Refused)))
		values.Set(liststate.ReasonParam, encodeReasons(result.Counts()))
	}

	redirectBack(w, r, sel, values)
}

// encodeReasons writes the refusals by reason ("server:1,state:2"), in the
// order of the reasons.
func encodeReasons(counts map[batch.Reason]int) string {
	var parts []string

	for _, reason := range batch.Reasons() {
		if n := counts[reason]; n > 0 {
			parts = append(parts, string(reason)+":"+strconv.Itoa(n))
		}
	}

	return strings.Join(parts, ",")
}

// ResumeSet serves POST /webui/transfers/resume: the transfers of the
// selection resume at once when the set still counts what the page showed
// (expected); otherwise, or without that count, the page that confirms the
// resumption says the set as it is now.
func (h *Handlers) ResumeSet(w http.ResponseWriter, r *http.Request) {
	sel, ok := h.selectionOf(w, r)
	if !ok {
		return
	}

	if expected, has := expectedOf(r); !has || expected != len(sel.ids) {
		h.renderSet(w, r, sel, batch.Resume, &setData{}, 0)

		return
	}

	result, err := batch.ResumeSet(h.db, sel.ids, time.Now())
	if err != nil {
		h.logger.Errorf("failed to resume the transfers of the selection: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	doneSet(w, r, sel, setResumed, result)
}

// RetrySetForm serves GET /webui/transfers/retry: the page that retries the
// selection, now or at a date, as the unit retry.
func (h *Handlers) RetrySetForm(w http.ResponseWriter, r *http.Request) {
	sel, ok := h.selectionOf(w, r)
	if !ok {
		return
	}

	h.renderSet(w, r, sel, batch.Retry, &setData{Date: &field{}}, 0)
}

// RetrySet serves POST /webui/transfers/retry: a new transfer for each
// transfer of the selection, now or at the date given.
func (h *Handlers) RetrySet(w http.ResponseWriter, r *http.Request) {
	sel, ok := h.selectionOf(w, r)
	if !ok {
		return
	}

	date := time.Now()
	value := strings.TrimSpace(r.Form.Get(fieldDate))

	if value != "" {
		parsed, valid := parseLocal(value)
		if !valid {
			h.renderSet(w, r, sel, batch.Retry, &setData{
				Date: &field{Value: value, Error: h.shell.Translator(r)("transfers.form.error.date")},
			}, http.StatusUnprocessableEntity)

			return
		}

		date = parsed
	}

	result, err := batch.RetrySet(h.db, sel.ids, date)
	if err != nil {
		h.logger.Errorf("failed to retry the transfers of the selection: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	doneSet(w, r, sel, setRetried, result)
}

// CancelSetForm serves GET /webui/transfers/cancel: the confirmation page of
// the cancellation of the selection (D10: a page), which names the set, how
// many transfers are running and will be stopped, and what follows.
func (h *Handlers) CancelSetForm(w http.ResponseWriter, r *http.Request) {
	sel, ok := h.selectionOf(w, r)
	if !ok {
		return
	}

	h.renderSet(w, r, sel, batch.Cancel, &setData{}, 0)
}

// CancelSet serves POST /webui/transfers/cancel: the transfers of the
// selection go to the history, cancelled; the running ones are stopped
// first.
func (h *Handlers) CancelSet(w http.ResponseWriter, r *http.Request) {
	sel, ok := h.selectionOf(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), setTimeout)
	defer cancel()

	result, err := batch.CancelSet(ctx, h.db, h.logger, sel.ids)
	if err != nil {
		h.logger.Errorf("failed to cancel the transfers of the selection: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	doneSet(w, r, sel, setCancelled, result)
}

// ------------------------------------------------------ notifications ----

// setNotice turns the result of a grouped action, carried by the
// redirection, into its notification: how many transfers it was applied to,
// then, when some were left aside, how many and why.
func setNotice(translate gui.Translator, values url.Values) (gui.Notice, bool) {
	done := values.Get(liststate.DoneParam)
	if done == setEmpty {
		return gui.Notice{Kind: stateWrn, Title: translate(setKeyPrefix + "empty")}, true
	}

	if done != setResumed && done != setRetried && done != setCancelled {
		return gui.Notice{}, false
	}

	kind := strings.TrimPrefix(done, "set_")
	n := countParam(values, liststate.NameParam)
	notice := gui.Notice{Kind: stateOK, Title: countedText(translate, setKeyPrefix+kind, n)}

	if left := countParam(values, liststate.FailedParam); left > 0 {
		notice.Text = countedText(translate, setKeyPrefix+"left_"+kind, left) +
			translate(setKeyPrefix+"left_sep") + reasonsText(translate, values.Get(liststate.ReasonParam))

		if n == 0 {
			notice.Kind = stateWrn
		}
	}

	return notice, true
}

// countParam reads a count carried by the redirection, 0 for none.
func countParam(values url.Values, key string) int {
	n, err := strconv.Atoi(values.Get(key))
	if err != nil || n < 0 {
		return 0
	}

	return n
}

// reasonsText says the refusals by reason ("Server transfers: 1 · Transfers
// whose status does not allow this gesture: 2").
func reasonsText(translate gui.Translator, encoded string) string {
	var parts []string

	for part := range strings.SplitSeq(encoded, ",") {
		reason, count, ok := strings.Cut(part, ":")
		if !ok || !slices.Contains(batch.Reasons(), batch.Reason(reason)) {
			continue
		}

		parts = append(parts, translate(asideKeyPrefix+reason)+translate(setKeyPrefix+"reason_sep")+count)
	}

	return strings.Join(parts, " · ")
}

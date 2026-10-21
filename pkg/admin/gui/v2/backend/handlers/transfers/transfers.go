// Package transfers serves the "Transfer monitoring" screen of the web
// interface: the transfers and the history together (searched, filtered by
// state and by criteria, sorted, paged and refreshed without reload), the
// sheet of a transfer, the actions its state allows (pause, resume, cancel,
// retry) and the pages that add a client transfer and preregister a server
// transfer.
//
// A transfer lives in the table of the transfers while it may still run
// (planned, available, running, paused, interrupted, in error), then, done or
// cancelled, in the history; the normalized_transfers view joins both, and
// the screen shows them together.
package transfers

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

const (
	stylesheet = "/static_v2/transfers/transfers.css"
	script     = "/static_v2/transfers/transfers.js" // the refresh without reload of the list
	docPage    = "user/webui/suivi-des-transferts"
	pagePath   = "/transfers"

	// The cookie that keeps the state of the list for the browser session, so
	// that coming back through the sidebar finds the list as it was left.
	stateCookie = "webui_transfers_list"

	// The block of the list refreshed in place (?fragment=list), and how
	// often: every 5 seconds, as the page this screen replaces (decision of
	// 2026-10-21).
	listFragment  = "list"
	refreshPeriod = 5 * time.Second

	sortStart = "start"
	sortFile  = "file"
	sortState = "state"
	sortRule  = "rule"
	sortStop  = "stop"

	// States of a gui.Status: the shape and colour of a status.
	stateOK  = "ok"
	stateKO  = "ko"
	stateOff = "off"
	stateWrn = "warn"

	// Prefixes of the catalogue keys chosen at run time.
	stateKeyPrefix  = "transfers.state."  // the state of a transfer, "transfers.state.running"
	statesKeyPrefix = "transfers.states." // a state in the row of the states, "transfers.states.running"
	errorKeyPrefix  = "transfers.code."   // the label of an error code, "transfers.code.TeConnection"
	stepKeyPrefix   = "transfers.step."   // the step of a transfer, "transfers.step.StepData"
	toastKeyPrefix  = "transfers.toast."  // the result of an action, "transfers.toast.paused"
	sortKeyPrefix   = "transfers.sort."   // the name of a sorted column, "transfers.sort.start"
)

//nolint:gochecknoglobals // reference data of the screen
var (
	sortKeys  = []string{sortStart, sortFile, sortState, sortRule, sortStop}
	pageSizes = liststate.DefaultSizes

	// The list of this screen: its page, its session cookie, its sort keys
	// (the most recent transfers first), its rows per page and its filters.
	listSpec = liststate.Spec{
		Path: pagePath, Cookie: stateCookie, Sorts: sortKeys, Sizes: pageSizes, FirstDesc: true,
		ParseFilters: func(v url.Values) liststate.Filters { return parseFilters(v) },
		FilterParams: filterParams,
	}
)

// stateSpec is a state of a transfer as the screen shows it: the key of the
// row of the states and of the filter, the status of the product, the shape
// of its status.
type stateSpec struct {
	key    string
	status types.TransferStatus
	shape  string
}

// stateSpecs are the states in the order of the row of the states: what
// needs attention first, then what waits, then what is over.
func stateSpecs() []stateSpec {
	return []stateSpec{
		{"running", types.StatusRunning, stateOK},
		{"error", types.StatusError, stateKO},
		{"planned", types.StatusPlanned, stateOff},
		{"paused", types.StatusPaused, stateOff},
		{"interrupted", types.StatusInterrupted, stateWrn},
		{"available", types.StatusAvailable, stateOff},
		{"done", types.StatusDone, stateOK},
		{"cancelled", types.StatusCancelled, stateOff},
	}
}

// sortExpression is what the list orders by for a sort key: a column of the
// view, the name of the file (the source one, else the destination one, as
// a preregistered transfer may have only one), or the rank of the state in
// the row of the states. The expressions are written from constants only.
func sortExpression(key string) string {
	switch key {
	case sortFile:
		return "COALESCE(NULLIF(src_filename, ''), dest_filename)"
	case sortState:
		var cases strings.Builder

		cases.WriteString("CASE status")

		for rank, spec := range stateSpecs() {
			fmt.Fprintf(&cases, " WHEN '%s' THEN %d", spec.status, rank)
		}

		cases.WriteString(" END")

		return cases.String()
	case sortRule, sortStop:
		return key
	default:
		return "start"
	}
}

// specOfStatus is the state of a status of the product; an unknown status
// reads as an error.
func specOfStatus(status types.TransferStatus) stateSpec {
	for _, spec := range stateSpecs() {
		if spec.status == status {
			return spec
		}
	}

	return stateSpec{key: "error", status: status, shape: stateKO}
}

// specOfKey is the state of a key of the row of the states.
func specOfKey(key string) (stateSpec, bool) {
	for _, spec := range stateSpecs() {
		if spec.key == key {
			return spec, true
		}
	}

	return stateSpec{}, false
}

// statusOf is the status shown for a transfer: its shape and its label.
func statusOf(translate gui.Translator, status types.TransferStatus) gui.Status {
	spec := specOfStatus(status)

	return gui.Status{State: spec.shape, Label: translate(stateKeyPrefix + spec.key)}
}

// Handlers serves the pages and the actions of the screen.
type Handlers struct {
	db     *database.DB
	logger *log.Logger

	shell *common.Shell
}

func New(db *database.DB, logger *log.Logger) *Handlers {
	return &Handlers{db: db, logger: logger, shell: common.NewShell()}
}

// canWrite tells whether the user may act on the transfers. Reading is
// implied by reaching the screen; every action is checked again by the
// router. The templates hide what the rights forbid.
func canWrite(r *http.Request) bool {
	return common.GetUser(r).Permissions.HasPermission(model.PermTransfersWrite)
}

// ------------------------------------------------------------- reading ----

// fileName is the name an operator knows the file of a transfer by: the
// file on the gateway when it is known, else the name at the source, else
// the name at the destination (as the page this screen replaces).
func fileName(trans *model.NormalizedTransferView) string {
	for _, name := range []string{trans.LocalPath, trans.SrcFilename, trans.DestFilename} {
		if name != "" {
			return path.Base(strings.ReplaceAll(name, "\\", "/"))
		}
	}

	return ""
}

// direction says the direction of a transfer and who started it: "Send ·
// client", "Receive · server".
func direction(translate gui.Translator, trans *model.NormalizedTransferView) string {
	dir, role := "transfers.dir.receive", "transfers.role.client"
	if trans.IsSend {
		dir = "transfers.dir.send"
	}

	if trans.IsServer {
		role = "transfers.role.server"
	}

	return translate(dir) + " · " + translate(role)
}

// protocolLabel is the name of the protocol of a transfer.
func protocolLabel(key string) string {
	if key == "" {
		return ""
	}

	return protoform.ProtocolOf(key).Label
}

// errorLabel is the readable label of an error code, the technical code
// staying as a hint; "" without error.
func errorLabel(translate gui.Translator, code types.TransferErrorCode) string {
	if code == types.TeOk || !code.IsValid() {
		return ""
	}

	return translate(errorKeyPrefix + code.String())
}

// transferURL is a page of a transfer.
func transferURL(id int64, suffix string) string {
	return common.WebuiURL(pagePath + "/" + strconv.FormatInt(id, 10) + suffix)
}

// The list state carried by a request, the list as the user left it, and a
// URL carrying the state: the shared mechanics, on the spec of this list.
func ret(r *http.Request) string   { return listSpec.Ret(r) }
func listURL(ret string) string    { return listSpec.ListURL(ret) }
func withRet(u, ret string) string { return liststate.WithRet(u, ret) }

// crumbs is the breadcrumb: the list alone, or the list (as the user left
// it) then the page.
func crumbs(translate gui.Translator, last, carried string) []gui.Crumb {
	c := []gui.Crumb{
		{Label: translate("nav.home"), URL: common.WebuiURL("/home")},
		{Label: translate("nav.supervision")},
	}

	if last == "" {
		return append(c, gui.Crumb{Label: translate("transfers.title")})
	}

	return append(c,
		gui.Crumb{Label: translate("transfers.title"), URL: listURL(carried)},
		gui.Crumb{Label: last})
}

// screen is a page of the screen: the chrome around the data.
func (h *Handlers) screen(title string, crumbs []gui.Crumb, status int) common.Screen {
	return common.Screen{
		Title:       title,
		Section:     constants.SidebarSectionMonitoring,
		Menu:        constants.SidebarMenuTransfers,
		Doc:         docPage,
		Crumbs:      crumbs,
		Status:      status,
		Stylesheets: []string{stylesheet},
	}
}

// doneKinds are the results of the actions, each with its notification.
//
//nolint:gochecknoglobals // reference data of the screen
var doneKinds = []string{"paused", "resumed", "cancelled", "retried", "added", "preregistered"}

// notices turns the result of the previous action, carried by the
// redirection, into a notification.
func notices(translate gui.Translator, values url.Values) []gui.Notice {
	if done := values.Get(liststate.DoneParam); slices.Contains(doneKinds, done) {
		return []gui.Notice{{
			Kind: stateOK, Title: translate(toastKeyPrefix+done, "name", values.Get(liststate.NameParam)),
		}}
	}

	return nil
}

// notFoundData is the localised "not found" page.
type notFoundData struct {
	Title   string
	Text    string
	BackURL string
}

// transfer reads the {id} of the route; a missing transfer renders the
// localised "not found" page and returns false.
func (h *Handlers) transfer(w http.ResponseWriter, r *http.Request) (*model.NormalizedTransferView, bool) {
	id, parseErr := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

	// Eager: the transfer information comes with the transfer.
	trans := &model.NormalizedTransferView{}

	err := h.db.Get(trans, "id=?", id).Eager().Run()
	if parseErr == nil && err == nil {
		return trans, true
	}

	if parseErr == nil && !database.IsNotFound(err) {
		h.logger.Errorf("failed to retrieve the transfer %d: %v", id, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return nil, false
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("transfers.not_found.title")

	h.shell.Render(w, h.logger, r, h.screen(title, crumbs(translate, title, carried), http.StatusNotFound),
		notFoundView(notFoundData{Title: title, Text: translate("transfers.not_found.text"), BackURL: listURL(carried)}))

	return nil, false
}

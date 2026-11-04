// Package dashboard serves the "Dashboard" screen of the web interface
// (Monitoring): the services of the node, the running and failed transfers,
// the activity of the last 24 hours and the latest errors with their actions,
// in one page a production manager reads at a glance, refreshed without
// reloading. Nothing is new in the data: the page assembles what the service
// status and the transfer monitoring screens show, each on its own, so that
// a figure here is always the one of the screen it leads to.
package dashboard

import (
	"net/http"
	"sync"
	"time"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/transfers"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

const (
	stylesheet   = "/static_v2/dashboard/dashboard.css"
	script       = "/static_v2/dashboard/dashboard.js"
	bodyFragment = "body"
	docPage      = "user/webui/tableau-de-bord"
	pagePath     = constants.DashboardPath

	// The automatic refresh of the figures; one reading per period serves
	// every page open, whatever the number of browsers (screen sheet of
	// 2026-10-26, decision 5).
	refreshPeriod = 30 * time.Second
	// The activity window: the transfers started in the last 24 hours, by
	// the hour of their start (the indexed column; decision 3), the hour
	// under way included.
	windowHours = 24
	// The failed transfers the page lists: the most recent ones.
	latestErrors = 5
	// "Recent" failures: the transfers failed that started in the last hour.
	recentFailures = time.Hour
	// A transfer running for longer than this is "stuck", said by its tile (#835).
	stuckAfter = time.Hour
	timeLayout = time.TimeOnly

	// States of a gui.Status and variants of a tile: the shape and colour.
	stateOK   = "ok"
	stateKO   = "ko"
	stateWarn = "warn"
	stateOff  = "off"
	stateInfo = "info"
)

// Handlers serves the page and its fragment.
type Handlers struct {
	db       *database.DB
	logger   *log.Logger
	list     services.Lister
	info     Node     // the figures of the node, for the health strip
	monitors Monitors // the partner monitors of the node, for the health strip
	node     string   // name of this node of the cluster, whose health the page shows
	shell    *common.Shell
	now      func() time.Time                      // the clock, fixed in the tests
	read     func(now time.Time) (*figures, error) // readFigures, a failing base in the tests

	mu   sync.Mutex
	last reading // the last reading of the figures, shared by every page open
}

// reading is what the last attempt to read the figures left: the figures of
// the last successful reading and when it was made, and since when the
// readings fail (zero while they succeed).
type reading struct {
	figures *figures
	at      time.Time
	took    time.Duration // how long the last successful reading took
	failing time.Time
}

// New builds the handlers of the screen: the base the figures are read from,
// the services of the gateway (internal.ListServices, a stub in the tests),
// the figures of the node (GatewayNode) and its partner monitors
// (watchers.Counter). The node is named by its identifier in a cluster, by
// the gateway name otherwise.
func New(db *database.DB, logger *log.Logger, list services.Lister, info Node, monitors Monitors) *Handlers {
	node := db.Config.NodeID
	if node == "" {
		node = db.Config.GatewayName
	}

	h := &Handlers{
		db: db, logger: logger, list: list, info: info, monitors: monitors, node: node,
		shell: common.NewShell(), now: time.Now,
	}
	h.read = h.readFigures

	return h
}

// Page serves GET /webui/dashboard. With ?fragment=body, it renders the body
// (everything under the page header) and the resumption button of the header,
// for the refresh without reload.
func (h *Handlers) Page(w http.ResponseWriter, r *http.Request) {
	translate := h.shell.Translator(r)
	perms := common.GetUser(r).Permissions
	// An action that comes back here with its result (a transfer resumed, the
	// failed transfers resumed at once): the figures are read again, so that
	// the tiles and the resumption button say what the action left.
	fresh := r.URL.Query().Get(liststate.DoneParam) != ""
	// The page is the origin of every link to another screen (#840): each
	// offers the return to it, each action comes back to it.
	data := h.makePageData(translate,
		perms.HasPermission(model.PermTransfersRead), perms.HasPermission(model.PermTransfersWrite), fresh,
		liststate.Origin(r))

	if r.URL.Query().Get(constants.FragmentParam) == bodyFragment {
		h.shell.RenderFragment(w, h.logger, r, fragment(data))

		return
	}

	h.shell.Render(w, h.logger, r, common.Screen{
		Title:   translate("dashboard.title"),
		Section: constants.SidebarSectionMonitoring,
		Menu:    constants.SidebarMenuDashboard,
		Doc:     docPage,
		Crumbs: []gui.Crumb{
			{Label: translate("nav.home"), URL: common.WebuiURL("/home")},
			{Label: translate("nav.supervision")},
			{Label: translate("dashboard.title")},
		},
		Stylesheets: []string{stylesheet},
		Scripts:     []string{script},
		Notices:     transfers.Notices(translate, r.URL.Query()),
		Wide:        true, // an operations screen: the whole width (#833)
	}, page(data))
}

// pageData is the page as the templates render it.
type pageData struct {
	RefreshedAt    string
	RefreshSeconds int
	HealthTitle    string           // title of the health strip, naming the node
	Health         gui.HealthProps  // the health strip of the node (#834)
	Read           bool             // the user reads the transfers: their blocks are rendered
	Write          bool             // the user may act on the transfers: the menus of the errors
	Figures        bool             // figures were read, now or before: the blocks of the transfers have values
	Degraded       *gui.BannerProps // the base does not answer: since when, how old the figures shown are
	Tiles          []gui.TileProps  // running, failed, done in 24 hours
	Chart          *gui.ChartProps  // nil without any transfer in the window: the empty state instead
	Days           []dayValues      // the values of the chart by day, the most recent first
	ValuesMeta     string           // the totals of the window, in the heading of the values
	Errors         []errorRow       // the latest failed transfers, the most recent first
	ErrorsAll      string           // the link to the list of the failed transfers, with their number
	ErrorsURL      string           // the list of the failed transfers
	Resume         *resumeAction    // the grouped resumption of the failed transfers; nil without any, or to a reader
}

// resumeAction is the primary button of the page header (#818): the grouped
// resumption of the failed transfers, its label saying how many will resume,
// the form it posts (the transfer monitoring: the failed state, the whole
// filter, the count of failed transfers the page showed), which comes back
// to the dashboard.
type resumeAction struct {
	Label  string
	Action string
	Fields []liststate.Hidden
}

// makePageData reads what the page shows (again, when fresh). Without the
// right to read the transfers, the page shows the health of the node alone,
// and reads nothing else (the database cell then says what the last reading
// left). Every link to another screen carries the origin.
func (h *Handlers) makePageData(translate gui.Translator, read, write, fresh bool, origin string) pageData {
	now := h.now()
	data := pageData{
		RefreshedAt:    now.Format(timeLayout),
		RefreshSeconds: int(refreshPeriod.Seconds()),
		HealthTitle:    translate("dashboard.health_title", "node", h.node),
		Read:           read,
		Write:          write,
	}

	var (
		figs *figures
		last reading
	)

	if read {
		figs, last = h.figures(now, fresh)
	} else {
		last = h.lastReading()
	}

	var certs *certSummary
	if figs != nil {
		certs = &figs.Certs
	}

	data.Health = gui.HealthProps{Items: healthCells(translate, &healthInput{
		node: h.info, now: now, dbType: h.db.Config.Database.Type, last: last, certs: certs,
		services: services.Sum(translate, h.shell.Catalog().Has, h.list), monitors: h.monitors, origin: origin,
	})}

	if !read {
		return data
	}

	if !last.failing.IsZero() {
		text := translate("dashboard.degraded.none")
		if figs != nil {
			text = translate("dashboard.degraded.text", "time", last.at.Format(timeLayout))
		}

		data.Degraded = &gui.BannerProps{
			Kind:   stateKO,
			Title:  translate("dashboard.degraded.title", "time", last.failing.Format(timeLayout)),
			Text:   text,
			Action: &gui.Link{Label: translate("nav.services"), URL: liststate.WithOrigin(common.WebuiURL("/services"), origin)},
		}
	}

	if figs == nil {
		return data
	}

	data.Figures = true
	texts := texts{translate}
	data.Tiles = tiles(texts, figs, origin)
	data.Chart, data.Days, data.ValuesMeta = activity(texts, figs)
	data.Errors = errorRows(translate, figs, now, write, origin)
	data.ErrorsAll = texts.count(errorsKeys, "all", figs.Live[statusKey(types.StatusError)])
	data.ErrorsURL = liststate.WithOrigin(transfers.FilteredListURL(transfers.StateError, time.Time{}), origin)

	if write && figs.Resumable > 0 {
		action, fields := transfers.ResumeFailedForm(figs.Live[statusKey(types.StatusError)], origin)
		data.Resume = &resumeAction{Label: texts.count(resumeKeys, "resume", figs.Resumable), Action: action, Fields: fields}
	}

	return data
}

// figures gives the figures of the transfers: those read less than a period
// ago, else, or when fresh, a new reading. When the base does not answer,
// the last figures read stay, with the reading that says since when the base
// fails.
func (h *Handlers) figures(now time.Time, fresh bool) (*figures, reading) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !fresh && h.last.figures != nil && h.last.failing.IsZero() && now.Sub(h.last.at) < refreshPeriod {
		return h.last.figures, h.last
	}

	started := time.Now()
	figs, err := h.read(now)
	took := time.Since(started)

	if err != nil {
		h.logger.Errorf("Failed to read the figures of the dashboard: %v", err)

		if h.last.failing.IsZero() {
			h.last.failing = now
		}

		return h.last.figures, h.last
	}

	h.last = reading{figures: figs, at: now, took: took}

	return figs, h.last
}

// lastReading is what the last reading left, for a page that does not read
// the figures itself.
func (h *Handlers) lastReading() reading {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.last
}

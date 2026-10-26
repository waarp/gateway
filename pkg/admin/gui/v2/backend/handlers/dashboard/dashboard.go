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
	timeLayout     = time.TimeOnly

	// States of a gui.Status and variants of a tile: the shape and colour.
	stateOK   = "ok"
	stateKO   = "ko"
	stateWarn = "warn"
	stateOff  = "off"
	stateInfo = "info"
)

// Handlers serves the page and its fragment.
type Handlers struct {
	db     *database.DB
	logger *log.Logger
	list   services.Lister
	node   string // name of this node of the cluster, whose services the page shows
	shell  *common.Shell
	now    func() time.Time                      // the clock, fixed in the tests
	read   func(now time.Time) (*figures, error) // readFigures, a failing base in the tests

	mu   sync.Mutex
	last reading // the last reading of the figures, shared by every page open
}

// reading is what the last attempt to read the figures left: the figures of
// the last successful reading and when it was made, and since when the
// readings fail (zero while they succeed).
type reading struct {
	figures *figures
	at      time.Time
	failing time.Time
}

// New builds the handlers of the screen: the base the figures are read from
// and the services of the gateway (internal.ListServices, a stub in the
// tests). The node is named by its identifier in a cluster, by the gateway
// name otherwise.
func New(db *database.DB, logger *log.Logger, list services.Lister) *Handlers {
	node := db.Config.NodeID
	if node == "" {
		node = db.Config.GatewayName
	}

	h := &Handlers{db: db, logger: logger, list: list, node: node, shell: common.NewShell(), now: time.Now}
	h.read = h.readFigures

	return h
}

// Page serves GET /webui/dashboard. With ?fragment=body, it renders the body
// alone (everything under the page header), for the refresh without reload.
func (h *Handlers) Page(w http.ResponseWriter, r *http.Request) {
	translate := h.shell.Translator(r)
	perms := common.GetUser(r).Permissions
	data := h.makePageData(translate,
		perms.HasPermission(model.PermTransfersRead), perms.HasPermission(model.PermTransfersWrite))

	if r.URL.Query().Get(constants.FragmentParam) == bodyFragment {
		h.shell.RenderFragment(w, h.logger, r, body(data))

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
	}, page(data))
}

// pageData is the page as the templates render it.
type pageData struct {
	RefreshedAt    string
	RefreshSeconds int
	Node           string           // title of the row of the services, naming the node
	Services       []gui.TileProps  // the three groups of services, as the service status screen sums them up
	Read           bool             // the user reads the transfers: their blocks are rendered
	Write          bool             // the user may act on the transfers: the menus of the errors
	Figures        bool             // figures were read, now or before: the blocks of the transfers have values
	Degraded       *gui.BannerProps // the base does not answer: since when, how old the figures shown are
	Tiles          []gui.TileProps  // running, failed, done in 24 hours
	Chart          *gui.ChartProps  // nil without any transfer in the window: the empty state instead
	Values         []hourRow        // the values of the chart, one row per hour
	ValuesMeta     string           // the totals of the window, in the heading of the values
	Errors         []errorRow       // the latest failed transfers, the most recent first
	ErrorsAll      string           // the link to the list of the failed transfers, with their number
	ErrorsURL      string           // the list of the failed transfers
}

// makePageData reads what the page shows. Without the right to read the
// transfers, the page shows the services alone, and reads nothing else.
func (h *Handlers) makePageData(translate gui.Translator, read, write bool) pageData {
	now := h.now()
	data := pageData{
		RefreshedAt:    now.Format(timeLayout),
		RefreshSeconds: int(refreshPeriod.Seconds()),
		Node:           translate("dashboard.node", "node", h.node),
		Services:       services.Tiles(translate, h.shell.Catalog().Has, h.list),
		Read:           read,
		Write:          write,
	}

	if !read {
		return data
	}

	figs, last := h.figures(now)

	if !last.failing.IsZero() {
		text := translate("dashboard.degraded.none")
		if figs != nil {
			text = translate("dashboard.degraded.text", "time", last.at.Format(timeLayout))
		}

		data.Degraded = &gui.BannerProps{
			Kind:   stateKO,
			Title:  translate("dashboard.degraded.title", "time", last.failing.Format(timeLayout)),
			Text:   text,
			Action: &gui.Link{Label: translate("nav.services"), URL: common.WebuiURL("/services")},
		}
	}

	if figs == nil {
		return data
	}

	data.Figures = true
	texts := texts{translate}
	data.Tiles = tiles(texts, figs)
	data.Chart, data.Values, data.ValuesMeta = activity(texts, figs)
	data.Errors = errorRows(translate, figs, now, write)
	data.ErrorsAll = texts.count(errorsKeys, "all", figs.Live[statusKey(types.StatusError)])
	data.ErrorsURL = transfers.FilteredListURL(transfers.StateError, time.Time{})

	return data
}

// figures gives the figures of the transfers: those read less than a period
// ago, else a new reading. When the base does not answer, the last figures
// read stay, with the reading that says since when the base fails.
func (h *Handlers) figures(now time.Time) (*figures, reading) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.last.figures != nil && h.last.failing.IsZero() && now.Sub(h.last.at) < refreshPeriod {
		return h.last.figures, h.last
	}

	figs, err := h.read(now)
	if err != nil {
		h.logger.Errorf("Failed to read the figures of the dashboard: %v", err)

		if h.last.failing.IsZero() {
			h.last.failing = now
		}

		return h.last.figures, h.last
	}

	h.last = reading{figures: figs, at: now}

	return figs, h.last
}

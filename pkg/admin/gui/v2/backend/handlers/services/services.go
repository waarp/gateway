// Package services serves the "Service status" screen of the web interface:
// the running state of the system services, local servers and local clients
// of the gateway, refreshed without reloading the page.
package services

import (
	"cmp"
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
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	stylesheet    = "/static_v2/services/services.css"
	script        = "/static_v2/services/services.js"
	stateFragment = "state"
	docPage       = "user/webui/etat-des-services"
	pagePath      = "/services"

	// The group whose table is shown (the tiles are its tabs), the sort of
	// that table and its page beyond pageSize rows.
	groupParam = "group"
	sortParam  = "sort"
	dirParam   = "dir"
	pageParam  = "page"
	pageSize   = 10

	sortName  = "name"
	sortState = "state"
	dirAsc    = "asc"
	dirDesc   = "desc"

	// Period of the automatic refresh of the states, the same as on the page
	// this screen replaces.
	refreshPeriod = 10 * time.Second
	timeLayout    = time.TimeOnly

	// Prefix of the catalogue keys of the functional names of the system
	// services ("services.core.Database"). A service without an entry keeps
	// its technical name.
	coreKeyPrefix = "services.core."

	// States of a gui.Status: the shape and colour of a status.
	stateOK      = "ok"
	stateKO      = "ko"
	stateOffline = "off"
	stateUnknown = "warn"
)

// Lister returns the services of the gateway by group. It is
// internal.ListServices on the gateway, a stub in the tests.
type Lister func() (core, servers, clients []internal.Service)

// groupSpec describes one of the three groups of the screen.
type groupSpec struct {
	id      string // value of groupParam, id of the panel, suffix of the catalogue keys
	kindKey string // catalogue key of the object name, heading of the first column
	path    string // historic page that lists the objects, linked from the empty state
	core    bool   // system services: functional names taken from the catalogue
}

type pageData struct {
	RefreshedAt    string
	RefreshSeconds int
	Groups         []group
}

// group is a tab of the screen: its tile sums up the whole group, its panel
// shows one page of its table, sorted as asked.
type group struct {
	ID       string
	Kind     string
	Caption  string
	Selected bool
	Sort     string // sorted column: sortName or sortState
	Dir      string // dirAsc or dirDesc
	Base     string // address of the group, which the column headers extend with the sort
	Rows     []row
	Tile     gui.TileProps   // the tile of the group: a tab that shows its panel, a link without JavaScript
	Pager    *gui.PagerProps // nil when every row fits on one page
	Empty    gui.EmptyProps  // the empty state of a group without any service
}

type row struct {
	Name   string
	Tech   string // technical name, shown under a functional one
	Status gui.Status
	Reason string // raw reason given by the service, never translated
}

// selection is what the request asks for: the group shown, the sort of its
// table and the page of it.
type selection struct {
	group string
	sort  string
	dir   string
	page  int
}

func groupSpecs() []groupSpec {
	return []groupSpec{
		{id: "core", kindKey: "services.col.core", core: true},
		{id: "servers", kindKey: "services.col.server", path: "/servers"},
		{id: "clients", kindKey: "services.col.client", path: "/clients"},
	}
}

// defaultSelection shows the first group, sorted by name, from its first page.
func defaultSelection(groupID string) selection {
	return selection{group: groupID, sort: sortName, dir: dirAsc, page: 1}
}

// Page serves GET /webui/services. With ?fragment=state, it renders the
// states block alone, for the refresh without reload.
func Page(logger *log.Logger, list Lister) http.HandlerFunc {
	shell := common.NewShell()

	return func(w http.ResponseWriter, r *http.Request) {
		translate := shell.Translator(r)
		data := makePageData(translate, shell.Catalog().Has, list, parseSelection(r.URL.Query()))

		if r.URL.Query().Get(constants.FragmentParam) == stateFragment {
			shell.RenderFragment(w, logger, r, state(data))

			return
		}

		shell.Render(w, logger, r, common.Screen{
			Title:   translate("services.title"),
			Section: constants.SidebarSectionMonitoring,
			Menu:    constants.SidebarMenuStatus,
			Doc:     docPage,
			Crumbs: []gui.Crumb{
				{Label: translate("nav.home"), URL: common.WebuiURL("/home")},
				{Label: translate("nav.supervision")},
				{Label: translate("services.title")},
			},
			Stylesheets: []string{stylesheet},
			Scripts:     []string{script},
		}, page(data))
	}
}

// parseSelection reads the group, the sort and the page asked for; anything
// unknown or out of range falls back to the default.
func parseSelection(query url.Values) selection {
	specs := groupSpecs()
	sel := defaultSelection(specs[0].id)

	if slices.ContainsFunc(specs, func(spec groupSpec) bool { return spec.id == query.Get(groupParam) }) {
		sel.group = query.Get(groupParam)
	}

	if query.Get(sortParam) == sortState {
		sel.sort = sortState
	}

	if query.Get(dirParam) == dirDesc {
		sel.dir = dirDesc
	}

	if page, err := strconv.Atoi(query.Get(pageParam)); err == nil && page > 1 {
		sel.page = page
	}

	return sel
}

func makePageData(translate gui.Translator, has func(key string) bool, list Lister, sel selection) pageData {
	core, servers, clients := list()
	lists := [][]internal.Service{core, servers, clients}
	specs := groupSpecs()
	groups := make([]group, 0, len(specs))

	for i, spec := range specs {
		// The sort and the page apply to the group shown; the others keep
		// their default.
		selected := spec.id == sel.group
		groupSel := defaultSelection(spec.id)

		if selected {
			groupSel = sel
		}

		groups = append(groups, makeGroup(translate, has, spec, lists[i], groupSel, selected))
	}

	return pageData{
		RefreshedAt:    time.Now().Format(timeLayout),
		RefreshSeconds: int(refreshPeriod.Seconds()),
		Groups:         groups,
	}
}

func makeGroup(translate gui.Translator, has func(key string) bool, spec groupSpec,
	services []internal.Service, sel selection, selected bool,
) group {
	title := translate("services.group." + spec.id)
	caption := translate("services.caption." + spec.id)
	rows := makeRows(translate, has, spec.core, services)
	sortRows(rows, sel.sort, sel.dir)

	tab := makeTile(translate, title, spec.id, rows)
	tab.Selected = selected

	empty := gui.EmptyProps{Title: translate("state.none_yet")}
	if spec.path != "" {
		empty.Title = translate("services.empty." + spec.id + ".title")
		empty.Text = translate("services.empty." + spec.id + ".text")
		empty.Action = &gui.Link{
			Label: translate("services.empty." + spec.id + ".link"),
			URL:   common.WebuiURL(spec.path),
		}
	}

	result := group{
		ID:       spec.id,
		Kind:     translate(spec.kindKey),
		Caption:  caption,
		Selected: selected,
		Sort:     sel.sort,
		Dir:      sel.dir,
		Base:     groupURL(defaultSelection(spec.id)),
		Rows:     rows,
		Tile:     tab,
		Empty:    empty,
	}

	if pages := (len(rows) + pageSize - 1) / pageSize; pages > 1 {
		page := min(max(sel.page, 1), pages)
		first, last := (page-1)*pageSize, min(page*pageSize, len(rows))

		result.Rows = rows[first:last]
		result.Caption = translate("services.caption.page", "caption", caption, "page", page, "pages", pages)
		result.Pager = &gui.PagerProps{
			Info:   translate("services.pager.info", "group", title, "from", first+1, "to", last, "total", len(rows)),
			Page:   page,
			Pages:  pages,
			Hidden: groupQuery(sel),
		}

		if page > 1 {
			prev := sel
			prev.page = page - 1
			result.Pager.PrevURL = groupURL(prev)
		}

		if page < pages {
			next := sel
			next.page = page + 1
			result.Pager.NextURL = groupURL(next)
		}
	}

	return result
}

// groupQuery is the query of a selection without its page: the group, and
// the sort when it is not the default one.
func groupQuery(sel selection) map[string]string {
	query := map[string]string{groupParam: sel.group}

	if sel.sort != sortName || sel.dir != dirAsc {
		query[sortParam] = sel.sort
		query[dirParam] = sel.dir
	}

	return query
}

// groupURL is the address of a selection: one page of the table of a group,
// sorted as asked; it is also the server fallback of the tile of the group.
func groupURL(sel selection) string {
	query := url.Values{}
	for name, value := range groupQuery(sel) {
		query.Set(name, value)
	}

	if sel.page > 1 {
		query.Set(pageParam, strconv.Itoa(sel.page))
	}

	return common.WebuiURL(pagePath) + "?" + query.Encode()
}

// makeRows gives the system services their functional name.
func makeRows(translate gui.Translator, has func(key string) bool, core bool,
	services []internal.Service,
) []row {
	rows := make([]row, 0, len(services))

	for _, service := range services {
		line := row{Name: service.Name, Status: Status(translate, service.State), Reason: service.Reason}

		if key := coreKeyPrefix + service.Name; core && has(key) {
			line.Name, line.Tech = translate(key), service.Name
		}

		rows = append(rows, line)
	}

	return rows
}

// sortRows sorts the rows by name (the technical one, as the page this
// screen replaces did) or by state, from the healthiest to the worst; ties
// keep the name order.
func sortRows(rows []row, key, dir string) {
	slices.SortStableFunc(rows, func(a, b row) int {
		var order int
		if key == sortState {
			order = cmp.Compare(stateRank(a.Status.State), stateRank(b.Status.State))
		} else {
			order = strings.Compare(nameKey(a), nameKey(b))
		}

		if dir == dirDesc {
			order = -order
		}

		if order == 0 {
			return strings.Compare(nameKey(a), nameKey(b))
		}

		return order
	})
}

// nameKey is the name a row sorts by: the technical name of a system
// service, the name of the others.
func nameKey(line row) string {
	if line.Tech != "" {
		return line.Tech
	}

	return line.Name
}

// stateRank orders the states from the healthiest to the worst.
func stateRank(state string) int {
	order := []string{stateOK, stateOffline, stateUnknown, stateKO}
	if rank := slices.Index(order, state); rank >= 0 {
		return rank
	}

	return len(order)
}

// Status turns the state code of a service into a shape, a colour and a
// label; a code outside the three known ones reads "unknown". The screens
// that show the state of one service (the SNMP agent) share it.
func Status(translate gui.Translator, code string) gui.Status {
	switch code {
	case utils.StateRunning.String():
		return gui.Status{State: stateOK, Label: translate("state.running")}
	case utils.StateOffline.String():
		return gui.Status{State: stateOffline, Label: translate("state.offline")}
	case utils.StateError.String():
		return gui.Status{State: stateKO, Label: translate("state.error")}
	default:
		return gui.Status{State: stateUnknown, Label: translate("state.unknown")}
	}
}

// makeTile counts the rows of a group by state: the tile shows how many
// services run, its border and its labels say what is wrong, if anything.
func makeTile(translate gui.Translator, label, groupID string, rows []row) gui.TileProps {
	counts := map[string]int{}
	for _, line := range rows {
		counts[line.Status.State]++
	}

	summary := gui.TileProps{
		Label:    label,
		Href:     groupURL(defaultSelection(groupID)),
		Controls: groupID,
		Value:    strconv.Itoa(counts[stateOK]),
		Total:    totalText(len(rows)),
		SR:       translate("services.summary.running", "running", counts[stateOK], "total", len(rows)),
	}

	switch {
	case len(rows) == 0:
		summary.Foot = translate("state.none_yet")
	case counts[stateKO] > 0:
		summary.Variant = stateKO
	case counts[stateOffline]+counts[stateUnknown] > 0:
		summary.Variant = stateUnknown
	default:
		summary.Variant = stateOK
		summary.Statuses = []gui.Status{{State: stateOK, Label: translate("state.all_running")}}
	}

	for _, problem := range []struct{ state, key string }{
		{stateKO, "services.summary.errors"},
		{stateOffline, "services.summary.offline"},
		{stateUnknown, "services.summary.unknown"},
	} {
		if n := counts[problem.state]; n > 0 {
			summary.Statuses = append(summary.Statuses,
				gui.Status{State: problem.state, Label: translate(problem.key, "n", n)})
		}
	}

	return summary
}

// totalText is the total of a tile, hidden when there is nothing to count.
func totalText(n int) string {
	if n == 0 {
		return ""
	}

	return strconv.Itoa(n)
}

// Tiles sums up the three groups of services for another screen (the
// dashboard): the tiles of this screen, each a link to its group here, without
// the tab it controls here. One source of truth for the figures of the
// services.
func Tiles(translate gui.Translator, has func(key string) bool, list Lister) []gui.TileProps {
	core, servers, clients := list()
	lists := [][]internal.Service{core, servers, clients}
	specs := groupSpecs()
	tiles := make([]gui.TileProps, 0, len(specs))

	for i, spec := range specs {
		rows := makeRows(translate, has, spec.core, lists[i])
		tile := makeTile(translate, translate("services.group."+spec.id), spec.id, rows)
		tile.Controls = ""
		tiles = append(tiles, tile)
	}

	return tiles
}

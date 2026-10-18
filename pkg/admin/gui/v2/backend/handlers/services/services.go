// Package services serves the "Service status" screen of the web interface:
// the running state of the system services, local servers and local clients
// of the gateway, refreshed without reloading the page.
package services

import (
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
	pageTemplateFile = "services/services.gohtml"
	stateTemplate    = "services/state"
	stateFragment    = "state"
	docPage          = "user/webui/etat-des-services"
	pagePath         = "/services"

	// The group whose table is shown (the tiles are its tabs), and the page
	// of that table beyond pageSize rows.
	groupParam = "group"
	pageParam  = "page"
	pageSize   = 25

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
// shows one page of its table.
type group struct {
	ID       string
	Kind     string
	Caption  string
	Selected bool
	Rows     []row
	Tile     tile
	Pager    *pager // nil when every row fits on one page
	Empty    emptyState
}

type row struct {
	Name   string
	Tech   string // technical name, shown under a functional one
	Status gui.Status
	Reason string // raw reason given by the service, never translated
}

// tile is the tile of a group, with the fields of "gui/tile": a tab that
// shows the panel named by Controls, or a link to Href without JavaScript.
type tile struct {
	Variant  string
	Label    string
	Href     string
	Controls string
	Selected bool
	SR       string
	Foot     string
	Value    int
	Total    int
	Statuses []gui.Status
}

// pager is the list foot of a group, with the fields of "gui/pager".
type pager struct {
	Info    string
	Page    int
	Pages   int
	PrevURL string
	NextURL string
	Hidden  map[string]string
}

// emptyState is the "gui/empty" of a group without any service (Icon empty:
// the kit's default).
type emptyState struct {
	Icon   string
	Title  string
	Text   string
	Action *gui.Link
}

// selection is what the request asks for: the group shown and the page of
// its table.
type selection struct {
	group string
	page  int
}

func groupSpecs() []groupSpec {
	return []groupSpec{
		{id: "core", kindKey: "services.col.core", core: true},
		{id: "servers", kindKey: "services.col.server", path: "/server_management"},
		{id: "clients", kindKey: "services.col.client", path: "/local_client_management"},
	}
}

// Page serves GET /webui/services. With ?fragment=state, it renders the
// states block alone, for the refresh without reload.
func Page(logger *log.Logger, list Lister) http.HandlerFunc {
	shell := common.NewShell(pageTemplateFile)

	return func(w http.ResponseWriter, r *http.Request) {
		translate := shell.Translator(r)
		data := makePageData(translate, shell.Catalog().Has, list, parseSelection(r.URL.Query()))

		if r.URL.Query().Get(constants.FragmentParam) == stateFragment {
			shell.RenderFragment(w, logger, r, stateTemplate, data)

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
			Data: data,
		})
	}
}

// parseSelection reads the group and the page asked for; anything unknown
// or out of range falls back to the first group and the first page.
func parseSelection(query url.Values) selection {
	specs := groupSpecs()
	sel := selection{group: specs[0].id, page: 1}

	if slices.ContainsFunc(specs, func(spec groupSpec) bool { return spec.id == query.Get(groupParam) }) {
		sel.group = query.Get(groupParam)
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
		page := 1
		if spec.id == sel.group {
			page = sel.page
		}

		groups = append(groups, makeGroup(translate, has, spec, lists[i], spec.id == sel.group, page))
	}

	return pageData{
		RefreshedAt:    time.Now().Format(timeLayout),
		RefreshSeconds: int(refreshPeriod.Seconds()),
		Groups:         groups,
	}
}

func makeGroup(translate gui.Translator, has func(key string) bool, spec groupSpec,
	services []internal.Service, selected bool, page int,
) group {
	title := translate("services.group." + spec.id)
	caption := translate("services.caption." + spec.id)
	rows := makeRows(translate, has, spec.core, services)

	tab := makeTile(translate, title, spec.id, rows)
	tab.Selected = selected

	empty := emptyState{Title: translate("state.none_yet")}
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
		Rows:     rows,
		Tile:     tab,
		Empty:    empty,
	}

	if pages := (len(rows) + pageSize - 1) / pageSize; pages > 1 {
		page = min(max(page, 1), pages)
		first, last := (page-1)*pageSize, min(page*pageSize, len(rows))

		result.Rows = rows[first:last]
		result.Caption = translate("services.caption.page", "caption", caption, "page", page, "pages", pages)
		result.Pager = &pager{
			Info:   translate("services.pager.info", "group", title, "from", first+1, "to", last, "total", len(rows)),
			Page:   page,
			Pages:  pages,
			Hidden: map[string]string{groupParam: spec.id},
		}

		if page > 1 {
			result.Pager.PrevURL = pageURL(spec.id, page-1)
		}

		if page < pages {
			result.Pager.NextURL = pageURL(spec.id, page+1)
		}
	}

	return result
}

// pageURL is the address of one page of the table of a group, which is also
// the server fallback of its tile.
func pageURL(groupID string, page int) string {
	query := url.Values{groupParam: {groupID}}
	if page > 1 {
		query.Set(pageParam, strconv.Itoa(page))
	}

	return common.WebuiURL(pagePath) + "?" + query.Encode()
}

// makeRows sorts the services by name, as the page this screen replaces
// did, and gives the system services their functional name.
func makeRows(translate gui.Translator, has func(key string) bool, core bool,
	services []internal.Service,
) []row {
	slices.SortFunc(services, func(a, b internal.Service) int {
		return strings.Compare(a.Name, b.Name)
	})

	rows := make([]row, 0, len(services))

	for _, service := range services {
		line := row{Name: service.Name, Status: status(translate, service.State), Reason: service.Reason}

		if key := coreKeyPrefix + service.Name; core && has(key) {
			line.Name, line.Tech = translate(key), service.Name
		}

		rows = append(rows, line)
	}

	return rows
}

// status turns the state code of a service into a shape, a colour and a
// label; a code outside the three known ones reads "unknown".
func status(translate gui.Translator, code string) gui.Status {
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
func makeTile(translate gui.Translator, label, groupID string, rows []row) tile {
	counts := map[string]int{}
	for _, line := range rows {
		counts[line.Status.State]++
	}

	summary := tile{
		Label:    label,
		Href:     pageURL(groupID, 1),
		Controls: groupID,
		Value:    counts[stateOK],
		Total:    len(rows),
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

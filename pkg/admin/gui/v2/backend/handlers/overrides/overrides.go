// Package overrides serves the "Configuration overrides" screen of the web
// interface (Administration): the addresses replaced on the node of a cluster
// that serves the page, what each rule concerns in the shared configuration
// (local servers, partners, administration), and the pages that add, edit
// and delete a rule.
//
// In a cluster, every gateway shares the same configuration; a local server
// is declared once, with the public address of the load balancer, which no
// node can listen to. An override is a setting of one node that wins over the
// shared configuration; today, the product knows one kind, the address
// indirection: on this node, whenever the shared configuration gives the
// configured address, the gateway uses the other one, to listen
// (administration, local servers) and to connect to the partners. A rule
// holds a host alone (the port is kept) or a host and its port, on both
// sides. The rules live in a file of the node (<instance name>.ini, next to
// the configuration file), not in the database: they exist only when the
// gateway starts with an instance name. Until this screen, they were managed
// through the REST API and the command line only.
package overrides

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	// The stylesheet of the screen, under the web interface.
	stylesheet = "/static_v2/overrides/overrides.css"

	docPage  = "user/webui/surcharges"
	pagePath = "/overrides"

	// The cookie that keeps the state of the list for the browser session.
	stateCookie = "webui_overrides_list"

	// The parameters of the list state and of the result of the previous
	// action, shared by the list screens; a rule is named by its configured
	// address, carried by targetParam.
	doneParam   = liststate.DoneParam
	nameParam   = liststate.NameParam
	dirDesc     = liststate.DirDesc
	targetParam = "target"

	sortTarget = "target"

	doneCreated = liststate.DoneCreated
	doneUpdated = liststate.DoneUpdated
	doneDeleted = liststate.DoneDeleted

	// Prefixes of the catalogue keys chosen at run time.
	sortKeyPrefix  = "overrides.sort."
	toastKeyPrefix = "overrides.toast."

	// Kinds of a gui.Notice and of a gui.Status.
	stateOK   = "ok"
	stateWarn = "warn"

	// Names shown in a cell before counting.
	namesShown = 1
)

//nolint:gochecknoglobals // reference data of the screen
var (
	sortKeys  = []string{sortTarget}
	doneKinds = []string{doneCreated, doneUpdated, doneDeleted}

	listSpec = liststate.Spec{Path: pagePath, Cookie: stateCookie, Sorts: sortKeys, Sizes: liststate.DefaultSizes}
)

// Handlers serves the pages and the actions of the screen.
type Handlers struct {
	db     *database.DB
	logger *log.Logger
	shell  *common.Shell
}

func New(db *database.DB, logger *log.Logger) *Handlers {
	return &Handlers{db: db, logger: logger, shell: common.NewShell()}
}

// node is the configuration of the node that serves the page: nil overrides
// when the gateway runs without an instance name.
func (h *Handlers) node() (*conf.ConfigOverride, string) {
	return h.db.Config.Overrides, h.db.Config.NodeID
}

// rights of the current user on the overrides: those of the administration.
// Reading is implied by reaching the screen; writing and deleting are checked
// again by the router on each action.
type rights struct{ Write, Delete bool }

func rightsOf(r *http.Request) rights {
	perms := common.GetUser(r).Permissions

	return rights{
		Write:  perms.HasPermission(model.PermAdminWrite),
		Delete: perms.HasPermission(model.PermAdminDelete),
	}
}

// hasPort tells whether an address carries a port; ok is false when the
// address cannot be read at all.
func hasPort(addr string) (withPort, ok bool) {
	_, _, err := net.SplitHostPort(addr)
	if err == nil {
		return true, true
	}

	var addrErr *net.AddrError
	if errors.As(err, &addrErr) && addrErr.Err == "missing port in address" {
		return false, true
	}

	return false, false
}

// rule is an address indirection of the node, with what it concerns in the
// shared configuration.
type rule struct {
	Target   string
	Real     string
	HostOnly bool
	uses     uses
}

// uses is what a rule replaces in the shared configuration: the local
// servers and the partners whose address it matches, the administration.
type uses struct {
	servers  []named
	partners []named
	admin    bool
}

type named struct {
	ID   int64
	Name string
}

// usage is the shared configuration the rules apply to: the addresses of the
// local servers, of the partners and of the administration.
type usage struct {
	servers  model.LocalAgents
	partners model.RemoteAgents
	admin    types.Address
}

func (h *Handlers) usage() (*usage, error) {
	u := &usage{}
	if err := h.db.Select(&u.servers).OrderBy("name", true).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	if err := h.db.Select(&u.partners).OrderBy("name", true).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	u.admin = types.Address{Host: h.db.Config.Admin.Host, Port: h.db.Config.Admin.Port}

	return u, nil
}

// ruleOf finds the rule an address goes through, as the product does: the
// rule of the host and the port first, else the rule of the host alone.
func ruleOf(rules map[string]*rule, host string, port uint16) *rule {
	if r, ok := rules[net.JoinHostPort(host, utils.FormatUint(port))]; ok {
		return r
	}

	return rules[host]
}

// rulesOf reads the rules of the node, sorted, each with what it concerns.
func rulesOf(indirections map[string]string, u *usage) []*rule {
	byTarget := make(map[string]*rule, len(indirections))
	list := make([]*rule, 0, len(indirections))

	for target, realAddr := range indirections {
		withPort, _ := hasPort(target)
		r := &rule{Target: target, Real: realAddr, HostOnly: !withPort}
		byTarget[target] = r
		list = append(list, r)
	}

	if u != nil {
		for _, s := range u.servers {
			if r := ruleOf(byTarget, s.Address.Host, s.Address.Port); r != nil {
				r.uses.servers = append(r.uses.servers, named{ID: s.ID, Name: s.Name})
			}
		}

		for _, p := range u.partners {
			if r := ruleOf(byTarget, p.Address.Host, p.Address.Port); r != nil {
				r.uses.partners = append(r.uses.partners, named{ID: p.ID, Name: p.Name})
			}
		}

		if r := ruleOf(byTarget, u.admin.Host, u.admin.Port); r != nil {
			r.uses.admin = true
		}
	}

	slices.SortStableFunc(list, func(a, b *rule) int { return strings.Compare(a.Target, b.Target) })

	return list
}

// use is an item of the "Concerns" cell: a link to a local server or a
// partner, a count, or the administration (a warning).
type use struct {
	Label string
	URL   string
	Warn  bool
}

// usesOf says what a rule concerns: the local servers and the partners, by
// name when there is one, else by count; the administration, as a warning.
func usesOf(translate gui.Translator, r *rule) []use {
	var out []use

	add := func(list []named, oneKey, manyKey, path string) {
		switch n := len(list); {
		case n == 0:
		case n <= namesShown:
			out = append(out, use{
				Label: translate(oneKey, "name", list[0].Name),
				URL:   common.WebuiURL(path + "/" + strconv.FormatInt(list[0].ID, 10)),
			})
		default:
			out = append(out, use{Label: translate(manyKey, "n", n)})
		}
	}

	add(r.uses.servers, "overrides.uses.server", "overrides.uses.servers", "/servers")
	add(r.uses.partners, "overrides.uses.partner", "overrides.uses.partners", "/partners")

	if r.uses.admin {
		out = append(out, use{Label: translate("overrides.uses.admin"), Warn: true})
	}

	return out
}

type listQuery = liststate.Query

func parseListQuery(v url.Values) *listQuery { return listSpec.Parse(v) }

func ret(r *http.Request) string   { return listSpec.Ret(r) }
func listURL(ret string) string    { return listSpec.ListURL(ret) }
func withRet(u, ret string) string { return liststate.WithRet(u, ret) }

// ruleURL is the address of a page of a rule: its configured address goes in
// a parameter (it holds dots, colons, brackets).
func ruleURL(action, target, carried string) string {
	return withRet(liststate.WithQuery(common.WebuiURL(pagePath+"/"+action), url.Values{targetParam: {target}}), carried)
}

type row struct {
	Target   string
	Real     string
	Scope    string
	Uses     []use
	Actions  []gui.Action
	HasAdmin bool
}

type listData struct {
	Node        string
	Enabled     bool
	AddURL      string
	Base        string
	Sort        string
	Dir         string
	Rows        []row
	Caption     string
	Pager       *liststate.Pager
	PagerHidden map[string]string
	Sizes       []int
	Size        int
	Empty       bool
	AdminWarn   bool
	Error       bool
	RetryURL    string
}

// List serves GET /webui/overrides: the rules of the node, or, on an
// instance started without an instance name, why there is none.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	if listSpec.Restore(w, r) {
		return
	}

	translate := h.shell.Translator(r)
	query := parseListQuery(r.URL.Query())
	screen := h.screen(translate, translate("overrides.title"), "", 0)
	screen.Crumbs = crumbs(translate, "", "")

	overrides, nodeID := h.node()
	if overrides == nil {
		h.shell.Render(w, h.logger, r, screen, listView(listData{}))

		return
	}

	u, err := h.usage()
	if err != nil {
		h.logger.Errorf("failed to read the configuration the overrides apply to: %v", err)

		screen.Status = http.StatusInternalServerError
		h.shell.Render(w, h.logger, r, screen, listView(listData{
			Enabled: true, Node: nodeID, Error: true, RetryURL: listSpec.ListURLOf(query.WithPage(query.Page)),
		}))

		return
	}

	rules := rulesOf(overrides.GetAllIndirections(), u)
	screen.Notices = notices(translate, r.URL.Query(), rules)
	h.shell.Render(w, h.logger, r, screen, listView(buildList(translate, rules, query, rightsOf(r), nodeID)))
}

// buildList computes the list: sort, page, row actions according to the
// rights.
func buildList(translate gui.Translator, rules []*rule, query *listQuery, user rights, nodeID string) listData {
	data := listData{
		Node:    nodeID,
		Enabled: true,
		Base:    listSpec.ListURLOf(query.FilterValues()),
		Sort:    query.Sort,
		Dir:     query.Dir,
		Size:    query.Size,
	}

	carried := query.Encode(query.Page)
	if user.Write {
		data.AddURL = withRet(common.WebuiURL(pagePath+"/new"), carried)
	}

	if len(rules) == 0 {
		data.Empty = true

		return data
	}

	sorted := slices.Clone(rules)
	if query.Dir == dirDesc {
		slices.Reverse(sorted)
	}

	b := query.Bounds(len(sorted))
	carried = query.Encode(b.Page)
	data.Caption = translate("overrides.caption",
		"page", b.Page, "pages", b.Pages, "sort", translate(sortKeyPrefix+query.Sort))

	for _, rl := range sorted[b.From:b.To] {
		data.Rows = append(data.Rows, row{
			Target:   rl.Target,
			Real:     rl.Real,
			Scope:    scopeText(translate, rl),
			Uses:     usesOf(translate, rl),
			Actions:  actions(translate, rl, user, carried),
			HasAdmin: rl.uses.admin,
		})
		data.AdminWarn = data.AdminWarn || rl.uses.admin
	}

	data.Sizes = liststate.DefaultSizes
	data.PagerHidden = query.PagerHidden()
	data.Pager = listSpec.Pager(query, translate("overrides.pager", "from", b.From+1, "to", b.To, "total", len(sorted)), b)

	return data
}

func scopeText(translate gui.Translator, r *rule) string {
	if r.HostOnly {
		return translate("overrides.scope.host")
	}

	return translate("overrides.scope.port")
}

// actions: the menu of a row: edit (write), delete (delete, last, red).
func actions(translate gui.Translator, r *rule, user rights, carried string) []gui.Action {
	var acts []gui.Action

	if user.Write {
		acts = append(acts, gui.Action{
			Icon: "pen", Label: translate("overrides.action.edit"), URL: ruleURL("edit", r.Target, carried),
		})
	}

	if user.Delete {
		acts = append(acts, gui.Action{
			Icon: "trash", Label: translate("overrides.action.delete"), URL: ruleURL("delete", r.Target, carried),
			Danger: true,
		})
	}

	return acts
}

// crumbs is the breadcrumb: the list alone, or the list (as the user left
// it) then the page.
func crumbs(translate gui.Translator, last, carried string) []gui.Crumb {
	c := []gui.Crumb{
		{Label: translate("nav.home"), URL: common.WebuiURL("/home")},
		{Label: translate("nav.administration")},
	}

	if last == "" {
		return append(c, gui.Crumb{Label: translate("overrides.title")})
	}

	return append(c,
		gui.Crumb{Label: translate("overrides.title"), URL: listURL(carried)},
		gui.Crumb{Label: last})
}

// notices turns the result of the previous action into a notification, with
// when it applies and, for a rule that concerns the administration, the
// warning.
func notices(translate gui.Translator, values url.Values, rules []*rule) []gui.Notice {
	done := values.Get(doneParam)
	if !slices.Contains(doneKinds, done) {
		return nil
	}

	target := values.Get(nameParam)
	notice := gui.Notice{
		Kind: stateOK, Title: translate(toastKeyPrefix+done, "name", target), Text: translate("overrides.timing"),
	}

	for _, r := range rules {
		if r.Target == target && r.uses.admin {
			notice.Text += " " + translate("overrides.admin_warning")
		}
	}

	return []gui.Notice{notice}
}

// screen is a page of the screen: the chrome around the data.
func (h *Handlers) screen(translate gui.Translator, title, carried string, status int) common.Screen {
	return common.Screen{
		Title:       title,
		Section:     constants.SidebarSectionAdministration,
		Menu:        constants.SidebarMenuOverrides,
		Doc:         docPage,
		Crumbs:      crumbs(translate, title, carried),
		Status:      status,
		Stylesheets: []string{stylesheet},
	}
}

// -------------------------------------------------------------- pages ----

type notFoundData struct {
	Title     string
	Text      string
	BackLabel string
	BackURL   string
}

// target reads the rule named by ?target= on a node that has overrides; a
// missing rule, or an instance without overrides, renders the localised "not
// found" page and returns false.
func (h *Handlers) target(w http.ResponseWriter, r *http.Request) (*conf.ConfigOverride, *rule, bool) {
	overrides, _ := h.node()
	target := r.URL.Query().Get(targetParam)

	if overrides != nil {
		if realAddr := overrides.GetIndirection(target); target != "" && realAddr != "" {
			withPort, _ := hasPort(target)

			return overrides, &rule{Target: target, Real: realAddr, HostOnly: !withPort}, true
		}
	}

	h.notFound(w, r)

	return nil, nil, false
}

func (h *Handlers) notFound(w http.ResponseWriter, r *http.Request) {
	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("overrides.not_found.title")

	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, http.StatusNotFound), notFoundView(notFoundData{
		Title:     title,
		Text:      translate("overrides.not_found.text"),
		BackLabel: translate("overrides.back"),
		BackURL:   listURL(carried),
	}))
}

// concerns completes a rule with what it concerns; a failure to read the
// shared configuration leaves it empty (logged): the page still serves.
func (h *Handlers) concerns(rl *rule) {
	u, err := h.usage()
	if err != nil {
		h.logger.Errorf("failed to read the configuration the overrides apply to: %v", err)

		return
	}

	*rl = *rulesOf(map[string]string{rl.Target: rl.Real}, u)[0]
}

type deleteData struct {
	Title     string
	Target    string
	Real      string
	Scope     string
	Uses      []use
	CancelURL string
	Action    string
}

// DeleteForm serves GET /webui/overrides/delete?target=…: the confirmation
// page, with what the rule concerns and when its removal applies.
func (h *Handlers) DeleteForm(w http.ResponseWriter, r *http.Request) {
	_, rl, ok := h.target(w, r)
	if !ok {
		return
	}

	h.concerns(rl)

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("overrides.delete.title", "name", rl.Target)

	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, 0), deleteView(deleteData{
		Title:     title,
		Target:    rl.Target,
		Real:      rl.Real,
		Scope:     scopeText(translate, rl),
		Uses:      usesOf(translate, rl),
		CancelURL: listURL(carried),
		Action:    ruleURL("delete", rl.Target, carried),
	}))
}

// Delete serves POST /webui/overrides/delete?target=…: the rule leaves the
// file of the node; the configured address is used as is again.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	overrides, rl, ok := h.target(w, r)
	if !ok {
		return
	}

	if err := overrides.RemoveIndirection(rl.Target); err != nil {
		h.logger.Errorf("failed to delete the address indirection %q: %v", rl.Target, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	listSpec.RedirectList(w, r, ret(r), url.Values{doneParam: {doneDeleted}, nameParam: {rl.Target}})
}

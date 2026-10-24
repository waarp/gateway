// Package clouds serves the "Cloud instances" screen of the web interface
// (Administration): the list of the cloud instances of the gateway with
// their type, their location and how they authenticate (searched, filtered,
// sorted and paged), the sheet of an instance in reading, and the pages that
// create, edit and delete one.
//
// A cloud instance is a remote storage the gateway uses instead of the local
// disk: once declared, its name prefixes the paths ("name:folder/file") of
// the rules, the servers and the transfers. It holds a name, a type among
// the five storages the product registers, a key and a secret whose meaning
// depends on the type, and options the storage library reads. The secret is
// encrypted at rest and never rendered; the product instantiates the storage
// at save time, so a refused configuration is never stored.
package clouds

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
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	// The stylesheet of the screen, under the web interface.
	stylesheet = "/static_v2/clouds/clouds.css"

	docPage  = "user/webui/instances-cloud"
	pagePath = "/clouds"

	// The sheet of a local server (#708), where a server that names the
	// instance leads.
	serversPath = "/servers"

	// The cookie that keeps the state of the list for the browser session, so
	// that coming back through the sidebar finds the list as it was left.
	stateCookie = "webui_clouds_list"

	// The parameters of the list state and of the result of the previous
	// action, shared by the list screens.
	retParam  = liststate.RetParam
	doneParam = liststate.DoneParam
	nameParam = liststate.NameParam
	dirDesc   = liststate.DirDesc

	sortName     = "name"
	sortType     = "type"
	sortLocation = "location"
	sortAuth     = "auth"

	doneCreated = liststate.DoneCreated
	doneUpdated = liststate.DoneUpdated
	doneDeleted = liststate.DoneDeleted

	// The filter of the list: the storage types.
	typeParam = "type"

	// Prefixes of the catalogue keys chosen at run time: the sort name of a
	// column ("clouds.sort.type"), how an instance authenticates
	// ("clouds.auth.env"), the result of an action ("clouds.toast.created").
	sortKeyPrefix  = "clouds.sort."
	authKeyPrefix  = "clouds.auth."
	toastKeyPrefix = "clouds.toast."

	// Kinds of a gui.Notice.
	stateOK = "ok"
)

//nolint:gochecknoglobals // reference data of the screen
var (
	sortKeys  = []string{sortName, sortType, sortLocation, sortAuth}
	doneKinds = []string{doneCreated, doneUpdated, doneDeleted}

	cloudsCounter = liststate.Counter{
		One: "clouds.count_one", Many: "clouds.count",
		ShownOne: "clouds.count_shown_one", ShownMany: "clouds.count_shown",
	}

	// The list of this screen (listSpec): its page, its session cookie, its
	// sort keys, its rows per page and the type filter.
	listSpec = liststate.Spec{
		Path: pagePath, Cookie: stateCookie, Sorts: sortKeys, Sizes: liststate.DefaultSizes,
		ParseFilters: parseFilter, FilterParams: []string{typeParam},
	}
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

// rights of the current user on the cloud instances: those of the
// administration. Reading is implied by reaching the screen; writing and
// deleting are checked again by the router on each action.
type rights struct{ Write, Delete bool }

func rightsOf(r *http.Request) rights {
	perms := common.GetUser(r).Permissions

	return rights{
		Write:  perms.HasPermission(model.PermAdminWrite),
		Delete: perms.HasPermission(model.PermAdminDelete),
	}
}

// choice is an option of a select; kv a row of a key/value list.
type (
	choice = protoform.Choice
	kv     = protoform.KV
)

// entry is a cloud instance with its storage type (nil when the product
// registers no storage under its type key).
type entry struct {
	cloud   *model.CloudInstance
	storage *storage
}

func entryOf(cloud *model.CloudInstance) entry {
	s, _ := storageOf(cloud.Type)

	return entry{cloud: cloud, storage: s}
}

// location is the value of the option that locates the storage (bucket,
// share, drive), "" when the storage has none or it is not set.
func (e *entry) location() string {
	if e.storage == nil || e.storage.Location == "" {
		return ""
	}

	return e.cloud.Options[e.storage.Location]
}

// auth says how the instance authenticates (see authOf).
func (e *entry) auth() string { return authOf(e.cloud.Key, e.cloud.Secret, e.cloud.Options) }

// --------------------------------------------------------------- list ----

// filter keeps the instances of the chosen storage types (an alias counts
// as its storage).
type filter struct {
	Types []string // canonical keys
}

// parseFilter reads the type filter from a query, dropping what is unknown.
func parseFilter(v url.Values) liststate.Filters {
	f := &filter{}

	for _, key := range v[typeParam] {
		if s, ok := storageOf(key); ok && !slices.Contains(f.Types, s.Key) {
			f.Types = append(f.Types, s.Key)
		}
	}

	return f
}

func (f *filter) Values() url.Values {
	v := url.Values{}
	for _, key := range f.Types {
		v.Add(typeParam, key)
	}

	return v
}

func (f *filter) Count() int { return len(f.Types) }

// Matches tells whether an instance passes the filter: no type chosen, or
// its storage among the chosen ones.
func (f *filter) Matches(e *entry) bool {
	if len(f.Types) == 0 {
		return true
	}

	return e.storage != nil && slices.Contains(f.Types, e.storage.Key)
}

// typeBox is a checkbox of the type filter.
type typeBox struct {
	Key     string
	Label   string
	Checked bool
}

func (f *filter) boxes(translate gui.Translator) []typeBox {
	boxes := make([]typeBox, 0, len(storages))
	for i := range storages {
		s := &storages[i]
		boxes = append(boxes, typeBox{Key: s.Key, Label: translate(s.labelKey()), Checked: slices.Contains(f.Types, s.Key)})
	}

	return boxes
}

// listQuery is the state of the list asked by the URL, with the type filter.
type listQuery = liststate.Query

func parseListQuery(v url.Values) *listQuery { return listSpec.Parse(v) }

func filterOf(q *listQuery) *filter {
	if f, ok := q.Filters.(*filter); ok {
		return f
	}

	return &filter{}
}

// The list state carried by a request, the list as the user left it, and a
// URL carrying the state: the shared mechanics, on the spec of this list.
func ret(r *http.Request) string   { return listSpec.Ret(r) }
func listURL(ret string) string    { return listSpec.ListURL(ret) }
func withRet(u, ret string) string { return liststate.WithRet(u, ret) }

func cloudURL(id int64, suffix string) string {
	return common.WebuiURL(pagePath + "/" + strconv.FormatInt(id, 10) + suffix)
}

type row struct {
	ID       int64
	Name     string
	URL      string
	Type     string // the label of the storage, with the stored key when it is an alias
	Location string // "" when the storage has none
	Auth     string
	Actions  []gui.Action
}

type listData struct {
	AddURL       string // empty without the write right: no coloured action at all
	Action       string // toolbar form action: the bare list
	Base         string // list with search and filters: base of the sort links
	Q            string
	Names        []string
	Types        []typeBox
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
	Empty        bool // no instance at all
	NoMatch      bool // instances exist, none matches
	NoMatchText  string
	NoMatchURL   string
	NoMatchLink  string
	Error        bool // the instances could not be read
	RetryURL     string
}

// List serves GET /webui/clouds.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	fragment := liststate.IsListFragment(r)
	if !fragment && listSpec.Restore(w, r) {
		return
	}

	translate := h.shell.Translator(r)
	query := parseListQuery(r.URL.Query())
	screen := h.screen(translate, translate("clouds.title"), "", 0)
	screen.Crumbs = crumbs(translate, "", "")
	screen.Notices = notices(translate, r.URL.Query())
	screen.Head = gui.LiveAssets()

	clouds, err := internal.ListClouds(h.db, "name", true, 0, 0)
	if err != nil {
		h.logger.Errorf("failed to list the cloud instances: %v", err)

		if fragment {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}

		screen.Status = http.StatusInternalServerError
		h.shell.Render(w, h.logger, r, screen,
			listView(listData{Error: true, RetryURL: listSpec.ListURLOf(query.WithPage(query.Page))}))

		return
	}

	entries := make([]entry, 0, len(clouds))
	for _, cloud := range clouds {
		entries = append(entries, entryOf(cloud))
	}

	data := buildList(translate, entries, query, rightsOf(r))
	if fragment {
		h.shell.RenderFragment(w, h.logger, r, listFragment(data))

		return
	}

	h.shell.Render(w, h.logger, r, screen, listView(data))
}

// buildList computes the list page: filter (name prefix, types), sort, page,
// toolbar state, row actions according to the rights.
func buildList(translate gui.Translator, all []entry, query *listQuery, user rights) listData {
	active := filterOf(query)
	data := listData{
		Action:       listSpec.ListURLOf(nil),
		Base:         listSpec.ListURLOf(query.FilterValues()),
		Q:            query.Q,
		Sort:         query.Sort,
		Dir:          query.Dir,
		Size:         query.Size,
		ResetURL:     listSpec.ResetURL(),
		Hidden:       query.ToolbarHidden(),
		Types:        active.boxes(translate),
		FiltersCount: active.Count(),
		FiltersOpen:  active.Count() > 0,
	}

	if user.Write {
		data.AddURL = withRet(common.WebuiURL(pagePath+"/new"), query.Encode(query.Page))
	}

	names := make([]string, 0, len(all))
	kept := make([]entry, 0, len(all))
	needle := strings.ToLower(query.Q)

	for i := range all {
		e := &all[i]
		names = append(names, e.cloud.Name)

		if needle != "" && !strings.HasPrefix(strings.ToLower(e.cloud.Name), needle) {
			continue
		}

		if !active.Matches(e) {
			continue
		}

		kept = append(kept, *e)
	}

	slices.Sort(names)
	data.Names = names
	sortEntries(translate, kept, query.Sort, query.Dir)

	data.Count = cloudsCounter.Text(translate, len(all), len(kept))

	switch {
	case len(all) == 0:
		data.Empty = true

		return data
	case len(kept) == 0:
		data.NoMatch = true

		if query.Q != "" {
			withoutQ := *query
			withoutQ.Q = ""
			data.NoMatchText = translate("clouds.none.title", "query", query.Q)
			data.NoMatchURL, data.NoMatchLink = listSpec.ExplicitListURL(withoutQ.Values()), translate("search.clear")
		} else {
			data.NoMatchText = translate("clouds.none.filters")
			data.NoMatchURL, data.NoMatchLink = listSpec.ResetURL(), translate("filters.reset")
		}

		return data
	}

	b := query.Bounds(len(kept))
	carried := query.Encode(b.Page)
	data.Caption = translate("clouds.caption",
		"page", b.Page, "pages", b.Pages, "sort", translate(sortKeyPrefix+query.Sort))

	for i := range kept[b.From:b.To] {
		e := &kept[b.From+i]
		data.Rows = append(data.Rows, row{
			ID:       e.cloud.ID,
			Name:     e.cloud.Name,
			URL:      withRet(cloudURL(e.cloud.ID, ""), carried),
			Type:     label(translate, e.cloud.Type),
			Location: e.location(),
			Auth:     translate(authKeyPrefix + e.auth()),
			Actions:  actions(translate, e, user, false, carried),
		})
	}

	// The rows per page are offered as soon as there are rows.
	data.Sizes = liststate.DefaultSizes
	data.PagerHidden = query.PagerHidden()
	data.Pager = listSpec.Pager(query, translate("clouds.pager", "from", b.From+1, "to", b.To, "total", len(kept)), b)

	return data
}

// sortEntries sorts by the asked column; ties keep the name order
// (regardless of case).
func sortEntries(translate gui.Translator, entries []entry, key, dir string) {
	byName := func(a, b *entry) int {
		return strings.Compare(strings.ToLower(a.cloud.Name), strings.ToLower(b.cloud.Name))
	}

	slices.SortStableFunc(entries, func(a, b entry) int {
		var order int

		switch key {
		case sortType:
			order = strings.Compare(label(translate, a.cloud.Type), label(translate, b.cloud.Type))
		case sortLocation:
			order = strings.Compare(a.location(), b.location())
		case sortAuth:
			order = strings.Compare(a.auth(), b.auth())
		default:
			order = byName(&a, &b)
		}

		if dir == dirDesc {
			order = -order
		}

		return cmp.Or(order, byName(&a, &b))
	})
}

// actions is the "⋯" menu of an instance, in the list (with "View the
// sheet" and "Edit") and on the sheet (without: the buttons are there).
// Entries follow the rights; the destructive one comes last. Every URL
// carries the list state.
func actions(translate gui.Translator, e *entry, user rights, onSheet bool, carried string) []gui.Action {
	id := e.cloud.ID

	var acts []gui.Action

	if !onSheet {
		acts = append(acts, gui.Action{
			Icon: "eye", Label: translate("clouds.action.detail"), URL: withRet(cloudURL(id, ""), carried),
		})

		if user.Write {
			acts = append(acts, gui.Action{
				Icon: "pen", Label: translate("clouds.action.edit"), URL: withRet(cloudURL(id, "/edit"), carried),
			})
		}
	}

	if user.Delete {
		acts = append(acts, gui.Action{
			Icon:   "trash",
			Label:  translate("clouds.action.delete"),
			URL:    withRet(cloudURL(id, "/delete"), carried),
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
		return append(c, gui.Crumb{Label: translate("clouds.title")})
	}

	return append(c,
		gui.Crumb{Label: translate("clouds.title"), URL: listURL(carried)},
		gui.Crumb{Label: last})
}

// notices turns the result of the previous action, carried by the
// redirection, into a notification.
func notices(translate gui.Translator, values url.Values) []gui.Notice {
	if done := values.Get(doneParam); slices.Contains(doneKinds, done) {
		return []gui.Notice{{Kind: stateOK, Title: translate(toastKeyPrefix+done, "name", values.Get(nameParam))}}
	}

	return nil
}

// -------------------------------------------------------------- pages ----

// notFoundData is the localised "not found" page.
type notFoundData struct {
	Title     string
	Text      string
	BackLabel string
	BackURL   string
}

// cloud reads the {id} of the route; a missing instance renders the
// localised "not found" page and returns false.
func (h *Handlers) cloud(w http.ResponseWriter, r *http.Request) (entry, bool) {
	id, parseErr := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

	cloud, err := internal.GetCloudByID(h.db, id)
	if parseErr == nil && err == nil {
		return entryOf(cloud), true
	}

	if err != nil && !database.IsNotFound(err) {
		h.logger.Errorf("failed to retrieve the cloud instance %d: %v", id, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return entry{}, false
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("clouds.not_found.title")

	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, http.StatusNotFound), notFoundView(notFoundData{
		Title:     title,
		Text:      translate("clouds.not_found.text"),
		BackLabel: translate("clouds.back"),
		BackURL:   listURL(carried),
	}))

	return entry{}, false
}

// screen is a page of the screen: the chrome around the data. The list
// itself replaces the breadcrumb and the notices.
func (h *Handlers) screen(translate gui.Translator, title, carried string, status int) common.Screen {
	return common.Screen{
		Title:       title,
		Section:     constants.SidebarSectionAdministration,
		Menu:        constants.SidebarMenuClouds,
		Doc:         docPage,
		Crumbs:      crumbs(translate, title, carried),
		Status:      status,
		Stylesheets: []string{stylesheet},
	}
}

// namer is an object whose path names the instance: its name and, when its
// screen is reworked, the link to its sheet.
type namer struct {
	Name string
	URL  string
}

// usage is what names the instance in a path: the rules by their local
// folder, the servers by their root or one of their folders, the transfers
// by their local path (counted: their names mean nothing to a reader).
type usage struct {
	Rules     []namer
	Servers   []namer
	Transfers int64
}

func (u *usage) none() bool { return len(u.Rules) == 0 && len(u.Servers) == 0 && u.Transfers == 0 }

// usageOf finds what names the instance: a path that starts with
// "name:" (pkg/fs/path.go).
func (h *Handlers) usageOf(name string) (usage, error) {
	prefix := name + ":%"

	var (
		found usage
		rules model.Rules
	)

	if err := h.db.Select(&rules).Where("local_dir LIKE ?", prefix).OrderBy("name", true).Run(); err != nil {
		return found, err //nolint:wrapcheck // logged by the caller with its context
	}

	for _, rule := range rules {
		found.Rules = append(found.Rules, namer{Name: rule.Name})
	}

	var servers model.LocalAgents

	if err := h.db.Select(&servers).Where("root_dir LIKE ? OR receive_dir LIKE ? OR send_dir LIKE ?",
		prefix, prefix, prefix).OrderBy("name", true).Run(); err != nil {
		return found, err //nolint:wrapcheck // logged by the caller with its context
	}

	for _, server := range servers {
		found.Servers = append(found.Servers, namer{
			Name: server.Name, URL: common.WebuiURL(serversPath + "/" + strconv.FormatInt(server.ID, 10)),
		})
	}

	transfers, err := h.db.Count(&model.Transfer{}).Where("local_path LIKE ?", prefix).Run()
	if err != nil {
		return found, err //nolint:wrapcheck // logged by the caller with its context
	}

	found.Transfers = int64(transfers)

	return found, nil
}

type detailData struct {
	Name     string
	Type     string
	TypeHint string // what the storage is; "" for an unknown type key
	Path     string // the path to use: "name:folder/file"
	BackURL  string
	EditURL  string
	Actions  []gui.Action
	KeyLabel string // what the key is for the storage; "" when the type is unknown
	Key      string
	Secret   bool // set or not: the value is never rendered
	EnvAuth  bool
	Options  []kv     // the known options, with their label; absent ones say the product default
	Others   []kv     // the options the form does not name, by raw key
	Masked   []string // the options the storage library masks: named, never rendered
	Usage    usage
}

// Detail serves GET /webui/clouds/{id}: the instance in reading, with what
// names it.
func (h *Handlers) Detail(w http.ResponseWriter, r *http.Request) {
	inst, ok := h.cloud(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	user := rightsOf(r)

	found, err := h.usageOf(inst.cloud.Name)
	if err != nil {
		h.logger.Errorf("failed to find what names the cloud instance %q: %v", inst.cloud.Name, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	data := detailData{
		Name:    inst.cloud.Name,
		Type:    label(translate, inst.cloud.Type),
		Path:    inst.cloud.Name + ":" + translate("clouds.detail.path_example"),
		BackURL: listURL(carried),
		Actions: actions(translate, &inst, user, true, carried),
		Key:     inst.cloud.Key,
		Secret:  inst.cloud.Secret != "",
		EnvAuth: inst.auth() == "env",
		Usage:   found,
	}

	if inst.storage != nil {
		data.TypeHint = translate(inst.storage.hintKey())
		data.KeyLabel = translate(inst.storage.keyLabelKey())
		data.Options = optionRows(translate, inst.storage, inst.cloud.Options)
	} else {
		data.KeyLabel = translate("clouds.form.key.any")
	}

	data.Others = otherRows(inst.storage, inst.cloud.Options)
	data.Masked = maskedKeys(inst.storage, inst.cloud.Options)

	if user.Write {
		data.EditURL = withRet(cloudURL(inst.cloud.ID, "/edit"), carried)
	}

	title := translate("clouds.detail.title", "name", inst.cloud.Name)
	screen := h.screen(translate, title, carried, 0)
	screen.Notices = notices(translate, r.URL.Query())
	h.shell.Render(w, h.logger, r, screen, detailView(data))
}

// optionRows reads the options the storage names, in its order: the value
// set, or what the product applies when it is absent (the library default),
// or that a required option is not set.
func optionRows(translate gui.Translator, s *storage, options map[string]string) []kv {
	resolved := s.resolved()
	rows := make([]kv, 0, len(resolved))

	for i := range resolved {
		o := &resolved[i]
		value, set := options[o.Key]

		switch {
		case o.Kind == kindCheck:
			rows = append(rows, kv{Label: translate(o.labelKey()), Value: yesNo(translate, strings.EqualFold(value, "true"))})
		case set && value != "":
			rows = append(rows, kv{Label: translate(o.labelKey()), Value: o.choiceLabel(translate, value), Mono: o.Mono})
		case o.Default != "":
			rows = append(rows, kv{Label: translate(o.labelKey()), Value: o.choiceLabel(translate, o.Default), Mono: o.Mono})
		case o.Required:
			rows = append(rows, kv{Label: translate(o.labelKey()), Value: translate("clouds.detail.unset")})
		default:
			rows = append(rows, kv{Label: translate(o.labelKey()), Value: translate("clouds.detail.default")})
		}
	}

	return rows
}

// otherRows reads the options the form does not name, by raw key, sorted;
// the secrets and the credentials never show, nor the environment switch
// (said with the access).
func otherRows(s *storage, options map[string]string) []kv {
	keys := make([]string, 0, len(options))
	masked := maskedKeys(s, options)

	for key := range options {
		if key == optEnvAuth || isSecret(key) || (s != nil && s.knows(key)) || slices.Contains(masked, key) {
			continue
		}

		keys = append(keys, key)
	}

	slices.Sort(keys)

	rows := make([]kv, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, kv{Label: key, Value: options[key], Mono: true})
	}

	return rows
}

// maskedKeys returns, sorted, the options of an instance that the storage
// library masks (secrets and identifiers set by the REST API or the command
// line): the sheet names them without their value, the form keeps them as
// they are. The credentials and the token are left out: never rendered,
// never written.
func maskedKeys(s *storage, options map[string]string) []string {
	if s == nil {
		return nil
	}

	lib, ok := libraryOf(s)
	if !ok {
		return nil
	}

	var keys []string

	for key := range options {
		if s.offers(key) && lib.sensitive(key) {
			keys = append(keys, key)
		}
	}

	slices.Sort(keys)

	return keys
}

func yesNo(translate gui.Translator, yes bool) string {
	if yes {
		return translate("clouds.detail.yes")
	}

	return translate("clouds.detail.no")
}

type deleteData struct {
	Name      string
	Type      string
	Location  string
	CancelURL string
	Action    string
	Usage     usage
}

// DeleteForm serves GET /webui/clouds/{id}/delete: the confirmation page,
// with what names the instance.
func (h *Handlers) DeleteForm(w http.ResponseWriter, r *http.Request) {
	e, ok := h.cloud(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)

	found, err := h.usageOf(e.cloud.Name)
	if err != nil {
		h.logger.Errorf("failed to find what names the cloud instance %q: %v", e.cloud.Name, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	data := deleteData{
		Name:      e.cloud.Name,
		Type:      label(translate, e.cloud.Type),
		Location:  e.location(),
		CancelURL: withRet(cloudURL(e.cloud.ID, ""), carried),
		Action:    withRet(cloudURL(e.cloud.ID, "/delete"), carried),
		Usage:     found,
	}

	title := translate("clouds.delete.title", "name", e.cloud.Name)
	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, 0), deleteView(data))
}

// Delete serves POST /webui/clouds/{id}/delete: the product refuses nothing,
// the paths that name the instance simply stop working.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	e, ok := h.cloud(w, r)
	if !ok {
		return
	}

	if err := internal.DeleteCloud(h.db, e.cloud); err != nil {
		h.logger.Errorf("failed to delete the cloud instance %q: %v", e.cloud.Name, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	listSpec.RedirectList(w, r, ret(r), url.Values{doneParam: {doneDeleted}, nameParam: {e.cloud.Name}})
}

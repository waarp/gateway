// Package keys serves the "Cryptographic keys" screen of the web interface
// (Treatments): the list of the keys the tasks of the rules use to encrypt,
// decrypt, sign and verify the files (searched, filtered by type, sorted and
// paged), with what their value says and the rules that use them, the sheet
// of a key in reading, and the pages that create, edit and delete one.
//
// A key holds a name, a type (AES, HMAC, PGP public, PGP private) and a
// value; the values of the AES, HMAC and PGP private keys are secrets, never
// rendered back. A task names its key in its arguments, without a link in
// the database: the screen keeps the name and the type of a key a rule uses,
// and replaces its deletion by the list of the rules to change first. Until
// this screen, the keys were managed through the REST API and the command
// line only.
package keys

import (
	"cmp"
	"net/http"
	"net/url"
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
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/ruleuse"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	// The stylesheet of the screen, under the web interface.
	stylesheet = "/static_v2/keys/keys.css"

	docPage  = "user/webui/cles"
	pagePath = "/keys"

	// The cookie that keeps the state of the list for the browser session.
	stateCookie = "webui_keys_list"

	doneParam = liststate.DoneParam
	nameParam = liststate.NameParam
	dirDesc   = liststate.DirDesc

	sortName = "name"
	sortType = "type"

	doneCreated = liststate.DoneCreated
	doneUpdated = liststate.DoneUpdated
	doneDeleted = liststate.DoneDeleted

	// The filter of the list: the types.
	typeParam = "type"

	// The short key ID of a PGP key: its last 8 hexadecimal digits.
	shortKeyID = 8

	// Prefixes of the catalogue keys chosen at run time.
	sortKeyPrefix  = "keys.sort."
	toastKeyPrefix = "keys.toast."
	typeKeyPrefix  = "keys.type."
	causeKeyPrefix = "keys.cause."
	roleKeyPrefix  = "keys.role."
)

//nolint:gochecknoglobals // reference data of the screen
var (
	sortKeys  = []string{sortName, sortType}
	doneKinds = []string{doneCreated, doneUpdated, doneDeleted}

	counter = liststate.Counter{
		One: "keys.count_one", Many: "keys.count", ShownOne: "keys.count_shown_one", ShownMany: "keys.count_shown",
	}

	listSpec = liststate.Spec{
		Path: pagePath, Cookie: stateCookie, Sorts: sortKeys, Sizes: liststate.DefaultSizes,
		ParseFilters: parseFilter, FilterParams: []string{typeParam},
	}
)

// kind is a type of key the product knows: the suffix of its value fields
// in the form, whether its value is a secret (never rendered back), whether
// it is a PGP armour (uploaded or pasted).
type kind struct {
	Key    string
	Slug   string
	Secret bool
	PGP    bool
}

//nolint:gochecknoglobals // reference data of the product, the types it accepts
var kinds = []kind{
	{Key: model.CryptoKeyTypeAES, Slug: "aes", Secret: true},
	{Key: model.CryptoKeyTypeHMAC, Slug: "hmac", Secret: true},
	{Key: model.CryptoKeyTypePGPPublic, Slug: "pgp_public", PGP: true},
	{Key: model.CryptoKeyTypePGPPrivate, Slug: "pgp_private", Secret: true, PGP: true},
}

func kindOf(key string) (*kind, bool) {
	for i := range kinds {
		if kinds[i].Key == key {
			return &kinds[i], true
		}
	}

	return nil, false
}

// Handlers serves the pages and the actions of the screen.
type Handlers struct {
	db     *database.DB
	logger *log.Logger
	shell  *common.Shell
	now    func() time.Time // the clock of the expiry states, fixed by the tests
}

func New(db *database.DB, logger *log.Logger) *Handlers {
	return &Handlers{db: db, logger: logger, shell: common.NewShell(), now: time.Now}
}

// rights of the current user on the keys: those of the administration.
type rights struct{ Write, Delete bool }

func rightsOf(r *http.Request) rights {
	perms := common.GetUser(r).Permissions

	return rights{
		Write:  perms.HasPermission(model.PermAdminWrite),
		Delete: perms.HasPermission(model.PermAdminDelete),
	}
}

type choice = protoform.Choice

func itoa(n int) string { return strconv.Itoa(n) }

// entry is a key with its type, what its value says and the rules whose
// tasks name it.
type entry struct {
	key  *model.CryptoKey
	kind *kind
	info keyInfo
	uses []ruleuse.Use
}

func (e *entry) typeLabel(translate gui.Translator) string {
	if e.kind == nil {
		return e.key.Type
	}

	return translate(typeKeyPrefix + e.kind.Key)
}

// detail is the short summary of the list: the size of an AES key, "secret
// set" for HMAC, the first identity of a PGP key; code is the short key ID
// of a PGP key (its last 8 hexadecimal digits), shown under the identity.
func (e *entry) detail(translate gui.Translator) (text, code string) {
	switch {
	case !e.info.Readable:
		return translate("keys.detail.unreadable"), ""
	case e.key.Type == model.CryptoKeyTypeAES:
		return "AES-" + itoa(e.info.Bits), ""
	case e.kind != nil && e.kind.PGP:
		code = e.info.KeyID[max(0, len(e.info.KeyID)-shortKeyID):]
		if len(e.info.Identities) > 0 {
			text = e.info.Identities[0]
		}

		return text, code
	default:
		return translate("keys.detail.secret"), ""
	}
}

// summary is the detail on one line, for the pages of a key.
func (e *entry) summary(translate gui.Translator) string {
	text, code := e.detail(translate)
	if text == "" || code == "" {
		return text + code
	}

	return text + " · " + code
}

// filter is the state of the filter of the list: the types.
type filter struct{ Types []string }

func parseFilter(v url.Values) liststate.Filters {
	f := &filter{}

	for _, key := range v[typeParam] {
		if _, ok := kindOf(key); ok && !slices.Contains(f.Types, key) {
			f.Types = append(f.Types, key)
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

type box struct {
	Name    string
	Value   string
	Label   string
	Checked bool
}

func (f *filter) boxes(translate gui.Translator) []box {
	out := make([]box, 0, len(kinds))
	for i := range kinds {
		out = append(out, box{
			Name: typeParam, Value: kinds[i].Key, Label: translate(typeKeyPrefix + kinds[i].Key),
			Checked: slices.Contains(f.Types, kinds[i].Key),
		})
	}

	return out
}

type listQuery = liststate.Query

func parseListQuery(v url.Values) *listQuery { return listSpec.Parse(v) }

func filterOf(q *listQuery) *filter {
	if f, ok := q.Filters.(*filter); ok {
		return f
	}

	return &filter{}
}

func ret(r *http.Request) string   { return listSpec.Ret(r) }
func listURL(ret string) string    { return listSpec.ListURL(ret) }
func withRet(u, ret string) string { return liststate.WithRet(u, ret) }

func keyURL(id int64, suffix string) string {
	return common.WebuiURL(pagePath + "/" + strconv.FormatInt(id, 10) + suffix)
}

// all reads the keys, by name, each with what its value says and the rules
// whose tasks name it.
func (h *Handlers) all() ([]entry, error) {
	var list model.CryptoKeys
	if err := h.db.Select(&list).OrderBy("name", true).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	uses, err := h.uses()
	if err != nil {
		return nil, err
	}

	now := h.now()
	entries := make([]entry, 0, len(list))

	for _, k := range list {
		entries = append(entries, entryOf(k, now, uses))
	}

	return entries, nil
}

func entryOf(k *model.CryptoKey, now time.Time, uses map[string][]ruleuse.Use) entry {
	e := entry{key: k, info: infoOf(k, now), uses: uses[k.Name]}
	e.kind, _ = kindOf(k.Type)

	return e
}

// uses reads the tasks that read a key and their rules: for every key name,
// the rules that use it.
func (h *Handlers) uses() (map[string][]ruleuse.Use, error) {
	return ruleuse.Read(h.db, keyParams) //nolint:wrapcheck // logged by the caller with its context
}

type row struct {
	ID      int64
	Name    string
	URL     string
	Type    string
	Detail  string
	Code    string
	State   gui.Status
	Uses    string
	Actions []gui.Action
}

type listData struct {
	AddURL       string
	Action       string
	Base         string
	Q            string
	Names        []string
	Types        []box
	FiltersOpen  bool
	FiltersCount int
	ResetURL     string
	Sort         string
	Dir          string
	Hidden       map[string]string
	Count        string
	Rows         []row
	Caption      string
	Pager        *liststate.Pager
	PagerHidden  map[string]string
	Sizes        []int
	Size         int
	Empty        bool
	NoMatch      bool
	NoMatchText  string
	NoMatchURL   string
	NoMatchLink  string
	Error        bool
	RetryURL     string
}

// List serves GET /webui/keys.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	fragment := liststate.IsListFragment(r)
	if !fragment && listSpec.Restore(w, r) {
		return
	}

	translate := h.shell.Translator(r)
	query := parseListQuery(r.URL.Query())
	screen := h.screen(translate, translate("keys.title"), "", 0)
	screen.Crumbs = crumbs(translate, "", "")
	screen.Notices = notices(translate, r.URL.Query())
	screen.Head = gui.LiveAssets()

	entries, err := h.all()
	if err != nil {
		h.logger.Errorf("failed to list the cryptographic keys: %v", err)

		if fragment {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}

		screen.Status = http.StatusInternalServerError
		h.shell.Render(w, h.logger, r, screen,
			listView(listData{Error: true, RetryURL: listSpec.ListURLOf(query.WithPage(query.Page))}))

		return
	}

	data := buildList(translate, entries, query, rightsOf(r), h.now())
	if fragment {
		h.shell.RenderFragment(w, h.logger, r, listFragment(data))

		return
	}

	h.shell.Render(w, h.logger, r, screen, listView(data))
}

// buildList computes the list page: filter (name prefix, types), sort, page,
// toolbar state, row actions according to the rights.
func buildList(translate gui.Translator, all []entry, query *listQuery, user rights, now time.Time) listData {
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
		names = append(names, e.key.Name)

		if needle != "" && !strings.HasPrefix(strings.ToLower(e.key.Name), needle) {
			continue
		}

		if len(active.Types) > 0 && !slices.Contains(active.Types, e.key.Type) {
			continue
		}

		kept = append(kept, *e)
	}

	slices.Sort(names)
	data.Names = names
	sortEntries(translate, kept, query.Sort, query.Dir)
	data.Count = counter.Text(translate, len(all), len(kept))

	switch {
	case len(all) == 0:
		data.Empty = true

		return data
	case len(kept) == 0:
		data.NoMatch = true

		if query.Q != "" {
			withoutQ := *query
			withoutQ.Q = ""
			data.NoMatchText = translate("keys.none.title", "query", query.Q)
			data.NoMatchURL, data.NoMatchLink = listSpec.ExplicitListURL(withoutQ.Values()), translate("search.clear")
		} else {
			data.NoMatchText = translate("keys.none.filters")
			data.NoMatchURL, data.NoMatchLink = listSpec.ResetURL(), translate("filters.reset")
		}

		return data
	}

	b := query.Bounds(len(kept))
	carried := query.Encode(b.Page)
	data.Caption = translate("keys.caption",
		"page", b.Page, "pages", b.Pages, "sort", translate(sortKeyPrefix+query.Sort))

	for i := range kept[b.From:b.To] {
		e := &kept[b.From+i]
		data.Rows = append(data.Rows, row{
			ID:      e.key.ID,
			Name:    e.key.Name,
			URL:     withRet(keyURL(e.key.ID, ""), carried),
			Type:    e.typeLabel(translate),
			State:   e.info.state(translate, now),
			Uses:    ruleuse.Count(translate, len(e.uses)),
			Actions: actions(translate, e, user, false, carried),
		})
		data.Rows[len(data.Rows)-1].Detail, data.Rows[len(data.Rows)-1].Code = e.detail(translate)
	}

	data.Sizes = liststate.DefaultSizes
	data.PagerHidden = query.PagerHidden()
	data.Pager = listSpec.Pager(query, translate("keys.pager", "from", b.From+1, "to", b.To, "total", len(kept)), b)

	return data
}

// sortEntries sorts by the asked column; ties keep the name order
// (regardless of case).
func sortEntries(translate gui.Translator, entries []entry, key, dir string) {
	byName := func(a, b *entry) int {
		return strings.Compare(strings.ToLower(a.key.Name), strings.ToLower(b.key.Name))
	}

	slices.SortStableFunc(entries, func(left, right entry) int {
		order := byName(&left, &right)
		if key == sortType {
			order = strings.Compare(left.typeLabel(translate), right.typeLabel(translate))
		}

		if dir == dirDesc {
			order = -order
		}

		return cmp.Or(order, byName(&left, &right))
	})
}

// actions: the menu of a row, or of the sheet (no "view" there).
func actions(translate gui.Translator, e *entry, user rights, onSheet bool, carried string) []gui.Action {
	var acts []gui.Action

	if !onSheet {
		acts = append(acts, gui.Action{
			Icon: "eye", Label: translate("keys.action.detail"), URL: withRet(keyURL(e.key.ID, ""), carried),
		})

		if user.Write {
			acts = append(acts, gui.Action{
				Icon: "pen", Label: translate("keys.action.edit"), URL: withRet(keyURL(e.key.ID, "/edit"), carried),
			})
		}
	}

	if user.Delete {
		acts = append(acts, gui.Action{
			Icon: "trash", Label: translate("keys.action.delete"), URL: withRet(keyURL(e.key.ID, "/delete"), carried),
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
		{Label: translate("nav.treatments")},
	}

	if last == "" {
		return append(c, gui.Crumb{Label: translate("keys.title")})
	}

	return append(c, gui.Crumb{Label: translate("keys.title"), URL: listURL(carried)}, gui.Crumb{Label: last})
}

// notices turns the result of the previous action into a notification.
func notices(translate gui.Translator, values url.Values) []gui.Notice {
	if done := values.Get(doneParam); slices.Contains(doneKinds, done) {
		return []gui.Notice{{Kind: stateOK, Title: translate(toastKeyPrefix+done, "name", values.Get(nameParam))}}
	}

	return nil
}

// screen is a page of the screen: the chrome around the data.
func (h *Handlers) screen(translate gui.Translator, title, carried string, status int) common.Screen {
	return common.Screen{
		Title:       title,
		Section:     constants.SidebarSectionTreatments,
		Menu:        constants.SidebarMenuCryptoKeys,
		Doc:         docPage,
		Crumbs:      crumbs(translate, title, carried),
		Status:      status,
		Stylesheets: []string{stylesheet, ruleuse.Stylesheet},
	}
}

// -------------------------------------------------------------- pages ----

type notFoundData struct {
	Title     string
	Text      string
	BackLabel string
	BackURL   string
}

// key reads the {id} of the route with what its value says and the rules
// whose tasks name it; a missing key renders the localised "not found" page.
func (h *Handlers) key(w http.ResponseWriter, r *http.Request) (entry, bool) {
	id, parseErr := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

	var k model.CryptoKey

	err := h.db.Get(&k, "id=?", id).Run()
	if parseErr == nil && err == nil {
		var uses map[string][]ruleuse.Use
		if uses, err = h.uses(); err == nil {
			return entryOf(&k, h.now(), uses), true
		}
	}

	if err != nil && !database.IsNotFound(err) {
		h.logger.Errorf("failed to retrieve the cryptographic key %d: %v", id, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return entry{}, false
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("keys.not_found.title")

	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, http.StatusNotFound), notFoundView(notFoundData{
		Title:     title,
		Text:      translate("keys.not_found.text"),
		BackLabel: translate("keys.back"),
		BackURL:   listURL(carried),
	}))

	return entry{}, false
}

type detailData struct {
	Name      string
	TypeLabel string
	TypeKey   string
	Role      string
	Secret    bool
	PGP       bool
	AESBits   int
	Info      keyInfo
	State     gui.Status
	Created   string
	Expires   string
	Uses      []ruleuse.Row
	BackURL   string
	EditURL   string
	Actions   []gui.Action
}

// Detail serves GET /webui/keys/{id}: the key in reading, never a secret,
// with the rules whose tasks name it.
func (h *Handlers) Detail(w http.ResponseWriter, r *http.Request) {
	item, ok := h.key(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	user := rightsOf(r)
	date := translate("auth.date")

	data := detailData{
		Name:      item.key.Name,
		TypeLabel: item.typeLabel(translate),
		TypeKey:   item.key.Type,
		Info:      item.info,
		State:     item.info.state(translate, h.now()),
		Uses:      ruleuse.Rows(translate, item.uses),
		BackURL:   listURL(carried),
		Actions:   actions(translate, &item, user, true, carried),
		AESBits:   item.info.Bits,
	}

	if item.kind != nil {
		data.Role = translate(roleKeyPrefix + item.kind.Key)
		data.Secret = item.kind.Secret
		data.PGP = item.kind.PGP
	}

	if !item.info.Created.IsZero() {
		data.Created = item.info.Created.Format(date)
	}

	data.Expires = translate("keys.detail.never")
	if !item.info.Expires.IsZero() {
		data.Expires = item.info.Expires.Format(date)
	}

	if user.Write {
		data.EditURL = withRet(keyURL(item.key.ID, "/edit"), carried)
	}

	title := translate("keys.detail.title", "name", item.key.Name)
	screen := h.screen(translate, title, carried, 0)
	screen.Notices = notices(translate, r.URL.Query())
	h.shell.Render(w, h.logger, r, screen, detailView(data))
}

type deleteData struct {
	Title     string
	Name      string
	TypeLabel string
	Detail    string
	Uses      []ruleuse.Row
	Used      bool
	Rules     string // how many rules use it
	CancelURL string
	Action    string
}

// DeleteForm serves GET /webui/keys/{id}/delete: the confirmation page; for
// a key a rule uses, the page that lists the rules to change first.
func (h *Handlers) DeleteForm(w http.ResponseWriter, r *http.Request) {
	item, ok := h.key(w, r)
	if !ok {
		return
	}

	h.renderDelete(w, r, &item, 0)
}

func (h *Handlers) renderDelete(w http.ResponseWriter, r *http.Request, item *entry, status int) {
	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("keys.delete.title", "name", item.key.Name)

	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, status), deleteView(deleteData{
		Title:     title,
		Name:      item.key.Name,
		TypeLabel: item.typeLabel(translate),
		Detail:    item.summary(translate),
		Uses:      ruleuse.Rows(translate, item.uses),
		Used:      len(item.uses) > 0,
		Rules:     ruleuse.Count(translate, len(item.uses)),
		CancelURL: withRet(keyURL(item.key.ID, ""), carried),
		Action:    withRet(keyURL(item.key.ID, "/delete"), carried),
	}))
}

// Delete serves POST /webui/keys/{id}/delete. A key a rule uses is not
// deleted (its tasks would fail at the next transfer, the product refusing
// nothing): the page lists the rules again (409).
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	item, ok := h.key(w, r)
	if !ok {
		return
	}

	if len(item.uses) > 0 {
		h.renderDelete(w, r, &item, http.StatusConflict)

		return
	}

	if err := h.db.Delete(item.key).Run(); err != nil {
		h.logger.Errorf("failed to delete the cryptographic key %q: %v", item.key.Name, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	listSpec.RedirectList(w, r, ret(r), url.Values{doneParam: {doneDeleted}, nameParam: {item.key.Name}})
}

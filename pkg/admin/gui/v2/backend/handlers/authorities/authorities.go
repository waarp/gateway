// Package authorities serves the "Authentication authorities" screen of the
// web interface (Administration): the list of the authorities the gateway
// trusts, with their type, what their public identity says and its validity,
// and the hosts they are allowed for (searched, filtered, sorted and paged),
// the sheet of an authority in reading, and the pages that create, edit and
// delete one.
//
// An authority is a trusted third party: a certificate it signs is accepted
// without being registered in the gateway first. It holds a name, a type (a
// TLS certificate authority, whose public identity is its certificate, or an
// SSH certificate authority, whose public identity is its public key) and the
// hosts it is allowed for, none meaning any host. Until this screen, they
// were managed through the REST API and the command line only.
package authorities

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
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/certinfo"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
)

const (
	// The stylesheet of the screen, under the web interface.
	stylesheet = "/static_v2/authorities/authorities.css"

	docPage  = "user/webui/autorites"
	pagePath = "/authorities"

	// The cookie that keeps the state of the list for the browser session.
	stateCookie = "webui_authorities_list"

	// The parameters of the list state and of the result of the previous
	// action, shared by the list screens.
	retParam  = liststate.RetParam
	doneParam = liststate.DoneParam
	nameParam = liststate.NameParam
	dirDesc   = liststate.DirDesc

	sortName     = "name"
	sortType     = "type"
	sortValidity = "validity"
	sortHosts    = "hosts"

	doneCreated = liststate.DoneCreated
	doneUpdated = liststate.DoneUpdated
	doneDeleted = liststate.DoneDeleted

	// The filters of the list: the types, the validity of the certificate.
	typeParam     = "type"
	validityParam = "validity"

	// Prefixes of the catalogue keys chosen at run time.
	sortKeyPrefix     = "authorities.sort."
	toastKeyPrefix    = "authorities.toast."
	validityKeyPrefix = "authorities.validity."
	ownerKeyPrefix    = "authorities.owner."
	pasteKeyPrefix    = "authorities.form.paste."
	hostsHintPrefix   = "authorities.form.hosts_hint."

	// The list shows two allowed hosts at most, then how many more.
	hostsShown = 2

	// Kinds of a gui.Notice.
	stateOK = "ok"
)

//nolint:gochecknoglobals // reference data of the screen
var (
	sortKeys       = []string{sortName, sortType, sortValidity, sortHosts}
	doneKinds      = []string{doneCreated, doneUpdated, doneDeleted}
	validityStates = []string{certinfo.StateOK, certinfo.StateWarn, certinfo.StateKO}

	counter = liststate.Counter{
		One: "authorities.count_one", Many: "authorities.count",
		ShownOne: "authorities.count_shown_one", ShownMany: "authorities.count_shown",
	}

	listSpec = liststate.Spec{
		Path: pagePath, Cookie: stateCookie, Sorts: sortKeys, Sizes: liststate.DefaultSizes,
		ParseFilters: parseFilter, FilterParams: []string{typeParam, validityParam},
	}
)

// kind is a type of authority the product registers: its key, its short label
// (the list), its full label (the sheet, the form) and what its public
// identity is.
type kind struct {
	Key    string
	Short  string // catalogue key of the short label
	Accept string // the files the form offers to upload
	TLS    bool   // the identity is a certificate (else an SSH public key)
}

//nolint:gochecknoglobals // reference data of the product, the keys it registers
var kinds = []kind{
	{Key: auth.AuthorityTLS, Short: "authorities.type_short.tls", Accept: ".pem,.crt,.cer", TLS: true},
	{Key: sftp.AuthoritySSHCert, Short: "authorities.type_short.ssh", Accept: ".pub"},
}

func kindOf(key string) (*kind, bool) {
	for i := range kinds {
		if kinds[i].Key == key {
			return &kinds[i], true
		}
	}

	return nil, false
}

func (k *kind) labelKey() string { return "authorities.type." + k.Key }

// Handlers serves the pages and the actions of the screen.
type Handlers struct {
	db     *database.DB
	logger *log.Logger
	shell  *common.Shell
	now    func() time.Time // the clock of the validity states, fixed by the tests
}

func New(db *database.DB, logger *log.Logger) *Handlers {
	return &Handlers{db: db, logger: logger, shell: common.NewShell(), now: time.Now}
}

// rights of the current user on the authorities: those of the
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

type (
	choice = protoform.Choice
)

// entry is an authority with its type (nil for a key the product does not
// register) and what its public identity says (Readable false when it cannot
// be read: an authority imported with a broken identity).
type entry struct {
	authority *model.Authority
	kind      *kind
	info      certinfo.Info
	Readable  bool
}

func entryOf(a *model.Authority) entry {
	e := entry{authority: a}
	e.kind, _ = kindOf(a.Type)
	e.info, e.Readable = identityOf(a.Type, a.PublicIdentity)

	return e
}

// identityOf reads a public identity for its type: a certificate for a TLS
// authority, a public key for an SSH one.
func identityOf(typ, identity string) (certinfo.Info, bool) {
	k, ok := kindOf(typ)
	if !ok {
		return certinfo.Info{}, false
	}

	if k.TLS {
		return certinfo.Certificate(identity)
	}

	key, err := sftp.ParseAuthorizedKey(identity)
	if err != nil {
		return certinfo.Info{}, false
	}

	info := certinfo.Key(key)
	info.Public = strings.TrimSpace(identity)

	return info, true
}

// typeLabel is the short label of the type, the raw key for an unknown one.
func (e *entry) typeLabel(translate gui.Translator) string {
	if e.kind == nil {
		return e.authority.Type
	}

	return translate(e.kind.Short)
}

// summary is what the list says of the identity: the subject of the
// certificate, the type of the key, or that it cannot be read.
func (e *entry) summary(translate gui.Translator) string {
	switch {
	case !e.Readable:
		return translate("authorities.identity.unreadable")
	case e.kind.TLS:
		return e.info.Subject
	default:
		return e.info.KeyType
	}
}

// validity is the state of the certificate of a TLS authority; nothing for
// an SSH one (a key does not expire) or an unreadable identity.
func (e *entry) validity(translate gui.Translator, now time.Time) gui.Status {
	if !e.Readable || !e.kind.TLS {
		return gui.Status{}
	}

	return e.info.Expiry(translate, now)
}

// hosts says the allowed hosts in a cell: any host, the first ones, then how
// many more.
func hostsText(translate gui.Translator, hosts []string) string {
	switch {
	case len(hosts) == 0:
		return translate("authorities.hosts.all")
	case len(hosts) <= hostsShown:
		return strings.Join(hosts, ", ")
	case len(hosts) == hostsShown+1:
		return translate("authorities.hosts.more_one", "hosts", strings.Join(hosts[:hostsShown], ", "))
	default:
		return translate("authorities.hosts.more",
			"hosts", strings.Join(hosts[:hostsShown], ", "), "n", len(hosts)-hostsShown)
	}
}

// --------------------------------------------------------------- list ----

// filter keeps the authorities of the chosen types and validity states.
type filter struct {
	Types    []string
	Validity []string
}

// parseFilter reads the filters from a query, dropping what is unknown.
func parseFilter(v url.Values) liststate.Filters {
	f := &filter{}

	for _, key := range v[typeParam] {
		if _, ok := kindOf(key); ok && !slices.Contains(f.Types, key) {
			f.Types = append(f.Types, key)
		}
	}

	for _, state := range v[validityParam] {
		if slices.Contains(validityStates, state) && !slices.Contains(f.Validity, state) {
			f.Validity = append(f.Validity, state)
		}
	}

	return f
}

func (f *filter) Values() url.Values {
	v := url.Values{}
	for _, key := range f.Types {
		v.Add(typeParam, key)
	}

	for _, state := range f.Validity {
		v.Add(validityParam, state)
	}

	return v
}

func (f *filter) Count() int { return len(f.Types) + len(f.Validity) }

// Matches tells whether an authority passes the filters: its type among the
// chosen ones (if any), the state of its certificate among the chosen ones
// (if any; an SSH authority has none).
func (f *filter) Matches(e *entry, state string) bool {
	if len(f.Types) > 0 && !slices.Contains(f.Types, e.authority.Type) {
		return false
	}

	return len(f.Validity) == 0 || slices.Contains(f.Validity, state)
}

// box is a checkbox of a filter.
type box struct {
	Name    string
	Value   string
	Label   string
	Checked bool
}

func (f *filter) typeBoxes(translate gui.Translator) []box {
	boxes := make([]box, 0, len(kinds))
	for i := range kinds {
		k := &kinds[i]
		boxes = append(boxes, box{
			Name: typeParam, Value: k.Key, Label: translate(k.labelKey()), Checked: slices.Contains(f.Types, k.Key),
		})
	}

	return boxes
}

func (f *filter) validityBoxes(translate gui.Translator) []box {
	boxes := make([]box, 0, len(validityStates))
	for _, state := range validityStates {
		boxes = append(boxes, box{
			Name: validityParam, Value: state, Label: translate(validityKeyPrefix + state),
			Checked: slices.Contains(f.Validity, state),
		})
	}

	return boxes
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

func authorityURL(id int64, suffix string) string {
	return common.WebuiURL(pagePath + "/" + strconv.FormatInt(id, 10) + suffix)
}

type row struct {
	ID       int64
	Name     string
	URL      string
	Type     string
	Identity string
	Readable bool
	Validity gui.Status
	Hosts    string
	Actions  []gui.Action
}

type listData struct {
	AddURL       string
	Action       string
	Base         string
	Q            string
	Names        []string
	Types        []box
	Validity     []box
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

// all reads the authorities, by name.
func (h *Handlers) all() ([]entry, error) {
	var list model.Authorities
	if err := h.db.Select(&list).OrderBy("name", true).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	entries := make([]entry, 0, len(list))
	for _, a := range list {
		entries = append(entries, entryOf(a))
	}

	return entries, nil
}

// List serves GET /webui/authorities.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	fragment := liststate.IsListFragment(r)
	if !fragment && listSpec.Restore(w, r) {
		return
	}

	translate := h.shell.Translator(r)
	query := parseListQuery(r.URL.Query())
	screen := h.screen(translate, translate("authorities.title"), "", 0)
	screen.Crumbs = crumbs(translate, "", "")
	screen.Notices = notices(translate, r.URL.Query())
	screen.Head = gui.LiveAssets()

	entries, err := h.all()
	if err != nil {
		h.logger.Errorf("failed to list the authentication authorities: %v", err)

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

// buildList computes the list page: filter (name prefix, types, validity),
// sort, page, toolbar state, row actions according to the rights.
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
		Types:        active.typeBoxes(translate),
		Validity:     active.validityBoxes(translate),
		FiltersCount: active.Count(),
		FiltersOpen:  active.Count() > 0,
	}

	if user.Write {
		data.AddURL = withRet(common.WebuiURL(pagePath+"/new"), query.Encode(query.Page))
	}

	names := make([]string, 0, len(all))
	kept := make([]entry, 0, len(all))

	for i := range all {
		e := &all[i]
		names = append(names, e.authority.Name)

		if !query.Search.Matches(e.authority.Name) {
			continue
		}

		if !active.Matches(e, e.validity(translate, now).State) {
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
			data.NoMatchText = translate("authorities.none.title", "query", query.Q)
			data.NoMatchURL, data.NoMatchLink = listSpec.ExplicitListURL(withoutQ.Values()), translate("search.clear")
		} else {
			data.NoMatchText = translate("authorities.none.filters")
			data.NoMatchURL, data.NoMatchLink = listSpec.ResetURL(), translate("filters.reset")
		}

		return data
	}

	b := query.Bounds(len(kept))
	carried := query.Encode(b.Page)
	data.Caption = translate("authorities.caption",
		"page", b.Page, "pages", b.Pages, "sort", translate(sortKeyPrefix+query.Sort))

	for i := range kept[b.From:b.To] {
		e := &kept[b.From+i]
		data.Rows = append(data.Rows, row{
			ID:       e.authority.ID,
			Name:     e.authority.Name,
			URL:      withRet(authorityURL(e.authority.ID, ""), carried),
			Type:     e.typeLabel(translate),
			Identity: e.summary(translate),
			Readable: e.Readable,
			Validity: e.validity(translate, now),
			Hosts:    hostsText(translate, e.authority.ValidHosts),
			Actions:  actions(translate, e, user, false, carried),
		})
	}

	data.Sizes = liststate.DefaultSizes
	data.PagerHidden = query.PagerHidden()
	data.Pager = listSpec.Pager(query, translate("authorities.pager", "from", b.From+1, "to", b.To, "total", len(kept)), b)

	return data
}

// sortEntries sorts by the asked column; ties keep the name order
// (regardless of case). By validity, the closest end of validity first; the
// authorities without one (SSH, unreadable) last.
func sortEntries(translate gui.Translator, entries []entry, key, dir string) {
	byName := func(a, b *entry) int {
		return strings.Compare(strings.ToLower(a.authority.Name), strings.ToLower(b.authority.Name))
	}

	end := func(e *entry) time.Time {
		if !e.Readable || e.kind == nil || !e.kind.TLS {
			return time.Time{}
		}

		return e.info.NotAfter
	}

	slices.SortStableFunc(entries, func(left, right entry) int {
		var order int

		switch key {
		case sortType:
			order = strings.Compare(left.typeLabel(translate), right.typeLabel(translate))
		case sortValidity:
			ea, eb := end(&left), end(&right)

			switch {
			case ea.IsZero() && eb.IsZero():
			case ea.IsZero():
				order = 1
			case eb.IsZero():
				order = -1
			default:
				order = ea.Compare(eb)
			}
		case sortHosts:
			order = cmp.Compare(len(left.authority.ValidHosts), len(right.authority.ValidHosts))
		default:
			order = byName(&left, &right)
		}

		if dir == dirDesc {
			order = -order
		}

		return cmp.Or(order, byName(&left, &right))
	})
}

// actions is the "⋯" menu of an authority, in the list (with "View the
// sheet" and "Edit") and on the sheet (without: the buttons are there).
func actions(translate gui.Translator, e *entry, user rights, onSheet bool, carried string) []gui.Action {
	id := e.authority.ID

	var acts []gui.Action

	if !onSheet {
		acts = append(acts, gui.Action{
			Icon: "eye", Label: translate("authorities.action.detail"), URL: withRet(authorityURL(id, ""), carried),
		})

		if user.Write {
			acts = append(acts, gui.Action{
				Icon: "pen", Label: translate("authorities.action.edit"), URL: withRet(authorityURL(id, "/edit"), carried),
			})
		}
	}

	if user.Delete {
		acts = append(acts, gui.Action{
			Icon:   "trash",
			Label:  translate("authorities.action.delete"),
			URL:    withRet(authorityURL(id, "/delete"), carried),
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
		return append(c, gui.Crumb{Label: translate("authorities.title")})
	}

	return append(c,
		gui.Crumb{Label: translate("authorities.title"), URL: listURL(carried)},
		gui.Crumb{Label: last})
}

// notices turns the result of the previous action into a notification.
func notices(translate gui.Translator, values url.Values) []gui.Notice {
	if done := values.Get(doneParam); slices.Contains(doneKinds, done) {
		return []gui.Notice{{Kind: stateOK, Title: translate(toastKeyPrefix+done, "name", values.Get(nameParam))}}
	}

	return nil
}

// -------------------------------------------------------------- pages ----

type notFoundData struct {
	Title     string
	Text      string
	BackLabel string
	BackURL   string
}

// authority reads the {id} of the route; a missing authority renders the
// localised "not found" page and returns false.
func (h *Handlers) authority(w http.ResponseWriter, r *http.Request) (entry, bool) {
	id, parseErr := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

	var a model.Authority

	err := h.db.Get(&a, "id=?", id).Run()
	if parseErr == nil && err == nil {
		return entryOf(&a), true
	}

	if err != nil && !database.IsNotFound(err) {
		h.logger.Errorf("failed to retrieve the authentication authority %d: %v", id, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return entry{}, false
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("authorities.not_found.title")

	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, http.StatusNotFound), notFoundView(notFoundData{
		Title:     title,
		Text:      translate("authorities.not_found.text"),
		BackLabel: translate("authorities.back"),
		BackURL:   listURL(carried),
	}))

	return entry{}, false
}

// screen is a page of the screen: the chrome around the data.
func (h *Handlers) screen(translate gui.Translator, title, carried string, status int) common.Screen {
	return common.Screen{
		Title:       title,
		Section:     constants.SidebarSectionAdministration,
		Menu:        constants.SidebarMenuAuthorities,
		Doc:         docPage,
		Crumbs:      crumbs(translate, title, carried),
		Status:      status,
		Stylesheets: []string{stylesheet},
	}
}

type detailData struct {
	Name         string
	TypeLabel    string // the full label, or the raw key of an unknown type
	TypeKey      string
	TypeHint     string
	TLS          bool
	Readable     bool
	Info         certinfo.Info
	Validity     gui.Status
	Period       string // "from … to …" of a certificate
	Hosts        []string
	HostsHint    string
	Signed       []signed
	IdentityMeta string // the summaries of the folded cards (#810)
	HostsMeta    string
	SignedMeta   string
	BackURL      string
	EditURL      string
	Actions      []gui.Action
}

// Detail serves GET /webui/authorities/{id}: the authority in reading, with
// the certificates registered in the gateway that it signs (TLS).
func (h *Handlers) Detail(w http.ResponseWriter, r *http.Request) {
	item, ok := h.authority(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	user := rightsOf(r)

	found, err := h.signedBy(&item)
	if err != nil {
		h.logger.Errorf("failed to find the certificates signed by %q: %v", item.authority.Name, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	data := detailData{
		Name:     item.authority.Name,
		TypeKey:  item.authority.Type,
		Readable: item.Readable,
		Info:     item.info,
		Validity: item.validity(translate, h.now()),
		Hosts:    item.authority.ValidHosts,
		Signed:   found,
		BackURL:  listURL(carried),
		Actions:  actions(translate, &item, user, true, carried),
	}

	data.TypeLabel = item.authority.Type
	if item.kind != nil {
		data.TypeLabel = translate(item.kind.labelKey())
		data.TypeHint = translate("authorities.type_hint." + item.kind.Key)
		data.HostsHint = translate("authorities.detail.hosts_hint." + item.kind.Key)
		data.TLS = item.kind.TLS
	}

	if data.TLS && item.Readable {
		date := translate("auth.date")
		data.Period = translate("auth.detail.validity_text",
			"from", item.info.NotBefore.Format(date), "to", item.info.NotAfter.Format(date))
	}

	data.IdentityMeta = identityMeta(translate, &data)
	data.HostsMeta = hostsMeta(translate, len(data.Hosts))
	data.SignedMeta = signedMeta(translate, len(data.Signed))

	if user.Write {
		data.EditURL = withRet(authorityURL(item.authority.ID, "/edit"), carried)
	}

	title := translate("authorities.detail.title", "name", item.authority.Name)
	screen := h.screen(translate, title, carried, 0)
	screen.Notices = notices(translate, r.URL.Query())
	h.shell.Render(w, h.logger, r, screen, detailView(data))
}

type deleteData struct {
	Name      string
	TypeLabel string
	Identity  string
	Hosts     string
	Signed    []signed
	TLS       bool
	CancelURL string
	Action    string
}

// DeleteForm serves GET /webui/authorities/{id}/delete: the confirmation
// page, with the consequence and the certificates the authority signs.
func (h *Handlers) DeleteForm(w http.ResponseWriter, r *http.Request) {
	item, ok := h.authority(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)

	found, err := h.signedBy(&item)
	if err != nil {
		h.logger.Errorf("failed to find the certificates signed by %q: %v", item.authority.Name, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	data := deleteData{
		Name:      item.authority.Name,
		TypeLabel: item.authority.Type,
		Identity:  item.summary(translate),
		Hosts:     hostsText(translate, item.authority.ValidHosts),
		Signed:    found,
		TLS:       item.kind != nil && item.kind.TLS,
		CancelURL: withRet(authorityURL(item.authority.ID, ""), carried),
		Action:    withRet(authorityURL(item.authority.ID, "/delete"), carried),
	}

	if item.kind != nil {
		data.TypeLabel = translate(item.kind.labelKey())
	}

	title := translate("authorities.delete.title", "name", item.authority.Name)
	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, 0), deleteView(data))
}

// Delete serves POST /webui/authorities/{id}/delete: the product refuses
// nothing; the certificates it signs are no longer accepted, unless they are
// registered or signed by a system authority.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	e, ok := h.authority(w, r)
	if !ok {
		return
	}

	if err := h.db.Delete(e.authority).Run(); err != nil {
		h.logger.Errorf("failed to delete the authentication authority %q: %v", e.authority.Name, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	listSpec.RedirectList(w, r, ret(r), url.Values{doneParam: {doneDeleted}, nameParam: {e.authority.Name}})
}

// identityMeta sums up the public identity card for its folded heading
// (#810): the subject and the expiry of a certificate, the type of a key, or
// that the identity cannot be read.
func identityMeta(translate gui.Translator, d *detailData) string {
	switch {
	case !d.Readable:
		return translate("authorities.identity.unreadable")
	case d.TLS:
		expires := translate("authorities.identity.meta_expires", "date", d.Info.NotAfter.Format(translate("auth.date")))

		return d.Info.Subject + " · " + expires
	default:
		return d.Info.KeyType
	}
}

// hostsMeta sums up the allowed hosts card for its folded heading (#810).
func hostsMeta(translate gui.Translator, n int) string {
	switch n {
	case 0:
		return translate("authorities.hosts.all")
	case 1:
		return translate("authorities.hosts.meta_one")
	default:
		return translate("authorities.hosts.meta", "n", n)
	}
}

// signedMeta sums up the signed certificates card for its folded heading (#810).
func signedMeta(translate gui.Translator, n int) string {
	switch n {
	case 0:
		return translate("authorities.signed.meta_none")
	case 1:
		return translate("authorities.signed.meta_one")
	default:
		return translate("authorities.signed.meta", "n", n)
	}
}

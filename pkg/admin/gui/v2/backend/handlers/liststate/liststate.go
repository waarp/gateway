// Package liststate is the state of a list screen (local servers, local
// clients...), shared by the screens: the search, the filters of the
// screen, the sort, the page and the rows per page, carried by the URL of
// the list, kept for the browser session in a cookie, and carried by the
// action pages (?ret=) so that coming back finds the list as it was left.
// It also builds the list foot and the counts.
package liststate

import (
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
)

// The parameters of the state of a list, and the ones that carry the result
// of the previous action through the redirection.
const (
	QueryParam = "q"
	SortParam  = "sort"
	DirParam   = "dir"
	PageParam  = "page"
	SizeParam  = "size"
	RetParam   = "ret"
	// ResetParam is the explicit reset: a request without any criterion
	// that clears the state kept for the session ("Reset", "Clear the
	// search").
	ResetParam = "reset"

	DirAsc  = "asc"
	DirDesc = "desc"

	DoneParam   = "done"
	FailedParam = "failed"
	NameParam   = "name"
	ReasonParam = "reason"
	// KindParam says what object the previous action concerned, when a
	// screen has several kinds (a server, one of its means, one of its
	// accounts); the screen renders the right notification from it.
	KindParam = "kind"

	// DoneCreated, DoneUpdated and DoneDeleted are the results of the actions
	// every screen has.
	DoneCreated = "created"
	DoneUpdated = "updated"
	DoneDeleted = "deleted"
)

// DefaultSizes are the rows per page of a list, the first one by default.
//
//nolint:gochecknoglobals // reference data of the lists
var DefaultSizes = []int{10, 25, 50, 100}

// Filters is the part of the state that belongs to the screen (the families
// and the TLS of the protocol screens): parsed and encoded by it.
type Filters interface {
	// Values encodes the filters, nothing when none is set.
	Values() url.Values
	// Count is the number of active filters.
	Count() int
}

// Spec describes the list of a screen: its page, its cookie, its sort keys
// (the first one by default), its rows per page (the first one by default)
// and the filters it owns.
type Spec struct {
	Path   string // the list page, under the web interface prefix ("/servers")
	Cookie string // the cookie that keeps the state for the browser session
	Sorts  []string
	Sizes  []int
	// FirstDesc sorts by the first sort key in the descending order by
	// default (the most recent transfers first); the other keys start
	// ascending.
	FirstDesc bool
	// DefaultSize is the rows per page by default, one of Sizes; the first
	// one when zero.
	DefaultSize int
	// ParseFilters reads the filters of the screen from a query, dropping
	// what is unknown; nil when the screen has none.
	ParseFilters func(url.Values) Filters
	// FilterParams are the parameters the filters use: a request that
	// carries one of them says something about the state of the list.
	FilterParams []string
}

// Query is the state of a list asked by a URL.
type Query struct {
	Q       string
	Sort    string
	Dir     string
	Page    int
	Size    int
	Filters Filters

	sorts       []string
	sizes       []int
	firstDesc   bool
	defaultSize int
}

// Parse reads ?q=, ?sort= (among the given keys, the first one by default),
// ?dir=, ?page= and ?size= (among the given sizes, the first one by
// default). Anything unknown falls back to the default; the page is bounded
// later, once the count is known.
func Parse(v url.Values, sorts []string, sizes []int) *Query {
	query := &Query{
		Q: strings.TrimSpace(v.Get(QueryParam)), Sort: sorts[0], Dir: DirAsc, Page: 1, Size: sizes[0],
		sorts: sorts, sizes: sizes,
	}

	if slices.Contains(sorts, v.Get(SortParam)) {
		query.Sort = v.Get(SortParam)
	}

	if v.Get(DirParam) == DirDesc {
		query.Dir = DirDesc
	}

	if page, err := strconv.Atoi(v.Get(PageParam)); err == nil && page > 1 {
		query.Page = page
	}

	if size, err := strconv.Atoi(v.Get(SizeParam)); err == nil && slices.Contains(sizes, size) {
		query.Size = size
	}

	return query
}

// Parse reads the state of the list of the screen, its filters included.
func (s *Spec) Parse(v url.Values) *Query {
	query := Parse(v, s.Sorts, s.Sizes)
	if s.FirstDesc {
		query.firstDesc = true
		if query.Sort == s.Sorts[0] && v.Get(DirParam) != DirAsc {
			query.Dir = DirDesc
		}
	}

	if slices.Contains(s.Sizes, s.DefaultSize) {
		query.defaultSize = s.DefaultSize
		if size, err := strconv.Atoi(v.Get(SizeParam)); err != nil || !slices.Contains(s.Sizes, size) {
			query.Size = s.DefaultSize
		}
	}

	if s.ParseFilters != nil {
		query.Filters = s.ParseFilters(v)
	}

	return query
}

// FilterValues carries the search and the filters only: the base of the
// sort links.
func (q *Query) FilterValues() url.Values {
	v := url.Values{}
	if q.Q != "" {
		v.Set(QueryParam, q.Q)
	}

	if q.Filters != nil {
		maps.Copy(v, q.Filters.Values())
	}

	return v
}

// Values carries the search, the filters, the sort and the rows per page
// when they differ from the default; the page is added by WithPage.
func (q *Query) Values() url.Values {
	v := q.FilterValues()
	if q.Sort != q.sorts[0] || q.Dir != q.defaultDir() {
		v.Set(SortParam, q.Sort)
		v.Set(DirParam, q.Dir)
	}

	if q.Size != q.sizeByDefault() {
		v.Set(SizeParam, strconv.Itoa(q.Size))
	}

	return v
}

// defaultDir is the order of the default sort: descending for a list sorted
// by its first key in that order by default (Spec.FirstDesc).
func (q *Query) defaultDir() string {
	if q.firstDesc {
		return DirDesc
	}

	return DirAsc
}

// sizeByDefault is the rows per page by default (Spec.DefaultSize).
func (q *Query) sizeByDefault() int {
	if q.defaultSize != 0 {
		return q.defaultSize
	}

	return q.sizes[0]
}

// WithPage is the state with the given page.
func (q *Query) WithPage(page int) url.Values {
	v := q.Values()
	if page > 1 {
		v.Set(PageParam, strconv.Itoa(page))
	}

	return v
}

// Encode is the list state carried by the action pages (?ret=).
func (q *Query) Encode(page int) string { return q.WithPage(page).Encode() }

// ToolbarHidden is what the toolbar form (search and filters) must carry so
// that a new search keeps the sort and the rows per page.
func (q *Query) ToolbarHidden() map[string]string {
	hidden := map[string]string{}

	for _, key := range []string{SortParam, DirParam, SizeParam} {
		if value := q.Values().Get(key); value != "" {
			hidden[key] = value
		}
	}

	return hidden
}

// PagerHidden is what the list foot form must carry besides the page and the
// rows per page it sets itself.
func (q *Query) PagerHidden() map[string]string {
	hidden := map[string]string{}

	for key, values := range q.Values() {
		if key != SizeParam {
			hidden[key] = values[0]
		}
	}

	return hidden
}

// FiltersCount is the number of active filters, shown next to the "Filters"
// button.
func (q *Query) FiltersCount() int {
	if q.Filters == nil {
		return 0
	}

	return q.Filters.Count()
}

// Bounds are the page of a list once the matching rows are known: the page
// bounded by the count, the rows it shows.
type Bounds struct{ Page, Pages, From, To int }

// Bounds computes the page of the query for a number of matching rows.
func (q *Query) Bounds(total int) Bounds {
	pages := (total + q.Size - 1) / q.Size
	page := min(max(q.Page, 1), pages)

	return Bounds{Page: page, Pages: pages, From: (page - 1) * q.Size, To: min(page*q.Size, total)}
}

// RetOf reads a carried list state; unknown or invalid parameters are
// dropped, so that a forged value cannot lead anywhere else than the list.
func (s *Spec) RetOf(ret string) *Query {
	values, err := url.ParseQuery(ret)
	if err != nil {
		return s.Parse(nil)
	}

	return s.Parse(values)
}

// Ret is the list state carried by the request (?ret=), cleaned.
func (s *Spec) Ret(r *http.Request) string {
	q := s.RetOf(r.URL.Query().Get(RetParam))

	return q.Encode(q.Page)
}

// ListURLOf is the list with the given parameters.
func (s *Spec) ListURLOf(v url.Values) string {
	if encoded := v.Encode(); encoded != "" {
		return common.WebuiURL(s.Path) + "?" + encoded
	}

	return common.WebuiURL(s.Path)
}

// ListURL is the list as the user left it: the "Back to the list" links and
// the redirections after an action.
func (s *Spec) ListURL(ret string) string {
	q := s.RetOf(ret)

	return s.ListURLOf(q.WithPage(q.Page))
}

// ResetURL is the list without any criterion, said explicitly so that the
// state kept for the session is cleared instead of restored.
func (s *Spec) ResetURL() string { return s.ListURLOf(url.Values{ResetParam: {"1"}}) }

// ExplicitListURL is the list with the given criteria, or the explicit
// reset when there is none left.
func (s *Spec) ExplicitListURL(v url.Values) string {
	if len(v) == 0 {
		return s.ResetURL()
	}

	return s.ListURLOf(v)
}

// HasParams tells whether the request says anything about the state of the
// list, be it a criterion or the explicit reset.
func (s *Spec) HasParams(params url.Values) bool {
	keys := append([]string{QueryParam, SortParam, DirParam, PageParam, SizeParam, ResetParam}, s.FilterParams...)

	return slices.ContainsFunc(keys, params.Has)
}

// KeptState reads the state kept for the session; a default or unreadable
// state is no state.
func (s *Spec) KeptState(r *http.Request) (*Query, bool) {
	cookie, err := r.Cookie(s.Cookie)
	if err != nil {
		return nil, false
	}

	state, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return nil, false
	}

	kept := s.RetOf(state)

	return kept, kept.Encode(kept.Page) != ""
}

// KeepState stores the state of the list for the browser session, or
// forgets it when the state is the default one.
func (s *Spec) KeepState(w http.ResponseWriter, query *Query) {
	cookie := &http.Cookie{
		Name:     s.Cookie,
		Value:    url.QueryEscape(query.Encode(query.Page)),
		Path:     common.WebuiURL(s.Path),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	if cookie.Value == "" {
		cookie.MaxAge = -1
	}

	http.SetCookie(w, cookie)
}

// Restore redirects a request that says nothing about the state of the list
// to the state kept for the session, carrying the result of the previous
// action; it returns false when there is nothing to restore. A request that
// says something keeps its state for the session.
func (s *Spec) Restore(w http.ResponseWriter, r *http.Request) bool {
	params := r.URL.Query()
	if s.HasParams(params) {
		s.KeepState(w, s.Parse(params))

		return false
	}

	kept, ok := s.KeptState(r)
	if !ok {
		return false
	}

	target := kept.WithPage(kept.Page)
	for _, name := range []string{DoneParam, FailedParam, NameParam, ReasonParam} {
		if params.Has(name) {
			target.Set(name, params.Get(name))
		}
	}

	http.Redirect(w, r, s.ListURLOf(target), http.StatusSeeOther)

	return true
}

// RedirectList sends back to the list as the user left it, with the result
// of the action.
func (s *Spec) RedirectList(w http.ResponseWriter, r *http.Request, carried string, result url.Values) {
	q := s.RetOf(carried)
	v := q.WithPage(q.Page)

	maps.Copy(v, result)

	http.Redirect(w, r, s.ListURLOf(v), http.StatusSeeOther)
}

// WithRet appends the carried list state to a URL.
func WithRet(u, ret string) string {
	if ret == "" {
		return u
	}

	return WithQuery(u, url.Values{RetParam: {ret}})
}

// WithQuery appends parameters to a URL.
func WithQuery(u string, v url.Values) string {
	encoded := v.Encode()
	if encoded == "" {
		return u
	}

	sep := "?"
	if strings.Contains(u, "?") {
		sep = "&"
	}

	return u + sep + encoded
}

// Pager is the list foot, with the fields of "gui/pager".
type Pager struct {
	Info    string
	Page    int
	Pages   int
	PrevURL string
	NextURL string
}

// Pager builds the list foot of the list page: the previous and next pages
// keep the state.
func (s *Spec) Pager(query *Query, info string, b Bounds) *Pager {
	pager := &Pager{Info: info, Page: b.Page, Pages: b.Pages}

	if b.Page > 1 {
		pager.PrevURL = s.ListURLOf(query.WithPage(b.Page - 1))
	}

	if b.Page < b.Pages {
		pager.NextURL = s.ListURLOf(query.WithPage(b.Page + 1))
	}

	return pager
}

// Counter is the set of keys of a count: "N objects", "N objects · n shown".
type Counter struct{ One, Many, ShownOne, ShownMany string }

// Text is the count of a list, with the shown rows when a search or a
// filter is active.
func (c Counter) Text(translate gui.Translator, total, shown int) string {
	switch {
	case shown == total && total == 1:
		return translate(c.One)
	case shown == total:
		return translate(c.Many, "total", total)
	case shown == 1:
		return translate(c.ShownOne, "total", total)
	default:
		return translate(c.ShownMany, "total", total, "shown", shown)
	}
}

// FailureReason is what the notification of a failed action says: the
// reason the service reports, or, when it reports none (a service left
// offline by a listener that could not open), the cause the start or stop
// returned.
func FailureReason(reason string, err error) string {
	if reason == "" && err != nil {
		return err.Error()
	}

	return reason
}

// Props is the list foot for the kit: the pager with the hidden fields of
// the toolbar form and the rows-per-page choices.
func (p *Pager) Props(hidden map[string]string, sizes []int, size int) gui.PagerProps {
	return gui.PagerProps{
		Info: p.Info, Page: p.Page, Pages: p.Pages, PrevURL: p.PrevURL, NextURL: p.NextURL,
		Hidden: hidden, Sizes: sizes, Size: size,
	}
}

// Hidden is a hidden field of a toolbar form.
type Hidden struct {
	Name  string
	Value string
}

// SortedHidden gives the hidden fields of a form in the order of their
// names, as html/template ranged over the map.
func SortedHidden(fields map[string]string) []Hidden {
	out := make([]Hidden, 0, len(fields))
	for _, name := range slices.Sorted(maps.Keys(fields)) {
		out = append(out, Hidden{Name: name, Value: fields[name]})
	}

	return out
}

// AddLink is the action of an empty state that leads to the creation, nil
// without the write right (no URL).
func AddLink(href, label string) *gui.Link {
	if href == "" {
		return nil
	}

	return &gui.Link{Label: label, URL: href, Icon: "plus"}
}

// CountSuffix is the " · n" shown on the "Filters" button when filters apply.
func CountSuffix(n int) string {
	if n == 0 {
		return ""
	}

	return " · " + strconv.Itoa(n)
}

// RedirectTo sends to a page (already carrying the list state) with the
// result of the action.
func RedirectTo(w http.ResponseWriter, r *http.Request, target string, result url.Values) {
	u, err := url.Parse(target)
	if err != nil {
		http.Redirect(w, r, target, http.StatusSeeOther)

		return
	}

	query := u.Query()
	for name, values := range result {
		query.Set(name, values[0])
	}

	u.RawQuery = query.Encode()

	http.Redirect(w, r, u.String(), http.StatusSeeOther)
}

// ------------------------------------------------------------ tab list ----

// TabList is the state of the list of a tab of a sheet: the toolbar form,
// the base of the sort links, the count, the rows per page and the list
// foot. The sheet carries the state of the list of its screen (?ret=)
// through every link and form.
type TabList struct {
	Action      string
	Base        string
	Q           string
	Names       []string
	Hidden      map[string]string
	Count       string
	Sort        string
	Dir         string
	Caption     string
	Pager       *Pager
	PagerHidden map[string]string
	Sizes       []int
	Size        int
	Empty       bool
	NoMatch     bool
	NoMatchText string
	NoMatchURL  string
	NoMatchLink string
}

// NewTabList builds the state of a tab list from its query.
func NewTabList(base string, query *Query, carried string) TabList {
	hidden, pagerHidden := query.ToolbarHidden(), query.PagerHidden()
	if carried != "" {
		hidden[RetParam], pagerHidden[RetParam] = carried, carried
	}

	return TabList{
		Action:      base,
		Base:        WithRet(WithQuery(base, query.FilterValues()), carried),
		Q:           query.Q,
		Hidden:      hidden,
		Sort:        query.Sort,
		Dir:         query.Dir,
		PagerHidden: pagerHidden,
		Size:        query.Size,
	}
}

// SetNoMatch marks the empty result of a search: the way out clears the search.
func (l *TabList) SetNoMatch(text, clearURL, clearLabel string) {
	l.NoMatch = true
	l.NoMatchText = text
	l.NoMatchURL, l.NoMatchLink = clearURL, clearLabel
}

// Foot fills the list foot: the rows per page (offered as soon as there are
// rows) and the pages.
func (l *TabList) Foot(info string, b Bounds, query *Query, base, carried string) {
	l.Sizes = query.sizes
	l.Pager = &Pager{Info: info, Page: b.Page, Pages: b.Pages}

	if b.Page > 1 {
		l.Pager.PrevURL = WithRet(WithQuery(base, query.WithPage(b.Page-1)), carried)
	}

	if b.Page < b.Pages {
		l.Pager.NextURL = WithRet(WithQuery(base, query.WithPage(b.Page+1)), carried)
	}
}

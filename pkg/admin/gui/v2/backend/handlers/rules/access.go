package rules

import (
	"cmp"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"code.waarp.fr/lib/gui"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/means"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// The Access tab of a rule: who may use it. A rule without any access is open
// to everyone; the first access closes it to every other one, on both sides:
// the servers and local accounts are checked when a partner connects, the
// partners and remote accounts when the gateway connects. The tab says so,
// and adds and removes the access as the REST API does.

// The kinds of access, as the address names them.
const (
	kindServer        = "server"
	kindLocalAccount  = "local_account"
	kindPartner       = "partner"
	kindRemoteAccount = "remote_account"

	// The target of an access in the form: its kind, then its ID ("server:3").
	fieldTarget = "target"

	// The side of the tab an addition comes from: its form offers the kinds
	// of that side only.
	sideParam = "side"
	sideIn    = "in"
	sideOut   = "out"

	doneAccessAdded   = "access_added"
	doneAccessRemoved = "access_removed"
	doneAccessOpened  = "access_opened"

	causeGranted = "already been granted"

	accessKindKeyPrefix  = "rules.access.kind."
	accessGroupKeyPrefix = "rules.access.group."
)

//nolint:gochecknoglobals // reference data of the screen
var accessKinds = []string{kindServer, kindLocalAccount, kindPartner, kindRemoteAccount}

// incoming says whether a kind of access is checked when a partner connects
// (a server, a local account), rather than when the gateway connects.
func incoming(kind string) bool { return kind == kindServer || kind == kindLocalAccount }

// kindsOf are the kinds of access of a side of the tab: when a partner
// connects (in), when the gateway connects (out), or all of them.
func kindsOf(side string) []string {
	switch side {
	case sideIn:
		return []string{kindServer, kindLocalAccount}
	case sideOut:
		return []string{kindPartner, kindRemoteAccount}
	default:
		return accessKinds
	}
}

// sideOf reads the side of the tab an addition comes from, "" for none.
func sideOf(r *http.Request) string {
	if side := r.URL.Query().Get(sideParam); side == sideIn || side == sideOut {
		return side
	}

	return ""
}

// addURL is the page that adds an access from a side of the tab.
func addURL(id int64, side, carried string) string {
	return withRet(liststate.WithQuery(accessURL(id, "/new"), url.Values{sideParam: {side}}), carried)
}

// grant is an access of a rule: its kind, the ID of its target and what the
// screen shows of it.
type grant struct {
	Kind  string
	ID    int64
	Label string // the server or the partner, or the login and its owner
}

// targets are the servers, partners and accounts of the gateway, which the
// access of the rules name.
type targets struct {
	servers  map[int64]*model.LocalAgent
	partners map[int64]*model.RemoteAgent
	locals   map[int64]*model.LocalAccount
	remotes  map[int64]*model.RemoteAccount
}

func (h *Handlers) targets() (*targets, error) {
	var (
		servers  model.LocalAgents
		partners model.RemoteAgents
		locals   model.LocalAccounts
		remotes  model.RemoteAccounts
	)

	// The database keeps to the servers and partners of this gateway.
	if err := h.db.Select(&servers).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	if err := h.db.Select(&partners).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	if err := h.db.Select(&locals).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	if err := h.db.Select(&remotes).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	found := &targets{
		servers: map[int64]*model.LocalAgent{}, partners: map[int64]*model.RemoteAgent{},
		locals: map[int64]*model.LocalAccount{}, remotes: map[int64]*model.RemoteAccount{},
	}

	for _, s := range servers {
		found.servers[s.ID] = s
	}

	for _, p := range partners {
		found.partners[p.ID] = p
	}

	// The accounts of the servers and partners of this gateway only.
	for _, a := range locals {
		if found.servers[a.LocalAgentID] != nil {
			found.locals[a.ID] = a
		}
	}

	for _, a := range remotes {
		if found.partners[a.RemoteAgentID] != nil {
			found.remotes[a.ID] = a
		}
	}

	return found, nil
}

// label is what the screen shows of a target: the name of a server or a
// partner, the login of an account with its server or its partner.
func (t *targets) label(translate gui.Translator, kind string, id int64) (string, bool) {
	switch kind {
	case kindServer:
		if s := t.servers[id]; s != nil {
			return s.Name, true
		}
	case kindPartner:
		if p := t.partners[id]; p != nil {
			return p.Name, true
		}
	case kindLocalAccount:
		if a := t.locals[id]; a != nil {
			return translate("rules.access.of_server", "login", a.Login, "server", t.servers[a.LocalAgentID].Name), true
		}
	case kindRemoteAccount:
		if a := t.remotes[id]; a != nil {
			return translate("rules.access.of_partner", "login", a.Login, "partner", t.partners[a.RemoteAgentID].Name), true
		}
	}

	return "", false
}

// target is the object of the model an access names, for the queries of the
// product (model.AccessTarget).
func (t *targets) target(kind string, id int64) (model.AccessTarget, bool) {
	switch kind {
	case kindServer:
		s, ok := t.servers[id]

		return s, ok
	case kindPartner:
		p, ok := t.partners[id]

		return p, ok
	case kindLocalAccount:
		a, ok := t.locals[id]

		return a, ok
	case kindRemoteAccount:
		a, ok := t.remotes[id]

		return a, ok
	}

	return nil, false
}

// kindOf is the kind and the target ID of an access row.
func kindOf(a *model.RuleAccess) (string, int64) {
	switch {
	case a.LocalAgentID.Valid:
		return kindServer, a.LocalAgentID.Int64
	case a.LocalAccountID.Valid:
		return kindLocalAccount, a.LocalAccountID.Int64
	case a.RemoteAgentID.Valid:
		return kindPartner, a.RemoteAgentID.Int64
	default:
		return kindRemoteAccount, a.RemoteAccountID.Int64
	}
}

// grantsOf reads the access of a rule, sorted by kind then label.
func (h *Handlers) grantsOf(translate gui.Translator, rule *model.Rule, t *targets) ([]grant, error) {
	var rows model.RuleAccesses
	if err := h.db.Select(&rows).Where("rule_id=?", rule.ID).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	out := make([]grant, 0, len(rows))

	for _, a := range rows {
		kind, id := kindOf(a)

		label, ok := t.label(translate, kind, id)
		if !ok {
			label = translate("rules.access.unknown", "id", id) // a target of another gateway of the cluster
		}

		out = append(out, grant{Kind: kind, ID: id, Label: label})
	}

	slices.SortFunc(out, func(a, b grant) int {
		return cmp.Or(slices.Index(accessKinds, a.Kind)-slices.Index(accessKinds, b.Kind), strings.Compare(a.Label, b.Label))
	})

	return out, nil
}

func accessURL(id int64, suffix string) string { return ruleURL(id, "/access"+suffix) }

func grantURL(id int64, g *grant, suffix string) string {
	return accessURL(id, "/"+g.Kind+"/"+strconv.FormatInt(g.ID, 10)+suffix)
}

// accessRow is an access in its block, with its menu.
type accessRow struct {
	ID      string
	Kind    string
	Label   string
	Actions []gui.Action
}

type accessData struct {
	Title     string
	Name      string
	Tabs      []means.Tab
	BackURL   string
	Actions   []gui.Action // the menu of the rule: its transfers, its deletion
	OpenURL   string
	Open      bool
	OneSided  string // the note of a rule restricted on one side only
	In, Out   accessSide
	StateMeta string // open to all, or restricted, in the head of the state card
}

// accessSide is one side of the tab: its access, how many, the page that
// adds one (empty without the write right) and what it says when empty.
type accessSide struct {
	Rows   []accessRow
	Count  string
	AddURL string
	None   string
}

// AccessTab serves GET /webui/rules/{id}/access: who may use the rule.
func (h *Handlers) AccessTab(w http.ResponseWriter, r *http.Request) {
	item, ok := h.load(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)

	t, err := h.targets()
	if err != nil {
		h.internal(w, "read the servers and partners", err)

		return
	}

	grants, err := h.grantsOf(translate, item.rule, t)
	if err != nil {
		h.internal(w, "read the access of the transfer rule", err)

		return
	}

	carried := ret(r)
	user := rightsOf(r)
	data := accessData{
		Title:   ruleTitle(translate, item.rule),
		Name:    item.rule.Name,
		Tabs:    tabs(translate, &item, tabAccess),
		BackURL: listURL(carried),
		Actions: actions(translate, &item, user, true, carried),
		Open:    len(grants) == 0,
	}

	if user.Write {
		data.In.AddURL = addURL(item.rule.ID, sideIn, carried)
		data.Out.AddURL = addURL(item.rule.ID, sideOut, carried)
		if !data.Open {
			data.OpenURL = withRet(accessURL(item.rule.ID, "/open"), carried)
		}
	}

	for i := range grants {
		g := &grants[i]
		row := accessRow{
			ID: g.Kind + "-" + strconv.FormatInt(g.ID, 10), Kind: translate(accessKindKeyPrefix + g.Kind), Label: g.Label,
		}

		if user.Write {
			row.Actions = []gui.Action{{
				Icon: "trash", Label: translate("rules.access.remove"), URL: withRet(grantURL(item.rule.ID, g, "/delete"), carried),
				Danger: true,
			}}
		}

		if incoming(g.Kind) {
			data.In.Rows = append(data.In.Rows, row)
		} else {
			data.Out.Rows = append(data.Out.Rows, row)
		}
	}

	data.In.Count, data.Out.Count = accessCount(translate, len(data.In.Rows)), accessCount(translate, len(data.Out.Rows))
	data.StateMeta = translate("rules.access.restricted_title")
	data.In.None, data.Out.None = translate("rules.access.in_none"), translate("rules.access.out_none")

	if data.Open {
		data.StateMeta = translate("rules.access.open")
		data.In.None = translate("rules.access.side_open")
		data.Out.None = data.In.None
	}

	switch {
	case len(data.In.Rows) > 0 && len(data.Out.Rows) == 0:
		data.OneSided = translate("rules.access.one_sided_in")
	case len(data.Out.Rows) > 0 && len(data.In.Rows) == 0:
		data.OneSided = translate("rules.access.one_sided_out")
	}

	screen := h.screen(translate, data.Title, carried, 0)
	screen.Notices = notices(translate, r.URL.Query())
	h.shell.Render(w, h.logger, r, screen, accessView(data))
}

// accessCount says how many access a side gives, in the head of its card.
func accessCount(translate gui.Translator, n int) string {
	if n == 0 {
		return translate("rules.access.count_none")
	}

	return translate("rules.access.count", "n", n)
}

// accessForm is the page that adds an access: the targets it offers, the one
// posted and what was wrong with it.
type accessForm struct {
	Title     string
	Action    string
	CancelURL string
	Groups    []gui.ComboGroup // the targets of the side, one group per kind
	TargetErr string
	HasErrors bool
	Refused   string
	Note      string // the rule is open: the first access closes it

	target string // the target posted, "kind:id"
	side   string // the side of the tab the addition comes from, "" for both
}

// targetValue is a target in the form: its kind, then its ID.
func targetValue(kind string, id int64) string { return kind + ":" + strconv.FormatInt(id, 10) }

// choices builds the targets of the side for the list to type in (Driss,
// 2026-10-23, as the type of a task), one group per kind: a server or a
// partner with its protocol, an account with its server or partner, so that
// typing the name of one finds its accounts too. The ones already granted are
// left out; the servers and partners sorted by name, the accounts by owner.
func choices(translate gui.Translator, known *targets, granted []grant, form *accessForm) {
	has := func(kind string, id int64) bool {
		return slices.ContainsFunc(granted, func(g grant) bool { return g.Kind == kind && g.ID == id })
	}

	for _, kind := range kindsOf(form.side) {
		var list []gui.ComboChoice

		add := func(id int64, label, code string) {
			if value := targetValue(kind, id); !has(kind, id) {
				list = append(list, gui.ComboChoice{Value: value, Label: label, Code: code, Selected: value == form.target})
			}
		}

		switch kind {
		case kindServer:
			for id, s := range known.servers {
				add(id, s.Name, protoform.ProtocolOf(s.Protocol).Label)
			}
		case kindPartner:
			for id, p := range known.partners {
				add(id, p.Name, protoform.ProtocolOf(p.Protocol).Label)
			}
		case kindLocalAccount:
			for id, a := range known.locals {
				add(id, a.Login, known.servers[a.LocalAgentID].Name)
			}
		case kindRemoteAccount:
			for id, a := range known.remotes {
				add(id, a.Login, known.partners[a.RemoteAgentID].Name)
			}
		}

		if len(list) == 0 {
			continue
		}

		slices.SortFunc(list, func(a, b gui.ComboChoice) int {
			if kind == kindLocalAccount || kind == kindRemoteAccount {
				return cmp.Or(strings.Compare(a.Code, b.Code), strings.Compare(a.Label, b.Label))
			}

			return strings.Compare(a.Label, b.Label)
		})
		form.Groups = append(form.Groups, gui.ComboGroup{Label: translate(accessGroupKeyPrefix + kind), Choices: list})
	}
}

// AccessNewForm serves GET /webui/rules/{id}/access/new.
func (h *Handlers) AccessNewForm(w http.ResponseWriter, r *http.Request) {
	h.accessForm(w, r, false)
}

// AccessCreate serves POST /webui/rules/{id}/access/new: the access is added
// as the REST API does (the product refuses a target already granted).
func (h *Handlers) AccessCreate(w http.ResponseWriter, r *http.Request) {
	h.accessForm(w, r, true)
}

func (h *Handlers) accessForm(w http.ResponseWriter, r *http.Request, post bool) {
	item, ok := h.load(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)

	known, err := h.targets()
	if err != nil {
		h.internal(w, "read the servers and partners", err)

		return
	}

	granted, err := h.grantsOf(translate, item.rule, known)
	if err != nil {
		h.internal(w, "read the access of the transfer rule", err)

		return
	}

	side := sideOf(r)
	form := &accessForm{
		Title:     translate("rules.access.new_title", "name", item.rule.Name),
		Action:    withRet(accessURL(item.rule.ID, "/new"), carried),
		side:      side,
		CancelURL: withRet(accessURL(item.rule.ID, ""), carried),
	}

	if side != "" {
		form.Action = addURL(item.rule.ID, side, carried)
	}

	if len(granted) == 0 {
		form.Note = translate("rules.access.first_note")
	}

	status := 0

	if post {
		if parseErr := r.ParseForm(); parseErr != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)

			return
		}

		target, label, valid := form.read(translate, r, known)
		if valid {
			access := &model.RuleAccess{RuleID: item.rule.ID}
			target.SetAccessTarget(access)

			insErr := h.db.Insert(access).Run()

			switch {
			case insErr == nil:
				liststate.RedirectTo(w, r, withRet(accessURL(item.rule.ID, ""), carried),
					url.Values{doneParam: {doneAccessAdded}, nameParam: {label}})

				return
			case database.IsValidationError(insErr) && strings.Contains(insErr.Error(), causeGranted):
				form.TargetErr = translate("rules.access.error.granted")
			case database.IsValidationError(insErr):
				form.Refused = insErr.Error()
			default:
				h.internal(w, "add an access to the transfer rule", insErr)

				return
			}

			form.HasErrors = true
		}

		status = http.StatusUnprocessableEntity
	}

	choices(translate, known, granted, form)

	screen := h.screen(translate, form.Title, carried, status)
	screen.Head = gui.ComboboxAssets() // the targets in the list to type in
	h.shell.Render(w, h.logger, r, screen, accessFormView(form))
}

// read takes the target of the posted form, "kind:id", of a kind of the side.
func (f *accessForm) read(translate gui.Translator, r *http.Request, t *targets) (model.AccessTarget, string, bool) {
	f.target = r.PostFormValue(fieldTarget)
	kind, rawID, _ := strings.Cut(f.target, ":")
	id, err := strconv.ParseInt(rawID, 10, 64)

	target, ok := t.target(kind, id)
	if err != nil || !ok || !slices.Contains(kindsOf(f.side), kind) {
		f.TargetErr, f.HasErrors = translate("rules.access.error.target"), true

		return nil, "", false
	}

	label, _ := t.label(translate, kind, id)

	return target, label, true
}

type removeData struct {
	Title     string
	Kind      string
	Label     string
	Last      bool // the rule becomes open to all
	CancelURL string
	Action    string
}

// grantOfRoute reads the {kind} and {target} of the route among the access of
// the rule; one it does not have renders the "not found" page.
func (h *Handlers) grantOfRoute(w http.ResponseWriter, r *http.Request, item *entry,
) (*grant, []grant, *targets, bool) {
	translate := h.shell.Translator(r)

	t, err := h.targets()
	if err != nil {
		h.internal(w, "read the servers and partners", err)

		return nil, nil, nil, false
	}

	granted, err := h.grantsOf(translate, item.rule, t)
	if err != nil {
		h.internal(w, "read the access of the transfer rule", err)

		return nil, nil, nil, false
	}

	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["target"], 10, 64) //nolint:errcheck // a bad number matches no access

	for i := range granted {
		if granted[i].Kind == vars["kind"] && granted[i].ID == id {
			return &granted[i], granted, t, true
		}
	}

	carried := ret(r)
	title := translate("rules.access.not_found.title")
	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, http.StatusNotFound),
		means.NotFoundView(means.NotFoundData{
			Title: title, Text: translate("rules.access.not_found.text"), Icon: "shield",
			BackLabel: translate("rules.access.back"), BackURL: withRet(accessURL(item.rule.ID, ""), carried),
		}))

	return nil, nil, nil, false
}

// AccessDeleteForm serves GET /webui/rules/{id}/access/{kind}/{target}/delete.
func (h *Handlers) AccessDeleteForm(w http.ResponseWriter, r *http.Request) {
	h.accessDelete(w, r, false)
}

// AccessDelete serves POST /webui/rules/{id}/access/{kind}/{target}/delete:
// the access is removed as the REST API does; without access left, the rule
// is open to all again.
func (h *Handlers) AccessDelete(w http.ResponseWriter, r *http.Request) {
	h.accessDelete(w, r, true)
}

func (h *Handlers) accessDelete(w http.ResponseWriter, r *http.Request, post bool) {
	item, ok := h.load(w, r)
	if !ok {
		return
	}

	access, granted, known, ok := h.grantOfRoute(w, r, &item)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)

	if post {
		target, known := known.target(access.Kind, access.ID)
		if !known {
			http.Error(w, "Not found", http.StatusNotFound)

			return
		}

		cond, arg := target.GenAccessSelectCond()
		if err := h.db.DeleteAll(&model.RuleAccess{}).Where("rule_id=?", item.rule.ID).Where(cond, arg).Run(); err != nil {
			h.internal(w, "remove an access of the transfer rule", err)

			return
		}

		liststate.RedirectTo(w, r, withRet(accessURL(item.rule.ID, ""), carried),
			url.Values{doneParam: {doneAccessRemoved}, nameParam: {access.Label}})

		return
	}

	title := translate("rules.access.remove_title", "name", access.Label)
	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, 0), removeView(removeData{
		Title:     title,
		Kind:      translate(accessKindKeyPrefix + access.Kind),
		Label:     access.Label,
		Last:      len(granted) == 1,
		CancelURL: withRet(accessURL(item.rule.ID, ""), carried),
		Action:    withRet(grantURL(item.rule.ID, access, "/delete"), carried),
	}))
}

type openData struct {
	Title     string
	Name      string
	Count     int
	CancelURL string
	Action    string
}

// AccessOpenForm serves GET /webui/rules/{id}/access/open.
func (h *Handlers) AccessOpenForm(w http.ResponseWriter, r *http.Request) {
	h.accessOpen(w, r, false)
}

// AccessOpen serves POST /webui/rules/{id}/access/open: every access of the
// rule is removed, which opens it to all (the "allow all" of the REST API).
func (h *Handlers) AccessOpen(w http.ResponseWriter, r *http.Request) {
	h.accessOpen(w, r, true)
}

func (h *Handlers) accessOpen(w http.ResponseWriter, r *http.Request, post bool) {
	item, ok := h.load(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)

	if post {
		if err := h.db.DeleteAll(&model.RuleAccess{}).Where("rule_id=?", item.rule.ID).Run(); err != nil {
			h.internal(w, "open the transfer rule to all", err)

			return
		}

		liststate.RedirectTo(w, r, withRet(accessURL(item.rule.ID, ""), carried),
			url.Values{doneParam: {doneAccessOpened}, nameParam: {item.rule.Name}})

		return
	}

	title := translate("rules.access.open_title", "name", item.rule.Name)
	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, 0), openView(openData{
		Title:     title,
		Name:      item.rule.Name,
		Count:     item.access,
		CancelURL: withRet(accessURL(item.rule.ID, ""), carried),
		Action:    withRet(accessURL(item.rule.ID, "/open"), carried),
	}))
}

// internal logs a failure of the database and answers 500.
func (h *Handlers) internal(w http.ResponseWriter, what string, err error) {
	h.logger.Errorf("failed to %s: %v", what, err)
	http.Error(w, "Internal error", http.StatusInternalServerError)
}

package watchers

import (
	"net/http"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/features"
)

// Names of the fields of the form. The account and the client are chosen
// in the block of the chosen protocol: their fields carry the protocol
// ("account-sftp"), so that every block can be posted without JavaScript
// and the server reads the one of the protocol.
const (
	fieldName     = "name"
	fieldPattern  = "pattern"
	fieldInterval = "interval"
	fieldUnit     = "unit"
	fieldProtocol = "protocol"
	fieldAccount  = "account"
	fieldClient   = "client"
	fieldRule     = "rule"
	fieldSkip     = "skip_duplicates"
	fieldEnabled  = "enabled"

	unitMinutes = "min"
	unitHours   = "h"

	// The causes the model gives when it refuses a watcher.
	causeListing  = "does not support listing files"
	causeSendRule = "must be a receive rule"
	causePattern  = "invalid filewatcher pattern"
)

type (
	field  = protoform.Field
	choice = protoform.Choice
)

func accountField(protocol string) string { return fieldAccount + "-" + protocol }
func clientField(protocol string) string  { return fieldClient + "-" + protocol }

// accountGroup is the accounts of one partner, as a group of the account
// select; the partner names the group.
type accountGroup struct {
	Partner string
	Options []choice
}

// catalog is what the form offers: the protocols that list files, among
// those registered in the gateway; the accounts of the partners of such a
// protocol; the clients of such a protocol; the receive rules.
type catalog struct {
	protocols []string
	partners  map[int64]*model.RemoteAgent
	accounts  []*model.RemoteAccount
	clients   []*model.Client
	rules     []*model.Rule
}

// listingProtocols is the protocols that know how to list a folder, by
// their label.
func listingProtocols() []string {
	var list []string

	for key := range protocols.List() {
		if features.Supports(key, features.Listing) {
			list = append(list, key)
		}
	}

	slices.SortFunc(list, func(a, b string) int { return strings.Compare(protoLabel(a), protoLabel(b)) })

	return list
}

func (h *Handlers) catalog() (*catalog, error) {
	c := &catalog{protocols: listingProtocols()}

	var err error
	if c.partners, err = h.partners(); err != nil {
		return nil, err
	}

	var accounts model.RemoteAccounts
	if serr := h.db.Select(&accounts).OrderBy("login", true).Run(); serr != nil {
		return nil, serr //nolint:wrapcheck // logged by the caller with its context
	}

	for _, a := range accounts {
		if p, ok := c.partners[a.RemoteAgentID]; ok && slices.Contains(c.protocols, p.Protocol) {
			c.accounts = append(c.accounts, a)
		}
	}

	var clients model.Clients
	if serr := h.db.Select(&clients).OrderBy("name", true).Run(); serr != nil {
		return nil, serr //nolint:wrapcheck // logged by the caller with its context
	}

	for _, cl := range clients {
		if slices.Contains(c.protocols, cl.Protocol) {
			c.clients = append(c.clients, cl)
		}
	}

	var rules model.Rules
	if serr := h.db.Select(&rules).Where("is_send=?", false).OrderBy("name", true).Run(); serr != nil {
		return nil, serr //nolint:wrapcheck // logged by the caller with its context
	}

	c.rules = rules

	return c, nil
}

// protoLabel names a protocol as the product shows it ("R66-TLS").
func protoLabel(key string) string {
	if p := protoform.ProtocolOf(key); p != nil {
		return p.Label
	}

	return key
}

func (c *catalog) account(id int64) (*model.RemoteAccount, *model.RemoteAgent) {
	for _, a := range c.accounts {
		if a.ID == id {
			return a, c.partners[a.RemoteAgentID]
		}
	}

	return nil, nil
}

func (c *catalog) client(id int64) *model.Client {
	for _, cl := range c.clients {
		if cl.ID == id {
			return cl
		}
	}

	return nil
}

func (c *catalog) rule(id int64) *model.Rule {
	for _, r := range c.rules {
		if r.ID == id {
			return r
		}
	}

	return nil
}

// protocolBlock is the block of one protocol in the partner card: the
// accounts of its partners, by partner, and its clients. The block of the
// chosen protocol carries the errors of the account and of the client.
type protocolBlock struct {
	Key        string
	Label      string
	Account    field
	Accounts   []accountGroup
	NoAccounts bool // no partner of this protocol has an account
	Client     field
	Clients    []choice
	NoClients  bool // no client of this protocol
}

type formData struct {
	Title     string
	Action    string
	CancelURL string
	Edit      bool
	HasErrors bool
	Refused   string // a refusal of the model no field carries

	Name     field
	Pattern  field
	Interval field // the number, with the unit
	Unit     string
	Units    []choice
	Protocol field // the key of the protocol, among those that list files
	// Protocols are the protocols offered; Blocks, the account and the
	// client of each one.
	Protocols []choice
	Blocks    []protocolBlock
	Account   field // the identifier of the remote account
	Client    field
	// NoAccounts: no partner that lists files has an account; the form says
	// it instead of the fields, and leads to the partners.
	NoAccounts  bool
	PartnersURL string
	Rule        field
	Rules       []choice
	Skip        bool // skip the files already transferred
	Enabled     bool // start with the gateway
	// The partner and the rule cards fold (the first card does not); their
	// summary says what is chosen.
	PartnerOpen bool
	PartnerMeta string
	RuleOpen    bool
	RuleMeta    string

	stored   *model.FileWatcher
	account  *model.RemoteAccount
	partner  *model.RemoteAgent
	client   *model.Client
	rule     *model.Rule
	interval time.Duration
}

// newForm builds the form of a new watcher (the files already transferred
// skipped, started with the gateway, every minute as the unit), or the form
// of an existing one filled with its values.
func newForm(translate gui.Translator, e *entry, carried string) *formData {
	form := &formData{
		Title:       translate("watchers.new.title"),
		Action:      withRet(common.WebuiURL(pagePath+"/new"), carried),
		CancelURL:   listURL(carried),
		PartnersURL: common.WebuiURL("/partners"),
		Unit:        unitMinutes,
		Skip:        true,
		Enabled:     true,
		PartnerOpen: true,
		RuleOpen:    true,
	}

	if e == nil {
		return form
	}

	fw := e.fw
	form.Title = translate("watchers.edit.title", "name", fw.Flow)
	form.Action = withRet(watcherURL(fw.ID, "/edit"), carried)
	form.Edit = true
	form.stored = fw
	form.Name = field{Value: fw.Flow}
	form.Pattern = field{Value: fw.Pattern}
	form.Protocol = field{Value: fw.Client.Protocol}
	form.Account = field{Value: strconv.FormatInt(fw.RemoteAccountID, 10)}
	form.Client = field{Value: strconv.FormatInt(fw.ClientID, 10)}
	form.Rule = field{Value: strconv.FormatInt(fw.RuleID, 10)}
	form.Skip = !fw.NoDuplicateCheck
	form.Enabled = !fw.Disabled

	// In hours when they are whole, in minutes otherwise.
	switch {
	case fw.Interval > 0 && fw.Interval%time.Hour == 0:
		form.Interval, form.Unit = field{Value: strconv.Itoa(int(fw.Interval / time.Hour))}, unitHours
	case fw.Interval > 0:
		form.Interval = field{Value: strconv.Itoa(int(fw.Interval / time.Minute))}
	}

	return form
}

// read takes the posted values: the account and the client of the chosen
// protocol.
func (f *formData) read(r *http.Request) {
	f.Name = field{Value: strings.TrimSpace(r.PostFormValue(fieldName))}
	f.Pattern = field{Value: strings.TrimSpace(r.PostFormValue(fieldPattern))}
	f.Interval = field{Value: strings.TrimSpace(r.PostFormValue(fieldInterval))}
	f.Unit = unitMinutes

	if r.PostFormValue(fieldUnit) == unitHours {
		f.Unit = unitHours
	}

	f.Protocol = field{Value: r.PostFormValue(fieldProtocol)}
	f.Account = field{Value: r.PostFormValue(accountField(f.Protocol.Value))}
	f.Client = field{Value: r.PostFormValue(clientField(f.Protocol.Value))}
	f.Rule = field{Value: r.PostFormValue(fieldRule)}
	f.Skip = r.PostFormValue(fieldSkip) != ""
	f.Enabled = r.PostFormValue(fieldEnabled) != ""
}

// validate checks what the screen can check before the model, with what the
// form offers: the name (given, free), the interval (a whole number of
// minutes or hours, one minute at least: a watcher at zero never polls),
// the pattern (a glob), the protocol (one that lists files), the account
// (of a partner of that protocol), the client (of that protocol), the rule
// (a receive rule).
func (f *formData) validate(translate gui.Translator, db *database.DB, c *catalog) bool {
	fail := func(fld *field, key string) {
		fld.Error = translate("watchers.form.error." + key)
		f.HasErrors = true
	}

	if f.Name.Value == "" {
		fail(&f.Name, "required")
	} else if taken, err := f.nameTaken(db); err != nil || taken {
		fail(&f.Name, "name_taken")
	}

	n, err := strconv.Atoi(f.Interval.Value)

	switch {
	case err != nil || n < 1:
		fail(&f.Interval, "interval")
	case f.Unit == unitHours:
		f.interval = time.Duration(n) * time.Hour
	default:
		f.interval = time.Duration(n) * time.Minute
	}

	if _, merr := path.Match(f.Pattern.Value, ""); merr != nil {
		fail(&f.Pattern, "pattern")
	}

	// The account and the client are checked in the block of the protocol:
	// without a valid protocol, no block shows, and the protocol says it.
	if !slices.Contains(c.protocols, f.Protocol.Value) {
		fail(&f.Protocol, "protocol")
	} else {
		f.validatePartner(fail, c)
	}

	if f.rule = c.rule(parseID(f.Rule.Value)); f.rule == nil {
		fail(&f.Rule, "rule")
	}

	return !f.HasErrors
}

// validatePartner checks the account (of a partner of the protocol) and
// the client (of the protocol too).
func (f *formData) validatePartner(fail func(*field, string), c *catalog) {
	switch f.account, f.partner = c.account(parseID(f.Account.Value)); {
	case f.account == nil:
		fail(&f.Account, "account")
	case f.partner.Protocol != f.Protocol.Value:
		fail(&f.Account, "account_protocol")
	}

	switch f.client = c.client(parseID(f.Client.Value)); {
	case f.client == nil:
		fail(&f.Client, "client")
	case f.client.Protocol != f.Protocol.Value, f.partner != nil && f.client.Protocol != f.partner.Protocol:
		fail(&f.Client, "client_protocol")
	}
}

// parseID reads the identifier a select posted: 0 for anything else.
func parseID(value string) int64 {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}

	return id
}

// nameTaken says whether another watcher has the name.
func (f *formData) nameTaken(db *database.DB) (bool, error) {
	query := db.Count(&model.FileWatcher{}).Where("flow=?", f.Name.Value)
	if f.stored != nil {
		query = db.Count(&model.FileWatcher{}).Where("flow=? AND id<>?", f.Name.Value, f.stored.ID)
	}

	n, err := query.Run()

	return n > 0, err //nolint:wrapcheck // the field says it
}

// apply writes the validated form into the watcher. The model reads the
// account, the client and the rule from the nested objects.
func (f *formData) apply(fw *model.FileWatcher) {
	fw.Flow = f.Name.Value
	fw.Pattern = f.Pattern.Value
	fw.Interval = f.interval
	fw.NoDuplicateCheck = !f.Skip
	fw.Disabled = !f.Enabled
	fw.RemoteAccount, fw.RemoteAccountID = *f.account, f.account.ID
	fw.Client, fw.ClientID = *f.client, f.client.ID
	fw.Rule, fw.RuleID = *f.rule, f.rule.ID
}

// refuse marks the form with the refusal of the model: the known causes go
// to their field, the others to the banner.
func (f *formData) refuse(translate gui.Translator, err error) {
	msg := err.Error()

	switch {
	case strings.Contains(msg, causeListing):
		f.Client.Error = translate("watchers.form.error.client")
	case strings.Contains(msg, causeSendRule):
		f.Rule.Error = translate("watchers.form.error.rule")
	case strings.Contains(msg, causePattern):
		f.Pattern.Error = translate("watchers.form.error.pattern")
	default:
		f.Refused = msg
	}

	f.HasErrors = true
}

// prepare builds what the template renders: the units, the protocols and
// the block of each one (the accounts by partner, the clients), the rules,
// the chosen ones selected, the summaries of the folded cards.
func (f *formData) prepare(translate gui.Translator, c *catalog) {
	f.Units = []choice{
		{Value: unitMinutes, Label: translate("watchers.form.unit_minutes"), Selected: f.Unit == unitMinutes},
		{Value: unitHours, Label: translate("watchers.form.unit_hours"), Selected: f.Unit == unitHours},
	}

	f.Protocols = make([]choice, 0, len(c.protocols))
	f.Blocks = make([]protocolBlock, 0, len(c.protocols))
	f.NoAccounts = len(c.accounts) == 0

	for _, key := range c.protocols {
		f.Protocols = append(f.Protocols, choice{Value: key, Label: protoLabel(key), Selected: f.Protocol.Value == key})
		f.Blocks = append(f.Blocks, f.block(key, c))
	}

	f.Rules = make([]choice, 0, len(c.rules))
	for _, r := range c.rules {
		id := strconv.FormatInt(r.ID, 10)
		f.Rules = append(f.Rules, choice{Value: id, Label: r.Name, Selected: f.Rule.Value == id})
	}

	f.summaries()
}

// block is the block of one protocol: its accounts by partner, its clients;
// the fields of the chosen protocol carry the errors.
func (f *formData) block(key string, c *catalog) protocolBlock {
	blk := protocolBlock{Key: key, Label: protoLabel(key)}
	if f.Protocol.Value == key {
		blk.Account, blk.Client = f.Account, f.Client
	}

	groups := map[int64]*accountGroup{}
	order := []int64{}

	for _, a := range c.accounts {
		p := c.partners[a.RemoteAgentID]
		if p.Protocol != key {
			continue
		}

		g, ok := groups[p.ID]
		if !ok {
			g = &accountGroup{Partner: p.Name}
			groups[p.ID] = g
			order = append(order, p.ID)
		}

		id := strconv.FormatInt(a.ID, 10)
		g.Options = append(g.Options, choice{Value: id, Label: a.Login, Selected: blk.Account.Value == id})
	}

	for _, id := range order {
		blk.Accounts = append(blk.Accounts, *groups[id])
	}

	for _, cl := range c.clients {
		if cl.Protocol != key {
			continue
		}

		id := strconv.FormatInt(cl.ID, 10)
		blk.Clients = append(blk.Clients, choice{Value: id, Label: cl.Name, Selected: blk.Client.Value == id})
	}

	blk.NoAccounts, blk.NoClients = len(blk.Accounts) == 0, len(blk.Clients) == 0

	return blk
}

// summaries says, in the head of the folded cards, what is chosen: the
// protocol, the partner and the account; the rule.
func (f *formData) summaries() {
	for i := range f.Blocks {
		blk := &f.Blocks[i]
		if blk.Key != f.Protocol.Value {
			continue
		}

		f.PartnerMeta = blk.Label

		for _, g := range blk.Accounts {
			for _, o := range g.Options {
				if o.Selected {
					f.PartnerMeta = blk.Label + " · " + g.Partner + " · " + o.Label
				}
			}
		}
	}

	for _, r := range f.Rules {
		if r.Selected {
			f.RuleMeta = r.Label
		}
	}
}

// render shows the form with what the configuration offers.
func (h *Handlers) render(w http.ResponseWriter, r *http.Request, form *formData, status int) {
	translate := h.shell.Translator(r)

	c, err := h.catalog()
	if err != nil {
		h.logger.Errorf("failed to read the configuration for the partner watcher form: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	form.prepare(translate, c)
	h.shell.Render(w, h.logger, r, h.screen(translate, form.Title, ret(r), status), formView(form))
}

// NewForm serves GET /webui/watchers/new.
func (h *Handlers) NewForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, newForm(h.shell.Translator(r), nil, ret(r)), 0)
}

// Create serves POST /webui/watchers/new.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	h.save(w, r, nil)
}

// EditForm serves GET /webui/watchers/{id}/edit.
func (h *Handlers) EditForm(w http.ResponseWriter, r *http.Request) {
	e, ok := h.watcher(w, r)
	if !ok {
		return
	}

	h.render(w, r, newForm(h.shell.Translator(r), &e, ret(r)), 0)
}

// Update serves POST /webui/watchers/{id}/edit.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	e, ok := h.watcher(w, r)
	if !ok {
		return
	}

	h.save(w, r, &e)
}

// save reads, validates and stores the posted form: a new watcher when e is
// nil, the given one otherwise. A refused form, by the screen or by the
// model, is rendered again with its errors (422); a saved one redirects to
// the list with a notification. A new watcher starts at once unless it is
// disabled; an edited one that was running starts again on its new
// configuration, as the REST API does.
func (h *Handlers) save(w http.ResponseWriter, r *http.Request, existing *entry) {
	translate := h.shell.Translator(r)
	carried := ret(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	c, err := h.catalog()
	if err != nil {
		h.logger.Errorf("failed to read the configuration for the partner watcher form: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	form := newForm(translate, existing, carried)
	form.read(r)

	if form.validate(translate, h.db, c) {
		done, fw := doneCreated, &model.FileWatcher{}
		if existing != nil {
			stored := *existing.fw
			done, fw = doneUpdated, &stored
		}

		form.apply(fw)

		if h.store(r, existing, fw, done, translate, form) {
			listSpec.RedirectList(w, r, carried, url.Values{doneParam: {done}, nameParam: {fw.Flow}})

			return
		}

		if form.Refused == "" && !form.HasErrors {
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}
	}

	form.prepare(translate, c)
	screen := h.screen(translate, form.Title, carried, http.StatusUnprocessableEntity)
	h.shell.Render(w, h.logger, r, screen, formView(form))
}

// store writes the watcher and tends its service: false when the model
// refused it (the form says why) or the database failed (logged, the form
// untouched).
func (h *Handlers) store(r *http.Request, existing *entry, watcher *model.FileWatcher, done string,
	translate gui.Translator,
	form *formData,
) bool {
	ctx := r.Context()

	var wasRunning bool

	if existing != nil {
		stopped, err := h.services.Stop(ctx, existing.fw)
		if err != nil {
			h.logger.Warningf("failed to stop the partner watcher %q before its edition: %v", existing.fw.Flow, err)
		}

		wasRunning = stopped
	}

	var err error
	if done == doneCreated {
		err = h.db.Insert(watcher).Run()
	} else {
		err = h.db.Update(watcher).Run()
	}

	switch {
	case err == nil:
	case database.IsValidationError(err):
		form.refuse(translate, err)

		return false
	default:
		h.logger.Errorf("failed to save the partner watcher %q: %v", watcher.Flow, err)

		return false
	}

	if (done == doneCreated && !watcher.Disabled) || wasRunning {
		if serr := h.services.Start(ctx, watcher); serr != nil {
			h.logger.Warningf("failed to start the partner watcher %q after its save: %v", watcher.Flow, serr)
		}
	}

	return true
}

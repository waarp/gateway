package transfers

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/compatibility"
)

// Names of the fields of the pages that add and preregister a transfer.
const (
	fieldFile    = "file"
	fieldOutput  = "output"
	fieldRule    = "rule"
	fieldAccount = "account"
	fieldClient  = "client"
	fieldStart   = "start"
	fieldInfo    = "transfer_info" // not "info": the id of an icon of the kit
	fieldTries   = "tries"
	fieldDelay   = "delay"
	fieldFactor  = "factor"

	maxTries = 127 // the attempts are an int8
)

type field = protoform.Field

// option is a choice of a select, in a group (the direction of a rule, the
// partner or the server of an account).
type option struct {
	Value    string
	Label    string
	Selected bool
}

type optGroup struct {
	Label   string
	Options []option
}

// formData is the page that adds a client transfer (Server false) or
// preregisters a server transfer (Server true).
type formData struct {
	Title     string
	Action    string
	CancelURL string
	Server    bool
	HasErrors bool
	Refused   string // a refusal of the model no field carries

	File     field
	Output   field
	Rule     field
	Rules    []optGroup
	Account  field
	Accounts []optGroup
	Client   field
	Clients  []option
	Start    field // the start of a client transfer, the due date of a server one
	Info     field
	Tries    field
	Delay    field
	Factor   field
}

// pageOf is the page that adds (or preregisters) a transfer, without its
// choices: its title, where it posts, where "Cancel" leads; a due date one
// day ahead for a server transfer.
func pageOf(translate gui.Translator, r *http.Request, server bool) *formData {
	form := &formData{
		Title: translate("transfers.new.title"), Action: r.URL.RequestURI(), CancelURL: listURL(ret(r)), Server: server,
	}

	if server {
		form.Title = translate("transfers.preregister.title")
		form.Start.Value = time.Now().Add(24 * time.Hour).Format(localLayout) //nolint:mnd // one day ahead
	}

	return form
}

// choices reads what the selects of the page offer: the rules by direction,
// the accounts by partner (or by server), the clients.
func (h *Handlers) choices(translate gui.Translator, form *formData) error {
	rules, err := internal.ListRules(h.db, "name", true, 0, 0)
	if err != nil {
		return err //nolint:wrapcheck // logged by the caller
	}

	send := optGroup{Label: translate("transfers.form.rules_send")}
	receive := optGroup{Label: translate("transfers.form.rules_receive")}

	for _, rule := range rules {
		opt := option{Value: strconv.FormatInt(rule.ID, 10), Label: rule.Name}
		opt.Selected = opt.Value == form.Rule.Value

		if rule.IsSend {
			send.Options = append(send.Options, opt)
		} else {
			receive.Options = append(receive.Options, opt)
		}
	}

	form.Rules = []optGroup{send, receive}

	if form.Server {
		return h.serverChoices(form)
	}

	return h.clientChoices(form)
}

// clientChoices are the remote accounts by partner, and the clients.
func (h *Handlers) clientChoices(form *formData) error {
	var partners model.RemoteAgents
	if err := h.db.Select(&partners).OrderBy("name", true).Run(); err != nil {
		return err //nolint:wrapcheck // logged by the caller
	}

	var accounts model.RemoteAccounts
	if err := h.db.Select(&accounts).OrderBy("login", true).Run(); err != nil {
		return err //nolint:wrapcheck // logged by the caller
	}

	for _, partner := range partners {
		group := optGroup{Label: partner.Name}

		for _, account := range accounts {
			if account.RemoteAgentID == partner.ID {
				group.Options = append(group.Options, accountOption(account.ID, account.Login, form.Account.Value))
			}
		}

		if len(group.Options) > 0 {
			form.Accounts = append(form.Accounts, group)
		}
	}

	var clients model.Clients
	if err := h.db.Select(&clients).OrderBy("name", true).Run(); err != nil {
		return err //nolint:wrapcheck // logged by the caller
	}

	for _, client := range clients {
		form.Clients = append(form.Clients, option{
			Value: client.Name, Label: client.Name + " · " + protocolLabel(client.Protocol),
			Selected: client.Name == form.Client.Value,
		})
	}

	return nil
}

// serverChoices are the local accounts by server.
func (h *Handlers) serverChoices(form *formData) error {
	var servers model.LocalAgents
	if err := h.db.Select(&servers).OrderBy("name", true).Run(); err != nil {
		return err //nolint:wrapcheck // logged by the caller
	}

	var accounts model.LocalAccounts
	if err := h.db.Select(&accounts).OrderBy("login", true).Run(); err != nil {
		return err //nolint:wrapcheck // logged by the caller
	}

	for _, server := range servers {
		group := optGroup{Label: server.Name}

		for _, account := range accounts {
			if account.LocalAgentID == server.ID {
				group.Options = append(group.Options, accountOption(account.ID, account.Login, form.Account.Value))
			}
		}

		if len(group.Options) > 0 {
			form.Accounts = append(form.Accounts, group)
		}
	}

	return nil
}

func accountOption(id int64, login, chosen string) option {
	value := strconv.FormatInt(id, 10)

	return option{Value: value, Label: login, Selected: value == chosen}
}

// read fills the form with the posted values.
func (f *formData) read(values url.Values) {
	get := func(name string) string { return strings.TrimSpace(values.Get(name)) }

	f.File.Value, f.Output.Value = get(fieldFile), get(fieldOutput)
	f.Rule.Value, f.Account.Value, f.Client.Value = get(fieldRule), get(fieldAccount), get(fieldClient)
	f.Start.Value, f.Info.Value = get(fieldStart), values.Get(fieldInfo)
	f.Tries.Value, f.Delay.Value, f.Factor.Value = get(fieldTries), get(fieldDelay), get(fieldFactor)
}

// parsed is what the form gives once checked.
type parsed struct {
	rule    *model.Rule
	start   time.Time
	info    map[string]any
	tries   int8
	delay   int32
	factor  float32
	account int64
}

// validate checks the form before the model: the file, the rule, the
// account, the date, the transfer information and the retries.
func (h *Handlers) validate(translate gui.Translator, form *formData) (parsed, bool) {
	var out parsed

	fail := func(fld *field, key string, args ...any) {
		fld.Error = translate(key, args...)
		form.HasErrors = true
	}

	if form.File.Value == "" {
		fail(&form.File, "transfers.form.error.required")
	}

	if id, err := strconv.ParseInt(form.Rule.Value, 10, 64); err != nil {
		fail(&form.Rule, "transfers.form.error.choose")
	} else if rule, getErr := internal.GetRuleByID(h.db, id); getErr != nil {
		fail(&form.Rule, "transfers.form.error.choose")
	} else {
		out.rule = rule
	}

	if id, err := strconv.ParseInt(form.Account.Value, 10, 64); err != nil {
		fail(&form.Account, "transfers.form.error.choose")
	} else {
		out.account = id
	}

	switch start, ok := parseLocal(form.Start.Value); {
	case ok && form.Server && !start.After(time.Now()):
		// As the REST API: a due date in the past would never be met.
		fail(&form.Start, "transfers.form.error.due_past")
	case ok:
		out.start = start
	case form.Start.Value != "" || form.Server:
		fail(&form.Start, "transfers.form.error.date")
	}

	info, bad := parseInfo(form.Info.Value)
	if bad != "" {
		fail(&form.Info, "transfers.form.error.info", "line", bad)
	}

	out.info = info
	out.tries, out.delay, out.factor = form.retries(fail)

	return out, !form.HasErrors
}

// retries reads the retries of a client transfer, each one optional: the
// attempts, the first delay in seconds, the growth factor (a decimal comma
// is accepted).
func (f *formData) retries(fail func(fld *field, key string, args ...any)) (tries int8, delay int32, factor float32) {
	if f.Tries.Value != "" {
		if n, err := strconv.ParseInt(f.Tries.Value, 10, 8); err != nil || n < 0 {
			fail(&f.Tries, "transfers.form.error.tries", "max", maxTries)
		} else {
			tries = int8(n)
		}
	}

	if f.Delay.Value != "" {
		if n, err := strconv.ParseInt(f.Delay.Value, 10, 32); err != nil || n < 0 {
			fail(&f.Delay, "transfers.form.error.delay")
		} else {
			delay = int32(n)
		}
	}

	if f.Factor.Value != "" {
		value := strings.Replace(f.Factor.Value, ",", ".", 1)
		if x, err := strconv.ParseFloat(value, 32); err != nil || x < 0 {
			fail(&f.Factor, "transfers.form.error.factor")
		} else {
			factor = float32(x)
		}
	}

	return tries, delay, factor
}

// parseInfo reads the transfer information, one "key: value" per line; it
// returns the first line it cannot read.
func parseInfo(text string) (map[string]any, string) {
	info := map[string]any{}

	for line := range strings.Lines(text) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if key = strings.TrimSpace(key); !ok || key == "" {
			return nil, line
		}

		info[key] = strings.TrimSpace(value)
	}

	return info, ""
}

// render shows the page, with its choices.
func (h *Handlers) render(w http.ResponseWriter, r *http.Request, form *formData, status int) {
	translate := h.shell.Translator(r)
	h.shell.Render(w, h.logger, r, h.screen(form.Title, crumbs(translate, form.Title, ret(r)), status),
		formView(form))
}

// NewForm serves GET /webui/transfers/new.
func (h *Handlers) NewForm(w http.ResponseWriter, r *http.Request) { h.showForm(w, r, false) }

// PreregisterForm serves GET /webui/transfers/preregister.
func (h *Handlers) PreregisterForm(w http.ResponseWriter, r *http.Request) { h.showForm(w, r, true) }

func (h *Handlers) showForm(w http.ResponseWriter, r *http.Request, server bool) {
	translate := h.shell.Translator(r)

	form := pageOf(translate, r, server)
	if err := h.choices(translate, form); err != nil {
		h.logger.Errorf("failed to read the choices of the transfer form: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	h.render(w, r, form, 0)
}

// Create serves POST /webui/transfers/new: a client transfer, planned at the
// given start (now by default). The list follows, with a notification.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) { h.save(w, r, false) }

// Preregister serves POST /webui/transfers/preregister: a server transfer,
// available until its due date.
func (h *Handlers) Preregister(w http.ResponseWriter, r *http.Request) { h.save(w, r, true) }

// save reads, checks and stores the posted form. A refused form, by the
// screen or by the model, is rendered again with its errors (422).
func (h *Handlers) save(w http.ResponseWriter, r *http.Request, server bool) {
	translate := h.shell.Translator(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	form := pageOf(translate, r, server)
	form.read(r.PostForm)

	values, ok := h.validate(translate, form)

	var (
		trans  *model.Transfer
		result = "added"
		err    error
	)

	if ok {
		if server {
			trans, err = h.preregister(form.File.Value, &values)
			result = "preregistered"
		} else {
			trans, err = h.add(translate, form, &values)
		}
	}

	switch {
	case ok && err == nil && trans != nil:
		listSpec.RedirectList(w, r, ret(r), url.Values{
			liststate.DoneParam: {result}, liststate.NameParam: {strconv.FormatInt(trans.ID, 10)},
		})

		return
	case err != nil && !database.IsValidationError(err) && !database.IsNotFound(err):
		h.logger.Errorf("failed to add a transfer: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	case err != nil:
		// A refusal a field carries says it there; the others in the banner.
		if !form.HasErrors {
			form.Refused = err.Error()
		}

		form.HasErrors = true
	}

	if choiceErr := h.choices(translate, form); choiceErr != nil {
		h.logger.Errorf("failed to read the choices of the transfer form: %v", choiceErr)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	h.render(w, r, form, http.StatusUnprocessableEntity)
}

// add plans a client transfer: the client chosen, or the client of the
// protocol of the partner as the REST API chooses it.
func (h *Handlers) add(translate gui.Translator, form *formData, checked *parsed) (*model.Transfer, error) {
	account, err := internal.GetPartnerAccountByID(h.db, checked.account)
	if err != nil {
		form.Account.Error, form.HasErrors = translate("transfers.form.error.choose"), true

		return nil, err //nolint:wrapcheck // turned into the error of the field
	}

	var partner model.RemoteAgent
	if getErr := h.db.Get(&partner, "id=?", account.RemoteAgentID).Run(); getErr != nil {
		return nil, getErr //nolint:wrapcheck // logged by the caller
	}

	var client *model.Client

	if form.Client.Value != "" {
		if client, err = internal.GetClient(h.db, form.Client.Value); err != nil {
			form.Client.Error, form.HasErrors = translate("transfers.form.error.choose"), true

			return nil, err //nolint:wrapcheck // turned into the error of the field
		}
	} else if client, err = compatibility.GetDefaultTransferClient(h.db, partner.Protocol); err != nil {
		if database.IsValidationError(err) {
			form.Client.Error = translate("transfers.form.error.client", "protocol", protocolLabel(partner.Protocol))
			form.HasErrors = true
		}

		return nil, err //nolint:wrapcheck // turned into the error of the field
	}

	output := form.Output.Value
	if output == "" {
		output = baseName(form.File.Value)
	}

	//nolint:wrapcheck // the model says why
	return internal.InsertNewTransfer(h.db, form.File.Value, output, checked.rule, account, client, checked.start,
		checked.tries, checked.delay, checked.factor, checked.info)
}

// preregister makes a server transfer available until its due date.
func (h *Handlers) preregister(file string, checked *parsed) (*model.Transfer, error) {
	account, err := internal.GetServerAccountByID(h.db, checked.account)
	if err != nil {
		return nil, err //nolint:wrapcheck // turned into a refusal
	}

	//nolint:wrapcheck // the model says why
	return internal.RegisterNewTransfer(h.db, file, checked.rule, account, checked.start, checked.info)
}

// baseName is the last element of a path, "/" or "\" separated.
func baseName(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}

	return p
}

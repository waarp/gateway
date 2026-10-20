package clients

import (
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// Names of the fields the screen owns; the protocol and its options are the
// shared protocol form (protoform). The delay before the first retry is
// entered as hours, minutes and seconds, as the historic page did (Driss,
// 2026-10-19), and stored in seconds.
const (
	fieldName        = "name"
	fieldNoAutoStart = "no_autostart"
	fieldHost        = "host"
	fieldPort        = "port"
	fieldAttempts    = "attempts"
	fieldDelayHours  = "delay_h"
	fieldDelayMins   = "delay_m"
	fieldDelaySecs   = "delay_s"
	fieldFactor      = "factor"
)

// field is a text or number field of the form: its value, kept when the form
// is refused, and the message of its error.
type field = protoform.Field

type formData struct {
	Title     string
	Action    string
	CancelURL string
	Edit      bool
	HasErrors bool
	// Refused is the cause given by the model when it refuses the client for
	// another reason than a field shown with its own error.
	Refused string

	Fields map[string]field // text and number fields, by name
	Checks map[string]bool  // checkboxes, by name
	Proto  *protoform.Form  // the protocol: its family, TLS and options

	// The secondary cards fold (Driss, 2026-10-19): open when they hold a
	// value or an error, their summary says what they hold.
	AddressOpen    bool
	AddressSummary string
	RetriesOpen    bool
	RetriesSummary string

	// The values once validated.
	port     uint16
	attempts int8
	delay    int32
	factor   float32
}

// summarize decides which secondary cards open and what their summary says,
// from the values of the form (filled or posted) and their errors.
func (f *formData) summarize(translate gui.Translator) {
	value := func(name string) string { return f.Fields[name].Value }
	failed := func(names ...string) bool {
		for _, name := range names {
			if f.Fields[name].Error != "" {
				return true
			}
		}

		return false
	}

	host, port := value(fieldHost), value(fieldPort)
	f.AddressOpen = host != "" || port != "" || failed(fieldHost, fieldPort)
	f.AddressSummary = translate("clients.address_auto")

	switch {
	case host != "" && port != "":
		f.AddressSummary = host + ":" + port
	case host != "":
		f.AddressSummary = host
	case port != "":
		f.AddressSummary = "*:" + port
	}

	attempts := value(fieldAttempts)
	retryFields := []string{fieldAttempts, fieldDelayHours, fieldDelayMins, fieldDelaySecs, fieldFactor}
	f.RetriesOpen = failed(retryFields...)

	for _, name := range retryFields {
		if v := value(name); v != "" && v != "0" {
			f.RetriesOpen = true
		}
	}

	f.RetriesSummary = translate("clients.detail.retries_none")

	if attempts != "" && attempts != "0" {
		factor := value(fieldFactor)
		if factor == "" {
			factor = "1"
		}

		key := "clients.form.retries_summary"
		if attempts == "1" {
			key = "clients.form.retries_summary_one"
		}

		f.RetriesSummary = translate(key,
			"attempts", attempts, "delay", formatDelay(translate, f.postedDelay()), "factor", factor)
	}
}

// postedDelay reads the delay fields as they are, in seconds, for the
// summary; an unreadable part counts for nothing (validate flags it).
func (f *formData) postedDelay() int32 {
	var total int64

	for _, part := range []struct {
		name  string
		scale int64
	}{
		{fieldDelayHours, secondsPerHour}, {fieldDelayMins, secondsPerMinute}, {fieldDelaySecs, 1},
	} {
		if n, err := internal.ParseInt[int32](f.Fields[part.name].Value); err == nil && n > 0 {
			total += int64(n) * part.scale
		}
	}

	if total > math.MaxInt32 {
		return math.MaxInt32
	}

	return int32(total)
}

// newForm builds the form of a new client, or the form of an existing one
// filled with its values.
func (h *Handlers) newForm(translate gui.Translator, client *model.Client, carried string) *formData {
	form := &formData{
		Title:     translate("clients.new.title"),
		Action:    withRet(common.WebuiURL(pagePath+"/new"), carried),
		CancelURL: listURL(carried),
		Fields:    map[string]field{},
		Checks:    map[string]bool{},
		Proto:     protoform.NewForm(protoform.RoleClient),
	}

	if client != nil {
		form.Title = translate("clients.edit.title", "name", client.Name)
		form.Action = withRet(clientURL(client.ID, "/edit"), carried)
		form.CancelURL = withRet(clientURL(client.ID, ""), carried)
		form.Edit = true
		form.fill(client)
	}

	form.summarize(translate)
	form.Proto.Prepare(translate)

	return form
}

// fill reads the values of an existing client into the form.
func (f *formData) fill(client *model.Client) {
	set := func(name, value string) { f.Fields[name] = field{Value: value} }
	number := func(name string, n int32) {
		if n != 0 {
			set(name, strconv.FormatInt(int64(n), 10))
		} else {
			set(name, "")
		}
	}

	set(fieldName, client.Name)
	set(fieldHost, client.LocalAddress.Host)
	number(fieldPort, int32(client.LocalAddress.Port))
	number(fieldAttempts, int32(client.NbOfAttempts))

	hours, minutes, secs := splitDelay(client.FirstRetryDelay)
	number(fieldDelayHours, hours)
	number(fieldDelayMins, minutes)
	number(fieldDelaySecs, secs)

	if client.RetryIncrementFactor != 0 {
		set(fieldFactor, strconv.FormatFloat(float64(client.RetryIncrementFactor), 'f', -1, 32))
	} else {
		set(fieldFactor, "")
	}

	f.Checks[fieldNoAutoStart] = client.Disabled
	f.Proto.Fill(client.Protocol, client.ProtoConfig)
}

// read takes the posted values into the form: every field, whatever the
// protocol, so that a refused form comes back as the user filled it.
func (f *formData) read(values url.Values) {
	for _, name := range []string{
		fieldName, fieldHost, fieldPort, fieldAttempts, fieldDelayHours, fieldDelayMins, fieldDelaySecs, fieldFactor,
	} {
		f.Fields[name] = field{Value: strings.TrimSpace(values.Get(name))}
	}

	f.Checks[fieldNoAutoStart] = values.Get(fieldNoAutoStart) != ""
	f.Proto.Read(values)
}

// validate checks the fields the screen owns (the model checks the rest):
// the name, the optional local address, the retries, the protocol and the
// numbers of its options. It marks the errors on the form and returns
// whether it is valid.
//
//nolint:funlen // one check per field, in the order of the form
func (f *formData) validate(translate gui.Translator) bool {
	fail := func(name, key string) {
		fld := f.Fields[name]
		fld.Error = translate(key)
		f.Fields[name] = fld
		f.HasErrors = true
	}
	value := func(name string) string { return f.Fields[name].Value }

	if value(fieldName) == "" {
		fail(fieldName, "clients.form.error.required")
	}

	// The local address is optional: empty, the system chooses the
	// interface and the port; a port alone binds every interface.
	if v := value(fieldPort); v != "" {
		port, err := internal.ParseUint[uint16](v)
		if err != nil || port == 0 {
			fail(fieldPort, "clients.form.error.port")
		}

		f.port = port
	}

	if v := value(fieldAttempts); v != "" {
		attempts, err := internal.ParseInt[int8](v)
		if err != nil || attempts < 0 {
			fail(fieldAttempts, "clients.form.error.attempts")
		}

		f.attempts = attempts
	}

	var delay int64

	for _, part := range []struct {
		name  string
		scale int64
	}{
		{fieldDelayHours, secondsPerHour}, {fieldDelayMins, secondsPerMinute}, {fieldDelaySecs, 1},
	} {
		v := value(part.name)
		if v == "" {
			continue
		}

		n, err := internal.ParseInt[int32](v)
		if err != nil || n < 0 {
			fail(part.name, "clients.form.error.number")

			continue
		}

		delay += int64(n) * part.scale
	}

	if delay > math.MaxInt32 {
		fail(fieldDelayHours, "clients.form.error.number")
	}

	f.delay = int32(delay)

	if v := value(fieldFactor); v != "" {
		factor, err := strconv.ParseFloat(strings.ReplaceAll(v, ",", "."), 32)
		if err != nil || factor <= 0 {
			fail(fieldFactor, "clients.form.error.factor")
		}

		f.factor = float32(factor)
	}

	if !f.Proto.Validate(translate) {
		f.HasErrors = true
	}

	return !f.HasErrors
}

// apply writes the validated form into the client: its fields, its retries,
// its protocol and the options of the protocol, with the keys the product
// reads.
func (f *formData) apply(client *model.Client) {
	client.Name = f.Fields[fieldName].Value
	client.Protocol = f.Proto.Protocol()
	client.Disabled = f.Checks[fieldNoAutoStart]
	client.LocalAddress = types.Addr(f.Fields[fieldHost].Value, f.port)
	client.NbOfAttempts = f.attempts
	client.FirstRetryDelay = f.delay
	client.RetryIncrementFactor = f.factor
	client.ProtoConfig = f.Proto.Apply()
}

// refuse marks the form with the refusal of the model: the known causes go
// to their field, the others to the banner.
func (f *formData) refuse(translate gui.Translator, err error) {
	msg := err.Error()
	fail := func(name, key string) {
		fld := f.Fields[name]
		fld.Error = translate(key)
		f.Fields[name] = fld
	}

	switch {
	case strings.Contains(msg, "already exist"):
		fail(fieldName, "clients.form.error.name_taken")
	case strings.Contains(msg, "address validation failed"):
		fail(fieldHost, "clients.form.error.address")
	default:
		f.Refused = msg
	}

	f.HasErrors = true
}

// NewForm serves GET /webui/clients/new.
func (h *Handlers) NewForm(w http.ResponseWriter, r *http.Request) {
	translate := h.shell.Translator(r)
	carried := ret(r)
	f := h.newForm(translate, nil, carried)

	screen := h.screen(translate, f.Title, carried, 0)
	screen.Stylesheets = formStylesheets()
	h.shell.Render(w, h.logger, r, screen, formView(f))
}

// Create serves POST /webui/clients/new.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	h.save(w, r, nil)
}

// EditForm serves GET /webui/clients/{id}/edit.
func (h *Handlers) EditForm(w http.ResponseWriter, r *http.Request) {
	e, ok := h.client(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	f := h.newForm(translate, e.client, carried)

	screen := h.screen(translate, f.Title, carried, 0)
	screen.Stylesheets = formStylesheets()
	h.shell.Render(w, h.logger, r, screen, formView(f))
}

// Update serves POST /webui/clients/{id}/edit.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	e, ok := h.client(w, r)
	if !ok {
		return
	}

	h.save(w, r, e.client)
}

// save reads, validates and stores the posted form: a new client when client
// is nil, the given one otherwise. A refused form is rendered again with its
// errors (422); a saved one redirects to the list with a notification.
func (h *Handlers) save(w http.ResponseWriter, r *http.Request, client *model.Client) {
	translate := h.shell.Translator(r)
	carried := ret(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	form := h.newForm(translate, client, carried)
	form.read(r.PostForm)

	if form.validate(translate) {
		done := doneUpdated

		if client == nil {
			client = &model.Client{}
			done = doneCreated
		}

		form.apply(client)

		var err error
		if done == doneCreated {
			err = h.ctl.Add(h.db, client)
		} else {
			err = internal.UpdateClient(h.db, client)
		}

		switch {
		case err == nil:
			// A new client shows in the list; an edited one comes back to its
			// page (Driss, 2026-10-20).
			result := url.Values{doneParam: {done}, nameParam: {client.Name}}
			if done == doneUpdated {
				liststate.RedirectTo(w, r, withRet(clientURL(client.ID, ""), carried), result)
			} else {
				listSpec.RedirectList(w, r, carried, result)
			}

			return
		case database.IsValidationError(err):
			form.refuse(translate, err)
		default:
			h.logger.Errorf("failed to save the client %q: %v", client.Name, err)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}
	}

	form.summarize(translate)
	form.Proto.Prepare(translate)
	screen := h.screen(translate, form.Title, carried, http.StatusUnprocessableEntity)
	screen.Stylesheets = formStylesheets()
	h.shell.Render(w, h.logger, r, screen, formView(form))
}

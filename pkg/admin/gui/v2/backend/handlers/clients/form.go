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

	// The values once validated.
	port     uint16
	attempts int8
	delay    int32
	factor   float32
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
	translate := h.form.Translator(r)
	carried := ret(r)
	f := h.newForm(translate, nil, carried)

	h.form.Render(w, h.logger, r, h.screen(translate, f.Title, carried, 0, f))
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

	translate := h.form.Translator(r)
	carried := ret(r)
	f := h.newForm(translate, e.client, carried)

	h.form.Render(w, h.logger, r, h.screen(translate, f.Title, carried, 0, f))
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
	translate := h.form.Translator(r)
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
			listSpec.RedirectList(w, r, carried, url.Values{doneParam: {done}, nameParam: {client.Name}})

			return
		case database.IsValidationError(err):
			form.refuse(translate, err)
		default:
			h.logger.Errorf("failed to save the client %q: %v", client.Name, err)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}
	}

	form.Proto.Prepare(translate)
	h.form.Render(w, h.logger, r, h.screen(translate, form.Title, carried, http.StatusUnprocessableEntity, form))
}

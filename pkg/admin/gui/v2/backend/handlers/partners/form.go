package partners

import (
	"errors"
	"maps"
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
// shared protocol form (protoform).
const (
	fieldName = "name"
	fieldHost = "host"
	fieldPort = "port"
)

var errPortOutOfRange = errors.New("port out of range")

// field is a text or number field of the form: its value, kept when the form
// is refused, and the message of its error.
type field = protoform.Field

// formData is the form of a partner: its identity (name, protocol), the
// address the gateway connects to, the options of the protocol and the TLS
// block (the shared protocol form).
type formData struct {
	Title     string
	Action    string
	CancelURL string
	Edit      bool
	HasErrors bool
	// Refused is the cause given by the model when it refuses the partner
	// for another reason than a field shown with its own error.
	Refused string

	Fields map[string]field // text and number fields, by name
	Proto  *protoform.Form  // the protocol: its family, TLS and options

	port uint16 // the port once validated
}

// newForm builds the form of a new partner, or the form of an existing one
// filled with its values.
func (h *Handlers) newForm(translate gui.Translator, agent *model.RemoteAgent, carried string) *formData {
	form := &formData{
		Title:     translate("partners.new.title"),
		Action:    withRet(common.WebuiURL(pagePath+"/new"), carried),
		CancelURL: listURL(carried),
		Fields:    map[string]field{},
		Proto:     protoform.NewForm(protoform.RolePartner),
	}

	if agent != nil {
		form.Title = translate("partners.edit.title", "name", agent.Name)
		form.Action = withRet(partnerURL(agent.ID, editSuffix), carried)
		form.CancelURL = withRet(partnerURL(agent.ID, ""), carried)
		form.Edit = true
		form.fill(agent)
	}

	form.Proto.Prepare(translate)

	return form
}

// fill reads the values of an existing partner into the form.
func (f *formData) fill(agent *model.RemoteAgent) {
	f.Fields[fieldName] = field{Value: agent.Name}
	f.Fields[fieldHost] = field{Value: agent.Address.Host}
	f.Fields[fieldPort] = field{Value: strconv.FormatUint(uint64(agent.Address.Port), 10)}
	f.Proto.Fill(agent.Protocol, agent.ProtoConfig)
}

// read takes the posted values into the form: every field, whatever the
// protocol, so that a refused form comes back as the user filled it.
func (f *formData) read(values url.Values) {
	for _, name := range []string{fieldName, fieldHost, fieldPort} {
		f.Fields[name] = field{Value: strings.TrimSpace(values.Get(name))}
	}

	f.Proto.Read(values)
}

// validate checks the fields the screen owns (the model checks the rest):
// the required ones, the port, the protocol and the values of its options.
// It marks the errors on the form and returns whether it is valid.
func (f *formData) validate(translate gui.Translator) bool {
	fail := func(name, key string) {
		fld := f.Fields[name]
		fld.Error = translate(key)
		f.Fields[name] = fld
		f.HasErrors = true
	}

	if f.Fields[fieldName].Value == "" {
		fail(fieldName, "partners.form.error.required")
	}

	// A partner always has an address: the gateway connects to it.
	if f.Fields[fieldHost].Value == "" {
		fail(fieldHost, "partners.form.error.host")
	}

	port, err := parsePort(f.Fields[fieldPort].Value)
	if err != nil {
		fail(fieldPort, "partners.form.error.port")
	}

	f.port = port

	if !f.Proto.Validate(translate) {
		f.HasErrors = true
	}

	return !f.HasErrors
}

func parsePort(s string) (uint16, error) {
	port, err := internal.ParseUint[uint16](s)
	if err != nil {
		return 0, err
	}

	if port == 0 {
		return 0, errPortOutOfRange
	}

	return port, nil
}

// apply writes the validated form into the partner: its fields, its protocol
// and the options of the protocol, with the keys the product reads. The
// options the screen does not show (the deprecated R66 password and TLS
// flag) stay as they are in the configuration.
func (f *formData) apply(agent *model.RemoteAgent) {
	agent.Name = f.Fields[fieldName].Value
	agent.Protocol = f.Proto.Protocol()
	agent.Address = types.Addr(f.Fields[fieldHost].Value, f.port)

	kept := model.Map[any]{}

	for _, key := range []string{"serverPassword", "isTLS"} {
		if v, ok := agent.ProtoConfig[key]; ok {
			kept[key] = v
		}
	}

	agent.ProtoConfig = f.Proto.Apply()
	maps.Copy(agent.ProtoConfig, kept)
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
	case strings.Contains(msg, "same name"):
		fail(fieldName, "partners.form.error.name_taken")
	case strings.Contains(msg, "address validation failed"):
		fail(fieldHost, "partners.form.error.address")
	case strings.Contains(msg, "async") && strings.Contains(msg, "MDN"):
		f.Proto.Fail("as2_async_mdn", translate("partners.form.error.async_mdn"))
	default:
		f.Refused = msg
	}

	f.HasErrors = true
}

// NewForm serves GET /webui/partners/new.
func (h *Handlers) NewForm(w http.ResponseWriter, r *http.Request) {
	translate := h.shell.Translator(r)
	carried := ret(r)
	f := h.newForm(translate, nil, carried)

	screen := h.screen(translate, f.Title, carried, 0)
	screen.Stylesheets = formStylesheets()
	h.shell.Render(w, h.logger, r, screen, formView(f))
}

// Create serves POST /webui/partners/new.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	h.save(w, r, nil)
}

// EditForm serves GET /webui/partners/{id}/edit.
func (h *Handlers) EditForm(w http.ResponseWriter, r *http.Request) {
	e, ok := h.partner(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	f := h.newForm(translate, e.agent, carried)

	screen := h.screen(translate, f.Title, carried, 0)
	screen.Stylesheets = formStylesheets()
	h.shell.Render(w, h.logger, r, screen, formView(f))
}

// Update serves POST /webui/partners/{id}/edit.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	e, ok := h.partner(w, r)
	if !ok {
		return
	}

	h.save(w, r, e.agent)
}

// save reads, validates and stores the posted form: a new partner when agent
// is nil, the given one otherwise. A refused form is rendered again with its
// errors (422); a saved one redirects to the list with a notification.
func (h *Handlers) save(w http.ResponseWriter, r *http.Request, agent *model.RemoteAgent) {
	translate := h.shell.Translator(r)
	carried := ret(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	form := h.newForm(translate, agent, carried)
	form.read(r.PostForm)

	if form.validate(translate) {
		done := doneUpdated

		if agent == nil {
			agent = &model.RemoteAgent{}
			done = doneCreated
		}

		form.apply(agent)

		var err error
		if done == doneCreated {
			err = internal.InsertPartner(h.db, agent)
		} else {
			err = internal.UpdatePartner(h.db, agent)
		}

		switch {
		case err == nil:
			redirectList(w, r, carried, url.Values{doneParam: {done}, nameParam: {agent.Name}})

			return
		case database.IsValidationError(err):
			form.refuse(translate, err)
		default:
			h.logger.Errorf("failed to save the partner %q: %v", agent.Name, err)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}
	}

	form.Proto.Prepare(translate)
	screen := h.screen(translate, form.Title, carried, http.StatusUnprocessableEntity)
	screen.Stylesheets = formStylesheets()
	h.shell.Render(w, h.logger, r, screen, formView(form))
}

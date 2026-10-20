package servers

import (
	"errors"
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
	fieldName          = "name"
	fieldNoAutoStart   = "no_autostart"
	fieldHost          = "host"
	fieldPort          = "port"
	fieldRootDir       = "root_dir"
	fieldReceiveDir    = "receive_dir"
	fieldTmpReceiveDir = "tmp_receive_dir"
	fieldSendDir       = "send_dir"
)

var errPortOutOfRange = errors.New("port out of range")

// field is a text or number field of the form: its value, kept when the form
// is refused, and the message of its error.
type field = protoform.Field

type formData struct {
	Title     string
	Action    string
	CancelURL string
	Edit      bool
	HasErrors bool
	// Refused is the cause given by the model when it refuses the server for
	// another reason than a field shown with its own error.
	Refused string

	Fields map[string]field // text and number fields, by name
	Checks map[string]bool  // checkboxes, by name
	Proto  *protoform.Form  // the protocol: its family, TLS and options

	// The directories card folds (Driss, 2026-10-19): open when it holds a
	// value or an error, its summary says what it holds.
	DirsOpen    bool
	DirsSummary string

	port uint16 // the port once validated
}

// summarize decides whether the directories card opens and what its summary
// says, from the values of the form (filled or posted) and their errors.
func (f *formData) summarize(translate gui.Translator) {
	dirs := []string{fieldRootDir, fieldReceiveDir, fieldTmpReceiveDir, fieldSendDir}
	set := 0

	for _, name := range dirs {
		fld := f.Fields[name]
		if fld.Value != "" {
			set++
		}

		if fld.Error != "" {
			f.DirsOpen = true
		}
	}

	f.DirsOpen = f.DirsOpen || set > 0

	switch {
	case f.Fields[fieldRootDir].Value != "":
		f.DirsSummary = translate("servers.form.dirs_root_summary", "root", f.Fields[fieldRootDir].Value)
	case set > 0:
		f.DirsSummary = translate("servers.form.dirs_count_summary", "n", set)
	default:
		f.DirsSummary = translate("servers.form.dirs_default_summary")
	}
}

// newForm builds the form of a new server, or the form of an existing one
// filled with its values.
func (h *Handlers) newForm(translate gui.Translator, agent *model.LocalAgent, carried string) *formData {
	form := &formData{
		Title:     translate("servers.new.title"),
		Action:    withRet(common.WebuiURL(pagePath+"/new"), carried),
		CancelURL: listURL(carried),
		Fields:    map[string]field{},
		Checks:    map[string]bool{},
		Proto:     protoform.NewForm(protoform.RoleServer),
	}

	if agent != nil {
		form.Title = translate("servers.edit.title", "name", agent.Name)
		form.Action = withRet(serverURL(agent.ID, "/edit"), carried)
		form.CancelURL = withRet(serverURL(agent.ID, ""), carried)
		form.Edit = true
		form.fill(agent)
	}

	form.summarize(translate)
	form.Proto.Prepare(translate)

	return form
}

// fill reads the values of an existing server into the form.
func (f *formData) fill(agent *model.LocalAgent) {
	set := func(name, value string) { f.Fields[name] = field{Value: value} }

	set(fieldName, agent.Name)
	set(fieldHost, agent.Address.Host)
	set(fieldPort, strconv.FormatUint(uint64(agent.Address.Port), 10))
	set(fieldRootDir, agent.RootDir)
	set(fieldReceiveDir, agent.ReceiveDir)
	set(fieldTmpReceiveDir, agent.TmpReceiveDir)
	set(fieldSendDir, agent.SendDir)
	f.Checks[fieldNoAutoStart] = agent.Disabled
	f.Proto.Fill(agent.Protocol, agent.ProtoConfig)
}

// read takes the posted values into the form: every field, whatever the
// protocol, so that a refused form comes back as the user filled it.
func (f *formData) read(values url.Values) {
	for _, name := range []string{
		fieldName, fieldHost, fieldPort, fieldRootDir, fieldReceiveDir, fieldTmpReceiveDir, fieldSendDir,
	} {
		f.Fields[name] = field{Value: strings.TrimSpace(values.Get(name))}
	}

	f.Checks[fieldNoAutoStart] = values.Get(fieldNoAutoStart) != ""
	f.Proto.Read(values)
}

// validate checks the fields the screen owns (the model checks the rest):
// the required ones, the port, the protocol and the numbers of its options.
// It marks the errors on the form and returns whether it is valid.
func (f *formData) validate(translate gui.Translator) bool {
	fail := func(name, key string) {
		fld := f.Fields[name]
		fld.Error = translate(key)
		f.Fields[name] = fld
		f.HasErrors = true
	}

	// The host is optional: empty, the server listens on every interface.
	if f.Fields[fieldName].Value == "" {
		fail(fieldName, "servers.form.error.required")
	}

	port, err := parsePort(f.Fields[fieldPort].Value)
	if err != nil {
		fail(fieldPort, "servers.form.error.port")
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

// apply writes the validated form into the server: its fields, its protocol
// and the options of the protocol, with the keys the product reads.
func (f *formData) apply(agent *model.LocalAgent) {
	value := func(name string) string { return f.Fields[name].Value }

	agent.Name = value(fieldName)
	agent.Protocol = f.Proto.Protocol()
	agent.Disabled = f.Checks[fieldNoAutoStart]
	agent.RootDir = value(fieldRootDir)
	agent.ReceiveDir = value(fieldReceiveDir)
	agent.TmpReceiveDir = value(fieldTmpReceiveDir)
	agent.SendDir = value(fieldSendDir)
	agent.Address = types.Addr(value(fieldHost), f.port)
	agent.ProtoConfig = f.Proto.Apply()
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
		fail(fieldName, "servers.form.error.name_taken")
	case strings.Contains(msg, "same address"):
		fail(fieldHost, "servers.form.error.address_taken")
	case strings.Contains(msg, "address validation failed"):
		fail(fieldHost, "servers.form.error.address")
	default:
		f.Refused = msg
	}

	f.HasErrors = true
}

// NewForm serves GET /webui/servers/new.
func (h *Handlers) NewForm(w http.ResponseWriter, r *http.Request) {
	translate := h.shell.Translator(r)
	carried := ret(r)
	f := h.newForm(translate, nil, carried)

	screen := h.screen(translate, f.Title, carried, 0)
	screen.Stylesheets = formStylesheets()
	h.shell.Render(w, h.logger, r, screen, formView(f))
}

// Create serves POST /webui/servers/new.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	h.save(w, r, nil)
}

// EditForm serves GET /webui/servers/{id}/edit.
func (h *Handlers) EditForm(w http.ResponseWriter, r *http.Request) {
	e, ok := h.server(w, r)
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

// Update serves POST /webui/servers/{id}/edit.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	e, ok := h.server(w, r)
	if !ok {
		return
	}

	h.save(w, r, e.agent)
}

// save reads, validates and stores the posted form: a new server when agent
// is nil, the given one otherwise. A refused form is rendered again with its
// errors (422); a saved one redirects to the list with a notification.
func (h *Handlers) save(w http.ResponseWriter, r *http.Request, agent *model.LocalAgent) {
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
			agent = &model.LocalAgent{}
			done = doneCreated
		}

		form.apply(agent)

		var err error
		if done == doneCreated {
			err = h.ctl.Add(h.db, agent)
		} else {
			err = internal.UpdateServer(h.db, agent)
		}

		switch {
		case err == nil:
			// A new server shows in the list; an edited one comes back to its
			// sheet (Driss, 2026-10-20).
			if done == doneUpdated {
				redirectSheet(w, r, withRet(serverURL(agent.ID, ""), carried), "", done, agent.Name)
			} else {
				redirectList(w, r, carried, url.Values{doneParam: {done}, nameParam: {agent.Name}})
			}

			return
		case database.IsValidationError(err):
			form.refuse(translate, err)
		default:
			h.logger.Errorf("failed to save the server %q: %v", agent.Name, err)
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

package smtp

import (
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/ruleuse"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// Names of the fields of the form.
const (
	fieldSender   = "sender"
	fieldHost     = "host"
	fieldPort     = "port"
	fieldLogin    = "login"
	fieldPassword = "password"

	// The port proposed for a new credential: message submission, with
	// STARTTLS.
	defaultPort = 587

	// The causes the model gives when it refuses a credential.
	causeTaken   = "already exists"
	causeInvalid = "sender address is invalid"
)

type field = protoform.Field

type formData struct {
	Title     string
	Action    string
	CancelURL string
	Edit      bool
	HasErrors bool
	Refused   string // a refusal of the model no field carries

	// Locked: rules use the credential, its sender is kept (their EMAIL
	// tasks name it); Rules says how many.
	Locked bool
	Rules  string

	Sender   field
	Host     field
	Port     field
	Login    field
	Password field // its error only: a password is never rendered back

	// PasswordSet: on edition, a password is in place and an empty field
	// keeps it.
	PasswordSet bool

	stored   *model.SMTPCredential
	port     uint16
	password string
}

// newForm builds the form of a new credential (the submission port
// proposed), or the form of an existing one filled with its values (never
// its password).
func newForm(translate gui.Translator, e *entry, carried string) *formData {
	form := &formData{
		Title:     translate("smtp.new.title"),
		Action:    withRet(common.WebuiURL(pagePath+"/new"), carried),
		CancelURL: listURL(carried),
		Port:      field{Value: strconv.Itoa(defaultPort)},
	}

	if e != nil {
		addr := e.cred.ServerAddress
		form.Title = translate("smtp.edit.title", "name", e.cred.EmailAddress)
		form.Action = withRet(credURL(e.cred.ID, "/edit"), carried)
		form.CancelURL = withRet(credURL(e.cred.ID, ""), carried)
		form.Edit = true
		form.stored = e.cred
		form.Sender = field{Value: e.cred.EmailAddress}
		form.Host = field{Value: addr.Host}
		form.Port = field{}
		form.Login = field{Value: e.cred.Login}
		form.PasswordSet = e.cred.Password != ""
		form.Locked = len(e.uses) > 0
		form.Rules = ruleuse.Count(translate, len(e.uses))

		if addr.Port != 0 {
			form.Port = field{Value: strconv.Itoa(int(addr.Port))}
		}
	}

	return form
}

// read takes the posted values: the sender (the one in place for a
// credential the rules use), the server, the login and the password.
func (f *formData) read(values url.Values) {
	if !f.Locked {
		f.Sender = field{Value: strings.TrimSpace(values.Get(fieldSender))}
	}

	f.Host = field{Value: strings.TrimSpace(values.Get(fieldHost))}
	f.Port = field{Value: strings.TrimSpace(values.Get(fieldPort))}
	f.Login = field{Value: strings.TrimSpace(values.Get(fieldLogin))}
	f.password = values.Get(fieldPassword)
}

// validate checks what the screen can check before the model: a sender that
// is an address alone (one in place with a display name is kept as it is),
// a host, a port from 1 to 65535, and no password without a login.
func (f *formData) validate(translate gui.Translator) bool {
	fail := func(fld *field, key string) {
		fld.Error = translate(key)
		f.HasErrors = true
	}

	switch addr, err := mail.ParseAddress(f.Sender.Value); {
	case f.Sender.Value == "":
		fail(&f.Sender, "smtp.form.error.required")
	case f.stored != nil && f.Sender.Value == f.stored.EmailAddress:
	case err != nil || addr.Name != "" || addr.Address != f.Sender.Value:
		fail(&f.Sender, "smtp.form.error.sender")
	}

	if f.Host.Value == "" {
		fail(&f.Host, "smtp.form.error.host")
	}

	port, err := internal.ParseUint[uint16](f.Port.Value)
	if err != nil || port == 0 {
		fail(&f.Port, "smtp.form.error.port")
	}

	f.port = port

	if f.Login.Value == "" && f.password != "" {
		fail(&f.Password, "smtp.form.error.password_without_login")
	}

	return !f.HasErrors
}

// apply writes the validated form into the credential: an empty password
// keeps the stored one; an empty login removes the authentication, password
// included.
func (f *formData) apply(c *model.SMTPCredential) {
	c.EmailAddress = f.Sender.Value
	c.ServerAddress = types.Addr(f.Host.Value, f.port)
	c.Login = f.Login.Value

	switch {
	case c.Login == "":
		c.Password = ""
	case f.password != "":
		c.Password = f.password
	}
}

// refuse marks the form with the refusal of the model: the known causes go
// to their field, the others to the banner.
func (f *formData) refuse(translate gui.Translator, err error) {
	msg := err.Error()

	switch {
	case strings.Contains(msg, causeTaken):
		f.Sender.Error = translate("smtp.form.error.sender_taken")
	case strings.Contains(msg, causeInvalid):
		f.Sender.Error = translate("smtp.form.error.sender")
	default:
		f.Refused = msg
	}

	f.HasErrors = true
}

// render shows the form.
func (h *Handlers) render(w http.ResponseWriter, r *http.Request, form *formData, status int) {
	translate := h.shell.Translator(r)
	h.shell.Render(w, h.logger, r, h.screen(translate, form.Title, ret(r), status), formView(form))
}

// NewForm serves GET /webui/smtp/new.
func (h *Handlers) NewForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, newForm(h.shell.Translator(r), nil, ret(r)), 0)
}

// Create serves POST /webui/smtp/new.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	h.save(w, r, nil)
}

// EditForm serves GET /webui/smtp/{id}/edit.
func (h *Handlers) EditForm(w http.ResponseWriter, r *http.Request) {
	e, ok := h.credential(w, r)
	if !ok {
		return
	}

	h.render(w, r, newForm(h.shell.Translator(r), &e, ret(r)), 0)
}

// Update serves POST /webui/smtp/{id}/edit.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	e, ok := h.credential(w, r)
	if !ok {
		return
	}

	h.save(w, r, &e)
}

// save reads, validates and stores the posted form: a new credential when e
// is nil, the given one otherwise. A refused form, by the screen or by the
// model, is rendered again with its errors (422), without the password
// typed; a saved one redirects with a notification: a new credential shows
// in the list, an edited one comes back to its sheet.
func (h *Handlers) save(w http.ResponseWriter, r *http.Request, e *entry) {
	translate := h.shell.Translator(r)
	carried := ret(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	form := newForm(translate, e, carried)
	form.read(r.PostForm)

	if form.validate(translate) {
		// A copy: a refused edition leaves the credential as it was read.
		done, cred := doneCreated, &model.SMTPCredential{}
		if e != nil {
			stored := *e.cred
			done, cred = doneUpdated, &stored
		}

		form.apply(cred)

		var err error
		if done == doneCreated {
			err = h.db.Insert(cred).Run()
		} else {
			err = h.db.Update(cred).Run()
		}

		switch {
		case err == nil:
			result := url.Values{doneParam: {done}, nameParam: {cred.EmailAddress}}
			if done == doneUpdated {
				liststate.RedirectTo(w, r, withRet(credURL(cred.ID, ""), carried), result)
			} else {
				listSpec.RedirectList(w, r, carried, result)
			}

			return
		case database.IsValidationError(err):
			form.refuse(translate, err)
		default:
			h.logger.Errorf("failed to save the SMTP credential %q: %v", cred.EmailAddress, err)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}
	}

	h.render(w, r, form, http.StatusUnprocessableEntity)
}

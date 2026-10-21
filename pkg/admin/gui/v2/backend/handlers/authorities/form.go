package authorities

import (
	"net/http"
	"net/url"
	"slices"
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/means"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// Names of the fields of the form. The public identity of each type has its
// own block ("tls_authority_text", "ssh_cert_authority_file"): the type select
// shows the one of its value (data-switch), the server reads the one of the
// chosen type.
const (
	fieldName  = "name"
	fieldType  = "type"
	fieldHosts = "hosts"

	suffixText = "_text"
	suffixFile = "_file"

	// The causes the model gives when it refuses an authority.
	causeTaken      = "already exists"
	causeNotCA      = "not a CA certificate"
	causeValidation = "could not validate the authority's public identity value: "
	causeType       = "is not a valid authority type"
)

type field = protoform.Field

// identityView is the block of the public identity of one type of authority:
// shown for the chosen type.
type identityView struct {
	Key         string
	TLS         bool
	Label       string
	Hint        string
	Placeholder string
	Accept      string
	Text        string // as pasted, kept when the form is refused
	Error       string
	Current     string // what is in place, on edition of an authority of this type
}

func (v *identityView) textID() string { return v.Key + suffixText }
func (v *identityView) fileID() string { return v.Key + suffixFile }

type formData struct {
	Title     string
	Action    string
	CancelURL string
	Edit      bool
	HasErrors bool
	Refused   string // a refusal of the model no field carries

	Name    field
	Type    string // the chosen type, "" when none
	Types   []choice
	TypeErr string

	Identities      []identityView
	IdentitySummary string // in the header of the folded identity card
	Hosts           field
	HostsOpen       bool
	HostsSummary    string

	// What the authority held before the edition: its type and its public
	// identity, kept when the field is left empty and the type unchanged.
	stored *model.Authority

	identity string            // the public identity read for the chosen type
	texts    map[string]string // the pasted texts, by type
	errors   map[string]string // the errors of the identity blocks, by type
	hosts    []string
}

// newForm builds the form of a new authority, or the form of an existing
// one filled with its values.
func newForm(translate gui.Translator, e *entry, carried string) *formData {
	form := &formData{
		Title:     translate("authorities.new.title"),
		Action:    withRet(common.WebuiURL(pagePath+"/new"), carried),
		CancelURL: listURL(carried),
		texts:     map[string]string{},
		errors:    map[string]string{},
	}

	if e != nil {
		form.Title = translate("authorities.edit.title", "name", e.authority.Name)
		form.Action = withRet(authorityURL(e.authority.ID, "/edit"), carried)
		form.CancelURL = withRet(authorityURL(e.authority.ID, ""), carried)
		form.Edit = true
		form.stored = e.authority
		form.Name = field{Value: e.authority.Name}
		form.Type = e.authority.Type
		form.Hosts = field{Value: strings.Join(e.authority.ValidHosts, "\n")}
	}

	return form
}

// read takes the posted values: the name, the type, the pasted texts of
// every type (kept when the form is refused), the public identity of the
// chosen type (an uploaded file first, else the pasted text), the hosts.
func (f *formData) read(r *http.Request) error {
	f.Name = field{Value: strings.TrimSpace(r.PostFormValue(fieldName))}
	f.Hosts = field{Value: strings.ReplaceAll(r.PostFormValue(fieldHosts), "\r\n", "\n")}
	f.Type = ""

	if k, ok := kindOf(r.PostFormValue(fieldType)); ok {
		f.Type = k.Key
	}

	for i := range kinds {
		f.texts[kinds[i].Key] = r.PostFormValue(kinds[i].Key + suffixText)
	}

	if f.Type == "" {
		return nil
	}

	identity, err := means.FileOrText(r, f.Type+suffixFile, f.Type+suffixText)
	if err != nil {
		return err //nolint:wrapcheck // the caller answers 400
	}

	f.identity = identity
	f.hosts = parseHosts(f.Hosts.Value)

	return nil
}

// parseHosts reads the allowed hosts: one per line, blanks and duplicates
// dropped, in the order given.
func parseHosts(text string) []string {
	var hosts []string

	for line := range strings.SplitSeq(text, "\n") {
		if host := strings.TrimSpace(line); host != "" && !slices.Contains(hosts, host) {
			hosts = append(hosts, host)
		}
	}

	return hosts
}

// validate checks what the screen can check before the model: the name, the
// type, the public identity given (required for a new authority or a new
// type) and readable for its type.
func (f *formData) validate(translate gui.Translator) bool {
	if f.Name.Value == "" {
		f.Name.Error = translate("authorities.form.error.required")
		f.HasErrors = true
	}

	k, ok := kindOf(f.Type)
	if !ok {
		f.TypeErr = translate("authorities.form.error.type")
		f.HasErrors = true

		return false
	}

	switch {
	case f.identity == "" && (f.stored == nil || f.stored.Type != f.Type):
		f.errors[k.Key] = translate("authorities.form.error.required")
		f.HasErrors = true
	case f.identity == "":
	default:
		if _, readable := identityOf(k.Key, f.identity); !readable {
			f.errors[k.Key] = translate("authorities.form.error.unreadable." + k.Key)
			f.HasErrors = true
		}
	}

	return !f.HasErrors
}

// apply writes the validated form into the authority: an empty identity
// keeps the stored one (same type); the hosts replace the stored ones.
func (f *formData) apply(a *model.Authority) {
	a.Name = f.Name.Value
	a.Type = f.Type
	a.ValidHosts = f.hosts

	if f.identity != "" {
		a.PublicIdentity = f.identity
	}
}

// refuse marks the form with the refusal of the model: the known causes go
// to their field, the others to the banner.
func (f *formData) refuse(translate gui.Translator, err error) {
	msg := err.Error()

	switch {
	case strings.Contains(msg, causeTaken):
		f.Name.Error = translate("authorities.form.error.name_taken")
	case strings.Contains(msg, causeNotCA):
		f.errors[f.Type] = translate("authorities.form.error.not_ca")
	case strings.Contains(msg, causeValidation):
		_, cause, _ := strings.Cut(msg, causeValidation)
		f.errors[f.Type] = translate("authorities.form.error.not_verified", "cause", cause)
	case strings.Contains(msg, causeType):
		f.TypeErr = translate("authorities.form.error.type")
	default:
		f.Refused = msg
	}

	f.HasErrors = true
}

// prepare builds what the templates render: the type choices, the identity
// block of every type with the text as pasted and what is in place (current:
// the summary of the stored identity), the summaries of the folded cards.
func (f *formData) prepare(translate gui.Translator, current string) {
	f.Types = make([]choice, 0, len(kinds))
	f.Identities = make([]identityView, 0, len(kinds))

	for i := range kinds {
		k := &kinds[i]
		f.Types = append(f.Types, choice{Value: k.Key, Label: translate(k.labelKey()), Selected: f.Type == k.Key})

		view := identityView{
			Key: k.Key, TLS: k.TLS, Accept: k.Accept,
			Label:       translate("authorities.form.identity." + k.Key),
			Hint:        translate("authorities.form.identity_hint." + k.Key),
			Placeholder: translate("authorities.form.identity_example." + k.Key),
			Text:        f.texts[k.Key],
			Error:       f.errors[k.Key],
		}

		if f.stored != nil && f.stored.Type == k.Key {
			view.Current = current
		}

		f.Identities = append(f.Identities, view)
	}

	f.IdentitySummary = translate("authorities.form.to_give")
	if f.Edit && current != "" {
		f.IdentitySummary = current
	}

	hosts := parseHosts(f.Hosts.Value)
	f.HostsOpen = len(hosts) > 0 || f.Hosts.Error != ""
	f.HostsSummary = translate("authorities.hosts.all")

	switch n := len(hosts); {
	case n == 1:
		f.HostsSummary = translate("authorities.form.hosts_one")
	case n > 1:
		f.HostsSummary = translate("authorities.form.hosts_count", "n", n)
	}
}

// render shows the form, with the summary of what is in place: the subject
// of the certificate and its validity, or the type of the key.
func (h *Handlers) render(w http.ResponseWriter, r *http.Request, form *formData, status int) {
	translate := h.shell.Translator(r)

	var current string

	if form.stored != nil {
		if e := entryOf(form.stored); e.Readable {
			current = e.summary(translate)
			if state := e.validity(translate, h.now()); state.Label != "" {
				current += " · " + state.Label
			}
		}
	}

	form.prepare(translate, current)
	h.shell.Render(w, h.logger, r, h.screen(translate, form.Title, ret(r), status), formView(form))
}

// NewForm serves GET /webui/authorities/new.
func (h *Handlers) NewForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, newForm(h.shell.Translator(r), nil, ret(r)), 0)
}

// Create serves POST /webui/authorities/new.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	h.save(w, r, nil)
}

// EditForm serves GET /webui/authorities/{id}/edit.
func (h *Handlers) EditForm(w http.ResponseWriter, r *http.Request) {
	e, ok := h.authority(w, r)
	if !ok {
		return
	}

	h.render(w, r, newForm(h.shell.Translator(r), &e, ret(r)), 0)
}

// Update serves POST /webui/authorities/{id}/edit.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	e, ok := h.authority(w, r)
	if !ok {
		return
	}

	h.save(w, r, &e)
}

// save reads, validates and stores the posted form (multipart with a file,
// or not): a new authority when e is nil, the given one otherwise. A refused
// form, by the screen or by the model (which checks the certificate), is
// rendered again with its errors (422); a saved one redirects with a
// notification: a new authority shows in the list, an edited one comes back
// to its sheet.
func (h *Handlers) save(w http.ResponseWriter, r *http.Request, e *entry) {
	translate := h.shell.Translator(r)
	carried := ret(r)

	if err := means.ParseForm(r); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	form := newForm(translate, e, carried)
	if err := form.read(r); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	if form.validate(translate) {
		// A copy: a refused edition leaves the authority as it was read.
		done, authority := doneCreated, &model.Authority{}
		if e != nil {
			stored := *e.authority
			done, authority = doneUpdated, &stored
		}

		form.apply(authority)

		var err error
		if done == doneCreated {
			err = h.db.Insert(authority).Run()
		} else {
			err = h.db.Update(authority).Run()
		}

		switch {
		case err == nil:
			result := url.Values{doneParam: {done}, nameParam: {authority.Name}}
			if done == doneUpdated {
				liststate.RedirectTo(w, r, withRet(authorityURL(authority.ID, ""), carried), result)
			} else {
				listSpec.RedirectList(w, r, carried, result)
			}

			return
		case database.IsValidationError(err):
			form.refuse(translate, err)
		default:
			h.logger.Errorf("failed to save the authentication authority %q: %v", authority.Name, err)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}
	}

	h.render(w, r, form, http.StatusUnprocessableEntity)
}

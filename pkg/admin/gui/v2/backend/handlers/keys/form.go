package keys

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/means"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// Names of the fields of the form. The value of each type has its own block
// ("value_aes", "value_pgp_public_file"): the type select shows the one of
// its value (data-switch), the server reads the one of the chosen type.
const (
	fieldName = "name"
	fieldType = "type"

	prefixValue = "value_"
	suffixText  = "_text"
	suffixFile  = "_file"

	// The files the form offers for a PGP key: the armour the product stores.
	pgpAccept = ".asc,.key,.txt"

	// The causes the model gives when it refuses a key.
	causeTaken   = "already exists"
	causeAESSize = "AES keys must be"
	causeNotPriv = "is not a private key"
	causeNotPub  = "is not a public key"
	causeUnknown = "unknown cryptographic key type"

	// Prefixes of the catalogue keys of the value blocks, completed by the
	// suffix of the type ("aes", "pgp_public"), and of the errors.
	valueKeyPrefix   = "keys.form.value."
	pasteKeyPrefix   = "keys.form.paste."
	exampleKeyPrefix = "keys.form.example."
	keepKeyPrefix    = "keys.form.keep."
	hintKeyPrefix    = "keys.form.hint."
	errorKeyPrefix   = "keys.form.error."
)

type field = protoform.Field

// valueView is the block of the value of one type of key, shown for the
// chosen type: a secret field (AES, HMAC), or a file or the pasted armour
// (PGP).
type valueView struct {
	Key     string
	Slug    string
	PGP     bool
	Accept  string
	Text    string // the pasted public armour, kept when the form is refused
	Error   string
	Current string // what is in place, on edition of a key of this type
}

func (v *valueView) id() string     { return prefixValue + v.Slug }
func (v *valueView) textID() string { return v.id() + suffixText }
func (v *valueView) fileID() string { return v.id() + suffixFile }

type formData struct {
	Title     string
	Action    string
	CancelURL string
	Edit      bool
	HasErrors bool
	Refused   string // a refusal of the model no field carries

	// Locked: the key is used by rules, its name and type are kept (the
	// tasks name it); Rules says how many.
	Locked    bool
	Rules     string
	TypeLabel string

	Name    field
	Type    string // the chosen type, "" when none
	Types   []choice
	TypeErr string

	Values       []valueView
	ValueSummary string // in the header of the value card

	// What the key held before the edition: its type and its value, kept
	// when the field is left empty and the type unchanged.
	stored *model.CryptoKey

	value  string            // the value read for the chosen type
	texts  map[string]string // the pasted public armours, by type
	errors map[string]string // the errors of the value blocks, by type
}

// newForm builds the form of a new key, or the form of an existing one
// filled with its name and type (never its value).
func newForm(translate gui.Translator, e *entry, carried string) *formData {
	form := &formData{
		Title:     translate("keys.new.title"),
		Action:    withRet(common.WebuiURL(pagePath+"/new"), carried),
		CancelURL: listURL(carried),
		texts:     map[string]string{},
		errors:    map[string]string{},
	}

	if e != nil {
		form.Title = translate("keys.edit.title", "name", e.key.Name)
		form.Action = withRet(keyURL(e.key.ID, "/edit"), carried)
		form.CancelURL = withRet(keyURL(e.key.ID, ""), carried)
		form.Edit = true
		form.stored = e.key
		form.Name = field{Value: e.key.Name}
		form.Type = e.key.Type
		form.TypeLabel = e.typeLabel(translate)
		form.Locked = len(e.uses) > 0
		form.Rules = usesText(translate, len(e.uses))
	}

	return form
}

// read takes the posted values: the name and the type (those in place for a
// key the rules use), the pasted public armour (kept when the form is
// refused), the value of the chosen type (a secret field, or for PGP an
// uploaded file first, else the pasted armour).
func (f *formData) read(r *http.Request) error {
	if !f.Locked {
		f.Name = field{Value: strings.TrimSpace(r.PostFormValue(fieldName))}
		f.Type = ""

		if k, ok := kindOf(r.PostFormValue(fieldType)); ok {
			f.Type = k.Key
		}
	}

	for i := range kinds {
		if k := &kinds[i]; k.PGP && !k.Secret {
			f.texts[k.Key] = r.PostFormValue(prefixValue + k.Slug + suffixText)
		}
	}

	k, ok := kindOf(f.Type)
	if !ok {
		return nil
	}

	if !k.PGP {
		// An AES key and an HMAC secret are in base64: spaces around are not
		// part of them.
		f.value = strings.TrimSpace(r.PostFormValue(prefixValue + k.Slug))

		return nil
	}

	value, err := means.FileOrText(r, prefixValue+k.Slug+suffixFile, prefixValue+k.Slug+suffixText)
	if err != nil {
		return err //nolint:wrapcheck // the caller answers 400
	}

	f.value = value

	return nil
}

// validate checks what the screen can check before the model: the name, the
// type, the value given (required for a new key or a new type) and usable by
// the tasks, as the model reads it (#736): an AES key in base64 of 16, 24 or
// 32 bytes, an HMAC secret in base64, a PGP key readable, of the right kind
// and not locked.
func (f *formData) validate(translate gui.Translator) bool {
	if f.Name.Value == "" {
		f.Name.Error = translate("keys.form.error.required")
		f.HasErrors = true
	}

	kind, ok := kindOf(f.Type)
	if !ok {
		f.TypeErr = translate("keys.form.error.type")
		f.HasErrors = true

		return false
	}

	switch {
	case f.value == "" && (f.stored == nil || f.stored.Type != f.Type):
		f.errors[kind.Key] = translate("keys.form.error.required")
		f.HasErrors = true
	case f.value == "":
	case kind.Key == model.CryptoKeyTypeAES:
		if info := aesInfo(f.value); !info.Readable {
			f.errors[kind.Key] = translate("keys.form.error.aes")
			f.HasErrors = true
		}
	case kind.Key == model.CryptoKeyTypeHMAC:
		if info := hmacInfo(f.value); !info.Readable {
			f.errors[kind.Key] = translate("keys.form.error.hmac")
			f.HasErrors = true
		}
	case kind.PGP:
		// The dates do not matter here: an expired key is still saved.
		if info := pgpInfo(kind.Key, f.value, time.Time{}); !info.Readable {
			f.errors[kind.Key] = translate(errorKeyPrefix + info.Cause)
			f.HasErrors = true

			if info.Cause == causeNotPublic {
				delete(f.texts, kind.Key) // a private key pasted as a public one is a secret
			}
		}
	}

	return !f.HasErrors
}

// apply writes the validated form into the key: an empty value keeps the
// stored one (same type).
func (f *formData) apply(k *model.CryptoKey) {
	k.Name = f.Name.Value
	k.Type = f.Type

	if f.value != "" {
		k.Key = f.value
	}
}

// refuse marks the form with the refusal of the model: the known causes go
// to their field, the others to the banner.
func (f *formData) refuse(translate gui.Translator, err error) {
	msg := err.Error()

	switch {
	case strings.Contains(msg, causeTaken):
		f.Name.Error = translate("keys.form.error.name_taken")
	case strings.Contains(msg, causeAESSize):
		f.errors[f.Type] = translate("keys.form.error.aes")
	case strings.Contains(msg, causeNotPriv):
		f.errors[f.Type] = translate(errorKeyPrefix + causeNotPrivate)
	case strings.Contains(msg, causeNotPub):
		f.errors[f.Type] = translate(errorKeyPrefix + causeNotPublic)
		delete(f.texts, f.Type)
	case strings.Contains(msg, causeUnknown):
		f.TypeErr = translate("keys.form.error.type")
	default:
		f.Refused = msg
	}

	f.HasErrors = true
}

// prepare builds what the templates render: the type choices, the value
// block of every type with what is in place (current: the summary of the
// stored value, never the value), the summary of the value card.
func (f *formData) prepare(translate gui.Translator, current string) {
	f.Types = make([]choice, 0, len(kinds))
	f.Values = make([]valueView, 0, len(kinds))

	for i := range kinds {
		k := &kinds[i]
		f.Types = append(f.Types, choice{Value: k.Key, Label: translate(typeKeyPrefix + k.Key), Selected: f.Type == k.Key})

		view := valueView{
			Key: k.Key, Slug: k.Slug, PGP: k.PGP, Accept: pgpAccept,
			Text:  f.texts[k.Key],
			Error: f.errors[k.Key],
		}

		if f.stored != nil && f.stored.Type == k.Key {
			view.Current = current
		}

		f.Values = append(f.Values, view)
	}

	f.ValueSummary = translate("keys.form.to_give")
	if f.Edit && current != "" {
		f.ValueSummary = current
	}
}

// render shows the form, with the summary of what is in place: the size of
// an AES key, "secret set", the identity and key ID of a PGP key.
func (h *Handlers) render(w http.ResponseWriter, r *http.Request, form *formData, status int) {
	translate := h.shell.Translator(r)

	var current string

	if form.stored != nil {
		e := entryOf(form.stored, h.now(), nil)
		current = e.summary(translate)
	}

	form.prepare(translate, current)
	h.shell.Render(w, h.logger, r, h.screen(translate, form.Title, ret(r), status), formView(form))
}

// NewForm serves GET /webui/keys/new.
func (h *Handlers) NewForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, newForm(h.shell.Translator(r), nil, ret(r)), 0)
}

// Create serves POST /webui/keys/new.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	h.save(w, r, nil)
}

// EditForm serves GET /webui/keys/{id}/edit.
func (h *Handlers) EditForm(w http.ResponseWriter, r *http.Request) {
	e, ok := h.key(w, r)
	if !ok {
		return
	}

	h.render(w, r, newForm(h.shell.Translator(r), &e, ret(r)), 0)
}

// Update serves POST /webui/keys/{id}/edit.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	e, ok := h.key(w, r)
	if !ok {
		return
	}

	h.save(w, r, &e)
}

// save reads, validates and stores the posted form (multipart with a file,
// or not): a new key when e is nil, the given one otherwise. A refused form,
// by the screen or by the model, is rendered again with its errors (422),
// without the secrets typed; a saved one redirects with a notification: a
// new key shows in the list, an edited one comes back to its sheet.
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
		// A copy: a refused edition leaves the key as it was read.
		done, key := doneCreated, &model.CryptoKey{}
		if e != nil {
			stored := *e.key
			done, key = doneUpdated, &stored
		}

		form.apply(key)

		var err error
		if done == doneCreated {
			err = h.db.Insert(key).Run()
		} else {
			err = h.db.Update(key).Run()
		}

		switch {
		case err == nil:
			result := url.Values{doneParam: {done}, nameParam: {key.Name}}
			if done == doneUpdated {
				liststate.RedirectTo(w, r, withRet(keyURL(key.ID, ""), carried), result)
			} else {
				listSpec.RedirectList(w, r, carried, result)
			}

			return
		case database.IsValidationError(err):
			form.refuse(translate, err)
		default:
			h.logger.Errorf("failed to save the cryptographic key %q: %v", key.Name, err)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}
	}

	h.render(w, r, form, http.StatusUnprocessableEntity)
}

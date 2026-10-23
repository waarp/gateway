package users

import (
	"net/http"
	"net/url"
	"slices"
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/rights"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// Names of the fields of the form. The rights are checkboxes named by their
// domain ("servers"), one value per right letter ("r", "w", "d"), as the
// product writes them.
const (
	fieldLogin    = "login"
	fieldPassword = "password"
	fieldConfirm  = "confirm"

	// The limit of bcrypt: a longer password is refused by the hashing.
	maxPasswordBytes = 72
)

// field is a field of the form: its value, kept when the form is refused,
// and the message of its error. The password fields carry an error only:
// no password is ever rendered back.
type field = protoform.Field

type formData struct {
	Title     string
	Action    string
	CancelURL string
	Edit      bool
	Self      bool // the signed-in user's own account: a warning says the rights apply at once
	LastAdmin bool // and they are the last administrator: the warning says what dropping the write right means
	HasErrors bool
	// Refused is the cause given by the model when it refuses the user for
	// another reason than a field shown with its own error.
	Refused string

	Fields       map[string]field // by name
	PasswordHint string           // how the password is kept, or, on edition, that an empty one keeps it
	Rights       []rights.Row     // the checkboxes, by domain: checked when the right is held

	// The posted passwords and the validated mask, never rendered.
	password string
	confirm  string
	mask     model.PermsMask
}

// newForm builds the form of a new user, or the form of an existing one
// filled with its login and its rights.
func (h *Handlers) newForm(translate gui.Translator, user *model.User, c caller, carried string) *formData {
	form := &formData{
		Title:        translate("users.new.title"),
		Action:       withRet(common.WebuiURL(pagePath+"/new"), carried),
		CancelURL:    listURL(carried),
		Fields:       map[string]field{},
		PasswordHint: translate("users.form.password_hint"),
		Rights:       rights.Rows(translate, 0),
	}

	if user != nil {
		form.Title = translate("users.edit.title", "name", user.Username)
		form.Action = withRet(userURL(user.ID, "/edit"), carried)
		form.CancelURL = withRet(userURL(user.ID, ""), carried)
		form.Edit = true
		form.PasswordHint = translate("users.form.password_keep")
		form.Self = c.is(user)
		form.Fields[fieldLogin] = field{Value: user.Username}
		form.Rights = rights.Rows(translate, user.Permissions)
	}

	return form
}

// read takes the posted values into the form: the login, the password (kept
// aside, never rendered) and the rights.
func (f *formData) read(values url.Values) {
	f.Fields[fieldLogin] = field{Value: strings.TrimSpace(values.Get(fieldLogin))}
	f.password = values.Get(fieldPassword)
	f.confirm = values.Get(fieldConfirm)

	for i := range f.Rights {
		posted := values[f.Rights[i].Key]
		f.Rights[i].Read = slices.Contains(posted, "r")
		f.Rights[i].Write = slices.Contains(posted, "w")
		f.Rights[i].Delete = !f.Rights[i].NoDelete && slices.Contains(posted, "d")
	}
}

// validate checks the login, the password (required on creation, entered
// twice, within the limit of bcrypt) and builds the mask of the rights. It
// marks the errors on the form and returns whether it is valid.
func (f *formData) validate(translate gui.Translator) bool {
	fail := func(name, key string) {
		fld := f.Fields[name]
		fld.Error = translate(key)
		f.Fields[name] = fld
		f.HasErrors = true
	}

	if f.Fields[fieldLogin].Value == "" {
		fail(fieldLogin, "users.form.error.required")
	}

	switch {
	case f.password == "" && !f.Edit:
		fail(fieldPassword, "users.form.error.password_required")
	case len(f.password) > maxPasswordBytes:
		fail(fieldPassword, "users.form.error.too_long")
	case f.password != f.confirm:
		fail(fieldConfirm, "users.form.error.mismatch")
	}

	f.mask = 0

	for i, d := range rights.Domains {
		row := f.Rights[i]
		if row.Read {
			f.mask |= d.Read
		}

		if row.Write {
			f.mask |= d.Write
		}

		if row.Delete {
			f.mask |= d.Delete
		}
	}

	return !f.HasErrors
}

// apply writes the validated form into the user: the login, the password
// when one was given (an empty one keeps the current hash), the rights.
func (f *formData) apply(user *model.User) error {
	user.Username = f.Fields[fieldLogin].Value
	user.Permissions = f.mask

	if f.password != "" {
		hash, err := utils.HashPassword(f.password)
		if err != nil {
			return err //nolint:wrapcheck // logged by the caller with its context
		}

		user.PasswordHash = hash
	}

	return nil
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
		fail(fieldLogin, "users.form.error.login_taken")
	case strings.Contains(msg, "password cannot be empty"):
		fail(fieldPassword, "users.form.error.password_required")
	default:
		f.Refused = msg
	}

	f.HasErrors = true
}

// formPage renders the form, with the warning of the signed-in user's own
// account.
func (h *Handlers) formPage(w http.ResponseWriter, r *http.Request, form *formData, user *model.User, status int) {
	translate := h.shell.Translator(r)

	if form.Self {
		last, err := h.lastAdmin(user)
		if err != nil {
			h.logger.Errorf("failed to count the administrators: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}

		form.LastAdmin = last
	}

	h.shell.Render(w, h.logger, r, h.screen(translate, form.Title, ret(r), status), formView(form))
}

// NewForm serves GET /webui/users/new.
func (h *Handlers) NewForm(w http.ResponseWriter, r *http.Request) {
	h.formPage(w, r, h.newForm(h.shell.Translator(r), nil, callerOf(r), ret(r)), nil, 0)
}

// Create serves POST /webui/users/new.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	h.save(w, r, nil)
}

// EditForm serves GET /webui/users/{id}/edit.
func (h *Handlers) EditForm(w http.ResponseWriter, r *http.Request) {
	user, ok := h.user(w, r)
	if !ok {
		return
	}

	h.formPage(w, r, h.newForm(h.shell.Translator(r), user, callerOf(r), ret(r)), user, 0)
}

// Update serves POST /webui/users/{id}/edit.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	user, ok := h.user(w, r)
	if !ok {
		return
	}

	h.save(w, r, user)
}

// save reads, validates and stores the posted form: a new user when user is
// nil, the given one otherwise. A refused form is rendered again with its
// errors (422); a saved one redirects with a notification: a new user shows
// in the list, an edited one comes back to their sheet, and the signed-in
// user who just dropped their own read right on the users lands on "My
// account".
func (h *Handlers) save(w http.ResponseWriter, r *http.Request, user *model.User) {
	translate := h.shell.Translator(r)
	carried := ret(r)
	c := callerOf(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	form := h.newForm(translate, user, c, carried)
	form.read(r.PostForm)

	if form.validate(translate) {
		done := doneUpdated
		if user == nil {
			user = &model.User{}
			done = doneCreated
		}

		if err := form.apply(user); err != nil {
			h.logger.Errorf("failed to hash the password of %q: %v", user.Username, err)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}

		var err error
		if done == doneCreated {
			err = internal.InsertUser(h.db, user)
		} else {
			err = internal.UpdateUser(h.db, user)
		}

		switch {
		case err == nil:
			h.saved(w, r, user, done, carried, form.Self)

			return
		case database.IsValidationError(err):
			form.refuse(translate, err)
		default:
			h.logger.Errorf("failed to save the user %q: %v", user.Username, err)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}
	}

	h.formPage(w, r, form, user, http.StatusUnprocessableEntity)
}

// saved redirects after a saved form, with the result of the action.
func (h *Handlers) saved(w http.ResponseWriter, r *http.Request, user *model.User, done, carried string, self bool) {
	result := url.Values{doneParam: {done}, nameParam: {user.Username}}

	switch {
	case self && !user.HasPermission(model.PermUsersRead):
		liststate.RedirectTo(w, r, common.WebuiURL(accountPath), url.Values{doneParam: {doneAccount}})
	case done == doneUpdated:
		liststate.RedirectTo(w, r, withRet(userURL(user.ID, ""), carried), result)
	default:
		listSpec.RedirectList(w, r, carried, result)
	}
}

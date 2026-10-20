// Package me serves the "My account" page of the web interface: what the
// signed-in user is (login, instance, rights, in reading) and the change of
// their own password. Any authenticated user reaches it; it only ever acts on
// the user of the session.
package me

import (
	"net/http"
	"net/url"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/rights"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	pagePath     = "/account"
	passwordPath = "/account/password"

	fieldCurrent = "current"
	fieldNew     = "new"
	fieldConfirm = "confirm"

	doneParam    = "done"
	donePassword = "password"
)

// passwordForm is the form that changes the password: no value is ever
// rendered back, only the errors.
type passwordForm struct {
	Action    string
	CancelURL string
	HasErrors bool
	Current   string // error of the field
	New       string
	Confirm   string
}

type pageData struct {
	Username string
	Instance string
	Rights   []rights.Row // the rights of the user by domain, in reading
	Form     passwordForm
}

// Handlers serves the page and the change of password.
type Handlers struct {
	db     *database.DB
	logger *log.Logger
	shell  *common.Shell
}

func New(db *database.DB, logger *log.Logger) *Handlers {
	return &Handlers{db: db, logger: logger, shell: common.NewShell()}
}

func (h *Handlers) data(translate gui.Translator, user *model.User) pageData {
	return pageData{
		Username: user.Username,
		Instance: user.Owner,
		Rights:   rights.Rows(translate, user.Permissions),
		Form:     passwordForm{Action: common.WebuiURL(passwordPath), CancelURL: common.WebuiURL(pagePath)},
	}
}

func (h *Handlers) render(w http.ResponseWriter, r *http.Request, data *pageData, status int, notices []gui.Notice) {
	translate := h.shell.Translator(r)

	h.shell.Render(w, h.logger, r, common.Screen{
		Title: translate("me.title"),
		Crumbs: []gui.Crumb{
			{Label: translate("home.title"), URL: common.WebuiURL("/home")},
			{Label: translate("me.title")},
		},
		Status:  status,
		Notices: notices,
	}, page(data))
}

// notices turns the result of the previous action into a notification.
func notices(translate gui.Translator, v url.Values) []gui.Notice {
	if v.Get(doneParam) == donePassword {
		return []gui.Notice{{Kind: "ok", Title: translate("me.toast.password")}}
	}

	return nil
}

// Page serves GET /webui/account.
func (h *Handlers) Page(w http.ResponseWriter, r *http.Request) {
	translate := h.shell.Translator(r)
	data := h.data(translate, common.GetUser(r))
	h.render(w, r, &data, 0, notices(translate, r.URL.Query()))
}

// ChangePassword serves POST /webui/account/password: the current password
// must match, the new one must be given twice. A refused form comes back with
// its errors and without any value (422); a changed password redirects to the
// page with a notification.
func (h *Handlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	translate := h.shell.Translator(r)
	user := common.GetUser(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	data := h.data(translate, user)
	current, next, confirm := r.PostForm.Get(fieldCurrent), r.PostForm.Get(fieldNew), r.PostForm.Get(fieldConfirm)

	if !utils.CheckPassword(utils.BcryptRounds, user.PasswordHash, current) {
		data.Form.Current = translate("me.error.current")
	}

	switch {
	case next == "":
		data.Form.New = translate("me.error.required")
	case next != confirm:
		data.Form.Confirm = translate("me.error.mismatch")
	}

	if data.Form.Current != "" || data.Form.New != "" || data.Form.Confirm != "" {
		data.Form.HasErrors = true
		h.render(w, r, &data, http.StatusUnprocessableEntity, nil)

		return
	}

	hash, hashErr := utils.HashPassword(next)
	if hashErr != nil {
		h.logger.Errorf("failed to hash the new password of %q: %v", user.Username, hashErr)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	// The user of the session, read again so that nothing else than the
	// password changes.
	var dbUser model.User
	if getErr := h.db.Get(&dbUser, "id=?", user.ID).Run(); getErr != nil {
		h.logger.Errorf("failed to retrieve the user %q: %v", user.Username, getErr)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	dbUser.PasswordHash = hash

	if updateErr := h.db.Update(&dbUser).Run(); updateErr != nil {
		h.logger.Errorf("failed to change the password of %q: %v", user.Username, updateErr)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	http.Redirect(w, r, common.WebuiURL(pagePath)+"?"+doneParam+"="+donePassword, http.StatusSeeOther)
}

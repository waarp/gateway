package servers

// The authentication means of a server (its identity) or of a local account
// (what the partner presents): the "Authentication" tab, the page of a means,
// the form that adds or edits one and the confirmation of a deletion are the
// shared pages of the means (handlers/means); the screen resolves the owner
// from the route and hands it over.

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/means"
)

// credPage resolves the owner of the route and describes it to the shared
// pages; it returns nil when the "not found" page was rendered.
func (h *Handlers) credPage(w http.ResponseWriter, r *http.Request) *means.Owner {
	own, ok := h.owner(w, r)
	if !ok {
		return nil
	}

	return h.meansOwner(h.shell.Translator(r), own, ret(r), rightsOf(r))
}

// Auth serves GET /webui/servers/{id}/auth and GET
// /webui/servers/{id}/accounts/{aid}/auth: the means of the owner.
func (h *Handlers) Auth(w http.ResponseWriter, r *http.Request) {
	if own := h.credPage(w, r); own != nil {
		h.means.Auth(w, r, own, notices(h.shell.Translator(r), r.URL.Query()))
	}
}

// NewCredForm serves GET …/auth/new.
func (h *Handlers) NewCredForm(w http.ResponseWriter, r *http.Request) {
	if own := h.credPage(w, r); own != nil {
		h.means.NewForm(w, r, own)
	}
}

// CreateCred serves POST …/auth/new.
func (h *Handlers) CreateCred(w http.ResponseWriter, r *http.Request) {
	if own := h.credPage(w, r); own != nil {
		h.means.Create(w, r, own)
	}
}

// EditCredForm serves GET …/auth/{cid}/edit.
func (h *Handlers) EditCredForm(w http.ResponseWriter, r *http.Request) {
	if own := h.credPage(w, r); own != nil {
		h.means.EditForm(w, r, own)
	}
}

// UpdateCred serves POST …/auth/{cid}/edit.
func (h *Handlers) UpdateCred(w http.ResponseWriter, r *http.Request) {
	if own := h.credPage(w, r); own != nil {
		h.means.Update(w, r, own)
	}
}

// CredDetail serves GET …/auth/{cid}.
func (h *Handlers) CredDetail(w http.ResponseWriter, r *http.Request) {
	if own := h.credPage(w, r); own != nil {
		h.means.Detail(w, r, own)
	}
}

// DeleteCredForm serves GET …/auth/{cid}/delete: the confirmation page.
func (h *Handlers) DeleteCredForm(w http.ResponseWriter, r *http.Request) {
	if own := h.credPage(w, r); own != nil {
		h.means.DeleteForm(w, r, own)
	}
}

// DeleteCred serves POST …/auth/{cid}/delete.
func (h *Handlers) DeleteCred(w http.ResponseWriter, r *http.Request) {
	if own := h.credPage(w, r); own != nil {
		h.means.Delete(w, r, own)
	}
}

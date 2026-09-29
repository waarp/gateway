package gui

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
)

// redirectToParentList is used by the pages showing the elements of a parent
// object (partner, server, rule) when that object cannot be loaded from the
// request: it logs the request and sends the user back to the parent list page.
func redirectToParentList(w http.ResponseWriter, r *http.Request, logger *log.Logger,
	parent, id, listPage string,
) {
	logger.Warningf("Page %q requested for an unknown %s %q, redirecting to %q",
		r.URL.Path, parent, id, listPage)
	http.Redirect(w, r, listPage, http.StatusSeeOther)
}

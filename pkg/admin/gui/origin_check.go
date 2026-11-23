package gui

import (
	"net/http"

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
)

const (
	// The message of a request refused for its origin, among the messages
	// of the refusals of the web interface.
	crossOriginKey = "cross_origin"
	// The length kept of each value a refused request sent, when it is
	// written to the log: such a request needs no session.
	loggedValueLength = 200
)

// logged returns a value received from a request as it is written to the
// log: its start, when it is longer than the length kept.
func logged(value string) string {
	if len(value) <= loggedValueLength {
		return value
	}

	return value[:loggedValueLength] + "[...]"
}

// originCheck refuses the requests that may change state (every method but
// GET, HEAD and OPTIONS) when they come from another origin than the web
// interface itself: the browser says so (Sec-Fetch-Site), or it names in
// Origin another host than the one the request is sent to (Host). A request
// that carries neither header does not come from a browser, and goes
// through: the check completes the session cookie, it does not replace the
// authentication.
//
// A refusal leaves a warning in the log with what was received, and tells
// the user the likely cause when the interface itself is refused: a reverse
// proxy that does not forward the Host header as it received it.
func originCheck(logger *log.Logger) mux.MiddlewareFunc {
	messages := locale.ParseLocalizationFile(locale.ErrorsFile)[crossOriginKey]

	protection := http.NewCrossOriginProtection()
	protection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Warningf("Refused the request %s %q, which comes from another origin: Origin %q, Host %q, "+
			"Sec-Fetch-Site %q. Behind a reverse proxy, the Host header must reach the gateway as the browser sent it.",
			r.Method, logged(r.URL.Path), logged(r.Header.Get("Origin")), logged(r.Host),
			logged(r.Header.Get("Sec-Fetch-Site")))

		message := messages[locale.GetLanguage(r)]
		if message == "" {
			message = messages[English]
		}

		http.Error(w, message, http.StatusForbidden)
	}))

	return protection.Handler
}

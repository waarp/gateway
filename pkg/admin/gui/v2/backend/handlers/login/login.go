// Package login renders the sign-in page of the web interface (#787), the
// first screen seen: the banner, the username and the password with its show
// button, the button, and under it the refusal or the information about the
// session that ended. It renders only: the session itself (the check of the
// credentials, the token, the cookie, the redirections) stays in pkg/admin/gui,
// which calls this package.
package login

import (
	"code.waarp.fr/lib/gui"
	"github.com/a-h/templ"
)

const (
	// ReasonParam is the query parameter that says why the user is on the
	// page: the middleware sets it when a session ended, the sign-out when
	// the user left.
	ReasonParam = "reason"
	// ReasonExpired: a session cookie was given and refused (expired, replaced
	// by a sign-in elsewhere, lost at a restart).
	ReasonExpired = "expired"
	// ReasonSignedOut: the user signed out.
	ReasonSignedOut = "signed_out"
)

// Data feeds the page: the username kept after a refusal (the password never
// is), the refusal (an alert under the button) or the information about the
// session (a status under the button), both already translated.
type Data struct {
	Username string
	Error    string
	Info     string
}

// Page is the component of the sign-in page, rendered in the plain page of
// the kit.
func Page(d Data) templ.Component { return page(d) }

// Title is the title of the page, which the button repeats.
func Title(translate gui.Translator) string { return translate("login.title") }

// Refusal is the sentence shown when the sign-in fails: the same one for a
// wrong username, a wrong password and a locked account (nothing tells
// whether the account exists nor whether it is locked), another one when the
// check itself could not be done (the cause stays in the log).
func Refusal(translate gui.Translator, credentials bool) string {
	if credentials {
		return translate("login.refused")
	}

	return translate("login.unavailable")
}

// Info translates the reason the user is on the page into the sentence shown
// under the button; empty for no reason or an unknown one.
func Info(translate gui.Translator, reason string) string {
	switch reason {
	case ReasonExpired:
		return translate("login.expired")
	case ReasonSignedOut:
		return translate("login.signed_out")
	default:
		return ""
	}
}

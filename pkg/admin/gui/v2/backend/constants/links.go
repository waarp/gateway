package constants

import (
	"path"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

//nolint:gochecknoglobals //cannot be constant: must be changed at runtime
var (
	AppName     = "Waarp Gateway"
	DocLinkHome = "https://doc.waarp.org/waarp-gateway/"
)

const (
	WebuiPrefix  = "/webui"
	StaticPrefix = "/static_v2/"
	// KitStaticPrefix is where the files of the GUI kit (styles, scripts,
	// fonts, icons) are served, under WebuiPrefix.
	KitStaticPrefix = "/static/gui/"

	// LanguageParam is the query parameter that selects the language of the
	// interface; the language middleware then stores it in a cookie.
	LanguageParam = "language"
	// FragmentParam is the query parameter that asks a screen for one of its
	// blocks alone, for an update without reload. Such a request is made by
	// the page itself, not by the user: it does not extend the session.
	FragmentParam = "fragment"
)

//nolint:staticcheck //remove once the doc has been translated
func DocLink(language string) string {
	language = "fr"

	num := version.Num
	if num == "dev" {
		num = "latest"
	}

	return DocLinkHome + path.Join(num, language) + "/"
}

// DocPage returns the URL of a page of the documentation, given relative to
// the guide ("user/webui/etat-des-services"), in the form the documentation
// site serves (a directory, with a trailing slash).
func DocPage(language, page string) string {
	return DocLink(language) + strings.Trim(page, "/") + "/"
}

// DocLatest is the home of the latest published documentation, whatever the
// version of the gateway: where the header of the web interface leads
// (Driss, 2026-10-19), in the only language the documentation exists in.
func DocLatest() string {
	return DocLinkHome + "latest/fr/"
}

package common

import (
	"bytes"
	"net/http"

	"code.waarp.fr/lib/gui"
	"github.com/a-h/templ"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

const productSlug = "gateway"

// Shell renders the screens built on the GUI kit: the kit chrome (header,
// sidebar, breadcrumb) around the component of a screen, in the language of
// the request. A handler creates its shell once, at initialisation, and uses
// it for every request.
type Shell struct {
	catalog locale.Catalog
}

// NewShell loads the translation catalogue. As for the other v2
// initialisations, a loading error is a programming error and panics.
func NewShell() *Shell {
	return &Shell{catalog: locale.MustLoadCatalog()}
}

// Options are the options of the kit for the web interface: where the pages
// live and where the static files of the kit are served.
func Options() gui.Options {
	return gui.Options{BasePath: constants.WebuiPrefix, StaticPath: constants.KitStaticPrefix}
}

// Catalog returns the translation catalogue.
func (s *Shell) Catalog() locale.Catalog { return s.catalog }

// Translator returns the catalogue bound to the language of the request.
func (s *Shell) Translator(r *http.Request) gui.Translator {
	return s.catalog.Translator(locale.GetLanguage(r))
}

// Screen is what a handler gives the shell to render a page.
type Screen struct {
	Title   string      // translated title of the screen
	Section string      // active section of the sidebar (constants.SidebarSection*)
	Menu    string      // active entry of the sidebar (constants.SidebarMenu*)
	Doc     string      // documentation page of the screen, relative to the guide
	Crumbs  []gui.Crumb // breadcrumb, the last item being the screen
	// Stylesheets and Scripts are the files of the screen, as paths under the
	// web interface ("/static_v2/services/services.css"), linked in <head>
	// and at the end of <body>.
	Stylesheets []string
	Scripts     []string
	// Editor loads the rich text field and the variables list of the kit
	// (gui.EditorAssets), for the screens that use them.
	Editor bool
	// Status is the HTTP status of the page: 0 (200 OK) for a page, 422 for a
	// form rendered again with its errors, 404 for a localised "not found".
	Status int
	// Notices are the results of the previous action, rendered as
	// notifications when the page loads (Post/Redirect/Get).
	Notices []gui.Notice
}

// Render writes the complete page: the kit chrome, filled from the request
// (user, language, rights), around the component of the screen.
//
//nolint:gocritic // hugeParam: one screen per request, built and consumed at once
func (s *Shell) Render(w http.ResponseWriter, logger *log.Logger, r *http.Request,
	screen Screen, body templ.Component,
) {
	s.execute(w, logger, r, gui.Shell(s.makePage(r, screen), body), screen.Status)
}

// RenderFragment writes one component of the screen alone, for the updates
// without reload (?fragment=...).
func (s *Shell) RenderFragment(w http.ResponseWriter, logger *log.Logger, r *http.Request, fragment templ.Component) {
	s.execute(w, logger, r, fragment, 0)
}

// execute renders a component in memory, with the translator of the request
// and the options of the web interface bound, then writes it with the given
// HTTP status (0 means 200 OK).
func (s *Shell) execute(w http.ResponseWriter, logger *log.Logger, r *http.Request, c templ.Component, status int) {
	// Rendered in memory first, so that a rendering error yields a clean
	// error response instead of a truncated page.
	var page bytes.Buffer
	if err := gui.Render(r.Context(), &page, s.Translator(r), Options(), c); err != nil {
		logger.Errorf("failed to render the page: %v", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if status != 0 {
		w.WriteHeader(status)
	}

	if _, writeErr := page.WriteTo(w); writeErr != nil {
		logger.Warningf("failed to write the page: %v", writeErr)
	}
}

//nolint:gocritic // hugeParam: see Render
func (s *Shell) makePage(r *http.Request, screen Screen) gui.Page {
	user := GetUser(r)
	lang := locale.GetLanguage(r)
	translate := s.catalog.Translator(lang)

	return gui.Page{
		Lang:        lang,
		Title:       screen.Title,
		Product:     constants.AppName,
		ProductSlug: productSlug,
		Instance:    user.Owner,
		Username:    user.Username,
		// The version alone: the revision stays on the About page (2026-10-20).
		Footer: translate("nav.footer", "product", constants.AppName, "version", version.Num),
		// The header leads to the latest documentation, not to the page of
		// the screen: the pages of the web interface guide are not published
		// yet (Driss, 2026-10-19); Screen.Doc is kept for that day.
		DocURL:     constants.DocLatest(),
		AccountURL: WebuiURL("/account"),
		AboutURL:   WebuiURL("/about"),
		About:      About(translate),
		LogoutURL:  WebuiURL("/logout"),
		HomeURL:    WebuiURL("/home"),
		Languages:  languages(r, lang),
		Nav:        makeNav(translate, user.Permissions, screen.Section, screen.Menu),
		Crumbs:     screen.Crumbs,
		Notices:    screen.Notices,
		Head:       headOf(screen.Stylesheets, screen.Editor),
		Scripts:    scriptsOf(screen.Scripts),
	}
}

// headOf links the stylesheets of a screen in <head>, after the files of the
// editor of the kit when the screen uses it; nil without any.
func headOf(paths []string, editor bool) templ.Component {
	switch {
	case editor:
		return templ.Join(gui.EditorAssets(), Stylesheets(paths))
	case len(paths) == 0:
		return nil
	default:
		return Stylesheets(paths)
	}
}

// scriptsOf loads the scripts of a screen at the end of <body>; nil without any.
func scriptsOf(paths []string) templ.Component {
	if len(paths) == 0 {
		return nil
	}

	return Scripts(paths)
}

// WebuiURL prefixes a path with the prefix of the web interface, as
// gui.URL does with the options bound to the context.
func WebuiURL(path string) string { return constants.WebuiPrefix + path }

// languages builds the language switcher: each entry reloads the current
// page with the language parameter, which the language middleware then
// stores in the language cookie.
func languages(r *http.Request, current string) []gui.Language {
	names := map[string]string{"fr": "Français", "en": "English"}
	switcher := make([]gui.Language, 0, len(names))

	for _, code := range locale.Languages() {
		query := r.URL.Query()
		query.Set(constants.LanguageParam, code)

		switcher = append(switcher, gui.Language{
			Code:    code,
			Label:   names[code],
			URL:     r.URL.Path + "?" + query.Encode(),
			Current: code == current,
		})
	}

	return switcher
}

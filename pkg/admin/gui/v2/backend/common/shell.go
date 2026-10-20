package common

import (
	"bytes"
	"html/template"
	"net/http"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/webfs"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

const (
	productSlug   = "gateway"
	shellTemplate = "gui/shell"
)

// Shell renders the screens built on the GUI kit: the kit chrome (header,
// sidebar, breadcrumb) around the "page" template of a screen, in the
// language of the request. A handler creates its shell once, at
// initialisation, and uses it for every request.
type Shell struct {
	tpl     *template.Template
	catalog locale.Catalog
}

// NewShell parses the kit templates, then the given screen templates read
// from the web file system. As for the other v2 initialisations, a parse
// error is a programming error and panics.
func NewShell(screenFiles ...string) *Shell {
	opts := gui.Options{BasePath: constants.WebuiPrefix, StaticPath: constants.KitStaticPrefix}

	tpl := template.Must(template.New("kit").Funcs(gui.Funcs(opts)).ParseFS(gui.Templates, gui.TemplatePatterns...))
	tpl = template.Must(tpl.ParseFS(webfs.WebFS, screenFiles...))

	return &Shell{tpl: tpl, catalog: locale.MustLoadCatalog()}
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
	Data    any         // screen data, .Data in the "page" template

	// Status is the HTTP status of the page: 0 (200 OK) for a page, 422 for a
	// form rendered again with its errors, 404 for a localised "not found".
	Status int
	// Notices are the results of the previous action, rendered as
	// notifications when the page loads (Post/Redirect/Get).
	Notices []gui.Notice
}

// Render writes the complete page: the kit chrome, filled from the request
// (user, language, rights), around the "page" template of the screen.
//
//nolint:gocritic // hugeParam: one screen per request, built and consumed at once
func (s *Shell) Render(w http.ResponseWriter, logger *log.Logger, r *http.Request, screen Screen) {
	s.execute(w, logger, r, shellTemplate, s.makePage(r, screen), screen.Status)
}

// RenderFragment writes one named template of the screen alone, for the
// updates without reload (?fragment=...).
func (s *Shell) RenderFragment(w http.ResponseWriter, logger *log.Logger, r *http.Request,
	name string, data any,
) {
	s.execute(w, logger, r, name, data, 0)
}

// execute renders a template in memory, then writes it with the given HTTP
// status (0 means 200 OK).
func (s *Shell) execute(w http.ResponseWriter, logger *log.Logger, r *http.Request, name string, data any, status int) {
	localized, err := gui.Localize(s.tpl, s.Translator(r))
	if err != nil {
		logger.Errorf("failed to localize the templates: %v", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)

		return
	}

	// Rendered in memory first, so that a template error yields a clean
	// error response instead of a truncated page.
	var page bytes.Buffer
	if execErr := localized.ExecuteTemplate(&page, name, data); execErr != nil {
		logger.Errorf("failed to render template %q: %v", name, execErr)
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
		Footer:    translate("nav.footer", "product", constants.AppName, "version", version.Num),
		DocURL:    constants.DocPage(lang, screen.Doc),
		LogoutURL: WebuiURL("/logout"),
		HomeURL:   WebuiURL("/home"),
		Languages: languages(r, lang),
		Nav:       makeNav(translate, user.Permissions, screen.Section, screen.Menu),
		Crumbs:    screen.Crumbs,
		Notices:   screen.Notices,
		Data:      screen.Data,
	}
}

// WebuiURL prefixes a path with the prefix of the web interface, as the "url"
// template helper does.
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

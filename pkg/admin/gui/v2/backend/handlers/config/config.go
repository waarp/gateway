// Package config is the "Configuration import and export" screen of the
// Administration section: one page, one form, three cards. The categories
// of the configuration are chosen once, for the export as for the import.
// The export downloads the configuration as JSON or YAML, the secrets and
// the certificates in clear or replaced by the markers of the migration
// tooling; the import reads such a file and simulates or applies it, in one
// transaction, with the options of the REST API (reset of the categories,
// restart of the services). The screen calls the backup package as the REST
// API does; the rights are those of the API: reading for the page and the
// export, writing for the import. Both actions are logged with their user.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/backup"
	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	// The stylesheet of the screen, under the web interface.
	stylesheet = "/static_v2/config/config.css"

	docPage    = "user/webui/import-export"
	pagePath   = "/config"
	exportPath = "/config/export"
	importPath = "/config/import"

	// The bound of a configuration file. The full export of the demonstration
	// instance weighs 52 KB (2026-10-24): a sizeable configuration stays far
	// under this bound, a wrong file does not.
	maxUpload = 64 << 20
	// What the multipart envelope and the other fields add to the file in the
	// request body.
	formOverhead = 64 << 10
	// What the multipart parser keeps in memory before it spills a file to
	// disk.
	formMemory = 1 << 20

	fieldTarget       = "target"
	fieldFormat       = "format"
	fieldRedact       = "redact"
	fieldFile         = "file"
	fieldRestart      = "restart"
	fieldReset        = "reset"
	fieldResetConfirm = "reset_confirm"
	fieldAction       = "action"

	actionSimulate = "simulate"
	actionImport   = "import"

	formatJSON = "json"
	formatYAML = "yaml"

	// The result of the previous action, carried by the redirection.
	doneParam       = liststate.DoneParam
	categoriesParam = "categories"
	restartedParam  = "restarted"
	doneSimulated   = "simulated"
	doneImported    = "imported"

	stateOK = "ok"

	// The start of the name of a downloaded file: the instance and the time
	// follow, then "-markers" when the secrets are replaced.
	filePrefix    = "waarp-gateway"
	markersSuffix = "-markers"
	timeLayout    = "20060102-150405"
)

// categories are the parts of the configuration, named as the REST API
// names its targets, in the order of the sidebar. The labels are the
// catalogue keys "config.categories.<name>".
//
//nolint:gochecknoglobals // reference data of the screen
var categories = []string{
	"servers", "partners", "clients", "rules", "users", "clouds",
	"snmp", "authorities", "keys", "email", "filewatchers",
}

// fileTypes are the extensions of a configuration file, as the backup
// package reads them: YAML for .yaml and .yml, JSON otherwise.
//
//nolint:gochecknoglobals // reference data of the screen
var fileTypes = []string{".json", ".yaml", ".yml"}

// unsafeInName matches what a file name must not carry: the name of the
// instance is written by the operator.
var unsafeInName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

type Handlers struct {
	db     *database.DB
	logger *log.Logger
	shell  *common.Shell
}

func New(db *database.DB, logger *log.Logger) *Handlers {
	return &Handlers{db: db, logger: logger, shell: common.NewShell()}
}

// category is a part of the configuration as a checkbox of the first card.
type category struct {
	Key     string
	Label   string
	Checked bool
}

// choice is a format of the export, one of two joined buttons.
type choice struct {
	Value    string
	Label    string
	Selected bool
}

// importForm is the card of the import: its options, their errors, and the
// refusal of the product when the simulation or the import failed.
type importForm struct {
	Restart      bool
	Reset        bool
	ResetConfirm bool
	FileError    string
	ConfirmError string
	// Refused is the message of the product when it refused the file or the
	// import: shown as is, as the REST API answers it.
	Refused string
}

// pageData is the page: the categories, the export card, the import card,
// the rights.
type pageData struct {
	ExportURL  string
	ImportURL  string
	Write      bool
	Categories []category
	// CategoriesError says that no category is chosen, for the export as for
	// the import.
	CategoriesError string
	Formats         []choice
	Redact          bool
	// ExportError says why the export did not start: the product failed to
	// build the file.
	ExportError string
	Import      importForm
}

// Page renders the screen, with the result of the previous action.
func (h *Handlers) Page(w http.ResponseWriter, r *http.Request) {
	translate := h.shell.Translator(r)
	screen := h.screen(translate, 0)
	screen.Notices = notices(translate, r.URL.Query())

	h.shell.Render(w, h.logger, r, screen, pageView(h.newPage(translate, r)))
}

// newPage is the page as it opens: every category checked, JSON, the
// secrets in clear, nothing asked of the import.
func (h *Handlers) newPage(translate gui.Translator, r *http.Request) pageData {
	return pageData{
		ExportURL:  common.WebuiURL(exportPath),
		ImportURL:  common.WebuiURL(importPath),
		Write:      common.GetUser(r).Permissions.HasPermission(model.PermAdminWrite),
		Categories: allCategories(translate, nil),
		Formats:    formats(translate, formatJSON),
	}
}

// Export downloads the configuration of the chosen categories, in the
// chosen format, the secrets and the certificates in clear or replaced by
// markers. The file is built whole before it is sent, so that a failure
// renders the page with its cause instead of a cut file. The export is
// logged with its user: a file in clear carries every secret.
func (h *Handlers) Export(w http.ResponseWriter, r *http.Request) {
	translate := h.shell.Translator(r)
	query := r.URL.Query()
	targets := chosen(query[fieldTarget])
	format := formatOf(query.Get(fieldFormat))
	redact := query.Get(fieldRedact) != ""

	if len(targets) == 0 {
		data := h.newPage(translate, r)
		data.Categories = allCategories(translate, append([]string{}, query[fieldTarget]...))
		data.Formats = formats(translate, format)
		data.Redact = redact
		data.CategoriesError = translate("config.categories_none")
		h.shell.Render(w, h.logger, r, h.screen(translate, http.StatusUnprocessableEntity), pageView(data))

		return
	}

	user := common.GetUser(r)
	name := fileName(user.Owner, format, redact, time.Now())

	data, err := backup.Collect(h.db, h.logger, targets)
	if err != nil {
		h.logger.Errorf("failed to export the configuration: %v", err)
		h.exportFailed(w, r, translate)

		return
	}

	if redact {
		redactSecrets(data)
	}

	var buf bytes.Buffer
	if serr := backup.Serialize(data, namedWriter{&buf, name}); serr != nil {
		h.logger.Errorf("failed to export the configuration: %v", serr)
		h.exportFailed(w, r, translate)

		return
	}

	if redact {
		h.logger.Infof("Configuration exported by user %q with the secrets and the certificates "+
			"replaced by markers: %s", user.Username, strings.Join(targets, ", "))
	} else {
		h.logger.Warningf("Configuration exported in clear by user %q, secrets included: %s",
			user.Username, strings.Join(targets, ", "))
	}

	w.Header().Set("Content-Type", mimeOf(format))
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(http.StatusOK)

	if _, werr := w.Write(buf.Bytes()); werr != nil {
		h.logger.Warningf("failed to send the exported configuration: %v", werr)
	}
}

// exportFailed renders the page when the product could not build the file.
func (h *Handlers) exportFailed(w http.ResponseWriter, r *http.Request, translate gui.Translator) {
	data := h.newPage(translate, r)
	data.ExportError = translate("config.export.failed")
	h.shell.Render(w, h.logger, r, h.screen(translate, http.StatusInternalServerError), pageView(data))
}

// Import reads the uploaded file and the options, then simulates or applies
// the import; the page comes back with its errors (422), or the screen
// loads again with the result (Post/Redirect/Get). A file that cannot be
// read is an error of the form, never a failure of the server; a file that
// still carries markers is refused. The import is logged with its user.
func (h *Handlers) Import(w http.ResponseWriter, r *http.Request) {
	translate := h.shell.Translator(r)
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload+formOverhead)

	data, posted, ok := h.readImport(translate, r)
	if !ok {
		h.refuse(w, r, translate, &posted)

		return
	}

	targets := chosen(r.Form[fieldTarget])
	dry := r.FormValue(fieldAction) != actionImport
	restart := posted.Import.Restart && !dry

	if err := backup.Import(h.db, h.logger, data, targets, dry, posted.Import.Reset, restart); err != nil {
		posted.Import.Refused = err.Error()
		h.refuse(w, r, translate, &posted)

		return
	}

	done, verb := doneImported, "applied"
	if dry {
		done, verb = doneSimulated, "simulated"
	}

	h.logger.Infof("Configuration import %s by user %q: %s (reset %t, restart %t)",
		verb, common.GetUser(r).Username, strings.Join(targets, ", "), posted.Import.Reset, restart)

	result := url.Values{doneParam: {done}, categoriesParam: {labels(translate, targets)}}
	if restart {
		result.Set(restartedParam, "1")
	}

	liststate.RedirectTo(w, r, common.WebuiURL(pagePath), result)
}

// readImport takes the posted fields and the file into the page, and parses
// the file; false, with the page carrying its errors, when something is
// missing or wrong.
func (h *Handlers) readImport(translate gui.Translator, r *http.Request) (*file.Data, pageData, bool) {
	posted := h.newPage(translate, r)

	if err := r.ParseMultipartForm(formMemory); err != nil {
		var tooBig *http.MaxBytesError

		switch {
		case errors.As(err, &tooBig):
			posted.Import.FileError = translate("config.import.file_size")
		case errors.Is(err, http.ErrNotMultipart), errors.Is(err, http.ErrMissingBoundary):
			posted.Import.FileError = translate("config.import.file_missing")
		default:
			posted.Import.FileError = translate("config.import.file_read")
		}

		return nil, posted, false
	}

	// Posted without any category is not the page as it opens: nothing
	// checked.
	posted.Categories = allCategories(translate, append([]string{}, r.Form[fieldTarget]...))
	posted.Import.Restart = r.FormValue(fieldRestart) != ""
	posted.Import.Reset = r.FormValue(fieldReset) != ""
	posted.Import.ResetConfirm = r.FormValue(fieldResetConfirm) != ""

	if len(chosen(r.Form[fieldTarget])) == 0 {
		posted.CategoriesError = translate("config.categories_none")
	}

	if posted.Import.Reset && !posted.Import.ResetConfirm {
		posted.Import.ConfirmError = translate("config.import.confirm_required")
	}

	content, name, ok := h.readFile(translate, r, &posted.Import)
	if !ok || posted.CategoriesError != "" || posted.Import.ConfirmError != "" {
		return nil, posted, false
	}

	data, err := backup.ParseData(namedReader{bytes.NewReader(content), name})
	if err != nil {
		posted.Import.FileError = translate("config.import.unreadable")
		posted.Import.Refused = err.Error()

		return nil, posted, false
	}

	return data, posted, true
}

// readFile takes the uploaded file: present, of a known type, within the
// bound, and free of the markers of a redacted export.
func (h *Handlers) readFile(translate gui.Translator, r *http.Request, form *importForm) ([]byte, string, bool) {
	upload, header, err := r.FormFile(fieldFile)
	if err != nil {
		form.FileError = translate("config.import.file_missing")
		if !errors.Is(err, http.ErrMissingFile) {
			form.FileError = translate("config.import.file_read")
		}

		return nil, "", false
	}
	defer upload.Close() //nolint:errcheck // read only

	if !slices.Contains(fileTypes, strings.ToLower(filepath.Ext(header.Filename))) {
		form.FileError = translate("config.import.file_type")

		return nil, "", false
	}

	content, err := io.ReadAll(io.LimitReader(upload, maxUpload+1))

	switch {
	case err != nil:
		form.FileError = translate("config.import.file_read")
	case len(content) > maxUpload:
		form.FileError = translate("config.import.file_size")
	case hasMarkers(content):
		form.FileError = translate("config.import.markers")
	}

	if form.FileError != "" {
		return nil, "", false
	}

	return content, header.Filename, true
}

// refuse renders the page again as posted, with its errors.
func (h *Handlers) refuse(w http.ResponseWriter, r *http.Request, translate gui.Translator, posted *pageData) {
	h.shell.Render(w, h.logger, r, h.screen(translate, http.StatusUnprocessableEntity), pageView(*posted))
}

// screen is the page of the screen: the chrome around the cards.
func (h *Handlers) screen(translate gui.Translator, status int) common.Screen {
	return common.Screen{
		Title:   translate("config.title"),
		Section: constants.SidebarSectionAdministration,
		Menu:    constants.SidebarMenuConfig,
		Doc:     docPage,
		Crumbs: []gui.Crumb{
			{Label: translate("nav.home"), URL: common.WebuiURL("/home")},
			{Label: translate("nav.administration")},
			{Label: translate("config.title")},
		},
		Status:      status,
		Stylesheets: []string{stylesheet},
	}
}

// notices turns the result of the previous action into a notification: the
// simulation passed, or the configuration is imported, with the categories
// and, when asked, the restart of the services.
func notices(translate gui.Translator, values url.Values) []gui.Notice {
	done := values.Get(doneParam)
	if done != doneSimulated && done != doneImported {
		return nil
	}

	notice := gui.Notice{Kind: stateOK, Title: translate("config.import."+done, "categories", values.Get(categoriesParam))}
	if values.Get(restartedParam) != "" {
		notice.Text = translate("config.import.restarted")
	}

	return []gui.Notice{notice}
}

// allCategories is every category as a checkbox: all checked when nothing
// was posted (the page opens so), else the posted ones.
func allCategories(translate gui.Translator, posted []string) []category {
	list := make([]category, 0, len(categories))
	for _, key := range categories {
		list = append(list, category{
			Key:     key,
			Label:   translate("config.categories." + key),
			Checked: posted == nil || slices.Contains(posted, key),
		})
	}

	return list
}

// chosen keeps the posted categories the screen knows, in the order of the
// sidebar: an unknown one is dropped, as the REST API ignores it.
func chosen(posted []string) []string {
	kept := make([]string, 0, len(categories))
	for _, key := range categories {
		if slices.Contains(posted, key) {
			kept = append(kept, key)
		}
	}

	return kept
}

// labels names the categories for a notification.
func labels(translate gui.Translator, targets []string) string {
	names := make([]string, 0, len(targets))
	for _, key := range targets {
		names = append(names, translate("config.categories."+key))
	}

	return strings.Join(names, ", ")
}

// formats are the two formats of the export, the chosen one selected.
func formats(translate gui.Translator, format string) []choice {
	return []choice{
		{Value: formatJSON, Label: translate("config.export.json"), Selected: format == formatJSON},
		{Value: formatYAML, Label: translate("config.export.yaml"), Selected: format == formatYAML},
	}
}

// formatOf reads the posted format, JSON unless YAML is asked.
func formatOf(posted string) string {
	if posted == formatYAML {
		return formatYAML
	}

	return formatJSON
}

// mimeOf is the content type of a downloaded file.
func mimeOf(format string) string {
	if format == formatYAML {
		return "application/yaml; charset=utf-8"
	}

	return "application/json; charset=utf-8"
}

// fileName names a downloaded file after the instance and the time:
// "waarp-gateway-<instance>-<date>-<time>[-markers].<format>".
func fileName(instance, format string, redact bool, now time.Time) string {
	instance = strings.Trim(unsafeInName.ReplaceAllString(instance, "-"), "-")
	if instance == "" {
		instance = "gateway"
	}

	suffix := ""
	if redact {
		suffix = markersSuffix
	}

	return fmt.Sprintf("%s-%s-%s%s.%s", filePrefix, instance, now.Format(timeLayout), suffix, format)
}

// namedWriter and namedReader give the backup package the name it reads the
// format from.
type namedWriter struct {
	io.Writer
	name string
}

func (n namedWriter) Name() string { return n.name }

type namedReader struct {
	io.Reader
	name string
}

func (n namedReader) Name() string { return n.name }

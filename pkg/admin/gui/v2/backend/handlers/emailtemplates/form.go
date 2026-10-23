package emailtemplates

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/means"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/ruleuse"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

// Names of the fields of the form.
const (
	fieldID          = "template_id" // the template, for the preview of an existing one
	fieldName        = "name"
	fieldFormat      = "format"
	fieldSubject     = "subject"
	fieldBodyText    = "body_text"
	fieldBodyHTML    = "body_html"
	fieldAttachments = "attachments"
	fieldBodyFile    = "body_file"

	// The size of the largest body file the form reads.
	maxBodyFile = 1 << 20

	// The files the body field offers to load.
	bodyFileAccept = ".html,.htm,.txt,text/html,text/plain"

	// The causes the model gives when it refuses a template.
	causeTaken = "already exists"
	causeMIME  = "MIME type is invalid"
)

// bodyFileTypes are the extensions of a body file, and whether the body is
// HTML.
//
//nolint:gochecknoglobals // reference data
var bodyFileTypes = map[string]bool{".html": true, ".htm": true, ".txt": false}

// tagPattern finds the tags of an HTML body, to tell whether it holds any
// text.
var tagPattern = regexp.MustCompile(`<[^>]*>|&nbsp;|\x{00a0}`)

type field = protoform.Field

type formData struct {
	Title     string
	Action    string
	CancelURL string
	Edit      bool
	HasErrors bool
	Refused   string // a refusal of the model no field carries

	// Locked: rules use the template, its name is kept (their EMAIL tasks name
	// it); Rules says how many.
	Locked bool
	Rules  string

	ID          int64
	Name        field
	Format      string        // the MIME type chosen
	Formats     []valueOption // Text, HTML, and the type the REST API set if another
	TextCases   string        // the formats whose body is a text (data-case)
	HTMLCase    string        // the format whose body is HTML
	Subject     field
	BodyText    field
	BodyHTML    field
	Attachments field
	BodyFile    field // the file the body is loaded from; only its error is kept

	Vars    gui.VarListProps
	Preview *previewData

	stored *model.EmailTemplate
}

// newForm builds the form of a new template (in HTML), or the form of an
// existing one filled with its values.
func newForm(translate gui.Translator, e *entry, carried string) *formData {
	form := &formData{
		Title:     translate("email_templates.new.title"),
		Action:    withRet(common.WebuiURL(pagePath+"/new"), carried),
		CancelURL: listURL(carried),
		Format:    mimeHTML,
		Vars:      varList(translate, time.Now()),
	}

	if e != nil {
		t := e.tpl
		form.Title = translate("email_templates.edit.title", "name", t.Name)
		form.Action = withRet(templateURL(t.ID, "/edit"), carried)
		form.CancelURL = withRet(templateURL(t.ID, ""), carried)
		form.Edit = true
		form.stored = t
		form.ID = t.ID
		form.Name = field{Value: t.Name}
		form.Format = t.MIMEType
		form.Subject = field{Value: t.Subject}
		form.Attachments = field{Value: strings.Join(t.Attachments, "\n")}
		form.Locked = len(e.uses) > 0
		form.Rules = ruleuse.Count(translate, len(e.uses))

		if form.Format == "" {
			form.Format = mimeText
		}

		if isHTML(form.Format) {
			form.BodyHTML = field{Value: t.Body}
		} else {
			form.BodyText = field{Value: t.Body}
		}
	}

	form.formats(translate)

	return form
}

// formats offers "Text" and "HTML", the MIME type in place kept as it is
// (with its parameters), and a type the REST API set as a third choice,
// whose body is a text.
func (f *formData) formats(translate gui.Translator) {
	text, htmlType, stored := mimeText, mimeHTML, ""
	if f.stored != nil {
		stored = f.stored.MIMEType
	}

	switch mediaType(stored) {
	case "":
	case mimeText:
		text = stored
	case mimeHTML:
		htmlType = stored
	default:
		f.TextCases = stored
	}

	f.TextCases = strings.TrimSpace(text + " " + f.TextCases)
	f.HTMLCase = htmlType
	f.Formats = []valueOption{
		{Value: text, Label: translate("email_templates.format.text")},
		{Value: htmlType, Label: translate("email_templates.format.html")},
	}

	if f.TextCases != text {
		f.Formats = append(f.Formats, valueOption{Value: stored, Label: stored})
	}

	for i := range f.Formats {
		f.Formats[i].Selected = f.Formats[i].Value == f.Format
	}
}

// read takes the posted values: the name (the one in place for a template
// the rules use), the format, the subject, the body of the format chosen
// and the attachments, one path per line.
func (f *formData) read(values url.Values) {
	if !f.Locked {
		f.Name = field{Value: strings.TrimSpace(values.Get(fieldName))}
	}

	if format := strings.TrimSpace(values.Get(fieldFormat)); format != "" {
		f.Format = format
	}

	f.Subject = field{Value: strings.TrimSpace(values.Get(fieldSubject))}
	f.BodyText = field{Value: values.Get(fieldBodyText)}
	f.BodyHTML = field{Value: values.Get(fieldBodyHTML)}

	var paths []string

	for line := range strings.Lines(values.Get(fieldAttachments)) {
		if p := strings.TrimSpace(line); p != "" {
			paths = append(paths, p)
		}
	}

	f.Attachments = field{Value: strings.Join(paths, "\n")}

	for i := range f.Formats {
		f.Formats[i].Selected = f.Formats[i].Value == f.Format
	}
}

// readFile takes the body from the uploaded file when there is one: an HTML
// file (.html, .htm) or a text file (.txt), in UTF-8, of maxBodyFile at most.
// The format follows the extension and the content replaces the body of that
// format; a refused file is said under its field. With JavaScript the page
// has already loaded the file in the body and sent the form without it.
func (f *formData) readFile(r *http.Request, translate gui.Translator) {
	if r.MultipartForm == nil {
		return
	}

	file, header, err := r.FormFile(fieldBodyFile)
	if err != nil {
		if !errors.Is(err, http.ErrMissingFile) {
			f.BodyFile.Error = translate("email_templates.form.error.body_file_read")
		}

		return
	}
	defer file.Close() //nolint:errcheck // read only

	toHTML, known := bodyFileTypes[strings.ToLower(filepath.Ext(header.Filename))]

	content, err := io.ReadAll(io.LimitReader(file, maxBodyFile+1))

	switch {
	case !known:
		f.BodyFile.Error = translate("email_templates.form.error.body_file_type")
	case err != nil:
		f.BodyFile.Error = translate("email_templates.form.error.body_file_read")
	case len(content) > maxBodyFile:
		f.BodyFile.Error = translate("email_templates.form.error.body_file_size")
	case !utf8.Valid(content):
		f.BodyFile.Error = translate("email_templates.form.error.body_file_encoding")
	}

	if f.BodyFile.Error != "" {
		return
	}

	body := strings.TrimSpace(strings.ReplaceAll(string(content), "\r\n", "\n"))
	if toHTML {
		f.Format, f.BodyHTML = f.HTMLCase, field{Value: body}
	} else {
		f.Format, f.BodyText = f.Formats[0].Value, field{Value: body}
	}

	for i := range f.Formats {
		f.Formats[i].Selected = f.Formats[i].Value == f.Format
	}
}

// bodyField is the body of the format chosen.
func (f *formData) bodyField() *field {
	if isHTML(f.Format) {
		return &f.BodyHTML
	}

	return &f.BodyText
}

// attachments are the paths of the attachments, one per line.
func (f *formData) attachments() []string {
	if f.Attachments.Value == "" {
		return nil
	}

	return strings.Split(f.Attachments.Value, "\n")
}

// template is the template the form holds.
func (f *formData) template() model.EmailTemplate {
	return model.EmailTemplate{
		ID: f.ID, Name: f.Name.Value, Subject: f.Subject.Value,
		MIMEType: f.Format, Body: f.bodyField().Value, Attachments: f.attachments(),
	}
}

// validate checks what the screen can check before the model: a name, a
// format among those offered, a subject and a body, and no marker the tasks
// do not substitute (unknown, or not implemented: the task would fail).
func (f *formData) validate(translate gui.Translator) bool {
	fail := func(fld *field, key string, args ...any) {
		fld.Error = translate(key, args...)
		f.HasErrors = true
	}

	if f.BodyFile.Error != "" {
		f.HasErrors = true
	}

	if f.Name.Value == "" {
		fail(&f.Name, "email_templates.form.error.required")
	}

	if f.Format != f.HTMLCase && !strings.Contains(" "+f.TextCases+" ", " "+f.Format+" ") {
		f.Refused = translate("email_templates.form.error.format")
		f.HasErrors = true
	}

	body := f.bodyField()

	for _, fld := range []*field{&f.Subject, body, &f.Attachments} {
		value := fld.Value
		if fld == &f.BodyHTML {
			value = tagPattern.ReplaceAllString(value, "")
		}

		if strings.TrimSpace(value) == "" && fld != &f.Attachments {
			fail(fld, "email_templates.form.error.required")

			continue
		}

		if unknown := tasks.UnknownMarkers(fld.Value); len(unknown) > 0 {
			fail(fld, "email_templates.form.error.markers", "markers", strings.Join(unknown, ", "))
		}
	}

	return !f.HasErrors
}

// apply writes the validated form into the template.
func (f *formData) apply(t *model.EmailTemplate) {
	t.Name = f.Name.Value
	t.Subject = f.Subject.Value
	t.MIMEType = f.Format
	t.Body = f.bodyField().Value
	t.Attachments = f.attachments()
}

// refuse marks the form with the refusal of the model: the known causes go
// to their field, the others to the banner.
func (f *formData) refuse(translate gui.Translator, err error) {
	msg := err.Error()

	switch {
	case strings.Contains(msg, causeTaken):
		f.Name.Error = translate("email_templates.form.error.name_taken")
	case strings.Contains(msg, causeMIME):
		f.Refused = translate("email_templates.form.error.format")
	default:
		f.Refused = msg
	}

	f.HasErrors = true
}

// render shows the form, with the preview of the template it holds.
func (h *Handlers) render(w http.ResponseWriter, r *http.Request, form *formData, uses []ruleuse.Use, status int) {
	translate := h.shell.Translator(r)
	tpl := form.template()
	form.Preview = h.preview(translate, &tpl, uses, "", false)

	screen := h.screen(translate, form.Title, ret(r), status)
	screen.Editor = true
	screen.Scripts = []string{script}
	h.shell.Render(w, h.logger, r, screen, formView(form))
}

// NewForm serves GET /webui/email-templates/new.
func (h *Handlers) NewForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, newForm(h.shell.Translator(r), nil, ret(r)), nil, 0)
}

// Create serves POST /webui/email-templates/new.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	h.save(w, r, nil)
}

// EditForm serves GET /webui/email-templates/{id}/edit.
func (h *Handlers) EditForm(w http.ResponseWriter, r *http.Request) {
	e, ok := h.template(w, r)
	if !ok {
		return
	}

	h.render(w, r, newForm(h.shell.Translator(r), &e, ret(r)), e.uses, 0)
}

// Update serves POST /webui/email-templates/{id}/edit.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	e, ok := h.template(w, r)
	if !ok {
		return
	}

	h.save(w, r, &e)
}

// save reads, validates and stores the posted form: a new template when e is
// nil, the given one otherwise. A refused form, by the screen or by the
// model, is rendered again with its errors (422); a saved one redirects with
// a notification: a new template shows in the list, an edited one comes back
// to its sheet.
func (h *Handlers) save(w http.ResponseWriter, r *http.Request, e *entry) {
	translate := h.shell.Translator(r)
	carried := ret(r)

	if err := means.ParseForm(r); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	form := newForm(translate, e, carried)
	form.read(r.PostForm)
	form.readFile(r, translate)

	var uses []ruleuse.Use
	if e != nil {
		uses = e.uses
	}

	if form.validate(translate) {
		// A copy: a refused edition leaves the template as it was read.
		done, tpl := doneCreated, &model.EmailTemplate{}
		if e != nil {
			stored := *e.tpl
			done, tpl = doneUpdated, &stored
		}

		form.apply(tpl)

		var err error
		if done == doneCreated {
			err = h.db.Insert(tpl).Run()
		} else {
			err = h.db.Update(tpl).Run()
		}

		switch {
		case err == nil:
			result := url.Values{doneParam: {done}, nameParam: {tpl.Name}}
			if done == doneUpdated {
				liststate.RedirectTo(w, r, withRet(templateURL(tpl.ID, ""), carried), result)
			} else {
				listSpec.RedirectList(w, r, carried, result)
			}

			return
		case database.IsValidationError(err):
			form.refuse(translate, err)
		default:
			h.logger.Errorf("failed to save the email template %q: %v", tpl.Name, err)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}
	}

	h.render(w, r, form, uses, http.StatusUnprocessableEntity)
}

// previewURL is the route the script of the screen posts the form to, for
// the preview of the template it holds.
func previewURL() string { return common.WebuiURL(pagePath + "/preview") }

// describe is the id of the hint of a field, or of its error.
func describe(id string, f field) string {
	if f.Error != "" {
		return id + "-e"
	}

	return id + "-h"
}

// rowsOf is the height of a text area: a few paths, or a message.
func rowsOf(id string) string {
	if id == fieldAttachments {
		return "3"
	}

	return "10"
}

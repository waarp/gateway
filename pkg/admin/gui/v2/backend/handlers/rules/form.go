package rules

import (
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// Names of the fields of the form.
const (
	fieldName      = "name"
	fieldDirection = "direction"
	fieldPath      = "path"
	fieldComment   = "comment"
	fieldLocal     = "local_dir"
	fieldRemote    = "remote_dir"
	fieldTmp       = "tmp_dir"

	// The columns of the name and of the path hold 100 characters on
	// PostgreSQL and MySQL; a longer value would fail there with a database
	// error.
	maxLength = 100

	// The causes the model gives when it refuses a rule.
	causeNameTaken  = "rule named"
	causePathTaken  = "with path"
	causeDescendant = "cannot be the descendant"
	causeAncestor   = "cannot be the ancestor"
	causeTmpLocal   = "tmp directory must be local"
)

type field = protoform.Field

// choice is a direction offered on the creation of a rule, with what it
// means.
type choice struct {
	Value   string
	Label   string
	Text    string
	Checked bool
}

type formData struct {
	Title     string
	Action    string
	CancelURL string
	Edit      bool
	HasErrors bool
	Refused   string // a refusal of the model no field carries

	Name       field
	Direction  string // "send" or "receive", "" when none is chosen yet
	Directions []choice
	DirErr     string
	DirLabel   string // the direction in reading, on edition
	Path       field
	Comment    field
	Local      field
	Remote     field
	Tmp        field

	// InHistory: the history names the rule as it is (edition), its entries
	// could no longer be restarted after a rename.
	InHistory bool

	stored *model.Rule
}

// newForm builds the form of a new rule, or the form of an existing one
// filled with its values.
func newForm(translate gui.Translator, e *entry, carried string) *formData {
	form := &formData{
		Title:     translate("rules.new.title"),
		Action:    withRet(common.WebuiURL(pagePath+"/new"), carried),
		CancelURL: listURL(carried),
	}

	if e != nil {
		rule := e.rule
		form.Title = translate("rules.edit.title", "name", rule.Name)
		form.Action = withRet(ruleURL(rule.ID, "/edit"), carried)
		form.CancelURL = withRet(ruleURL(rule.ID, ""), carried)
		form.Edit = true
		form.stored = rule
		form.Name = field{Value: rule.Name}
		form.Direction = directionOf(rule.IsSend)
		form.DirLabel = e.direction(translate)
		form.Path = field{Value: rule.Path}
		form.Comment = field{Value: rule.Comment}
		form.Local = field{Value: rule.LocalDir}
		form.Remote = field{Value: rule.RemoteDir}
		form.Tmp = field{Value: rule.TmpLocalRcvDir}
	}

	return form
}

// read takes the posted values; the direction of an existing rule is kept.
func (f *formData) read(r *http.Request) {
	f.Name = field{Value: strings.TrimSpace(r.PostFormValue(fieldName))}
	f.Path = field{Value: strings.TrimSpace(r.PostFormValue(fieldPath))}
	f.Comment = field{Value: strings.TrimSpace(r.PostFormValue(fieldComment))}
	f.Local = field{Value: strings.TrimSpace(r.PostFormValue(fieldLocal))}
	f.Remote = field{Value: strings.TrimSpace(r.PostFormValue(fieldRemote))}
	f.Tmp = field{Value: strings.TrimSpace(r.PostFormValue(fieldTmp))}

	if !f.Edit {
		f.Direction = ""
		if dir := r.PostFormValue(fieldDirection); dir == dirSend || dir == dirReceive {
			f.Direction = dir
		}
	}
}

// validate checks what the screen can check before the model: the name
// (given, without a slash, which the REST API could not address, and short
// enough for every database), the direction, the length of the path.
func (f *formData) validate(translate gui.Translator) bool {
	switch {
	case f.Name.Value == "":
		f.Name.Error = translate("rules.form.error.required")
	case strings.Contains(f.Name.Value, "/"):
		f.Name.Error = translate("rules.form.error.slash")
	case utf8.RuneCountInString(f.Name.Value) > maxLength:
		f.Name.Error = translate("rules.form.error.too_long", "n", maxLength)
	}

	if utf8.RuneCountInString(f.Path.Value) > maxLength {
		f.Path.Error = translate("rules.form.error.too_long", "n", maxLength)
	}

	if f.Direction == "" {
		f.DirErr = translate("rules.form.error.direction")
	}

	f.HasErrors = f.Name.Error != "" || f.Path.Error != "" || f.DirErr != ""

	return !f.HasErrors
}

// apply writes the validated form into the rule; an empty path becomes the
// name of the rule (the product's default).
func (f *formData) apply(rule *model.Rule) {
	rule.Name = f.Name.Value
	rule.IsSend = f.Direction == dirSend
	rule.Path = f.Path.Value
	rule.Comment = f.Comment.Value
	rule.LocalDir = f.Local.Value
	rule.RemoteDir = f.Remote.Value
	rule.TmpLocalRcvDir = f.Tmp.Value
}

// refuse marks the form with the refusal of the model: the known causes go
// to their field, the others to the banner.
func (f *formData) refuse(translate gui.Translator, err error) {
	msg := err.Error()
	dir := translate(directionKeyPrefix + f.Direction + "_lower")

	switch {
	case strings.Contains(msg, causeNameTaken):
		f.Name.Error = translate("rules.form.error.name_taken", "direction", dir)
	case strings.Contains(msg, causePathTaken):
		f.Path.Error = translate("rules.form.error.path_taken", "direction", dir)
	case strings.Contains(msg, causeDescendant):
		f.Path.Error = translate("rules.form.error.path_under", "direction", dir)
	case strings.Contains(msg, causeAncestor):
		f.Path.Error = translate("rules.form.error.path_above", "direction", dir)
	case strings.Contains(msg, causeTmpLocal):
		f.Tmp.Error = translate("rules.form.error.tmp_local")
	default:
		f.Refused = msg
	}

	f.HasErrors = true
}

// prepare builds the direction choices of a new rule.
func (f *formData) prepare(translate gui.Translator) {
	f.Directions = make([]choice, 0, len(directions))
	for _, dir := range directions {
		f.Directions = append(f.Directions, choice{
			Value: dir, Label: translate(directionKeyPrefix + dir), Text: translate("rules.form.direction_text." + dir),
			Checked: f.Direction == dir,
		})
	}
}

// inHistory says whether the history names a rule by its name and
// direction: one entry is enough, read without counting them all.
func (h *Handlers) inHistory(rule *model.Rule) bool {
	var entries model.HistoryEntries
	if err := h.db.Select(&entries).Where("rule=? AND is_send=?", rule.Name, rule.IsSend).
		Limit(1, 0).Run(); err != nil {
		h.logger.Warningf("failed to read the history of the transfer rule %q: %v", rule.Name, err)

		return false
	}

	return len(entries) > 0
}

func (h *Handlers) render(w http.ResponseWriter, r *http.Request, form *formData, status int) {
	translate := h.shell.Translator(r)

	if form.stored != nil {
		form.InHistory = h.inHistory(form.stored)
	}

	form.prepare(translate)
	h.shell.Render(w, h.logger, r, h.screen(translate, form.Title, ret(r), status), formView(form))
}

// NewForm serves GET /webui/rules/new.
func (h *Handlers) NewForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, newForm(h.shell.Translator(r), nil, ret(r)), 0)
}

// Create serves POST /webui/rules/new.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	h.save(w, r, nil)
}

// EditForm serves GET /webui/rules/{id}/edit.
func (h *Handlers) EditForm(w http.ResponseWriter, r *http.Request) {
	e, ok := h.load(w, r)
	if !ok {
		return
	}

	h.render(w, r, newForm(h.shell.Translator(r), &e, ret(r)), 0)
}

// Update serves POST /webui/rules/{id}/edit.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	e, ok := h.load(w, r)
	if !ok {
		return
	}

	h.save(w, r, &e)
}

// save reads, validates and stores the posted form: a new rule when e is
// nil, the given one otherwise, its tasks and its access entries untouched. A
// refused form, by the screen or by the model, is rendered again with its
// errors (422); a saved one leads to the sheet of the rule, with a
// notification.
func (h *Handlers) save(w http.ResponseWriter, r *http.Request, e *entry) {
	translate := h.shell.Translator(r)
	carried := ret(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	form := newForm(translate, e, carried)
	form.read(r)

	if form.validate(translate) {
		// A copy: a refused edition leaves the rule as it was read.
		done, rule := doneCreated, &model.Rule{}
		if e != nil {
			stored := *e.rule
			done, rule = doneUpdated, &stored
		}

		form.apply(rule)

		var err error
		if done == doneCreated {
			err = h.db.Insert(rule).Run()
		} else {
			err = h.db.Update(rule).Run()
		}

		switch {
		case err == nil:
			liststate.RedirectTo(w, r, withRet(ruleURL(rule.ID, ""), carried),
				url.Values{doneParam: {done}, nameParam: {rule.Name}})

			return
		case database.IsValidationError(err):
			form.refuse(translate, err)
		default:
			h.logger.Errorf("failed to save the transfer rule %q: %v", rule.Name, err)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}
	}

	h.render(w, r, form, http.StatusUnprocessableEntity)
}

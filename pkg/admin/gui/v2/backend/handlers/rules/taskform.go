package rules

import (
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

// The page of a task: the type to choose (a new task), then its parameters, each asked as the description of
// its type says (taskspec.go), checked as far as the form can before the product checks the task.

// The kinds of field of the form, as the template renders them.
const (
	fieldText    = "text"
	fieldBool    = "bool"
	fieldNumber  = "number"
	fieldDecimal = "decimal"
	fieldSelect  = "select"
	fieldJSON    = "json"
	fieldPairs   = "pairs"
)

// taskOption is a value offered by a select.
type taskOption struct {
	Value    string
	Label    string
	Selected bool
}

// taskGroup is a group of a select; one group without label for a flat
// list.
type taskGroup struct {
	Label   string
	Options []taskOption
}

// fieldView is an argument of the task in the form.
type fieldView struct {
	ID       string
	Kind     string // fieldText, fieldBool, fieldNumber, fieldDecimal, fieldSelect, fieldJSON, fieldPairs
	Label    string
	Hint     string
	Error    string
	Value    string
	Checked  bool
	Required bool
	Min, Max int
	None     string // the empty option of a select
	Groups   []taskGroup
	Rows     int
}

type familyView struct {
	Label string
	Types []taskOption
}

type taskFormData struct {
	Title     string
	Action    string
	CancelURL string
	Ret       string // the state of the list, carried by the GET form of the type
	Choose    bool   // the first step: the type to choose
	Families  []familyView
	TypeErr   string
	Type      string
	TypeLabel string
	TypeText  string
	ChangeURL string // a new task: back to the choice of the type
	Fields    []fieldView
	Required  bool   // a field is required: the form says what the star means
	Unknown   string // the arguments kept as they are
	HasErrors bool
	Refused   string
}

// fields builds the form of a task of the given type with its values; a
// value the lists do not hold stays offered, marked missing.
func fields(translate gui.Translator, spec *taskSpec, args map[string]string, found namedObjects) []fieldView {
	out := make([]fieldView, 0, len(spec.Args))
	extensions := strings.Join(tasks.ArchiveExtensions.Keys(), ", ")

	for i := range spec.Args {
		arg := &spec.Args[i]
		value := args[arg.Name]
		field := fieldView{
			ID: argPrefix + arg.Name, Kind: fieldText, Label: translate(argKey(spec, arg.Name, "label")),
			Hint: translate(argKey(spec, arg.Name, "hint"), "extensions", extensions), Value: value, Required: arg.Required,
		}

		switch arg.Kind {
		case argBool:
			field.Kind, field.Checked = fieldBool, value == trueValue
		case argInt, argMillis:
			field.Kind, field.Min, field.Max = fieldNumber, arg.Min, arg.Max
			if arg.Kind == argMillis {
				field.Hint += " " + translate("task_types.hint.millis")
			}
		case argDecimal:
			field.Kind = fieldDecimal
		case argDuration:
			field.Hint += " " + translate("task_types.hint.duration")
		case argOctal:
			field.Hint += " " + translate("task_types.hint.octal")
		case argJSON:
			field.Kind, field.Rows = fieldJSON, jsonLines
			field.Hint += " " + translate("task_types.hint.json")
		case argPairs:
			field.Kind, field.Rows, field.Value = fieldPairs, pairsLines, pairsText(args)
		case argChoice:
			field.Kind, field.None = fieldSelect, noneLabel(translate, arg)
			field.Groups = withMissing(translate, choiceGroups(translate, arg, value), value)
		case argRef:
			field.Kind, field.None = fieldSelect, noneLabel(translate, arg)
			field.Groups = withMissing(translate, selected(found[arg.Ref], value), value)
		case argText:
		}

		out = append(out, field)
	}

	return out
}

// choiceGroups are the values of a choice with their labels, the value
// selected.
func choiceGroups(translate gui.Translator, arg *argSpec, value string) []taskGroup {
	group := taskGroup{}

	for _, c := range arg.Choices() {
		label := c.Value
		if c.Key != "" {
			label = translate(c.Key)
		}

		group.Options = append(group.Options, taskOption{Value: c.Value, Label: label, Selected: c.Value == value})
	}

	return []taskGroup{group}
}

// noneLabel is the empty option of a select: "Choose" for a required
// argument, "None" (or the label of the description) for an optional one.
func noneLabel(translate gui.Translator, arg *argSpec) string {
	switch {
	case arg.Required:
		return translate("task_types.choice.choose")
	case arg.None != "":
		return translate(arg.None)
	default:
		return translate("rules.tasks.option_none")
	}
}

// selected copies the groups with the value selected.
func selected(groups []taskGroup, value string) []taskGroup {
	out := make([]taskGroup, 0, len(groups))

	for _, g := range groups {
		copied := taskGroup{Label: g.Label}
		for _, o := range g.Options {
			o.Selected = o.Value == value
			copied.Options = append(copied.Options, o)
		}

		out = append(out, copied)
	}

	return out
}

// withMissing adds a value the lists do not hold (an object deleted since, a
// variable), selected and marked, so that saving the form keeps it.
func withMissing(translate gui.Translator, groups []taskGroup, value string) []taskGroup {
	if value == "" || offers(groups, value) {
		return groups
	}

	missing := taskOption{Value: value, Label: translate("rules.tasks.missing", "name", value), Selected: true}

	return append([]taskGroup{{Options: []taskOption{missing}}}, groups...)
}

// pairsText is the SETINFO arguments, one "key=value" a line, by key.
func pairsText(args map[string]string) string {
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}

	slices.Sort(keys)

	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, k+pairsSeparator+args[k])
	}

	return strings.Join(lines, "\n")
}

// readArgs takes the posted arguments of a task: the values given (a ticked
// box as "true"; a box left empty and an empty field are left out), checked
// as far as the form can (required, numbers, JSON, key=value lines); the
// arguments of the stored task the description does not know are kept.
func readArgs(translate gui.Translator, r *http.Request, spec *taskSpec, stored map[string]string,
	form []fieldView,
) (map[string]string, bool) {
	args := map[string]string{}
	valid := true

	for i := range spec.Args {
		arg := &spec.Args[i]
		raw := strings.TrimSpace(r.PostFormValue(argPrefix + arg.Name))
		form[i].Value, form[i].Checked = raw, raw == trueValue

		if msg := checkArg(translate, arg, raw, args); msg != "" {
			form[i].Error, valid = msg, false
		}
	}

	for _, name := range unknownArgs(spec, stored) {
		args[name] = stored[name]
	}

	return args, valid
}

// checkArg checks one posted argument and stores it; it answers the message
// of a refused value.
func checkArg(translate gui.Translator, arg *argSpec, raw string, args map[string]string) string {
	if arg.Kind == argPairs {
		return readPairs(translate, raw, args)
	}

	if raw == "" || arg.Kind == argBool && raw != trueValue {
		if arg.Required {
			return translate("rules.tasks.error.required")
		}

		return ""
	}

	switch arg.Kind {
	case argInt, argMillis:
		n, err := strconv.Atoi(raw)

		switch {
		case err != nil:
			return translate("rules.tasks.error.number")
		case arg.Max > 0 && (n < arg.Min || n > arg.Max):
			return translate("rules.tasks.error.range", "min", arg.Min, "max", arg.Max)
		case n < arg.Min:
			return translate("rules.tasks.error.min", "min", arg.Min)
		}
	case argDecimal:
		if _, err := strconv.ParseFloat(raw, 64); err != nil {
			return translate("rules.tasks.error.decimal")
		}
	case argJSON:
		var object map[string]any
		if err := json.Unmarshal([]byte(raw), &object); err != nil {
			return translate("rules.tasks.error.json")
		}
	case argText, argBool, argDuration, argOctal, argChoice, argRef, argPairs:
	}

	args[arg.Name] = raw

	return ""
}

// readPairs reads the SETINFO lines, "key=value"; an empty value is kept (it
// removes the key from the transfer information when the task runs).
func readPairs(translate gui.Translator, raw string, args map[string]string) string {
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		key, value, ok := strings.Cut(line, pairsSeparator)
		if !ok || strings.TrimSpace(key) == "" {
			return translate("rules.tasks.error.pairs")
		}

		args[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	return ""
}

// unknownArgs are the arguments of a stored task its description does not
// know, kept as they are (none for SETINFO, whose keys are free).
func unknownArgs(spec *taskSpec, stored map[string]string) []string {
	var out []string

	for name := range stored {
		if !slices.ContainsFunc(spec.Args, func(a argSpec) bool { return a.Name == name || a.Kind == argPairs }) {
			out = append(out, name)
		}
	}

	slices.Sort(out)

	return out
}

// typeChoices are the types to choose from, by family.
func typeChoices(translate gui.Translator) []familyView {
	out := make([]familyView, 0, len(families))

	for _, family := range families {
		view := familyView{Label: translate(taskTypesPrefix + "family." + family)}

		for i := range taskSpecs {
			if taskSpecs[i].Family == family {
				view.Types = append(view.Types, taskOption{
					Value: taskSpecs[i].Type, Label: typeLabel(translate, taskSpecs[i].Type) + " (" + taskSpecs[i].Type + ")",
				})
			}
		}

		out = append(out, view)
	}

	return out
}

// TaskNewForm serves GET /webui/rules/{id}/tasks/{chain}/new: the type to
// choose, then (?type=) the arguments of a task of that type.
func (h *Handlers) TaskNewForm(w http.ResponseWriter, r *http.Request) {
	h.taskForm(w, r, false, false)
}

// TaskCreate serves POST /webui/rules/{id}/tasks/{chain}/new: the task is
// added at the end of the chain.
func (h *Handlers) TaskCreate(w http.ResponseWriter, r *http.Request) {
	h.taskForm(w, r, false, true)
}

// TaskEditForm serves GET /webui/rules/{id}/tasks/{chain}/{rank}/edit.
func (h *Handlers) TaskEditForm(w http.ResponseWriter, r *http.Request) {
	h.taskForm(w, r, true, false)
}

// TaskUpdate serves POST /webui/rules/{id}/tasks/{chain}/{rank}/edit: the
// task is replaced at its rank, its type kept.
func (h *Handlers) TaskUpdate(w http.ResponseWriter, r *http.Request) {
	h.taskForm(w, r, true, true)
}

// taskForm serves the four pages of the form: the choice of the type (a new
// task), then its arguments. A stored task of a type the screen does not
// know is not edited (404: the menu does not offer it).
func (h *Handlers) taskForm(w http.ResponseWriter, r *http.Request, edit, post bool) {
	item, ok := h.load(w, r)
	if !ok {
		return
	}

	place, list, ok := h.taskRoute(w, r, &item, edit)
	if !ok {
		return
	}

	if post {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)

			return
		}
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	chain := translate(chainLowerKey + place.slug)
	form := &taskFormData{
		Title:     translate("rules.tasks.new_title", "chain", chain),
		Action:    withRet(newTaskURL(item.rule.ID, place.slug), carried),
		CancelURL: withRet(tasksTabURL(item.rule.ID), carried),
	}

	var stored map[string]string

	switch {
	case edit:
		form.Type, stored = list[place.rank].Type, list[place.rank].Args
		form.Title = translate("rules.tasks.edit_title", "n", place.rank+1, "chain", chain)
		form.Action = withRet(taskURL(item.rule.ID, place.slug, place.rank, "/edit"), carried)
	case post:
		form.Type = r.PostFormValue(fieldType)
	default:
		form.Type = r.URL.Query().Get(fieldType)
	}

	spec := specOf(form.Type)

	switch {
	case spec == nil && edit:
		http.Error(w, "Not found", http.StatusNotFound)

		return
	case spec == nil:
		h.chooseType(w, r, form, place, post || form.Type != "")

		return
	case !edit:
		form.ChangeURL = withRet(newTaskURL(item.rule.ID, place.slug), carried)
	}

	found, err := h.namesFor(spec)
	if err != nil {
		h.internal(w, "read the objects the tasks name", err)

		return
	}

	form.TypeLabel = typeLabel(translate, spec.Type)
	form.TypeText = translate(taskTypesPrefix + spec.Slug + ".text")
	form.Fields = fields(translate, spec, stored, found)
	form.Required = slices.ContainsFunc(spec.Args, func(a argSpec) bool { return a.Required })

	if unknown := unknownArgs(spec, stored); len(unknown) > 0 {
		form.Unknown = translate("rules.tasks.keep_unknown", "names", strings.Join(unknown, ", "))
	}

	status := 0

	if post {
		if h.saveTask(w, r, form, spec, stored, place, list, edit) {
			return
		}

		form.HasErrors = true
		form.Fields = refill(translate, spec, form.Fields, found)
		status = http.StatusUnprocessableEntity
	}

	h.shell.Render(w, h.logger, r, h.screen(translate, form.Title, carried, status), taskFormView(form))
}

// chooseType renders the first step of a new task: the type, by family, in
// a GET form; a type the screen does not know is refused (422).
func (h *Handlers) chooseType(w http.ResponseWriter, r *http.Request, form *taskFormData, place *taskPlace,
	refused bool,
) {
	translate := h.shell.Translator(r)
	carried := ret(r)
	form.Choose, form.Families = true, typeChoices(translate)
	form.Action, form.Ret = newTaskURL(place.ruleID, place.slug), carried

	status := 0
	if refused {
		form.TypeErr, status = translate("rules.tasks.error.type"), http.StatusUnprocessableEntity
	}

	h.shell.Render(w, h.logger, r, h.screen(translate, form.Title, carried, status), taskFormView(form))
}

// saveTask reads the posted arguments and rewrites the chain with the task
// added at its end or replaced at its rank; it answers whether the page was
// answered (a redirection to the tab, or an internal error). A refusal of
// the product goes under the field it names, or in the banner.
func (h *Handlers) saveTask(w http.ResponseWriter, r *http.Request, form *taskFormData, spec *taskSpec,
	stored map[string]string, place *taskPlace, list []*model.Task, edit bool,
) bool {
	translate := h.shell.Translator(r)

	args, valid := readArgs(translate, r, spec, stored, form.Fields)
	if !valid {
		return false
	}

	task := &model.Task{Type: spec.Type, Args: args}
	next := slices.Clone(list)
	done := doneTaskAdded

	if edit {
		next[place.rank], done = task, doneTaskUpdated
	} else {
		next = append(next, task)
	}

	err := h.writeChain(place.ruleID, place.chain(), next)

	switch {
	case err == nil:
		liststate.RedirectTo(w, r, withRet(tasksTabURL(place.ruleID), ret(r)),
			url.Values{doneParam: {done}, nameParam: {form.TypeLabel}})

		return true
	case database.IsValidationError(err):
		refuse(translate, form, err.Error())

		return false
	default:
		h.internal(w, "save a task of the transfer rule", err)

		return true
	}
}

// refill renders the posted values again: the selects with the posted value
// selected, the other fields with what was typed.
func refill(translate gui.Translator, spec *taskSpec, posted []fieldView, found namedObjects) []fieldView {
	values := map[string]string{}
	for i := range spec.Args {
		values[spec.Args[i].Name] = posted[i].Value
	}

	fresh := fields(translate, spec, values, found)
	for i := range fresh {
		fresh[i].Error, fresh[i].Value, fresh[i].Checked = posted[i].Error, posted[i].Value, posted[i].Checked
	}

	return fresh
}

// refuse puts the refusal of the product under the field it names (an
// argument quoted in its message), or in the banner.
func refuse(translate gui.Translator, form *taskFormData, msg string) {
	text := translate("rules.tasks.refused_product", "cause", msg)

	for i := range form.Fields {
		if strings.Contains(msg, `"`+strings.TrimPrefix(form.Fields[i].ID, argPrefix)+`"`) {
			form.Fields[i].Error = text

			return
		}
	}

	form.Refused = text
}

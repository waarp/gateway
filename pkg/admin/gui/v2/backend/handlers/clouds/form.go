package clouds

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// Names of the fields the screen owns. The options of a storage are named
// "<type>_<option>" ("s3_bucket"): every storage has its card, the type
// select shows the one of its value (data-switch), the server reads the one
// of the chosen type.
const (
	fieldName    = "name"
	fieldType    = "type"
	fieldKey     = "access_key" // not "key": the icon sprite of the kit has a symbol of that id
	fieldSecret  = "access_secret"
	fieldEnvAuth = "env_auth"
	fieldOther   = "other"

	// The name "file" is reserved for the local disk (model.CloudInstance).
	reservedName = "file"

	// The prefix the model puts before the cause given by the storage
	// library when it refuses a configuration.
	backendRefusal = "invalid cloud instance configuration: "
)

var errOptionLine = errors.New("malformed option line")

// field is a text field of the form: its value, kept when the form is
// refused, and the message of its error. The secret carries an error only:
// its value is never rendered back.
type field = protoform.Field

// optionView is an option field of the form of a storage.
type optionView struct {
	Name        string // the field name ("s3_bucket")
	Code        string // the option key the product reads
	Kind        string
	Label       string
	Hint        string
	Placeholder string // the default of the product or an example, in the empty field
	Required    bool
	Mono        bool
	Value       string
	Error       string
	Checked     bool
	Choices     []choice
}

// acceptedView is a row of the list of the options the storage library
// accepts for a storage, under the "other options" zone.
type acceptedView struct {
	Key     string
	Default string // "" when the library applies nothing
	Help    string // in English, the words of the library
	Values  string // the values it proposes, when they are few
}

// typeView is the card of a storage in the form: shown for the chosen type,
// open when it holds a required option, a value that differs from the
// product default or an error (the rule of the options of the protocols),
// folded otherwise; its summary says which.
type typeView struct {
	Key     string
	Label   string
	Hint    string
	EnvAuth bool
	Options []optionView
	Open    bool
	Summary string
	// Accepted lists the options the storage library accepts beyond the
	// fields, as the choices of the list to type in under the "other options"
	// zone; empty when the library is not known.
	Accepted []gui.ComboGroup
}

type formData struct {
	Title     string
	Action    string
	CancelURL string
	Edit      bool
	HasErrors bool
	// Refused is the cause given by the model, or by the storage library,
	// when it refuses the instance for another reason than a field shown
	// with its own error.
	Refused string

	Fields  map[string]field // name, key, secret (error only), other
	Type    string           // the chosen storage (canonical key), "" when none
	Types   []choice
	TypeErr string
	EnvAuth bool
	Cases   string // keys of the storages that accept the environment credentials: the case of the checkbox
	Cards   []typeView
	// AccessSummary says how the instance authenticates, in the header of
	// the access card (open: it holds what the gateway needs to connect).
	AccessSummary string
	// OtherOpen is set when the "other options" card holds something or an
	// error; folded otherwise (its summary says so).
	OtherOpen bool

	// The option values as entered, by field name, whatever the storage: a
	// refused form comes back as the user filled it.
	values map[string]string
	checks map[string]bool
	errors map[string]string

	// What the instance held before the edition: the stored type key (kept
	// when it is an alias of the chosen storage) and the options the product
	// reads, so that an untouched instance keeps the same configuration.
	storedType    string
	storedOptions map[string]string
	storedSecret  bool

	// The parsed "other options" and the validated secret, never rendered.
	others map[string]string
	secret string
}

func fieldOf(s *storage, o *option) string { return s.Key + "_" + o.Key }

// newForm builds the form of a new instance, or the form of an existing one
// filled with its values.
func (h *Handlers) newForm(translate gui.Translator, e *entry, carried string) *formData {
	form := &formData{
		Title:     translate("clouds.new.title"),
		Action:    withRet(common.WebuiURL(pagePath+"/new"), carried),
		CancelURL: listURL(carried),
		Fields:    map[string]field{},
		values:    map[string]string{},
		checks:    map[string]bool{},
		errors:    map[string]string{},
	}

	if e != nil {
		form.Title = translate("clouds.edit.title", "name", e.cloud.Name)
		form.Action = withRet(cloudURL(e.cloud.ID, "/edit"), carried)
		form.CancelURL = withRet(cloudURL(e.cloud.ID, ""), carried)
		form.Edit = true
		form.fill(e)
	}

	return form
}

// fill reads the values of an existing instance into the form: the name,
// the key, the type, the options the storage names in their fields, the
// others in the "other options" zone.
func (f *formData) fill(e *entry) {
	f.Fields[fieldName] = field{Value: e.cloud.Name}
	f.Fields[fieldKey] = field{Value: e.cloud.Key}
	f.storedType = e.cloud.Type
	f.storedOptions = e.cloud.Options
	f.storedSecret = e.cloud.Secret != ""
	f.EnvAuth = e.auth() == "env"

	var others []string

	if e.storage != nil {
		f.Type = e.storage.Key

		for i := range e.storage.Options {
			o := &e.storage.Options[i]
			value := e.cloud.Options[o.Key]

			if o.Kind == kindCheck {
				f.checks[fieldOf(e.storage, o)] = strings.EqualFold(value, "true")
			} else {
				f.values[fieldOf(e.storage, o)] = value
			}
		}
	}

	for _, row := range otherRows(e.storage, e.cloud.Options) {
		others = append(others, row.Label+"="+row.Value)
	}

	f.Fields[fieldOther] = field{Value: strings.Join(others, "\n")}
}

// read takes the posted values into the form: every field of every storage,
// so that a refused form comes back as the user filled it.
func (f *formData) read(values url.Values) {
	f.Fields[fieldName] = field{Value: strings.TrimSpace(values.Get(fieldName))}
	f.Fields[fieldKey] = field{Value: strings.TrimSpace(values.Get(fieldKey))}
	f.Fields[fieldOther] = field{Value: strings.ReplaceAll(values.Get(fieldOther), "\r\n", "\n")}
	f.secret = values.Get(fieldSecret)
	f.EnvAuth = values.Get(fieldEnvAuth) != ""
	f.Type = ""

	if s, ok := storageOf(values.Get(fieldType)); ok {
		f.Type = s.Key
	}

	for i := range storages {
		s := &storages[i]

		for j := range s.Options {
			o := &s.Options[j]
			name := fieldOf(s, o)

			if o.Kind == kindCheck {
				f.checks[name] = values.Get(name) != ""
			} else {
				f.values[name] = strings.TrimSpace(values.Get(name))
			}
		}
	}
}

// validate checks the name, the type, the required options of the chosen
// storage and the "other options" lines. It marks the errors on the form and
// returns whether it is valid.
func (f *formData) validate(translate gui.Translator) bool {
	fail := func(name, key string) {
		fld := f.Fields[name]
		fld.Error = translate(key)
		f.Fields[name] = fld
		f.HasErrors = true
	}

	switch f.Fields[fieldName].Value {
	case "":
		fail(fieldName, "clouds.form.error.required")
	case reservedName:
		fail(fieldName, "clouds.form.error.name_file")
	}

	s, ok := storageOf(f.Type)
	if !ok {
		f.TypeErr = translate("clouds.form.error.type")
		f.HasErrors = true
	} else {
		for i := range s.Options {
			o := &s.Options[i]
			if o.Required && f.values[fieldOf(s, o)] == "" {
				f.errors[fieldOf(s, o)] = translate("clouds.form.error.required")
				f.HasErrors = true
			}
		}
	}

	others, order, err := parseOthers(f.Fields[fieldOther].Value)

	switch {
	case err != nil:
		fail(fieldOther, "clouds.form.error.option")
	case ok:
		if refused := s.refusals(translate, order); refused != "" {
			fld := f.Fields[fieldOther]
			fld.Error = refused
			f.Fields[fieldOther] = fld
			f.HasErrors = true
		}
	}

	f.others = others

	return !f.HasErrors
}

// refusals says, line by line, why options of the "other options" zone are
// refused: an option the form names in a field, a credential or an option
// the storage library masks (it would be kept and shown in clear), an
// option the library does not know (it would have no effect). "" when every
// line is accepted.
func (s *storage) refusals(translate gui.Translator, order []string) string {
	lib, known := libraryOf(s)
	label := translate(s.labelKey())

	var refused []string

	for _, key := range order {
		switch {
		case key == optEnvAuth || s.knows(key):
			refused = append(refused, translate("clouds.form.error.other_field", "key", key))
		case isSecret(key) || (known && lib.sensitive(key)):
			refused = append(refused, translate("clouds.form.error.other_secret", "key", key))
		case known && !lib.knows(key):
			refused = append(refused, translate("clouds.form.error.other_unknown", "key", key, "type", label))
		}
	}

	return strings.Join(refused, " ")
}

// parseOthers reads the "other options" zone: one "key=value" per line,
// blank lines ignored; the keys also come in the order of their lines.
func parseOthers(text string) (map[string]string, []string, error) {
	others := map[string]string{}

	var order []string

	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		key = strings.TrimSpace(key)

		if !found || key == "" {
			return nil, nil, errOptionLine
		}

		if _, seen := others[key]; !seen {
			order = append(order, key)
		}

		others[key] = strings.TrimSpace(value)
	}

	return others, order, nil
}

// apply writes the validated form into the instance: the name, the type
// (the stored key stays when it is an alias of the chosen storage), the key,
// the secret when one was given (an empty one keeps the current value), and
// the options: the fields of the chosen storage (an empty field, or a choice
// left at the product default that was not stored before, leaves the option
// out, so that the product applies its default), the environment switch,
// then the other options (never a credential nor a secret, never an option
// the storage names: its field is the place).
func (f *formData) apply(cloud *model.CloudInstance) {
	store, _ := storageOf(f.Type)
	stored, ok := storageOf(f.storedType)
	same := ok && stored.Key == store.Key

	cloud.Name = f.Fields[fieldName].Value
	cloud.Key = f.Fields[fieldKey].Value

	if f.secret != "" {
		cloud.Secret = f.secret
	}

	cloud.Type = store.Key
	if same {
		cloud.Type = f.storedType
	}

	options := f.fieldOptions(store)

	if f.EnvAuth && store.EnvAuth {
		options[optEnvAuth] = "true"
	}

	for key, value := range f.others {
		if isSecret(key) || key == optEnvAuth || store.knows(key) {
			continue
		}

		options[key] = value
	}

	// The options the storage library masks, set by the REST API or the
	// command line, are never shown nor entered here: they stay as they
	// are while the storage does not change.
	if same {
		for _, key := range maskedKeys(store, f.storedOptions) {
			options[key] = f.storedOptions[key]
		}
	}

	cloud.Options = options
}

// fieldOptions reads the fields of the chosen storage: an empty field, or a
// choice left at the product default that was not stored before, leaves the
// option out, so that the product applies its default.
func (f *formData) fieldOptions(store *storage) model.Map[string] {
	values := model.Map[string]{}

	options := store.resolved()

	for i := range options {
		o := &options[i]
		name := fieldOf(store, o)

		switch {
		case o.Kind == kindCheck:
			if f.checks[name] {
				values[o.Key] = "true"
			}
		case f.values[name] == "":
		case o.Kind == kindSelect && f.values[name] == o.Default && f.storedOptions[o.Key] == "":
		default:
			values[o.Key] = f.values[name]
		}
	}

	return values
}

// refuse marks the form with the refusal of the model: the known causes go
// to their field, the others to the banner, the refusal of the storage
// library with its cause.
func (f *formData) refuse(translate gui.Translator, err error) {
	msg := err.Error()
	fail := func(name, key string) {
		fld := f.Fields[name]
		fld.Error = translate(key)
		f.Fields[name] = fld
	}

	// The model prefixes the cause of the storage library ("… validation
	// failed: invalid cloud instance configuration: <cause>"); the library
	// may go on for lines (a provider's error with its JSON body): the page
	// says the first line, the log keeps the whole.
	cause := msg
	if _, after, found := strings.Cut(msg, backendRefusal); found {
		cause = firstLine(after)
	}

	switch {
	case strings.Contains(msg, "already exist"):
		fail(fieldName, "clouds.form.error.name_taken")
	case strings.Contains(msg, "name cannot be empty"):
		fail(fieldName, "clouds.form.error.required")
	case strings.Contains(msg, "is reserved for the local filesystem"):
		fail(fieldName, "clouds.form.error.name_file")
	case strings.Contains(msg, "unknown cloud type"):
		f.TypeErr = translate("clouds.form.error.type")
	case cause != msg:
		f.Refused = translate("clouds.form.refused_by_backend", "cause", cause)
	default:
		f.Refused = msg
	}

	f.HasErrors = true
}

// maxCauseLength bounds the cause of a refusal shown in the banner.
const maxCauseLength = 240

// firstLine keeps the first line of a cause, bounded, so that the banner
// stays readable; the full cause goes to the log.
func firstLine(cause string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(cause), "\n")
	line = strings.TrimSpace(line)

	if len(line) > maxCauseLength {
		return line[:maxCauseLength] + "…"
	}

	return line
}

// prepare builds what the templates render: the type choices, the card of
// every storage with its fields as entered, the state of the "other
// options" card.
func (f *formData) prepare(translate gui.Translator) {
	f.Types = make([]choice, 0, len(storages))
	f.Cards = make([]typeView, 0, len(storages))

	var cases []string

	for i := range storages {
		store := &storages[i]
		f.Types = append(f.Types, choice{Value: store.Key, Label: translate(store.labelKey()), Selected: f.Type == store.Key})

		if store.EnvAuth {
			cases = append(cases, store.Key)
		}

		card := typeView{
			Key: store.Key, Label: translate(store.labelKey()), Hint: translate(store.hintKey()), EnvAuth: store.EnvAuth,
		}
		set, missing := 0, false

		options := store.resolved()

		for j := range options {
			o := &options[j]
			name := fieldOf(store, o)
			view := optionView{
				Name: name, Code: o.Key, Kind: o.Kind, Label: translate(o.labelKey()), Hint: o.hint(translate),
				Placeholder: o.placeholder(), Required: o.Required, Mono: o.Mono,
				Value: f.values[name], Error: f.errors[name], Checked: f.checks[name],
			}

			if o.isSet(view.Value, view.Checked) {
				set++
			}

			missing = missing || (o.Required && view.Value == "")
			card.Open = card.Open || view.Error != ""

			if o.Kind == kindSelect {
				current := protoform.Or(view.Value, protoform.Or(o.Default, o.Initial))

				for _, c := range o.Choices {
					view.Choices = append(view.Choices, choice{Value: c, Label: o.choiceLabel(translate, c), Selected: c == current})
				}
			}

			card.Options = append(card.Options, view)
		}

		card.Open = card.Open || missing || set > 0
		card.Summary = protoform.OptionsSummary(translate, set)
		card.Accepted = acceptedChoices(translate, acceptedOf(store))

		if missing {
			card.Summary = translate("clouds.form.options_required")
		}

		f.Cards = append(f.Cards, card)
	}

	f.Cases = strings.Join(cases, " ")
	f.AccessSummary = f.accessSummary(translate)
	f.OtherOpen = f.Fields[fieldOther].Value != "" || f.Fields[fieldOther].Error != ""
}

// acceptedChoices are the options a storage accepts as the choices of the
// list to type in that writes them in the "other options" zone (Driss,
// 2026-10-23, as the TLS suites are suggested): the key, then what the
// library applies without it, the values it proposes and its help
// (English); nil when there is none.
func acceptedChoices(translate gui.Translator, accepted []acceptedView) []gui.ComboGroup {
	if len(accepted) == 0 {
		return nil
	}

	choices := make([]gui.ComboChoice, 0, len(accepted))

	for _, a := range accepted {
		parts := make([]string, 0, 3) //nolint:mnd // the default, the values, the help
		if a.Default != "" {
			parts = append(parts, translate("clouds.form.accepted_default", "value", a.Default))
		}

		if a.Values != "" {
			parts = append(parts, translate("clouds.form.accepted_values", "values", a.Values))
		}

		if a.Help != "" {
			parts = append(parts, a.Help)
		}

		choices = append(choices, gui.ComboChoice{Value: a.Key, Label: a.Key, Text: strings.Join(parts, " · ")})
	}

	return []gui.ComboGroup{{Choices: choices}}
}

// acceptedOf lists the options the storage library accepts for a storage,
// beyond the fields of the form; nil when the library is not known.
func acceptedOf(s *storage) []acceptedView {
	lib, ok := libraryOf(s)
	if !ok {
		return nil
	}

	listed := lib.listed()
	views := make([]acceptedView, 0, len(listed))

	for _, o := range listed {
		views = append(views, acceptedView{
			Key: o.Key, Default: o.Default, Help: o.Help, Values: strings.Join(o.Values, ", "),
		})
	}

	return views
}

// accessSummary says how the instance will authenticate, in the words of
// the list: by the environment variables, by a key (entered, or a secret
// kept from before the edition), or not at all.
func (f *formData) accessSummary(translate gui.Translator) string {
	secret := f.secret
	if secret == "" && f.storedSecret {
		secret = "kept"
	}

	var options map[string]string
	if f.EnvAuth {
		options = map[string]string{optEnvAuth: "true"}
	}

	return translate(authKeyPrefix + authOf(f.Fields[fieldKey].Value, secret, options))
}

// formScreen is a page of the form: the screen, with the list to type in of
// the kit (the accepted options).
func (h *Handlers) formScreen(translate gui.Translator, title, carried string, status int) common.Screen {
	screen := h.screen(translate, title, carried, status)
	screen.Head = gui.ComboboxAssets()

	return screen
}

// NewForm serves GET /webui/clouds/new.
func (h *Handlers) NewForm(w http.ResponseWriter, r *http.Request) {
	translate := h.shell.Translator(r)
	carried := ret(r)
	form := h.newForm(translate, nil, carried)
	form.prepare(translate)
	h.shell.Render(w, h.logger, r, h.formScreen(translate, form.Title, carried, 0), formView(form))
}

// Create serves POST /webui/clouds/new.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	h.save(w, r, nil)
}

// EditForm serves GET /webui/clouds/{id}/edit.
func (h *Handlers) EditForm(w http.ResponseWriter, r *http.Request) {
	e, ok := h.cloud(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	form := h.newForm(translate, &e, carried)
	form.prepare(translate)
	h.shell.Render(w, h.logger, r, h.formScreen(translate, form.Title, carried, 0), formView(form))
}

// Update serves POST /webui/clouds/{id}/edit.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	e, ok := h.cloud(w, r)
	if !ok {
		return
	}

	h.save(w, r, &e)
}

// save reads, validates and stores the posted form: a new instance when e is
// nil, the given one otherwise. A refused form, by the screen or by the
// model (which instantiates the storage), is rendered again with its errors
// (422); a saved one redirects with a notification: a new instance shows in
// the list, an edited one comes back to its sheet.
func (h *Handlers) save(w http.ResponseWriter, r *http.Request, e *entry) {
	translate := h.shell.Translator(r)
	carried := ret(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	form := h.newForm(translate, e, carried)
	form.read(r.PostForm)

	if form.validate(translate) {
		done := doneUpdated
		cloud := &model.CloudInstance{}

		if e == nil {
			done = doneCreated
		} else {
			cloud = e.cloud
		}

		form.apply(cloud)

		var err error
		if done == doneCreated {
			err = internal.InsertCloud(h.db, cloud)
		} else {
			err = internal.UpdateCloud(h.db, cloud)
		}

		switch {
		case err == nil:
			result := url.Values{doneParam: {done}, nameParam: {cloud.Name}}
			if done == doneUpdated {
				liststate.RedirectTo(w, r, withRet(cloudURL(cloud.ID, ""), carried), result)
			} else {
				listSpec.RedirectList(w, r, carried, result)
			}

			return
		case database.IsValidationError(err):
			// The whole cause (the storage library may give a long one).
			h.logger.Warningf("the cloud instance %q was refused: %v", cloud.Name, err)
			form.refuse(translate, err)
		default:
			h.logger.Errorf("failed to save the cloud instance %q: %v", cloud.Name, err)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}
	}

	form.prepare(translate)
	screen := h.formScreen(translate, form.Title, carried, http.StatusUnprocessableEntity)
	h.shell.Render(w, h.logger, r, screen, formView(form))
}

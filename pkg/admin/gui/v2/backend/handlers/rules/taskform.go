package rules

import (
	"cmp"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/transfers"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

// The page of a task, one page (prototype validated by Driss on 2026-10-23):
// the type of a new task makes its parameters appear at once, one card per
// group of its description (taskspec.go), side by side where they fit; two or
// three values to choose from as joined buttons, a box as a box. TRANSFER is
// asked as the page that adds a transfer asks it (the direction, the rules of
// that direction, the partner and its account, the client, the rows of the
// transfer information, the retries in seconds) and keeps what is its own (the
// copy of the information, the synchronous mode and its time limit). The page
// holds the fields of every type, named after their type, and shows those of
// the type chosen (the kit switches them; without JavaScript, "Continue" asks
// the page again with the type). Every value is checked as far as the form can
// before the product checks the task.

// The kinds of field of the form, as the template renders them.
const (
	fieldText    = "text"
	fieldBool    = "bool"
	fieldNumber  = "number"
	fieldDecimal = "decimal"
	fieldSelect  = "select"
	fieldChoice  = "choice" // two or three values, as joined buttons
	fieldFixed   = "fixed"  // the only value possible, said and sent
	fieldJSON    = "json"
	fieldPairs   = "pairs"

	// The most values a choice offers as joined buttons; beyond, a list.
	maxButtons = 3

	// The box of the card of the transfer information, one for TRANSFER and
	// PREREGISTER.
	fieldInfoCopy = "info_copy"

	// The fields of the exchange of a TRANSFER task, after its type.
	exchangeSens    = "sens"
	exchangeRule    = "rule_"
	exchangeAccount = "account"

	// The scheme of the address of an ICAP server, which the gateway adds.
	icapScheme = "icap://"

	// The parts of the value of a partner and its account.
	accountPartner = "partner"
	accountLogin   = "login"
)

// argName and argID: a field of the page is named after its type, since
// the page holds the fields of every type (two types may name an argument
// alike, a radio button of one would uncheck the one of another).
func argName(spec *taskSpec, arg string) string { return spec.Slug + "." + arg }

func argID(spec *taskSpec, arg string) string { return spec.Slug + "-" + arg }

// taskOption is a value offered by a select or a choice.
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

// fieldView is an argument of the task in the form. Switch names the fields
// that depend on this one (data-switch); CaseOf and CaseValue, the switch a
// field depends on and the value that shows it.
type fieldView struct {
	ID        string
	Name      string
	Kind      string
	Label     string
	Hint      string
	Error     string
	Value     string
	Checked   bool
	Required  bool
	NoMark    bool // a choice always made: no star, no "(optional)"
	Min, Max  int
	None      string // the empty option of a select
	Groups    []taskGroup
	Options   []taskOption // the joined buttons of a choice
	Fixed     string       // the label of the only value possible
	Rows      int
	Wide      bool
	Switch    string
	CaseOf    string
	CaseValue string
	Check     gui.Check // the type the page checks as soon as the user leaves the field
}

// groupView is a card of the page: its title, its note, its fields side by
// side, or the exchange of a TRANSFER task; Empty, the card of a type without
// parameter.
type groupView struct {
	Title    string
	Note     string
	Columns  int
	Fields   []fieldView
	Exchange *exchangeView
	Empty    bool
}

// typeBlock is what the page shows of a type: its label, its code and what it
// does, its cards before and after the card of the transfer information; only
// the block of the type chosen is shown.
type typeBlock struct {
	Type   string
	Label  string
	Tip    string
	Active bool
	Before []groupView
	After  []groupView
}

// infosView is the card of the transfer information, one for the types that
// ask for it (Cases, for the switch of the type): its box, its rows.
type infosView struct {
	Cases  string
	Active bool
	Copy   bool
	Rows   []transfers.InfoPair
	Error  string
}

// exchangeView is the exchange of a TRANSFER task, as the page that adds a
// transfer asks it: the direction, the rules of each direction, the partner
// and its account, the client.
type exchangeView struct {
	Slug       string
	Senses     []taskOption
	SensErr    string
	Rules      []ruleCase
	RuleErr    string
	Accounts   []taskGroup
	AccountErr string
	Client     fieldView
}

// ruleCase is the list of the rules of a direction, shown once the direction
// is chosen.
type ruleCase struct {
	Sens   string
	ID     string
	Name   string
	None   string
	Groups []taskGroup
}

type taskFormData struct {
	Title     string
	Action    string
	CancelURL string
	Ret       string // the state of the list, carried by "Continue" without JavaScript
	NewURL    string // where "Continue" asks the page again, with the type
	Edit      bool
	Families  []gui.ComboGroup // the types by family, for the list to type in
	Type      string
	TypeErr   string
	TypeTip   string // the type of a task edited: its label and what it does
	Blocks    []typeBlock
	Infos     *infosView
	Required  bool   // a field is required: the form says what the star means
	Unknown   string // the arguments kept as they are
	HasErrors bool
	Refused   string
}

// taskInput is what the form holds for the type chosen: the values of its
// arguments and their errors, the rows of its transfer information, its
// exchange; posted says whether the values come from the form (as typed) or
// from the task stored.
type taskInput struct {
	values   map[string]string
	errors   map[string]string
	rows     []transfers.InfoPair
	infoErr  string
	exchange exchangeInput
	posted   bool
}

// exchangeInput is the exchange of a TRANSFER task: its direction, its rule,
// the partner and the login of its account, with their errors.
type exchangeInput struct {
	sens, rule, partner, login   string
	sensErr, ruleErr, accountErr string
}

// inputOf is what the form shows of a stored task (none for a new one).
func inputOf(stored map[string]string) *taskInput {
	input := &taskInput{values: maps.Clone(stored), errors: map[string]string{}, rows: rowsOf(stored["info"])}
	if input.values == nil {
		input.values = map[string]string{}
	}

	exchange := &input.exchange
	switch {
	case stored["to"] != "":
		exchange.sens, exchange.partner = dirSend, stored["to"]
	case stored["from"] != "":
		exchange.sens, exchange.partner = dirReceive, stored["from"]
	}

	exchange.rule, exchange.login = stored["rule"], stored["as"]

	return input
}

// rowsOf are the rows of a stored transfer information.
func rowsOf(info string) []transfers.InfoPair {
	var object map[string]json.RawMessage
	if info == "" || json.Unmarshal([]byte(info), &object) != nil {
		return nil
	}

	return transfers.InfoPairs(object)
}

// accountValue is a partner and the login of its account, as one value of a
// list; accountOf reads it.
func accountValue(partner, login string) string {
	if partner == "" && login == "" {
		return ""
	}

	return url.Values{accountPartner: {partner}, accountLogin: {login}}.Encode()
}

func accountOf(value string) (partner, login string) {
	parts, err := url.ParseQuery(value)
	if err != nil {
		return "", ""
	}

	return parts.Get(accountPartner), parts.Get(accountLogin)
}

// secondsText is a duration in seconds, as the form asks it ("90" for
// 1m30s); a duration of another form (milliseconds, a variable) stays as it
// is.
func secondsText(value string) string {
	d, err := time.ParseDuration(value)
	if err != nil || d%time.Second != 0 {
		return value
	}

	return strconv.FormatInt(int64(d/time.Second), 10)
}

// fields builds the fields of a task of the given type with what the form
// holds (nothing for a type not chosen), one per argument in the order of the
// description; a value the lists do not hold stays offered, marked missing.
func fields(translate gui.Translator, spec *taskSpec, input *taskInput, found namedObjects) []fieldView {
	out := make([]fieldView, 0, len(spec.Args))
	extensions := strings.Join(tasks.ArchiveExtensions.Keys(), ", ")

	for i := range spec.Args {
		arg := &spec.Args[i]
		value := input.values[arg.Name]
		field := fieldView{
			ID: argID(spec, arg.Name), Name: argName(spec, arg.Name), Kind: fieldText,
			Label: translate(argKey(spec, arg.Name, "label")), Error: input.errors[arg.Name],
			Hint: translate(argKey(spec, arg.Name, "hint"), "extensions", extensions), Value: value,
			Required: arg.Required, Wide: arg.Wide, Check: arg.check(),
		}

		if arg.When != nil {
			field.CaseOf, field.CaseValue = argID(spec, arg.When.Arg), arg.When.Value
		}

		if slices.ContainsFunc(spec.Args, func(a argSpec) bool { return a.When != nil && a.When.Arg == arg.Name }) {
			field.Switch = field.ID
		}

		switch arg.Kind {
		case argBool:
			boolField(translate, arg, value, &field)
		case argInt:
			field.Kind, field.Min, field.Max = fieldNumber, arg.Min, arg.Max
		case argDecimal:
			field.Kind = fieldDecimal
		case argDuration:
			durationField(translate, arg, value, &field)
		case argOctal:
			field.Hint += " " + translate("task_types.hint.octal")
		case argJSON:
			field.Kind, field.Rows, field.Wide = fieldJSON, jsonLines, true
			field.Hint += " " + translate("task_types.hint.json")
		case argPairs:
			field.Kind, field.Rows, field.Wide = fieldPairs, pairsLines, true
			if !input.posted {
				field.Value = pairsText(input.values)
			}
		case argChoice:
			choiceField(translate, arg, value, &field)
		case argRef:
			field.Kind, field.None = fieldSelect, noneLabel(translate, arg)
			field.Groups = withMissing(translate, selected(found[arg.Ref], value), value)
		case argText:
		}

		out = append(out, field)
	}

	return out
}

// boolField: a box, or two joined buttons when the description names its
// two modes ("" and "true").
func boolField(translate gui.Translator, arg *argSpec, value string, field *fieldView) {
	if arg.Modes[0] == "" {
		field.Kind, field.Checked = fieldBool, value == trueValue

		return
	}

	field.Kind, field.NoMark = fieldChoice, true
	field.Options = []taskOption{
		{Value: "", Label: translate(arg.Modes[0]), Selected: value != trueValue},
		{Value: trueValue, Label: translate(arg.Modes[1]), Selected: value == trueValue},
	}
}

// durationField: a duration, in seconds when the description says so (a
// number field, unless the value stored has another form).
func durationField(translate gui.Translator, arg *argSpec, value string, field *fieldView) {
	switch {
	case arg.Millis:
		field.Hint += " " + translate("task_types.hint.duration")
		field.Value = millisText(value)
	case !arg.Seconds:
		field.Hint += " " + translate("task_types.hint.duration")
	default:
		field.Value = secondsText(value)
		if _, err := strconv.Atoi(field.Value); err == nil || field.Value == "" {
			field.Kind, field.Min = fieldNumber, 0
		}
	}
}

// durationValue reads a duration of the form as the product stores it: the
// seconds of a Seconds argument ("30s"), the milliseconds of a Millis one, the
// text otherwise; with the message of a refusal.
func durationValue(translate gui.Translator, arg *argSpec, raw string) (value, msg string) {
	switch {
	case arg.Millis:
		millis, ok := millisValue(raw)
		if !ok {
			return raw, translate("check." + string(gui.CheckDuration))
		}

		return millis, ""
	case arg.Seconds:
		duration, ok := secondsValue(raw)
		if !ok {
			return raw, translate("rules.tasks.error.seconds")
		}

		return duration, ""
	default:
		return raw, ""
	}
}

// millisText writes a stored number of milliseconds as the duration the form
// shows ("30s", "1m30s", "1h", "1.5s"); zero, empty or unreadable: nothing,
// the limit is absent.
func millisText(stored string) string {
	ms, err := strconv.ParseInt(stored, 10, 64)
	if err != nil || ms <= 0 {
		return ""
	}

	return compactDuration(time.Duration(ms) * time.Millisecond)
}

// millisValue reads a duration of the form as the number of milliseconds the
// product stores; empty: no limit, nothing stored.
func millisValue(raw string) (string, bool) {
	if raw == "" {
		return "", true
	}

	d, err := time.ParseDuration(raw)
	if err != nil || d < 0 {
		return "", false
	}

	return strconv.FormatInt(int64(d/time.Millisecond), 10), true
}

// compactDuration writes a duration without its zero parts ("1h", "1m30s");
// under the second, as Go writes it ("1.5s").
func compactDuration(d time.Duration) string {
	if d%time.Second != 0 {
		return d.String()
	}

	var out strings.Builder

	for _, part := range []struct {
		unit time.Duration
		mark string
	}{{time.Hour, "h"}, {time.Minute, "m"}, {time.Second, "s"}} {
		if n := d / part.unit; n > 0 {
			out.WriteString(strconv.FormatInt(int64(n), 10) + part.mark)
			d %= part.unit
		}
	}

	return out.String()
}

// choiceField: the only value possible is said and sent; two or three values
// (the empty one of an optional choice among them) are joined buttons; more,
// or a value the description does not offer, a list.
func choiceField(translate gui.Translator, arg *argSpec, value string, field *fieldView) {
	groups := choiceGroups(translate, arg, value)
	options := groups[0].Options
	count := len(options)

	if !arg.Required {
		count++
	}

	switch {
	case value != "" && !offers(groups, value):
	case arg.Required && len(options) == 1:
		field.Kind, field.Value, field.Fixed = fieldFixed, options[0].Value, options[0].Label

		return
	case count <= maxButtons:
		if !arg.Required {
			none := taskOption{Value: "", Label: noneLabel(translate, arg), Selected: value == ""}
			options = append([]taskOption{none}, options...)
		}

		field.Kind, field.Options = fieldChoice, options

		return
	}

	field.Kind, field.None = fieldSelect, noneLabel(translate, arg)
	field.Groups = withMissing(translate, groups, value)
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

// exchangeOf builds the exchange of a TRANSFER task: the rules of each
// direction (the one held selected, kept when the lists do not hold it), the
// accounts by partner, the client.
func exchangeOf(translate gui.Translator, spec *taskSpec, held *exchangeInput, byDirection map[string][]string,
	found namedObjects, client *fieldView,
) *exchangeView {
	view := &exchangeView{
		Slug: spec.Slug, SensErr: held.sensErr, RuleErr: held.ruleErr, AccountErr: held.accountErr, Client: *client,
	}

	for _, dir := range directions {
		view.Senses = append(view.Senses, taskOption{
			Value: dir, Label: translate(directionKeyPrefix + dir), Selected: held.sens == dir,
		})

		group := taskGroup{}
		for _, name := range byDirection[dir] {
			group.Options = append(group.Options, taskOption{
				Value: name, Label: name, Selected: held.sens == dir && held.rule == name,
			})
		}

		groups := []taskGroup{group}
		if held.sens == dir {
			groups = withMissing(translate, groups, held.rule)
		}

		view.Rules = append(view.Rules, ruleCase{
			Sens: dir, ID: argID(spec, exchangeRule+dir), Name: argName(spec, exchangeRule+dir),
			None: translate("task_types.transfer.rule_" + dir), Groups: groups,
		})
	}

	chosen := accountValue(held.partner, held.login)

	for _, g := range found[refRemoteAccount] {
		group := taskGroup{Label: g.Label}
		for _, o := range g.Options {
			value := accountValue(g.Label, o.Value)
			group.Options = append(group.Options, taskOption{Value: value, Label: o.Label, Selected: value == chosen})
		}

		view.Accounts = append(view.Accounts, group)
	}

	if chosen != "" && !offers(view.Accounts, chosen) {
		missing := taskOption{
			Value: chosen, Label: translate("rules.tasks.missing", "name", held.partner+" · "+held.login), Selected: true,
		}
		view.Accounts = append([]taskGroup{{Options: []taskOption{missing}}}, view.Accounts...)
	}

	return view
}

// layout puts the fields of a type in its cards, before and after the card
// of the transfer information; it says whether the type has that card.
func layout(translate gui.Translator, spec *taskSpec, all []fieldView, exchange *exchangeView,
) (before, after []groupView, infos bool) {
	if len(spec.Groups) == 0 {
		return []groupView{{Empty: true}}, nil, false
	}

	for i := range spec.Groups {
		g := &spec.Groups[i]
		if g.Kind == groupInfos {
			infos = true

			continue
		}

		view := groupView{Columns: cmp.Or(g.Columns, twoColumns)}
		if g.Title != "" {
			view.Title = translate(taskTypesPrefix + "group." + g.Title)
		}

		if g.Note != "" {
			view.Note = translate(g.Note)
		}

		if g.Kind == groupExchange {
			view.Exchange = exchange
		}

		for _, name := range g.Args {
			view.Fields = append(view.Fields, all[spec.arg(name)])
		}

		if infos {
			after = append(after, view)
		} else {
			before = append(before, view)
		}
	}

	return before, after, infos
}

// readArgs takes the posted arguments of a task: the values given (a ticked
// box as "true"; a box left empty and an empty field are left out, but a box
// the stored task has stays, as "false"), checked as far as the form can
// (required, numbers, seconds, JSON, key=value lines). An argument that only
// matters when another one has a value is read only then, and kept as stored
// otherwise; a number of seconds that says the stored duration keeps it as
// written; the arguments of the stored task the description does not know
// are kept. It answers what the form holds, with its errors.
func readArgs(translate gui.Translator, r *http.Request, spec *taskSpec, stored map[string]string,
) (map[string]string, *taskInput, bool) {
	input := &taskInput{values: map[string]string{}, errors: map[string]string{}, posted: true}
	args := map[string]string{}
	valid := true
	posted := func(name string) string { return strings.TrimSpace(r.PostFormValue(argName(spec, name))) }
	special := spec.special()

	for i := range spec.Args {
		arg := &spec.Args[i]
		if special[arg.Name] {
			continue
		}

		raw := posted(arg.Name)
		input.values[arg.Name] = raw
		old, had := stored[arg.Name]

		if keptAsStored(arg, raw, old, had, posted) {
			if had {
				args[arg.Name] = old
			}

			continue
		}

		if msg := checkArg(translate, arg, raw, args); msg != "" {
			input.errors[arg.Name], valid = msg, false
		}

		if had && arg.Kind == argBool && raw != trueValue {
			args[arg.Name] = falseValue
		}
	}

	if spec.has(groupInfos) {
		valid = readInfos(translate, r, stored, args, input) && valid
	}

	if spec.has(groupExchange) {
		valid = readExchange(translate, r, spec, args, input) && valid
	}

	for _, name := range unknownArgs(spec, stored) {
		args[name] = stored[name]
	}

	return args, input, valid
}

// keptAsStored says whether an argument keeps what the task stored: one that
// only matters with another one that does not have its value, or a value the
// user left as the product saved it (a number of seconds that says the
// stored duration among them), which is not judged again.
func keptAsStored(arg *argSpec, raw, old string, had bool, posted func(string) string) bool {
	switch {
	case arg.When != nil && posted(arg.When.Arg) != arg.When.Value:
		return true
	case !had:
		return false
	case arg.Seconds && raw != "" && secondsText(old) == raw:
		return true
	case arg.Millis && raw != "" && millisText(old) == raw:
		return true
	}

	return raw == old && arg.Kind != argPairs
}

// readInfos reads the card of the transfer information: the copy of the
// information of the transfer under way (a box stored stays, as "false"), and
// the rows, as the page that adds a transfer reads them; a stored empty
// object stays when no row is given.
func readInfos(translate gui.Translator, r *http.Request, stored, args map[string]string, input *taskInput) bool {
	if r.PostFormValue(fieldInfoCopy) == trueValue {
		args["copyInfo"], input.values["copyInfo"] = trueValue, trueValue
	} else if _, had := stored["copyInfo"]; had {
		args["copyInfo"] = falseValue
	}

	input.rows = transfers.ReadInfoPairs(r.PostForm)

	info, msg := transfers.InfoObject(translate, input.rows)
	if msg != "" {
		input.infoErr = msg

		return false
	}

	switch {
	case len(info) > 0:
		data, err := json.Marshal(info)
		if err != nil {
			input.infoErr = translate("rules.tasks.error.json")

			return false
		}

		args["info"] = string(data)
	case stored["info"] != "" && len(rowsOf(stored["info"])) == 0:
		args["info"] = stored["info"]
	}

	return true
}

// readExchange reads the exchange of a TRANSFER task: the direction gives "to"
// or "from" to the partner of the account chosen, whose login goes to "as";
// the rule is the one chosen in the list of that direction.
func readExchange(translate gui.Translator, r *http.Request, spec *taskSpec, args map[string]string,
	input *taskInput,
) bool {
	exchange := &input.exchange
	required := translate("rules.tasks.error.required")
	valid := true

	exchange.sens = r.PostFormValue(argName(spec, exchangeSens))
	switch exchange.sens {
	case dirSend, dirReceive:
		exchange.rule = strings.TrimSpace(r.PostFormValue(argName(spec, exchangeRule+exchange.sens)))
		if exchange.rule == "" {
			exchange.ruleErr, valid = required, false
		}
	default:
		exchange.sens, exchange.sensErr, valid = "", translate("rules.tasks.error.sens"), false
	}

	exchange.partner, exchange.login = accountOf(r.PostFormValue(argName(spec, exchangeAccount)))
	if exchange.partner == "" || exchange.login == "" {
		exchange.accountErr, valid = required, false
	}

	using := strings.TrimSpace(r.PostFormValue(argName(spec, "using")))
	input.values["using"] = using

	if !valid {
		return false
	}

	side := "from"
	if exchange.sens == dirSend {
		side = "to"
	}

	args[side], args["as"], args["rule"] = exchange.partner, exchange.login, exchange.rule
	if using != "" {
		args["using"] = using
	}

	return true
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

	if check := arg.check(); !check.Valid(raw) {
		return translate("check." + string(check))
	}

	switch arg.Kind {
	case argInt:
		if msg := checkNumber(translate, arg, raw); msg != "" {
			return msg
		}
	case argDecimal:
		if _, err := strconv.ParseFloat(raw, 64); err != nil {
			return translate("rules.tasks.error.decimal")
		}
	case argDuration:
		value, msg := durationValue(translate, arg, raw)
		if msg != "" {
			return msg
		}

		raw = value
	case argJSON:
		var object map[string]any
		if err := json.Unmarshal([]byte(raw), &object); err != nil {
			return translate("rules.tasks.error.json")
		}
	case argICAP:
		if !icapAddress(raw) {
			return translate("rules.tasks.error.icap")
		}
	case argText, argBool, argOctal, argChoice, argRef, argPairs:
	}

	args[arg.Name] = raw

	return ""
}

// icapAddress says whether a value is the address of an ICAP server as the
// gateway reads it: "icap://" added when it is not there, then a host, its
// port and its service.
func icapAddress(raw string) bool {
	if !strings.HasPrefix(raw, icapScheme) {
		raw = icapScheme + raw
	}

	address, err := url.Parse(raw)

	return err == nil && address.Hostname() != "" && !strings.ContainsAny(address.Hostname(), "/:@")
}

// checkNumber checks a whole number against the bounds of its argument.
func checkNumber(translate gui.Translator, arg *argSpec, raw string) string {
	n, err := strconv.Atoi(raw)

	switch {
	case err != nil:
		return translate("rules.tasks.error.number")
	case arg.Max > 0 && (n < arg.Min || n > arg.Max):
		return translate("rules.tasks.error.range", "min", arg.Min, "max", arg.Max)
	case n < arg.Min:
		return translate("rules.tasks.error.min", "min", arg.Min)
	}

	return ""
}

// secondsValue is a number of seconds typed as the duration stored ("90"
// gives "90s"); a duration of another form (1m30s) is accepted as it is.
func secondsValue(raw string) (string, bool) {
	if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
		return strconv.Itoa(n) + "s", true
	}

	_, err := time.ParseDuration(raw)

	return raw, err == nil
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

// typeChoices are the types to choose from, by family, each with its label,
// its code and what it does (Driss, 2026-10-23: the label in bold and its
// explanation, found by typing), the chosen one selected.
func typeChoices(translate gui.Translator, chosen string) []gui.ComboGroup {
	out := make([]gui.ComboGroup, 0, len(families))

	for _, family := range families {
		group := gui.ComboGroup{Label: translate(taskTypesPrefix + "family." + family)}

		for i := range taskSpecs {
			spec := &taskSpecs[i]
			if spec.Family == family {
				group.Choices = append(group.Choices, gui.ComboChoice{
					Value: spec.Type, Label: typeLabel(translate, spec.Type), Code: spec.Type,
					Text: translate(taskTypesPrefix + spec.Slug + ".text"), Selected: spec.Type == chosen,
				})
			}
		}

		out = append(out, group)
	}

	return out
}

// typeTip is the tooltip of a type: its label and what it does.
func typeTip(translate gui.Translator, spec *taskSpec) string {
	return translate("rules.tasks.tip", "label", typeLabel(translate, spec.Type),
		"text", translate(taskTypesPrefix+spec.Slug+".text"))
}

// TaskNewForm serves GET /webui/rules/{id}/tasks/{chain}/new: the page of a
// new task; with ?type=, that type chosen (the way without JavaScript).
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

// taskForm serves the four pages of the form. A new task offers every type,
// its chosen one shown; a stored task shows its own (its type is kept). A
// stored task of a type the screen does not know is not edited (404: the menu
// does not offer it). "Add an information" without JavaScript brings the page
// back as typed, with one more row, nothing saved.
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
	form, stored := newTaskForm(translate, r, place, list, edit, post)
	spec := specOf(form.Type)
	specs := make([]*taskSpec, 0, len(taskSpecs))

	for i := range taskSpecs {
		specs = append(specs, &taskSpecs[i])
	}

	if edit {
		if spec == nil {
			http.Error(w, "Not found", http.StatusNotFound)

			return
		}

		form.Edit, specs = true, []*taskSpec{spec}
		form.TypeTip = typeTip(translate, spec)
		form.Required = slices.ContainsFunc(spec.Args, func(a argSpec) bool { return a.Required })
	}

	input := inputOf(stored)
	status := 0

	switch {
	case spec == nil:
		if post || form.Type != "" {
			form.TypeErr, status = translate("rules.tasks.error.type"), http.StatusUnprocessableEntity
		}

		form.Type = ""
	case post && transfers.AddsInfo(r.PostForm):
		input = asTyped(translate, r, spec, stored)
	case post:
		answered, posted := h.saveTask(w, r, form, spec, stored, place, list, edit)
		if answered {
			return
		}

		input, form.HasErrors, status = posted, true, http.StatusUnprocessableEntity
	}

	if spec != nil {
		if unknown := unknownArgs(spec, stored); len(unknown) > 0 {
			form.Unknown = translate("rules.tasks.keep_unknown", "names", strings.Join(unknown, ", "))
		}
	}

	if err := h.taskPage(translate, form, specs, spec, input); err != nil {
		h.internal(w, "read the objects the tasks name", err)

		return
	}

	screen := h.screen(translate, form.Title, carried, status)
	screen.Head = taskFormHead()

	if form.Infos != nil {
		screen.Stylesheets = append(screen.Stylesheets, transfers.InfoStylesheet)
		screen.Scripts = append(screen.Scripts, transfers.InfoScript)
	}

	h.shell.Render(w, h.logger, r, screen, taskFormView(form))
}

// newTaskForm is the page of a new task, or of the task edited (its type and
// its arguments stored), the type posted or asked (?type=) otherwise.
func newTaskForm(translate gui.Translator, r *http.Request, place *taskPlace, list []*model.Task, edit, post bool,
) (*taskFormData, map[string]string) {
	carried := ret(r)
	chain := translate(chainLowerKey + place.slug)
	form := &taskFormData{
		Title:     translate("rules.tasks.new_title", "chain", chain),
		Action:    withRet(newTaskURL(place.ruleID, place.slug), carried),
		CancelURL: withRet(tasksTabURL(place.ruleID), carried),
		Ret:       carried,
		NewURL:    newTaskURL(place.ruleID, place.slug),
		Required:  true,
	}

	switch {
	case edit:
		form.Type = list[place.rank].Type
		form.Title = translate("rules.tasks.edit_title", "n", place.rank+1, "chain", chain)
		form.Action = withRet(withVersion(taskURL(place.ruleID, place.slug, place.rank, "/edit"), versionOf(list)), carried)

		return form, list[place.rank].Args
	case post:
		form.Type = r.PostFormValue(fieldType)
	default:
		form.Type = r.URL.Query().Get(fieldType)
	}

	return form, nil
}

// asTyped is the form as posted, nothing checked: the page that asked for one
// more row of transfer information without JavaScript.
func asTyped(translate gui.Translator, r *http.Request, spec *taskSpec, stored map[string]string) *taskInput {
	_, input, _ := readArgs(translate, r, spec, stored)
	input.errors, input.infoErr = map[string]string{}, ""
	input.exchange.sensErr, input.exchange.ruleErr, input.exchange.accountErr = "", "", ""

	for i := range input.rows {
		input.rows[i].Invalid = false
	}

	return input
}

// taskPage builds the blocks of the page: one per type offered, the chosen
// one with what the form holds, and the card of the transfer information.
func (h *Handlers) taskPage(translate gui.Translator, form *taskFormData, specs []*taskSpec, chosen *taskSpec,
	input *taskInput,
) error {
	found, err := h.namesFor(specs...)
	if err != nil {
		return err
	}

	var byDirection map[string][]string

	if slices.ContainsFunc(specs, func(s *taskSpec) bool { return s.has(groupExchange) }) {
		if byDirection, err = h.ruleNames(); err != nil {
			return err
		}
	}

	if !form.Edit {
		form.Families = typeChoices(translate, form.Type)
	}

	var cases []string

	empty := inputOf(nil)

	for _, spec := range specs {
		active := spec == chosen
		shown := empty

		if active {
			shown = input
		}

		all := fields(translate, spec, shown, found)

		var exchange *exchangeView
		if spec.has(groupExchange) {
			exchange = exchangeOf(translate, spec, &shown.exchange, byDirection, found, &all[spec.arg("using")])
		}

		block := typeBlock{
			Type: spec.Type, Label: typeLabel(translate, spec.Type), Tip: typeTip(translate, spec), Active: active,
		}

		var infos bool

		block.Before, block.After, infos = layout(translate, spec, all, exchange)
		form.Blocks = append(form.Blocks, block)

		if infos {
			cases = append(cases, spec.Type)
		}
	}

	if len(cases) > 0 {
		form.Infos = &infosView{Cases: strings.Join(cases, " ")}

		if chosen != nil && chosen.has(groupInfos) {
			form.Infos.Active, form.Infos.Copy = true, input.values["copyInfo"] == trueValue
			form.Infos.Rows, form.Infos.Error = slices.Clone(input.rows), input.infoErr
		}

		form.Infos.Rows = append(form.Infos.Rows, transfers.InfoPair{}) // the row to fill next
	}

	return nil
}

// saveTask reads the posted arguments and rewrites the chain with the task
// added at its end or replaced at its rank (the chain shown by the page, or
// the tab says it changed); it answers whether the page was answered (a
// redirection to the tab, or an internal error), and what the form holds
// otherwise. A refusal of the product goes under the field it names, or in
// the banner.
func (h *Handlers) saveTask(w http.ResponseWriter, r *http.Request, form *taskFormData, spec *taskSpec,
	stored map[string]string, place *taskPlace, list []*model.Task, edit bool,
) (bool, *taskInput) {
	translate := h.shell.Translator(r)

	args, input, valid := readArgs(translate, r, spec, stored)
	if !valid {
		return false, input
	}

	task := &model.Task{Type: spec.Type, Args: args}
	version, done := "", doneTaskAdded
	change := func(current []*model.Task) []*model.Task { return append(current, task) }

	if edit {
		version, done = versionOf(list), doneTaskUpdated
		change = func(current []*model.Task) []*model.Task {
			current[place.rank] = task

			return current
		}
	}

	err := h.modifyChain(place.ruleID, place.chain(), version, change)

	switch {
	case errors.Is(err, errChainChanged):
		h.stale(w, r, place.ruleID)

		return true, nil
	case err == nil:
		liststate.RedirectTo(w, r, withRet(tasksTabURL(place.ruleID), ret(r)),
			url.Values{doneParam: {done}, nameParam: {typeLabel(translate, spec.Type)}})

		return true, nil
	case database.IsValidationError(err):
		refuse(translate, form, spec, input, err.Error())

		return false, input
	default:
		h.internal(w, "save a task of the transfer rule", err)

		return true, nil
	}
}

// refuse puts the refusal of the product under the field it names (an
// argument quoted in its message), or in the banner.
func refuse(translate gui.Translator, form *taskFormData, spec *taskSpec, input *taskInput, msg string) {
	text := translate("rules.tasks.refused_product", "cause", msg)

	for i := range spec.Args {
		name := spec.Args[i].Name
		if !strings.Contains(msg, `"`+name+`"`) {
			continue
		}

		switch {
		case !spec.has(groupExchange) || !slices.Contains((&argGroup{Kind: groupExchange}).covers(), name):
			input.errors[name] = text
		case name == "rule":
			input.exchange.ruleErr = text
		case name == "using":
			input.errors[name] = text
		default:
			input.exchange.accountErr = text
		}

		if name == "info" {
			input.infoErr = text
		}

		return
	}

	form.Refused = text
}

package protoform

import (
	"net/url"
	"slices"
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

// Names of the fields that choose the protocol: the family and the TLS
// checkbox. The blocks of every family are posted together without
// JavaScript, so every option has its own field name; the server reads the
// block of the chosen family only.
const (
	FieldFamily = "protocol"
	FieldTLS    = "tls"
)

// Field is a text or number field of a form: its value, kept when the form
// is refused, and the message of its error.
type Field struct {
	Value string
	Error string
}

// Form is the protocol part of the form of a server, a client or a partner:
// the chosen family and TLS, and the options of every family of the role,
// read from an existing object or posted. Prepare builds what the templates
// "proto/choice" and "proto/options" render.
type Form struct {
	Role   Role
	Family string // chosen family key, "" when none
	TLS    bool
	Error  string // error on the choice of the family

	fields  map[string]Field
	checks  map[string]bool
	selects map[string]string
	lists   map[string][]string

	// Views for the templates, built by Prepare.
	Choice   ChoiceView
	Families []*FamilyView
	TLSBlock BlockView
	// The TLS card is foldable: open when its values differ from the
	// product defaults, its summary says the version and the suites.
	TLSOpen    bool
	TLSSummary string
}

// ChoiceView is the choice of the protocol: the select of the families and
// the TLS checkbox, shown for the families that have a TLS variant.
type ChoiceView struct {
	Families []Choice
	TLSCases string // keys of the families with a TLS variant, space separated: the case of the checkbox
	None     bool   // no family chosen yet: the placeholder is selected
	TLS      bool
	Error    string
}

// FamilyView is the card of a family: its options by kind, in the order of
// the form, and the ones of its TLS variant. The card folds: open when an
// option differs from the product default or carries an error, its summary
// says how many options are set.
type FamilyView struct {
	Key     string
	Label   string
	Note    string
	Plain   BlockView
	TLS     BlockView
	Open    bool
	Summary string
}

// HasOptions tells whether the card has anything to show.
func (v *FamilyView) HasOptions() bool { return v.Plain.any() || v.TLS.any() }

// BlockView is the options of a block, grouped by section when the table
// names one (the themes of a card: identification, synchronisation…); in a
// section, the text, number and select fields share a grid, then the
// checkboxes stand together, then the lists (Driss, 2026-10-20).
type BlockView struct {
	Sections []SectionView
}

// SectionView is one theme of a block: its title, when the block has two
// at least, and its options by kind.
type SectionView struct {
	Key    string
	Label  string
	Fields []OptionView
	Checks []OptionView
	Lists  []OptionView
}

func (b *BlockView) any() bool { return len(b.Sections) > 0 }

// Titled tells whether the sections of the block carry a title: two at least.
func (b *BlockView) Titled() bool { return len(b.Sections) > 1 }

func (b *BlockView) add(v *OptionView) {
	i := slices.IndexFunc(b.Sections, func(s SectionView) bool { return s.Key == v.Section })
	if i < 0 {
		b.Sections = append(b.Sections, SectionView{Key: v.Section, Label: v.SectionLabel})
		i = len(b.Sections) - 1
	}

	section := &b.Sections[i]

	switch v.Kind {
	case "check":
		section.Checks = append(section.Checks, *v)
	case "list":
		section.Lists = append(section.Lists, *v)
	default:
		section.Fields = append(section.Fields, *v)
	}
}

// FieldList lists the fields of a block, whatever their section: the TLS
// block template renders its select and its list apart.
func (b *BlockView) FieldList() []OptionView {
	var out []OptionView
	for _, s := range b.Sections {
		out = append(out, s.Fields...)
	}

	return out
}

// ListList lists the lists of a block, whatever their section.
func (b *BlockView) ListList() []OptionView {
	var out []OptionView
	for _, s := range b.Sections {
		out = append(out, s.Lists...)
	}

	return out
}

// OptionView is one option as the template renders it.
type OptionView struct {
	Name    string
	Kind    string // "text", "number", "check", "select", "list", "duration"
	Label   string
	Hint    string
	Code    string // the configuration key, shown as a technical hint
	Short   bool
	Min     int
	Max     int
	Value   string
	Error   string
	Checked bool
	Choices []Choice
	// The theme of the option in its card: its key and its translated label.
	Section      string
	SectionLabel string
	// A duration: its parts, as entered.
	Hours, Mins, Secs string
}

// NewForm builds the protocol part of a form for a role, with the defaults
// of the product selected.
func NewForm(role Role) *Form {
	f := &Form{
		Role:    role,
		fields:  map[string]Field{},
		checks:  map[string]bool{},
		selects: map[string]string{},
		lists:   map[string][]string{},
	}

	for _, opt := range f.options() {
		switch {
		case opt.Kind == KindSelect:
			f.selects[opt.Name] = opt.Default
		case opt.Kind == KindCheck && opt.DefaultOn:
			f.checks[opt.Name] = true
		}
	}

	return f
}

// options lists every option of the role, of every family, then the TLS block.
func (f *Form) options() []*Option {
	var out []*Option

	for _, family := range Families() {
		out = append(out, optionsOf(family.Key, f.Role)...)
	}

	return append(out, tlsOptionList()...)
}

// chosen lists the options that apply to the chosen family and TLS: what
// Validate checks and Apply stores.
func (f *Form) chosen() []*Option {
	var out []*Option

	for _, opt := range optionsOf(f.Family, f.Role) {
		if !opt.TLSOnly || f.tlsOn() {
			out = append(out, opt)
		}
	}

	if f.tlsOn() {
		out = append(out, tlsOptionList()...)
	}

	return out
}

// tlsOn tells whether the chosen protocol is a TLS variant: the checkbox
// counts only for a family that has one.
func (f *Form) tlsOn() bool { return f.TLS && familyOf(f.Family).HasTLS() }

// Fill reads the values of an existing object into the form.
func (f *Form) Fill(protocol string, cfg map[string]any) {
	p := ProtocolOf(protocol)
	f.Family, f.TLS = p.Family.Key, p.TLS

	for _, opt := range f.chosen() {
		switch opt.Kind {
		case KindCheck:
			f.checks[opt.Name] = CfgBool(cfg, opt.Cfg) || (opt.DefaultOn && !CfgHas(cfg, opt.Cfg))
		case KindTri:
			f.selects[opt.Name] = triOf(cfg, opt)
		case KindSelect:
			if v := CfgString(cfg, opt.Cfg); v != "" {
				f.selects[opt.Name] = v
			}
		case KindList:
			f.lists[opt.Name] = CfgList(cfg, opt.Cfg)
		case KindDuration:
			value := CfgString(cfg, opt.Cfg)
			f.fields[opt.Name] = Field{Value: value}
			hours, minutes, seconds := durationParts(value)
			f.setDurationParts(opt.Name, hours, minutes, seconds)
		default:
			f.fields[opt.Name] = Field{Value: CfgString(cfg, opt.Cfg)}
		}
	}
}

// Names of the fields of the parts of a duration.
const (
	suffixHours   = "_h"
	suffixMinutes = "_m"
	suffixSeconds = "_s"
)

func (f *Form) setDurationParts(name, hours, minutes, seconds string) {
	f.fields[name+suffixHours] = Field{Value: hours}
	f.fields[name+suffixMinutes] = Field{Value: minutes}
	f.fields[name+suffixSeconds] = Field{Value: seconds}
}

func (f *Form) durationParts(name string) (hours, minutes, seconds string) {
	return f.fields[name+suffixHours].Value, f.fields[name+suffixMinutes].Value, f.fields[name+suffixSeconds].Value
}

// Read takes the posted values into the form: the family, the TLS checkbox
// and every option of every family, so that a refused form comes back as the
// user filled it.
func (f *Form) Read(values url.Values) {
	f.Family = values.Get(FieldFamily)
	f.TLS = values.Get(FieldTLS) != ""

	for _, opt := range f.options() {
		switch opt.Kind {
		case KindCheck:
			f.checks[opt.Name] = values.Get(opt.Name) != ""
		case KindSelect, KindTri:
			if values.Has(opt.Name) {
				f.selects[opt.Name] = values.Get(opt.Name)
			}
		case KindList:
			f.lists[opt.Name] = nonEmpty(values[opt.Name])
		case KindDuration:
			f.setDurationParts(opt.Name, strings.TrimSpace(values.Get(opt.Name+suffixHours)),
				strings.TrimSpace(values.Get(opt.Name+suffixMinutes)), strings.TrimSpace(values.Get(opt.Name+suffixSeconds)))
			// A part that is not a whole number leaves the duration empty:
			// Validate marks it.
			composed, err := composeDuration(f.durationParts(opt.Name))
			if err != nil {
				composed = ""
			}

			f.fields[opt.Name] = Field{Value: composed}
		default:
			f.fields[opt.Name] = Field{Value: strings.TrimSpace(values.Get(opt.Name))}
		}
	}
}

func nonEmpty(values []string) []string {
	out := make([]string, 0, len(values))

	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}

	return out
}

// Validate checks the family and the numbers of the chosen options; it marks
// the errors on the form and returns whether the protocol part is valid.
func (f *Form) Validate(translate gui.Translator) bool {
	valid := true

	if !IsFamily(f.Family) {
		f.Family, f.Error = "", translate("proto.choose_error")
		valid = false
	}

	for _, opt := range f.chosen() {
		if opt.Kind == KindDuration {
			if _, err := composeDuration(f.durationParts(opt.Name)); err != nil {
				f.Fail(opt.Name, translate(Or(opt.ErrorKey, "proto.error_duration")))
				valid = false
			}

			continue
		}

		if opt.Parse == nil || (opt.Kind != KindNumber && opt.Kind != KindText) {
			continue
		}

		fld := f.fields[opt.Name]
		if fld.Value == "" {
			continue
		}

		if _, err := opt.Parse(fld.Value); err != nil {
			fld.Error = translate(Or(opt.ErrorKey, "proto.error_number"))
			f.fields[opt.Name] = fld
			valid = false
		}
	}

	return valid
}

// Protocol is the key the product stores for the chosen family and TLS, ""
// when the family is unknown.
func (f *Form) Protocol() string {
	key, _ := Resolve(f.Family, f.TLS)

	return key
}

// Apply writes the chosen options into a configuration, with the keys the
// product reads; an empty text or number leaves the product default.
func (f *Form) Apply() model.Map[any] {
	cfg := model.Map[any]{}

	for _, opt := range f.chosen() {
		switch opt.Kind {
		case KindCheck:
			cfg[opt.Cfg] = f.checks[opt.Name]
		case KindTri:
			// The client's value applies when nothing is chosen: the flag
			// stays absent, never written as false.
			switch f.selects[opt.Name] {
			case TriOn:
				cfg[opt.Cfg] = !opt.Inverted
			case TriOff:
				cfg[opt.Cfg] = opt.Inverted
			}
		case KindSelect:
			if v := f.selects[opt.Name]; v != "" {
				cfg[opt.Cfg] = v
			}
		case KindList:
			if values := f.lists[opt.Name]; len(values) > 0 {
				cfg[opt.Cfg] = values
			}
		case KindNumber:
			if v := f.fields[opt.Name].Value; v != "" {
				if n, err := opt.Parse(v); err == nil {
					cfg[opt.Cfg] = n
				}
			}
		default:
			if v := f.fields[opt.Name].Value; v != "" {
				cfg[opt.Cfg] = v
			}
		}
	}

	return cfg
}

// Fail marks an option with the refusal of the model, when the cause names
// one (an AS2 partner that handles asynchronous MDNs without an address).
func (f *Form) Fail(name, msg string) {
	fld := f.fields[name]
	fld.Error = msg
	f.fields[name] = fld
}

// HasErrors tells whether Validate marked anything.
func (f *Form) HasErrors() bool {
	if f.Error != "" {
		return true
	}

	for _, fld := range f.fields {
		if fld.Error != "" {
			return true
		}
	}

	return false
}

// Prepare builds the views the templates render: the choice of the family,
// one card per family with its options, the TLS block.
func (f *Form) Prepare(translate gui.Translator) {
	var tlsCases []string

	f.Choice = ChoiceView{TLS: f.TLS, Error: f.Error, None: f.Family == ""}
	f.Families = nil

	for _, family := range Families() {
		f.Choice.Families = append(f.Choice.Families,
			Choice{Value: family.Key, Label: family.Label, Selected: family.Key == f.Family})

		if family.HasTLS() {
			tlsCases = append(tlsCases, family.Key)
		}

		view := &FamilyView{Key: family.Key, Label: family.Label}
		if note, ok := familyNotes[family.Key]; ok {
			view.Note = translate(note)
		}

		set := 0

		for _, opt := range optionsOf(family.Key, f.Role) {
			if opt.TLSOnly {
				view.TLS.add(f.view(translate, opt))
			} else {
				view.Plain.add(f.view(translate, opt))
			}

			if f.isSet(opt) {
				set++
			}

			if f.fields[opt.Name].Error != "" {
				view.Open = true
			}
		}

		view.Open = view.Open || set > 0
		view.Summary = optionsSummary(translate, set)
		f.Families = append(f.Families, view)
	}

	f.Choice.TLSCases = strings.Join(tlsCases, " ")
	f.TLSBlock = BlockView{}

	for _, opt := range tlsOptionList() {
		f.TLSBlock.add(f.view(translate, opt))
	}

	f.TLSOpen, f.TLSSummary = f.tlsSummary(translate)
}

// isSet tells whether an option differs from the product default: a checked
// box (an unchecked one when the product takes the flag as on), a filled
// text or number, a select away from its default, a list with something
// chosen.
func (f *Form) isSet(opt *Option) bool {
	switch opt.Kind {
	case KindCheck:
		return f.checks[opt.Name] != opt.DefaultOn
	case KindSelect:
		return f.selects[opt.Name] != "" && f.selects[opt.Name] != opt.Default
	case KindTri:
		return f.selects[opt.Name] != TriClient
	case KindList:
		return len(f.lists[opt.Name]) > 0
	default:
		return f.fields[opt.Name].Value != ""
	}
}

// triOf reads a flag the product takes in three states: the client's value
// (absent), on or off, as the form shows it.
func triOf(cfg map[string]any, opt *Option) string {
	if !CfgHas(cfg, opt.Cfg) {
		return TriClient
	}

	if CfgBool(cfg, opt.Cfg) != opt.Inverted {
		return TriOn
	}

	return TriOff
}

// optionsSummary says how many options of a card are set, or that the
// product defaults apply.
func optionsSummary(translate gui.Translator, set int) string {
	switch set {
	case 0:
		return translate("proto.options_default")
	case 1:
		return translate("proto.options_set_one")
	default:
		return translate("proto.options_set", "n", set)
	}
}

// tlsSummary says whether the TLS card holds something else than the
// product defaults, and sums it up: the minimum version and the suites.
func (f *Form) tlsSummary(translate gui.Translator) (bool, string) {
	version := f.selects[FieldMinTLS]
	suites := f.lists[FieldTLSCiphers]
	open := (version != "" && version != protoutils.TLSv12) || len(suites) > 0

	label := TLSVersionLabel(version)
	chosen := translate("proto.opt.default_ciphers")

	switch len(suites) {
	case 0:
	case 1:
		chosen = translate("proto.tls_suite_one")
	default:
		chosen = translate("proto.tls_suites", "n", len(suites))
	}

	return open, translate("proto.tls_summary", "version", label, "suites", chosen)
}

// view renders one option with its current value.
func (f *Form) view(translate gui.Translator, opt *Option) *OptionView {
	item := &OptionView{
		Name: opt.Name, Label: translate(opt.Label), Code: opt.Cfg, Short: opt.Short, Min: opt.Min, Max: opt.Max,
	}
	if opt.Hint != "" {
		item.Hint = translate(opt.Hint)
	}

	if opt.Section != "" {
		item.Section, item.SectionLabel = opt.Section, translate(opt.Section)
	}

	switch opt.Kind {
	case KindCheck:
		item.Kind, item.Checked = "check", f.checks[opt.Name]
	case KindDuration:
		item.Kind, item.Value, item.Error = "duration", f.fields[opt.Name].Value, f.fields[opt.Name].Error
		item.Hours, item.Mins, item.Secs = f.durationParts(opt.Name)
	case KindSelect:
		item.Kind, item.Choices = "select", opt.Choices(translate, f.selects[opt.Name])
	case KindTri:
		item.Kind, item.Choices = "select", triChoices(translate, f.selects[opt.Name])
	case KindList:
		item.Kind = "list"
		if opt.Cfg == cfgCipherSuites {
			item.Choices = TLSCipherChoices(translate, f.lists[opt.Name])
		} else {
			item.Choices = sftpAlgoChoices(translate, opt.Values(f.Role), f.lists[opt.Name])
		}
	case KindNumber:
		item.Kind, item.Value, item.Error = "number", f.fields[opt.Name].Value, f.fields[opt.Name].Error
	default:
		item.Kind, item.Value, item.Error = "text", f.fields[opt.Name].Value, f.fields[opt.Name].Error
	}

	return item
}

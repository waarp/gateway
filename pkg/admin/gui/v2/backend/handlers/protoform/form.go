package protoform

import (
	"net/url"
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// Names of the fields that choose the protocol: the family and the TLS
// checkbox. The blocks of every family are posted together without
// JavaScript, so every option has its own field name; the server reads the
// block of the chosen family only.
const (
	FieldFamily = "protocol"
	FieldTLS    = "tls"
)

// TemplateFile is the file of the shared templates ("proto/field",
// "proto/choice", "proto/options"), to give the shell of every page that
// renders a protocol form or its options.
const TemplateFile = "proto/options.gohtml"

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
// the form, and the ones of its TLS variant.
type FamilyView struct {
	Key   string
	Label string
	Note  string
	Plain BlockView
	TLS   BlockView
}

// HasOptions tells whether the card has anything to show.
func (v *FamilyView) HasOptions() bool { return v.Plain.any() || v.TLS.any() }

// BlockView groups options by kind: the text, number and select fields in a
// grid, then the checkboxes, then the lists.
type BlockView struct {
	Fields []OptionView
	Checks []OptionView
	Lists  []OptionView
}

func (b *BlockView) any() bool { return len(b.Fields)+len(b.Checks)+len(b.Lists) > 0 }

func (b *BlockView) add(v *OptionView) {
	switch v.Kind {
	case "check":
		b.Checks = append(b.Checks, *v)
	case "list":
		b.Lists = append(b.Lists, *v)
	default:
		b.Fields = append(b.Fields, *v)
	}
}

// OptionView is one option as the template renders it.
type OptionView struct {
	Name    string
	Kind    string // "text", "number", "check", "select", "list"
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
		if opt.Kind == KindSelect {
			f.selects[opt.Name] = opt.Default
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
			f.checks[opt.Name] = CfgBool(cfg, opt.Cfg)
		case KindSelect:
			if v := CfgString(cfg, opt.Cfg); v != "" {
				f.selects[opt.Name] = v
			}
		case KindList:
			f.lists[opt.Name] = CfgList(cfg, opt.Cfg)
		default:
			f.fields[opt.Name] = Field{Value: CfgString(cfg, opt.Cfg)}
		}
	}
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
		case KindSelect:
			if values.Has(opt.Name) {
				f.selects[opt.Name] = values.Get(opt.Name)
			}
		case KindList:
			f.lists[opt.Name] = nonEmpty(values[opt.Name])
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
		if opt.Kind != KindNumber {
			continue
		}

		fld := f.fields[opt.Name]
		if fld.Value == "" {
			continue
		}

		if _, err := opt.Parse(fld.Value); err != nil {
			fld.Error = translate("proto.error_number")
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

		for _, opt := range optionsOf(family.Key, f.Role) {
			if opt.TLSOnly {
				view.TLS.add(f.view(translate, opt))
			} else {
				view.Plain.add(f.view(translate, opt))
			}
		}

		f.Families = append(f.Families, view)
	}

	f.Choice.TLSCases = strings.Join(tlsCases, " ")
	f.TLSBlock = BlockView{}

	for _, opt := range tlsOptionList() {
		f.TLSBlock.add(f.view(translate, opt))
	}
}

// view renders one option with its current value.
func (f *Form) view(translate gui.Translator, opt *Option) *OptionView {
	v := &OptionView{
		Name: opt.Name, Label: translate(opt.Label), Code: opt.Cfg, Short: opt.Short, Min: opt.Min, Max: opt.Max,
	}
	if opt.Hint != "" {
		v.Hint = translate(opt.Hint)
	}

	switch opt.Kind {
	case KindCheck:
		v.Kind, v.Checked = "check", f.checks[opt.Name]
	case KindSelect:
		v.Kind, v.Choices = "select", opt.Choices(translate, f.selects[opt.Name])
	case KindList:
		v.Kind = "list"
		if opt.Cfg == cfgCipherSuites {
			v.Choices = TLSCipherChoices(translate, f.lists[opt.Name])
		} else {
			v.Choices = multi(opt.Values(f.Role), f.lists[opt.Name])
		}
	case KindNumber:
		v.Kind, v.Value, v.Error = "number", f.fields[opt.Name].Value, f.fields[opt.Name].Error
	default:
		v.Kind, v.Value, v.Error = "text", f.fields[opt.Name].Value, f.fields[opt.Name].Error
	}

	return v
}

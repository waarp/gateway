package protoform

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"
	"golang.org/x/exp/constraints"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/format"
)

// Role is what the object is for the protocol: a server listens, a client
// connects, a partner is connected to. Each role has its own options.
type Role int

const (
	RoleServer Role = iota
	RoleClient
	RolePartner
)

// Kind is how an option is entered and shown.
type Kind int

const (
	KindText   Kind = iota // a text
	KindNumber             // a whole number, with bounds
	KindCheck              // a checkbox
	KindSelect             // one value among a list
	KindList               // several values among a list, in order (a Tags field)
	// KindTri is a flag the product reads in three states: absent (the value
	// of the client applies), on, off. A select with the three states; an
	// absent flag is never written as false.
	KindTri
	// KindDuration is a duration entered in hours, minutes and seconds, stored
	// as the product reads it ("1h30m", "10s"); empty: the product default.
	KindDuration
)

// Keys of the protocol options in the configuration of an object, as the
// product reads them (the same as the historic pages wrote).
const (
	cfgServerLogin            = "serverLogin"
	cfgBlockSize              = "blockSize"
	cfgNoFinalHash            = "noFinalHash"
	cfgFinalHashAlgo          = "finalHashAlgo"
	cfgCheckBlockHash         = "checkBlockHash"
	cfgMinTLSVersion          = "minTLSVersion"
	cfgCipherSuites           = "cipherSuites"
	cfgDisablePassiveMode     = "disablePassiveMode"
	cfgDisableActiveMode      = "disableActiveMode"
	cfgDisableEPSV            = "disableEPSV"
	cfgUseImplicitTLS         = "useImplicitTLS"
	cfgDisableTLSSessionReuse = "disableTLSSessionReuse"
	cfgPassiveModeMinPort     = "passiveModeMinPort"
	cfgPassiveModeMaxPort     = "passiveModeMaxPort"
	cfgEnableActiveMode       = "enableActiveMode"
	cfgActiveModeAddress      = "activeModeAddress"
	cfgActiveModeMinPort      = "activeModeMinPort"
	cfgActiveModeMaxPort      = "activeModeMaxPort"
	cfgTLSRequirement         = "tlsRequirement"
	cfgLogin                  = "login"
	cfgDisableRestart         = "disableRestart"
	cfgDisableCheckpoints     = "disableCheckpoints"
	cfgCheckpointSize         = "checkpointSize"
	cfgCheckpointWindow       = "checkpointWindow"
	cfgUseNSDU                = "useNSDU"
	cfgCompatibilityMode      = "compatibilityMode"
	cfgMaxMessageSize         = "maxMessageSize"
	cfgDisablePreConnection   = "disablePreConnection"
	cfgExpectsAck             = "expectsAck"
	cfgAckTimeout             = "ackTimeout"
	cfgNoPI37                 = "noPI37"
	cfgKeyExchanges           = "keyExchanges"
	cfgCiphers                = "ciphers"
	cfgMACs                   = "macs"
	cfgUseStat                = "useStat"
	cfgNoConcurrentReads      = "disableClientConcurrentReads"
	cfgMaxFileSize            = "maxFileSize"
	cfgMDNSignatureAlgo       = "mdnSignatureAlgorithm"
	cfgSignatureAlgo          = "signatureAlgorithm"
	cfgEncryptionAlgo         = "encryptionAlgorithm"
	cfgAsyncMDNAddress        = "asyncMDNAddress"
	cfgHandleAsyncMDN         = "handleAsyncMDN"

	maxPort = 65535
)

// FieldMinTLS and FieldTLSCiphers are the fields of the TLS block, the same
// for every family.
const (
	FieldMinTLS     = "min_tls"
	FieldTLSCiphers = "tls_ciphers"
)

// The three states of a KindTri option, as the form posts them.
const (
	TriClient = ""
	TriOn     = "on"
	TriOff    = "off"
)

var errPortOutOfRange = errors.New("port out of range")

// r66HashAlgos are the hash algorithms the R66 module accepts for the final
// check of a transfer, the product's default first (the module keeps its
// list in an internal package; the test checks that the module accepts each
// of these).
//
//nolint:gochecknoglobals // reference data of the product
var r66HashAlgos = []string{"SHA-256", "ADLER32", "MD5", "SHA-1", "SHA-384", "SHA-512"}

// Option is one option of a family for one or several roles: how it is
// entered (field name, kind, bounds, choices), how it is stored (the key the
// product reads) and how a details page shows it. The theme it belongs to
// (identification, synchronisation…) is the section of the table of the
// families it sits in (families.go, placed).
type Option struct {
	Name  string // form field name, unique in a form
	Cfg   string // configuration key the product reads
	Kind  Kind
	Roles []Role

	Label string // catalogue key of the label
	Hint  string // catalogue key of the hint, optional
	Short bool   // a short field (a number, a port)
	// TLSOnly options belong to the TLS variant of the family (the TLS
	// requirement of an FTPS server): shown with the TLS checkbox on.
	TLSOnly bool

	// KindNumber: the bounds shown to the browser and the parser that
	// validates and types the value the product stores. A KindText with a
	// Parse is checked too (a duration); ErrorKey is the catalogue key of
	// its error, "proto.error_number" by default.
	Min, Max int
	Parse    func(string) (any, error)
	ErrorKey string

	// KindSelect: the choices with the current one selected, and the value
	// chosen by default; KindList: the values offered for a role.
	Choices func(gui.Translator, string) []Choice
	Default string
	Values  func(Role) []string

	// The row of the details page: its label when it differs from the
	// label of the field (a "disable X" checkbox reads "X: enabled"), the
	// words of a checkbox (enabled/disabled by default, checked/unchecked
	// with Checked) and whether the stored value is the opposite of the row
	// (Inverted: "disableX" stored true shows "X: disabled").
	Row      string
	Checked  bool
	Inverted bool
	// DefaultOn is a KindCheck flag the product takes as on when it is
	// absent: the box is checked by default and the row reads "enabled".
	DefaultOn bool
	// DefaultKey is the catalogue key of the row when the option is not
	// set: the product's default.
	DefaultKey string
	// Bytes marks a number of bytes: the row of the details page says it
	// with its unit ("10 Mo (10 485 760 octets)", "32 700 octets").
	Bytes bool
}

func (o *Option) forRole(role Role) bool { return slices.Contains(o.Roles, role) }

// parseUnsigned and parseSigned type a whole number for the product.
func parseUnsigned[T constraints.Unsigned](s string) (any, error) {
	n, err := internal.ParseUint[T](s)
	if err != nil {
		return nil, err
	}

	return n, nil
}

func parseSigned[T constraints.Signed](s string) (any, error) {
	n, err := internal.ParseInt[T](s)
	if err != nil {
		return nil, err
	}

	return n, nil
}

// parsePort types a TCP port, 1 to 65535.
func parsePort(s string) (any, error) {
	port, err := internal.ParseUint[uint16](s)
	if err != nil {
		return nil, err
	}

	if port == 0 {
		return nil, errPortOutOfRange
	}

	return port, nil
}

// parseDuration checks a duration as the product reads it ("10s", "1m",
// "1h"); the product stores the text.
func parseDuration(s string) (any, error) {
	if _, err := time.ParseDuration(s); err != nil {
		return nil, fmt.Errorf("invalid duration: %w", err)
	}

	return s, nil
}

// composeDuration writes a duration as the product reads it ("1h30m", "10s")
// from its parts, each one a whole number of hours, minutes or seconds, or
// empty; all empty or zero: "" (the product default).
func composeDuration(hours, minutes, seconds string) (string, error) {
	var out strings.Builder

	for _, part := range []struct{ value, unit string }{{hours, "h"}, {minutes, "m"}, {seconds, "s"}} {
		if part.value == "" {
			continue
		}

		n, err := internal.ParseUint[uint32](part.value)
		if err != nil {
			return "", fmt.Errorf("invalid duration: %w", err)
		}

		if n > 0 {
			fmt.Fprintf(&out, "%d%s", n, part.unit)
		}
	}

	return out.String(), nil
}

// durationParts splits a duration the product stores into whole hours,
// minutes and seconds, each one empty when zero; unreadable: nothing.
func durationParts(stored string) (hours, minutes, seconds string) {
	d, err := time.ParseDuration(stored)
	if err != nil || d <= 0 {
		return "", "", ""
	}

	part := func(unit time.Duration) string {
		n := int64(d / unit)
		d %= unit

		if n == 0 {
			return ""
		}

		return fmt.Sprint(n)
	}

	return part(time.Hour), part(time.Minute), part(time.Second)
}

// FormatDuration says a stored duration in words ("1 h 30 min", "10 s");
// one the product could not read is shown as it is.
func FormatDuration(translate gui.Translator, stored string) string {
	hours, minutes, seconds := durationParts(stored)

	var parts []string

	for _, part := range []struct{ value, key string }{
		{hours, "proto.duration.h"}, {minutes, "proto.duration.min"}, {seconds, "proto.duration.s"},
	} {
		if part.value != "" {
			parts = append(parts, part.value+" "+translate(part.key))
		}
	}

	if len(parts) == 0 {
		return stored
	}

	return strings.Join(parts, " ")
}

// ------------------------------------------------- configuration reads ----

// CfgString reads a text or a number of a configuration as a string.
func CfgString(cfg map[string]any, key string) string {
	switch v := cfg[key].(type) {
	case nil:
		return ""
	case string:
		return v
	case json.Number:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprint(v)
	}
}

// CfgBool reads a flag of a configuration.
func CfgBool(cfg map[string]any, key string) bool {
	b, _ := cfg[key].(bool)

	return b
}

// CfgHas tells whether a configuration holds a value for the key: a flag the
// product reads in three states is absent when the client's value applies.
func CfgHas(cfg map[string]any, key string) bool {
	v, ok := cfg[key]

	return ok && v != nil
}

// CfgList reads a list of texts of a configuration.
func CfgList(cfg map[string]any, key string) []string {
	switch v := cfg[key].(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, fmt.Sprint(item))
		}

		return out
	default:
		return nil
	}
}

// --------------------------------------------------------- details rows ----

// KV is a row of a key/value list.
type KV struct {
	Label string
	Value string
	Mono  bool
	// URL leads to the sheet of the value when it has one (the rule, the
	// partner of a transfer); empty, the value is text.
	URL string
	// Section is the theme of the row, translated, when the family names
	// one: the details page groups its rows by it (Grouped).
	Section string
}

// RowGroup is the rows of one section of a details page, under its title;
// a page without sections has one group without title.
type RowGroup struct {
	Title string
	Rows  []KV
}

// Grouped gathers the rows of a details page by section, in the order of
// first appearance; without two sections at least, one group without title.
func Grouped(rows []KV) []RowGroup {
	var groups []RowGroup

	for _, row := range rows {
		i := slices.IndexFunc(groups, func(g RowGroup) bool { return g.Title == row.Section })
		if i < 0 {
			groups = append(groups, RowGroup{Title: row.Section})
			i = len(groups) - 1
		}

		groups[i].Rows = append(groups[i].Rows, row)
	}

	if len(groups) == 1 {
		groups[0].Title = ""
	}

	return groups
}

// Rows lists the protocol options of an object as its details page shows
// them: the effect for the operator first, in the order of the form, then
// the TLS block for a TLS variant; "no option" when the family has none.
func Rows(translate gui.Translator, role Role, p *Protocol, cfg map[string]any) []KV {
	var rows []KV

	for _, opt := range optionsOf(p.Family.Key, role) {
		if opt.TLSOnly && !p.TLS {
			continue
		}

		rows = append(rows, opt.sectioned(translate, cfg))
	}

	if p.TLS {
		for _, opt := range tlsOptionList() {
			rows = append(rows, opt.sectioned(translate, cfg))
		}
	}

	if len(rows) == 0 {
		rows = append(rows, KV{Label: translate("proto.options_label"), Value: translate("proto.opt.none")})
	}

	return rows
}

// row is the details row of one option.
func (o *Option) row(translate gui.Translator, cfg map[string]any) KV {
	label := translate(o.Label)
	if o.Row != "" {
		label = translate(o.Row)
	}

	switch o.Kind {
	case KindCheck:
		on := CfgBool(cfg, o.Cfg) != o.Inverted
		if o.DefaultOn && !CfgHas(cfg, o.Cfg) {
			on = true
		}

		return KV{Label: label, Value: translate(o.word(on))}
	case KindDuration:
		value := CfgString(cfg, o.Cfg)
		if value == "" {
			return KV{Label: label, Value: translate(o.DefaultKey)}
		}

		return KV{Label: label, Value: FormatDuration(translate, value)}
	case KindTri:
		if !CfgHas(cfg, o.Cfg) {
			return KV{Label: label, Value: translate(o.DefaultKey)}
		}

		return KV{Label: label, Value: translate(o.word(CfgBool(cfg, o.Cfg) != o.Inverted))}
	case KindSelect:
		return KV{Label: label, Value: o.choiceLabel(translate, CfgString(cfg, o.Cfg))}
	case KindList:
		values := CfgList(cfg, o.Cfg)
		if len(values) == 0 {
			return KV{Label: label, Value: translate(o.DefaultKey)}
		}

		if o.Cfg == cfgCipherSuites {
			labels := make([]string, 0, len(values))
			for _, name := range values {
				labels = append(labels, CipherLabel(name))
			}

			values = labels
		}

		return KV{Label: label, Value: strings.Join(values, ", "), Mono: true}
	default:
		value := CfgString(cfg, o.Cfg)
		if value == "" {
			return KV{Label: label, Value: translate(o.DefaultKey)}
		}

		if o.Bytes {
			if n, err := strconv.ParseInt(value, 10, 64); err == nil {
				return KV{Label: label, Value: format.WholeBytes(translate, n)}
			}
		}

		return KV{Label: label, Value: value, Mono: true}
	}
}

// sectioned is the row of the option with the theme it belongs to.
func (p placed) sectioned(translate gui.Translator, cfg map[string]any) KV {
	row := p.row(translate, cfg)
	if p.Section != "" {
		row.Section = translate(p.Section)
	}

	return row
}

// word is the catalogue key of the state of a checkbox on a details page.
func (o *Option) word(on bool) string {
	switch {
	case o.Checked && on:
		return "proto.opt.checked"
	case o.Checked:
		return "proto.opt.unchecked"
	case on:
		return "proto.opt.enabled"
	default:
		return "proto.opt.disabled"
	}
}

// choiceLabel is the label of the chosen value of a select, the default
// value when none is stored, the raw value when the product knows a value
// the interface does not.
func (o *Option) choiceLabel(translate gui.Translator, value string) string {
	if value == "" {
		value = o.Default
	}

	for _, c := range o.Choices(translate, value) {
		if c.Selected {
			return c.Label
		}
	}

	if value == "" && o.DefaultKey != "" {
		return translate(o.DefaultKey)
	}

	return value
}

// GUIChoices gives the choices of a list to the kit.
func GUIChoices(choices []Choice) []gui.Choice {
	out := make([]gui.Choice, 0, len(choices))
	for _, c := range choices {
		out = append(out, gui.Choice{Value: c.Value, Label: c.Label, Selected: c.Selected})
	}

	return out
}

// Or returns the first text, or the second when the first is empty: the
// refusal of the model when it has one, the generic sentence otherwise.
func Or(text, fallback string) string {
	if text != "" {
		return text
	}

	return fallback
}

// OptionsMeta sums up the protocol options card for its folded heading.
func OptionsMeta(translate gui.Translator, rows []KV) string {
	n := len(rows)

	switch n {
	case 0:
		return translate("proto.options_meta_none")
	case 1:
		return translate("proto.options_meta_one")
	default:
		return translate("proto.options_meta", "n", n)
	}
}

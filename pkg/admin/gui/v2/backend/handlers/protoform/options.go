package protoform

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"code.waarp.fr/lib/gui"
	"golang.org/x/exp/constraints"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
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
	KindList               // several values among a list, in order (dual list)
)

// Keys of the protocol options in the configuration of an object, as the
// product reads them (the same as the historic pages wrote).
const (
	cfgServerLogin          = "serverLogin"
	cfgBlockSize            = "blockSize"
	cfgNoFinalHash          = "noFinalHash"
	cfgFinalHashAlgo        = "finalHashAlgo"
	cfgCheckBlockHash       = "checkBlockHash"
	cfgMinTLSVersion        = "minTLSVersion"
	cfgCipherSuites         = "cipherSuites"
	cfgDisablePassiveMode   = "disablePassiveMode"
	cfgDisableActiveMode    = "disableActiveMode"
	cfgPassiveModeMinPort   = "passiveModeMinPort"
	cfgPassiveModeMaxPort   = "passiveModeMaxPort"
	cfgEnableActiveMode     = "enableActiveMode"
	cfgActiveModeAddress    = "activeModeAddress"
	cfgActiveModeMinPort    = "activeModeMinPort"
	cfgActiveModeMaxPort    = "activeModeMaxPort"
	cfgTLSRequirement       = "tlsRequirement"
	cfgDisableRestart       = "disableRestart"
	cfgDisableCheckpoints   = "disableCheckpoints"
	cfgCheckpointSize       = "checkpointSize"
	cfgCheckpointWindow     = "checkpointWindow"
	cfgCompatibilityMode    = "compatibilityMode"
	cfgMaxMessageSize       = "maxMessageSize"
	cfgDisablePreConnection = "disablePreConnection"
	cfgKeyExchanges         = "keyExchanges"
	cfgCiphers              = "ciphers"
	cfgMACs                 = "macs"
	cfgMaxFileSize          = "maxFileSize"
	cfgMDNSignatureAlgo     = "mdnSignatureAlgorithm"

	maxPort = 65535
)

// FieldMinTLS and FieldTLSCiphers are the fields of the TLS block, the same
// for every family.
const (
	FieldMinTLS     = "min_tls"
	FieldTLSCiphers = "tls_ciphers"
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
// product reads) and how a details page shows it.
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
	// validates and types the value the product stores.
	Min, Max int
	Parse    func(string) (any, error)

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
	// DefaultKey is the catalogue key of the row when the option is not
	// set: the product's default.
	DefaultKey string
}

func (o *Option) forRole(role Role) bool { return slices.Contains(o.Roles, role) }

//nolint:gochecknoglobals // reference data of the screens
var (
	all             = []Role{RoleServer, RoleClient, RolePartner}
	serverOnly      = []Role{RoleServer}
	clientOnly      = []Role{RoleClient}
	serverAndClient = []Role{RoleServer, RoleClient}

	// The options of each family (familyOptions), in the order of the form,
	// for every role; the note shown at the top of the card of a family
	// (familyNotes).
	familyOptions = map[string][]Option{
		FamilyR66: {
			{
				Name: "r66_login", Cfg: cfgServerLogin, Kind: KindText, Roles: serverOnly,
				Label: "proto.opt.r66_login", Hint: "proto.opt.r66_login_hint", DefaultKey: "proto.opt.default_login",
			},
			{
				Name: "r66_block", Cfg: cfgBlockSize, Kind: KindNumber, Roles: serverAndClient, Short: true, Min: 1,
				Parse: parseUnsigned[uint32], Label: "proto.opt.r66_block", Hint: "proto.opt.r66_block_hint",
				DefaultKey: "proto.opt.default_block",
			},
			{
				Name: "r66_final_hash_algo", Cfg: cfgFinalHashAlgo, Kind: KindSelect, Roles: clientOnly, Short: true,
				Choices: r66HashChoices, Default: r66HashAlgos[0], Label: "proto.opt.r66_final_hash_algo",
				Hint: "proto.opt.r66_final_hash_algo_hint",
			},
			{
				Name: "r66_no_final_hash", Cfg: cfgNoFinalHash, Kind: KindCheck, Roles: serverAndClient,
				Label: "proto.opt.r66_no_final_hash", Row: "proto.opt.r66_final_hash", Checked: true, Inverted: true,
			},
			{
				Name: "r66_check_block_hash", Cfg: cfgCheckBlockHash, Kind: KindCheck, Roles: serverAndClient,
				Label: "proto.opt.r66_check_block_hash", Row: "proto.opt.r66_block_hash", Checked: true,
			},
		},
		FamilyFTP: {
			{
				Name: "ftp_disable_passive", Cfg: cfgDisablePassiveMode, Kind: KindCheck, Roles: serverOnly,
				Label: "proto.opt.ftp_disable_passive", Row: "proto.opt.ftp_passive", Inverted: true,
			},
			{
				Name: "ftp_disable_active", Cfg: cfgDisableActiveMode, Kind: KindCheck, Roles: serverOnly,
				Label: "proto.opt.ftp_disable_active", Row: "proto.opt.ftp_active", Inverted: true,
			},
			{
				Name: "ftp_passive_min", Cfg: cfgPassiveModeMinPort, Kind: KindNumber, Roles: serverOnly, Short: true,
				Min: 1, Max: maxPort, Parse: parsePort, Label: "proto.opt.ftp_passive_min",
				Hint: "proto.opt.any_port_hint", DefaultKey: "proto.opt.any_port",
			},
			{
				Name: "ftp_passive_max", Cfg: cfgPassiveModeMaxPort, Kind: KindNumber, Roles: serverOnly, Short: true,
				Min: 1, Max: maxPort, Parse: parsePort, Label: "proto.opt.ftp_passive_max",
				DefaultKey: "proto.opt.any_port",
			},
			{
				Name: "ftp_enable_active", Cfg: cfgEnableActiveMode, Kind: KindCheck, Roles: clientOnly,
				Label: "proto.opt.ftp_enable_active", Row: "proto.opt.ftp_active",
			},
			{
				Name: "ftp_active_address", Cfg: cfgActiveModeAddress, Kind: KindText, Roles: clientOnly,
				Label: "proto.opt.ftp_active_address", Hint: "proto.opt.ftp_active_address_hint",
				DefaultKey: "proto.opt.default_active_address",
			},
			{
				Name: "ftp_active_min", Cfg: cfgActiveModeMinPort, Kind: KindNumber, Roles: clientOnly, Short: true,
				Min: 1, Max: maxPort, Parse: parsePort, Label: "proto.opt.ftp_active_min",
				Hint: "proto.opt.any_port_hint", DefaultKey: "proto.opt.any_port",
			},
			{
				Name: "ftp_active_max", Cfg: cfgActiveModeMaxPort, Kind: KindNumber, Roles: clientOnly, Short: true,
				Min: 1, Max: maxPort, Parse: parsePort, Label: "proto.opt.ftp_active_max",
				DefaultKey: "proto.opt.any_port",
			},
			{
				Name: "ftp_tls_requirement", Cfg: cfgTLSRequirement, Kind: KindSelect, Roles: serverOnly, TLSOnly: true,
				Choices: tlsRequirementChoices, Default: string(ftp.TLSOptional), Label: "proto.opt.tls_requirement",
			},
		},
		FamilyPeSIT: {
			{
				Name: "pesit_disable_restart", Cfg: cfgDisableRestart, Kind: KindCheck, Roles: serverAndClient,
				Label: "proto.opt.pesit_disable_restart", Row: "proto.opt.pesit_restart", Inverted: true,
			},
			{
				Name: "pesit_disable_checkpoints", Cfg: cfgDisableCheckpoints, Kind: KindCheck, Roles: serverAndClient,
				Label: "proto.opt.pesit_disable_checkpoints", Row: "proto.opt.pesit_checkpoints", Inverted: true,
			},
			{
				Name: "pesit_checkpoint_size", Cfg: cfgCheckpointSize, Kind: KindNumber, Roles: serverAndClient,
				Short: true, Min: 1, Parse: parseUnsigned[uint32], Label: "proto.opt.pesit_checkpoint_size",
				Hint: "proto.opt.bytes", DefaultKey: "proto.opt.default_value",
			},
			{
				Name: "pesit_checkpoint_window", Cfg: cfgCheckpointWindow, Kind: KindNumber, Roles: serverAndClient,
				Short: true, Min: 1, Max: math.MaxUint8, Parse: parseUnsigned[uint8],
				Label: "proto.opt.pesit_checkpoint_window", Hint: "proto.opt.pesit_checkpoint_window_hint",
				DefaultKey: "proto.opt.default_value",
			},
			{
				Name: "pesit_max_message", Cfg: cfgMaxMessageSize, Kind: KindNumber, Roles: serverOnly, Short: true,
				Min: 1, Max: maxPort, Parse: parseUnsigned[uint16], Label: "proto.opt.pesit_max_message",
				Hint: "proto.opt.bytes", DefaultKey: "proto.opt.default_value",
			},
			{
				Name: "pesit_mode", Cfg: cfgCompatibilityMode, Kind: KindSelect, Roles: serverOnly, Short: true,
				Choices: pesitModeChoices, Default: pesit.CompatibilityModeStandard, Label: "proto.opt.pesit_mode",
				Hint: "proto.opt.pesit_mode_hint",
			},
			{
				Name: "pesit_disable_preconnection", Cfg: cfgDisablePreConnection, Kind: KindCheck, Roles: serverOnly,
				Label: "proto.opt.pesit_disable_preconnection", Row: "proto.opt.pesit_preconnection", Inverted: true,
			},
		},
		FamilySFTP: {
			{
				Name: "sftp_kex", Cfg: cfgKeyExchanges, Kind: KindList, Roles: serverAndClient, Values: sftpKeyExchanges,
				Label: "proto.opt.sftp_kex", Hint: "proto.opt.sftp_kex_hint", DefaultKey: "proto.opt.default_algos",
			},
			{
				Name: "sftp_ciphers", Cfg: cfgCiphers, Kind: KindList, Roles: serverAndClient, Values: sftpCiphers,
				Label: "proto.opt.sftp_ciphers", Hint: "proto.opt.sftp_ciphers_hint", DefaultKey: "proto.opt.default_algos",
			},
			{
				Name: "sftp_macs", Cfg: cfgMACs, Kind: KindList, Roles: serverAndClient, Values: sftpMACs,
				Label: "proto.opt.sftp_macs", Hint: "proto.opt.sftp_macs_hint", DefaultKey: "proto.opt.default_algos",
			},
		},
		FamilyAS2: {
			{
				Name: "as2_max_file_size", Cfg: cfgMaxFileSize, Kind: KindNumber, Roles: serverAndClient, Short: true,
				Min: 1, Parse: parseSigned[int64], Label: "proto.opt.as2_max_file_size",
				Hint: "proto.opt.as2_max_file_size_hint", DefaultKey: "proto.opt.default_value",
			},
			{
				Name: "as2_mdn_algo", Cfg: cfgMDNSignatureAlgo, Kind: KindSelect, Roles: serverOnly, Short: true,
				Choices: mdnAlgoChoices, Label: "proto.opt.as2_mdn_algo", DefaultKey: "proto.opt.mdn.none",
			},
		},
	}

	familyNotes = map[string]string{FamilySFTP: "proto.opt.sftp_note"}

	// The TLS block (tlsOptions), the same for the TLS variant of every
	// family: the minimum version and the cipher suites.
	tlsOptions = []Option{
		{
			Name: FieldMinTLS, Cfg: cfgMinTLSVersion, Kind: KindSelect, Roles: all, Short: true,
			Choices: func(_ gui.Translator, current string) []Choice { return TLSVersionChoices(current) },
			Default: protoutils.TLSv12, Label: "proto.opt.min_tls", Hint: "proto.opt.min_tls_hint",
		},
		{
			Name: FieldTLSCiphers, Cfg: cfgCipherSuites, Kind: KindList, Roles: all,
			Values: func(Role) []string { return tlsCipherNames() },
			Label:  "proto.opt.tls_ciphers", Hint: "proto.opt.tls_ciphers_hint", Row: "proto.opt.tls_ciphers_short",
			DefaultKey: "proto.opt.default_ciphers",
		},
	}
)

// optionsOf lists the options of a family for a role, in the order of the form.
func optionsOf(family string, role Role) []*Option {
	var out []*Option

	opts := familyOptions[family]
	for i := range opts {
		if opts[i].forRole(role) {
			out = append(out, &opts[i])
		}
	}

	return out
}

// tlsOptionList lists the options of the TLS block.
func tlsOptionList() []*Option {
	out := make([]*Option, 0, len(tlsOptions))
	for i := range tlsOptions {
		out = append(out, &tlsOptions[i])
	}

	return out
}

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

		rows = append(rows, opt.row(translate, cfg))
	}

	if p.TLS {
		for _, opt := range tlsOptionList() {
			rows = append(rows, opt.row(translate, cfg))
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

		return KV{Label: label, Value: translate(o.word(on))}
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

		return KV{Label: label, Value: value, Mono: true}
	}
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

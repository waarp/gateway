package protoform

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

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
// product reads) and how a details page shows it.
type Option struct {
	Name  string // form field name, unique in a form
	Cfg   string // configuration key the product reads
	Kind  Kind
	Roles []Role

	Label string // catalogue key of the label
	Hint  string // catalogue key of the hint, optional
	Short bool   // a short field (a number, a port)
	// Section is the catalogue key of the theme the option belongs to in
	// its card (identification, synchronisation…); the form and the details
	// page group the options by section when the family names two at least.
	Section string
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
}

func (o *Option) forRole(role Role) bool { return slices.Contains(o.Roles, role) }

// The sections of the cards of options: the themes the options are grouped
// by (catalogue keys). A family that names two at least shows them.
const (
	secIdentity   = "proto.section.identity"
	secSync       = "proto.section.sync"
	secMessages   = "proto.section.messages"
	secAck        = "proto.section.ack"
	secTransfer   = "proto.section.transfer"
	secIntegrity  = "proto.section.integrity"
	secModes      = "proto.section.modes"
	secAlgorithms = "proto.section.algorithms"
	secCompat     = "proto.section.compat"
	secSecurity   = "proto.section.security"
	secMDN        = "proto.section.mdn"
	secTLS        = "proto.tls_title"
)

//nolint:gochecknoglobals // reference data of the screens
var (
	all              = []Role{RoleServer, RoleClient, RolePartner}
	serverOnly       = []Role{RoleServer}
	clientOnly       = []Role{RoleClient}
	partnerOnly      = []Role{RolePartner}
	serverAndClient  = []Role{RoleServer, RoleClient}
	serverAndPartner = []Role{RoleServer, RolePartner}
	clientAndPartner = []Role{RoleClient, RolePartner}

	// The options of each family (familyOptions), in the order of the form,
	// for every role; the note shown at the top of the card of a family
	// (familyNotes). A partner is read by the client of its protocol: its
	// options are what the product reads from the partner (the partner
	// configuration of each module), merged with the client's.
	familyOptions = map[string][]Option{
		FamilyR66: {
			{
				Name: "r66_login", Cfg: cfgServerLogin, Kind: KindText, Roles: serverOnly,
				Section: secIdentity,
				Label:   "proto.opt.r66_login", Hint: "proto.opt.r66_login_hint", DefaultKey: "proto.opt.default_login",
			},
			{
				Name: "r66_server_login", Cfg: cfgServerLogin, Kind: KindText, Roles: partnerOnly,
				Section: secIdentity,
				Label:   "proto.opt.r66_server_login", Hint: "proto.opt.r66_server_login_hint",
				DefaultKey: "proto.opt.default_partner_login",
			},
			{
				Name: "r66_block", Cfg: cfgBlockSize, Kind: KindNumber, Roles: all, Short: true, Min: 1,
				Section: secTransfer,
				Parse:   parseUnsigned[uint32], Label: "proto.opt.r66_block", Hint: "proto.opt.r66_block_hint",
				DefaultKey: "proto.opt.default_block",
			},
			{
				Name: "r66_final_hash_algo", Cfg: cfgFinalHashAlgo, Kind: KindSelect, Roles: clientAndPartner, Short: true,
				Section: secIntegrity,
				Choices: r66HashChoices, Default: r66HashAlgos[0], Label: "proto.opt.r66_final_hash_algo",
				Hint: "proto.opt.r66_final_hash_algo_hint",
			},
			{
				Name: "r66_no_final_hash", Cfg: cfgNoFinalHash, Kind: KindCheck, Roles: all,
				Section: secIntegrity,
				Label:   "proto.opt.r66_no_final_hash", Row: "proto.opt.r66_final_hash", Checked: true, Inverted: true,
			},
			{
				Name: "r66_check_block_hash", Cfg: cfgCheckBlockHash, Kind: KindCheck, Roles: all,
				Section: secIntegrity,
				Label:   "proto.opt.r66_check_block_hash", Row: "proto.opt.r66_block_hash", Checked: true,
			},
		},
		FamilyFTP: {
			{
				Name: "ftp_disable_passive", Cfg: cfgDisablePassiveMode, Kind: KindCheck, Roles: serverOnly,
				Section: secModes,
				Label:   "proto.opt.ftp_disable_passive", Row: "proto.opt.ftp_passive", Inverted: true,
			},
			{
				Name: "ftp_disable_active", Cfg: cfgDisableActiveMode, Kind: KindCheck, Roles: serverAndPartner,
				Section: secModes,
				Label:   "proto.opt.ftp_disable_active", Row: "proto.opt.ftp_active", Inverted: true,
			},
			{
				Name: "ftp_disable_epsv", Cfg: cfgDisableEPSV, Kind: KindCheck, Roles: partnerOnly,
				Section: secModes,
				Label:   "proto.opt.ftp_disable_epsv", Row: "proto.opt.ftp_epsv", Inverted: true,
			},
			{
				Name: "ftp_passive_min", Cfg: cfgPassiveModeMinPort, Kind: KindNumber, Roles: serverOnly, Short: true,
				Section: secModes,
				Min:     1, Max: maxPort, Parse: parsePort, Label: "proto.opt.ftp_passive_min",
				Hint: "proto.opt.any_port_hint", DefaultKey: "proto.opt.any_port",
			},
			{
				Name: "ftp_passive_max", Cfg: cfgPassiveModeMaxPort, Kind: KindNumber, Roles: serverOnly, Short: true,
				Section: secModes,
				Min:     1, Max: maxPort, Parse: parsePort, Label: "proto.opt.ftp_passive_max",
				DefaultKey: "proto.opt.any_port",
			},
			{
				Name: "ftp_enable_active", Cfg: cfgEnableActiveMode, Kind: KindCheck, Roles: clientOnly,
				Section: secModes,
				Label:   "proto.opt.ftp_enable_active", Row: "proto.opt.ftp_active",
			},
			{
				Name: "ftp_active_address", Cfg: cfgActiveModeAddress, Kind: KindText, Roles: clientOnly,
				Section: secModes,
				Label:   "proto.opt.ftp_active_address", Hint: "proto.opt.ftp_active_address_hint",
				DefaultKey: "proto.opt.default_active_address",
			},
			{
				Name: "ftp_active_min", Cfg: cfgActiveModeMinPort, Kind: KindNumber, Roles: clientOnly, Short: true,
				Section: secModes,
				Min:     1, Max: maxPort, Parse: parsePort, Label: "proto.opt.ftp_active_min",
				Hint: "proto.opt.any_port_hint", DefaultKey: "proto.opt.any_port",
			},
			{
				Name: "ftp_active_max", Cfg: cfgActiveModeMaxPort, Kind: KindNumber, Roles: clientOnly, Short: true,
				Section: secModes,
				Min:     1, Max: maxPort, Parse: parsePort, Label: "proto.opt.ftp_active_max",
				DefaultKey: "proto.opt.any_port",
			},
			{
				Name: "ftp_tls_requirement", Cfg: cfgTLSRequirement, Kind: KindSelect, Roles: serverOnly, TLSOnly: true,
				Choices: tlsRequirementChoices, Default: string(ftp.TLSOptional), Label: "proto.opt.tls_requirement",
			},
			{
				Name: "ftp_implicit_tls", Cfg: cfgUseImplicitTLS, Kind: KindCheck, Roles: partnerOnly, TLSOnly: true,
				Label: "proto.opt.ftp_implicit_tls", Hint: "proto.opt.ftp_implicit_tls_hint",
			},
			{
				Name: "ftp_no_session_reuse", Cfg: cfgDisableTLSSessionReuse, Kind: KindCheck, Roles: partnerOnly,
				Section: secTLS,
				TLSOnly: true, Label: "proto.opt.ftp_no_session_reuse", Row: "proto.opt.ftp_session_reuse", Inverted: true,
			},
		},
		FamilyPeSIT: {
			{
				Name: "pesit_login", Cfg: cfgLogin, Kind: KindText, Roles: partnerOnly,
				Section: secIdentity,
				Label:   "proto.opt.pesit_login", Hint: "proto.opt.pesit_login_hint", DefaultKey: "proto.opt.pesit_login_none",
			},
			{
				Name: "pesit_disable_restart", Cfg: cfgDisableRestart, Kind: KindCheck, Roles: serverAndClient,
				Section: secSync,
				Label:   "proto.opt.pesit_disable_restart", Row: "proto.opt.pesit_restart", Inverted: true,
			},
			{
				Name: "pesit_restart", Cfg: cfgDisableRestart, Kind: KindTri, Roles: partnerOnly,
				Section: secSync,
				Label:   "proto.opt.pesit_restart", Inverted: true, DefaultKey: "proto.opt.tristate.client",
			},
			{
				Name: "pesit_disable_checkpoints", Cfg: cfgDisableCheckpoints, Kind: KindCheck, Roles: serverAndClient,
				Section: secSync,
				Label:   "proto.opt.pesit_disable_checkpoints", Row: "proto.opt.pesit_checkpoints", Inverted: true,
			},
			{
				Name: "pesit_checkpoints", Cfg: cfgDisableCheckpoints, Kind: KindTri, Roles: partnerOnly,
				Section: secSync,
				Label:   "proto.opt.pesit_checkpoints", Inverted: true, DefaultKey: "proto.opt.tristate.client",
			},
			{
				Name: "pesit_checkpoint_size", Cfg: cfgCheckpointSize, Kind: KindNumber, Roles: all,
				Section: secSync,
				Short:   true, Min: 1, Parse: parseUnsigned[uint32], Label: "proto.opt.pesit_checkpoint_size",
				Hint: "proto.opt.bytes", DefaultKey: "proto.opt.default_value",
			},
			{
				Name: "pesit_checkpoint_window", Cfg: cfgCheckpointWindow, Kind: KindNumber, Roles: all,
				Section: secSync,
				Short:   true, Min: 1, Max: math.MaxUint8, Parse: parseUnsigned[uint8],
				Label: "proto.opt.pesit_checkpoint_window", Hint: "proto.opt.pesit_checkpoint_window_hint",
				DefaultKey: "proto.opt.default_value",
			},
			{
				Name: "pesit_nsdu", Cfg: cfgUseNSDU, Kind: KindCheck, Roles: partnerOnly, DefaultOn: true,
				Section: secMessages,
				Label:   "proto.opt.pesit_nsdu", Hint: "proto.opt.pesit_nsdu_hint",
			},
			{
				Name: "pesit_max_message", Cfg: cfgMaxMessageSize, Kind: KindNumber, Roles: serverAndPartner, Short: true,
				Section: secMessages,
				Min:     1, Max: maxPort, Parse: parseUnsigned[uint16], Label: "proto.opt.pesit_max_message",
				Hint: "proto.opt.bytes", DefaultKey: "proto.opt.default_value",
			},
			{
				Name: "pesit_mode", Cfg: cfgCompatibilityMode, Kind: KindSelect, Roles: serverAndPartner, Short: true,
				Section: secIdentity,
				Choices: pesitModeChoices, Default: pesit.CompatibilityModeStandard, Label: "proto.opt.pesit_mode",
				Hint: "proto.opt.pesit_mode_hint",
			},
			{
				Name: "pesit_disable_preconnection", Cfg: cfgDisablePreConnection, Kind: KindCheck, Roles: serverAndPartner,
				Section: secIdentity,
				Label:   "proto.opt.pesit_disable_preconnection", Row: "proto.opt.pesit_preconnection", Inverted: true,
			},
			{
				Name: "pesit_expects_ack", Cfg: cfgExpectsAck, Kind: KindCheck, Roles: partnerOnly,
				Section: secAck,
				Label:   "proto.opt.pesit_expects_ack",
			},
			{
				Name: "pesit_ack_timeout", Cfg: cfgAckTimeout, Kind: KindDuration, Roles: partnerOnly,
				Section: secAck,
				Parse:   parseDuration, ErrorKey: "proto.error_duration", Label: "proto.opt.pesit_ack_timeout",
				Hint: "proto.opt.pesit_ack_timeout_hint", DefaultKey: "proto.opt.pesit_ack_no_limit",
			},
			{
				Name: "pesit_no_pi37", Cfg: cfgNoPI37, Kind: KindCheck, Roles: partnerOnly,
				Section: secIdentity,
				Label:   "proto.opt.pesit_no_pi37", Row: "proto.opt.pesit_pi37", Inverted: true,
			},
		},
		FamilySFTP: {
			{
				Name: "sftp_kex", Cfg: cfgKeyExchanges, Kind: KindList, Roles: all, Values: sftpKeyExchanges,
				Section: secAlgorithms,
				Label:   "proto.opt.sftp_kex", Hint: "proto.opt.sftp_kex_hint", DefaultKey: "proto.opt.default_algos",
			},
			{
				Name: "sftp_ciphers", Cfg: cfgCiphers, Kind: KindList, Roles: all, Values: sftpCiphers,
				Section: secAlgorithms,
				Label:   "proto.opt.sftp_ciphers", Hint: "proto.opt.sftp_ciphers_hint", DefaultKey: "proto.opt.default_algos",
			},
			{
				Name: "sftp_macs", Cfg: cfgMACs, Kind: KindList, Roles: all, Values: sftpMACs,
				Section: secAlgorithms,
				Label:   "proto.opt.sftp_macs", Hint: "proto.opt.sftp_macs_hint", DefaultKey: "proto.opt.default_algos",
			},
			{
				Name: "sftp_use_stat", Cfg: cfgUseStat, Kind: KindCheck, Roles: partnerOnly,
				Section: secCompat,
				Label:   "proto.opt.sftp_use_stat",
			},
			{
				Name: "sftp_no_concurrent_reads", Cfg: cfgNoConcurrentReads, Kind: KindCheck, Roles: partnerOnly,
				Section: secCompat,
				Label:   "proto.opt.sftp_no_concurrent_reads", Hint: "proto.opt.sftp_no_concurrent_reads_hint",
				Row: "proto.opt.sftp_concurrent_reads", Inverted: true,
			},
		},
		FamilyAS2: {
			{
				Name: "as2_max_file_size", Cfg: cfgMaxFileSize, Kind: KindNumber, Roles: serverAndClient, Short: true,
				Section: secTransfer,
				Min:     1, Parse: parseSigned[int64], Label: "proto.opt.as2_max_file_size",
				Hint: "proto.opt.as2_max_file_size_hint", DefaultKey: "proto.opt.default_value",
			},
			{
				Name: "as2_mdn_algo", Cfg: cfgMDNSignatureAlgo, Kind: KindSelect, Roles: serverOnly, Short: true,
				Section: secSecurity,
				Choices: mdnAlgoChoices, Label: "proto.opt.as2_mdn_algo", DefaultKey: "proto.opt.mdn.none",
			},
			{
				Name: "as2_sign_algo", Cfg: cfgSignatureAlgo, Kind: KindSelect, Roles: partnerOnly, Short: true,
				Section: secSecurity,
				Choices: as2SignChoices, Label: "proto.opt.as2_sign_algo", DefaultKey: "proto.opt.default_value",
			},
			{
				Name: "as2_encrypt_algo", Cfg: cfgEncryptionAlgo, Kind: KindSelect, Roles: partnerOnly, Short: true,
				Section: secSecurity,
				Choices: as2EncryptChoices, Label: "proto.opt.as2_encrypt_algo", DefaultKey: "proto.opt.default_value",
			},
			{
				Name: "as2_async_mdn", Cfg: cfgAsyncMDNAddress, Kind: KindText, Roles: partnerOnly,
				Section: secMDN,
				Label:   "proto.opt.as2_async_mdn", Hint: "proto.opt.as2_async_mdn_hint",
				DefaultKey: "proto.opt.as2_async_mdn_none",
			},
			{
				Name: "as2_handle_async_mdn", Cfg: cfgHandleAsyncMDN, Kind: KindCheck, Roles: partnerOnly,
				Section: secMDN,
				Label:   "proto.opt.as2_handle_async_mdn", Hint: "proto.opt.as2_handle_async_mdn_hint",
			},
		},
	}

	familyNotes = map[string]string{FamilySFTP: "proto.opt.sftp_note"}

	// The TLS block (tlsOptions), the same for the TLS variant of every
	// family: the minimum version and the cipher suites.
	tlsOptions = []Option{
		{
			Name: FieldMinTLS, Cfg: cfgMinTLSVersion, Kind: KindSelect, Roles: all, Short: true,
			Section: secTLS,
			Choices: func(_ gui.Translator, current string) []Choice { return TLSVersionChoices(current) },
			Default: protoutils.TLSv12, Label: "proto.opt.min_tls", Hint: "proto.opt.min_tls_hint",
		},
		{
			Name: FieldTLSCiphers, Cfg: cfgCipherSuites, Kind: KindList, Roles: all,
			Section: secTLS,
			Values:  func(Role) []string { return tlsCipherNames() },
			Label:   "proto.opt.tls_ciphers", Hint: "proto.opt.tls_ciphers_hint", Row: "proto.opt.tls_ciphers_short",
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

		return KV{Label: label, Value: value, Mono: true}
	}
}

// sectioned is the row of the option with the theme it belongs to.
func (o *Option) sectioned(translate gui.Translator, cfg map[string]any) KV {
	row := o.row(translate, cfg)
	if o.Section != "" {
		row.Section = translate(o.Section)
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

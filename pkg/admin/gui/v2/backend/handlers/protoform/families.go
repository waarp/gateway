package protoform

import (
	"math"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/as2"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ftp"
	httpconst "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/webdav"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

// One table describes the protocol families: everything the interface
// knows of a family sits in its entry, and adding a family takes one entry.

// The families: the key of a family is the key of its plain variant.
const (
	FamilyR66    = "r66"
	FamilyHTTP   = "http"
	FamilySFTP   = "sftp"
	FamilyFTP    = "ftp"
	FamilyPeSIT  = "pesit"
	FamilyWebDAV = "webdav"
	FamilyAS2    = "as2"
)

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

// variant is one protocol key of the product and the name the product shows
// for it (a datum rather than a label: the historic pages show it, the
// documentation names it). Key "" when the product has no such variant (SFTP
// has no TLS variant).
type variant struct {
	Key  string
	Name string
}

// optionSection is a theme of the options card, as a catalogue key ("" for options
// without theme), and its options in the order of the form.
type optionSection struct {
	key     string
	options []Option
}

// family is one protocol family, as the interface shows it: its key (the key
// of its plain variant), its label, its plain and TLS variants, the note at
// the top of its card (a catalogue key, "" for none) and its options by
// theme, in the order of the form. A partner is read by the client of its
// protocol: its options are what the product reads from the partner (the
// partner configuration of each module), merged with the client's.
type family struct {
	key      string
	label    string
	plain    variant
	tls      variant
	note     string
	sections []optionSection
}

//nolint:gochecknoglobals // reference data of the product and of the screens
var (
	all              = []Role{RoleServer, RoleClient, RolePartner}
	serverOnly       = []Role{RoleServer}
	clientOnly       = []Role{RoleClient}
	partnerOnly      = []Role{RolePartner}
	serverAndClient  = []Role{RoleServer, RoleClient}
	serverAndPartner = []Role{RoleServer, RolePartner}
	clientAndPartner = []Role{RoleClient, RolePartner}

	// The table of the protocol families (families).
	families = []family{
		{
			key: FamilyR66, label: "R66", plain: variant{r66.R66, "R66"}, tls: variant{r66.R66TLS, "R66-TLS"},
			sections: []optionSection{
				{secIdentity, []Option{
					{
						Name: "r66_login", Cfg: cfgServerLogin, Kind: KindText, Roles: serverOnly,
						Label: "proto.opt.r66_login", Hint: "proto.opt.r66_login_hint", DefaultKey: "proto.opt.default_login",
					},
					{
						Name: "r66_server_login", Cfg: cfgServerLogin, Kind: KindText, Roles: partnerOnly,
						Label: "proto.opt.r66_server_login", Hint: "proto.opt.r66_server_login_hint",
						DefaultKey: "proto.opt.default_partner_login",
					},
				}},
				{secTransfer, []Option{
					{
						Name: "r66_block", Cfg: cfgBlockSize, Kind: KindNumber, Roles: all, Short: true, Min: 1,
						Parse: parseUnsigned[uint32], Label: "proto.opt.r66_block", Hint: "proto.opt.r66_block_hint",
						DefaultKey: "proto.opt.default_block",
					},
				}},
				{secIntegrity, []Option{
					{
						Name: "r66_final_hash_algo", Cfg: cfgFinalHashAlgo, Kind: KindSelect, Roles: clientAndPartner, Short: true,
						Choices: r66HashChoices, Default: r66HashAlgos[0], Label: "proto.opt.r66_final_hash_algo",
						Hint: "proto.opt.r66_final_hash_algo_hint",
					},
					{
						Name: "r66_no_final_hash", Cfg: cfgNoFinalHash, Kind: KindCheck, Roles: all,
						Label: "proto.opt.r66_no_final_hash", Row: "proto.opt.r66_final_hash", Checked: true, Inverted: true,
					},
					{
						Name: "r66_check_block_hash", Cfg: cfgCheckBlockHash, Kind: KindCheck, Roles: all,
						Label: "proto.opt.r66_check_block_hash", Row: "proto.opt.r66_block_hash", Checked: true,
					},
				}},
			},
		},
		{
			key: FamilyHTTP, label: "HTTP", plain: variant{httpconst.HTTP, "HTTP"}, tls: variant{httpconst.HTTPS, "HTTPS"},
		},
		{
			key: FamilySFTP, label: "SFTP", plain: variant{sftp.SFTP, "SFTP"}, note: "proto.opt.sftp_note",
			sections: []optionSection{
				{secAlgorithms, []Option{
					{
						Name: "sftp_kex", Cfg: cfgKeyExchanges, Kind: KindList, Roles: all, Values: sftpKeyExchanges,
						Label: "proto.opt.sftp_kex", Hint: "proto.opt.sftp_kex_hint", DefaultKey: "proto.opt.default_algos",
					},
					{
						Name: "sftp_ciphers", Cfg: cfgCiphers, Kind: KindList, Roles: all, Values: sftpCiphers,
						Label: "proto.opt.sftp_ciphers", Hint: "proto.opt.sftp_ciphers_hint", DefaultKey: "proto.opt.default_algos",
					},
					{
						Name: "sftp_macs", Cfg: cfgMACs, Kind: KindList, Roles: all, Values: sftpMACs,
						Label: "proto.opt.sftp_macs", Hint: "proto.opt.sftp_macs_hint", DefaultKey: "proto.opt.default_algos",
					},
				}},
				{secCompat, []Option{
					{
						Name: "sftp_use_stat", Cfg: cfgUseStat, Kind: KindCheck, Roles: partnerOnly,
						Label: "proto.opt.sftp_use_stat",
					},
					{
						Name: "sftp_no_concurrent_reads", Cfg: cfgNoConcurrentReads, Kind: KindCheck, Roles: partnerOnly,
						Label: "proto.opt.sftp_no_concurrent_reads", Hint: "proto.opt.sftp_no_concurrent_reads_hint",
						Row: "proto.opt.sftp_concurrent_reads", Inverted: true,
					},
				}},
			},
		},
		{
			key: FamilyFTP, label: "FTP", plain: variant{ftp.FTP, "FTP"}, tls: variant{ftp.FTPS, "FTPS"},
			sections: []optionSection{
				{secModes, []Option{
					{
						Name: "ftp_disable_passive", Cfg: cfgDisablePassiveMode, Kind: KindCheck, Roles: serverOnly,
						Label: "proto.opt.ftp_disable_passive", Row: "proto.opt.ftp_passive", Inverted: true,
					},
					{
						Name: "ftp_disable_active", Cfg: cfgDisableActiveMode, Kind: KindCheck, Roles: serverAndPartner,
						Label: "proto.opt.ftp_disable_active", Row: "proto.opt.ftp_active", Inverted: true,
					},
					{
						Name: "ftp_disable_epsv", Cfg: cfgDisableEPSV, Kind: KindCheck, Roles: partnerOnly,
						Label: "proto.opt.ftp_disable_epsv", Row: "proto.opt.ftp_epsv", Inverted: true,
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
				}},
				// The options of the TLS variant without a theme: shown with the TLS checkbox on.
				{"", []Option{
					{
						Name: "ftp_tls_requirement", Cfg: cfgTLSRequirement, Kind: KindSelect, Roles: serverOnly, TLSOnly: true,
						Choices: tlsRequirementChoices, Default: string(ftp.TLSOptional), Label: "proto.opt.tls_requirement",
					},
					{
						Name: "ftp_implicit_tls", Cfg: cfgUseImplicitTLS, Kind: KindCheck, Roles: partnerOnly, TLSOnly: true,
						Label: "proto.opt.ftp_implicit_tls", Hint: "proto.opt.ftp_implicit_tls_hint",
					},
				}},
				{secTLS, []Option{
					{
						Name: "ftp_no_session_reuse", Cfg: cfgDisableTLSSessionReuse, Kind: KindCheck, Roles: partnerOnly,
						TLSOnly: true, Label: "proto.opt.ftp_no_session_reuse", Row: "proto.opt.ftp_session_reuse", Inverted: true,
					},
				}},
			},
		},
		{
			key: FamilyPeSIT, label: "PeSIT", plain: variant{pesit.Pesit, "PeSIT"}, tls: variant{pesit.PesitTLS, "PeSIT-TLS"},
			// The identity comes in two parts, so that each role keeps the order
			// of its cards: the login (a partner option) leads the partner card;
			// the server, which has no login, opens on the synchronisation and
			// finds the mode and the pre-connection after the messages. The pages
			// gather the parts under one title.
			sections: []optionSection{
				{secIdentity, []Option{
					{
						Name: "pesit_login", Cfg: cfgLogin, Kind: KindText, Roles: partnerOnly,
						Label: "proto.opt.pesit_login", Hint: "proto.opt.pesit_login_hint", DefaultKey: "proto.opt.pesit_login_none",
					},
				}},
				{secSync, []Option{
					{
						Name: "pesit_disable_restart", Cfg: cfgDisableRestart, Kind: KindCheck, Roles: serverAndClient,
						Label: "proto.opt.pesit_disable_restart", Row: "proto.opt.pesit_restart", Inverted: true,
					},
					{
						Name: "pesit_restart", Cfg: cfgDisableRestart, Kind: KindTri, Roles: partnerOnly,
						Label: "proto.opt.pesit_restart", Inverted: true, DefaultKey: "proto.opt.tristate.client",
					},
					{
						Name: "pesit_disable_checkpoints", Cfg: cfgDisableCheckpoints, Kind: KindCheck, Roles: serverAndClient,
						Label: "proto.opt.pesit_disable_checkpoints", Row: "proto.opt.pesit_checkpoints", Inverted: true,
					},
					{
						Name: "pesit_checkpoints", Cfg: cfgDisableCheckpoints, Kind: KindTri, Roles: partnerOnly,
						Label: "proto.opt.pesit_checkpoints", Inverted: true, DefaultKey: "proto.opt.tristate.client",
					},
					{
						Name: "pesit_checkpoint_size", Cfg: cfgCheckpointSize, Kind: KindNumber, Roles: all,
						Short: true, Min: 1, Parse: parseUnsigned[uint32], Label: "proto.opt.pesit_checkpoint_size",
						Hint: "proto.opt.bytes", DefaultKey: "proto.opt.default_value", Bytes: true,
					},
					{
						Name: "pesit_checkpoint_window", Cfg: cfgCheckpointWindow, Kind: KindNumber, Roles: all,
						Short: true, Min: 1, Max: math.MaxUint8, Parse: parseUnsigned[uint8],
						Label: "proto.opt.pesit_checkpoint_window", Hint: "proto.opt.pesit_checkpoint_window_hint",
						DefaultKey: "proto.opt.default_value",
					},
				}},
				{secMessages, []Option{
					{
						Name: "pesit_nsdu", Cfg: cfgUseNSDU, Kind: KindCheck, Roles: partnerOnly, DefaultOn: true,
						Label: "proto.opt.pesit_nsdu", Hint: "proto.opt.pesit_nsdu_hint",
					},
					{
						Name: "pesit_max_message", Cfg: cfgMaxMessageSize, Kind: KindNumber, Roles: serverAndPartner, Short: true,
						Min: 1, Max: maxPort, Parse: parseUnsigned[uint16], Label: "proto.opt.pesit_max_message",
						Hint: "proto.opt.bytes", DefaultKey: "proto.opt.default_value", Bytes: true,
					},
				}},
				{secIdentity, []Option{
					{
						Name: "pesit_mode", Cfg: cfgCompatibilityMode, Kind: KindSelect, Roles: serverAndPartner, Short: true,
						Choices: pesitModeChoices, Default: pesit.CompatibilityModeStandard, Label: "proto.opt.pesit_mode",
						Hint: "proto.opt.pesit_mode_hint",
					},
					{
						Name: "pesit_disable_preconnection", Cfg: cfgDisablePreConnection, Kind: KindCheck, Roles: serverAndPartner,
						Label: "proto.opt.pesit_disable_preconnection", Row: "proto.opt.pesit_preconnection", Inverted: true,
					},
					{
						Name: "pesit_no_pi37", Cfg: cfgNoPI37, Kind: KindCheck, Roles: partnerOnly,
						Label: "proto.opt.pesit_no_pi37", Row: "proto.opt.pesit_pi37", Inverted: true,
					},
				}},
				{secAck, []Option{
					{
						Name: "pesit_expects_ack", Cfg: cfgExpectsAck, Kind: KindCheck, Roles: partnerOnly,
						Label: "proto.opt.pesit_expects_ack",
					},
					{
						Name: "pesit_ack_timeout", Cfg: cfgAckTimeout, Kind: KindDuration, Roles: partnerOnly,
						Parse: parseDuration, ErrorKey: "proto.error_duration", Label: "proto.opt.pesit_ack_timeout",
						Hint: "proto.opt.pesit_ack_timeout_hint", DefaultKey: "proto.opt.pesit_ack_default",
					},
				}},
			},
		},
		{
			key: FamilyWebDAV, label: "WebDAV", plain: variant{webdav.Webdav, "WebDAV"},
			tls: variant{webdav.WebdavTLS, "WebDAV over HTTPS"},
		},
		{
			key: FamilyAS2, label: "AS2", plain: variant{as2.AS2, "AS2"}, tls: variant{as2.AS2TLS, "AS2 over HTTPS"},
			sections: []optionSection{
				{secTransfer, []Option{
					{
						Name: "as2_max_file_size", Cfg: cfgMaxFileSize, Kind: KindNumber, Roles: serverAndClient, Short: true,
						Min: 1, Parse: parseSigned[int64], Label: "proto.opt.as2_max_file_size",
						Hint: "proto.opt.as2_max_file_size_hint", DefaultKey: "proto.opt.default_value", Bytes: true,
					},
				}},
				{secSecurity, []Option{
					{
						Name: "as2_mdn_algo", Cfg: cfgMDNSignatureAlgo, Kind: KindSelect, Roles: serverOnly, Short: true,
						Choices: mdnAlgoChoices, Label: "proto.opt.as2_mdn_algo", DefaultKey: "proto.opt.mdn.none",
					},
					{
						Name: "as2_sign_algo", Cfg: cfgSignatureAlgo, Kind: KindSelect, Roles: partnerOnly, Short: true,
						Choices: as2SignChoices, Label: "proto.opt.as2_sign_algo", DefaultKey: "proto.opt.default_value",
					},
					{
						Name: "as2_encrypt_algo", Cfg: cfgEncryptionAlgo, Kind: KindSelect, Roles: partnerOnly, Short: true,
						Choices: as2EncryptChoices, Label: "proto.opt.as2_encrypt_algo", DefaultKey: "proto.opt.default_value",
					},
				}},
				{secMDN, []Option{
					{
						Name: "as2_async_mdn", Cfg: cfgAsyncMDNAddress, Kind: KindText, Roles: partnerOnly,
						Label: "proto.opt.as2_async_mdn", Hint: "proto.opt.as2_async_mdn_hint",
						DefaultKey: "proto.opt.as2_async_mdn_none",
					},
					{
						Name: "as2_handle_async_mdn", Cfg: cfgHandleAsyncMDN, Kind: KindCheck, Roles: partnerOnly,
						Label: "proto.opt.as2_handle_async_mdn", Hint: "proto.opt.as2_handle_async_mdn_hint",
					},
				}},
			},
		},
	}

	// The TLS block (tlsSection), the same for the TLS variant of every
	// family: the minimum version and the cipher suites.
	tlsSection = optionSection{secTLS, []Option{
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
	}}
)

// familyByKey finds a family of the table by its key; nil for a key the
// interface does not know.
func familyByKey(key string) *family {
	for i := range families {
		if families[i].key == key {
			return &families[i]
		}
	}

	return nil
}

// variantOf finds the family of a protocol key of the product and says
// whether the key is its TLS variant; ok false for a key the interface does
// not know.
func variantOf(key string) (fam *family, withTLS, ok bool) {
	for i := range families {
		switch key {
		case families[i].plain.Key:
			return &families[i], false, true
		case families[i].tls.Key:
			return &families[i], true, true
		}
	}

	return nil, false, false
}

// placed is an option with the theme it belongs to: the section of the
// table it sits in (a catalogue key, "" for none).
type placed struct {
	*Option
	Section string
}

// optionsOf lists the options of a family for a role, in the order of the
// form, each with its theme; nothing for a family the interface does not know.
func optionsOf(familyKey string, role Role) []placed {
	fam := familyByKey(familyKey)
	if fam == nil {
		return nil
	}

	var out []placed

	for i := range fam.sections {
		for _, p := range fam.sections[i].placed() {
			if p.forRole(role) {
				out = append(out, p)
			}
		}
	}

	return out
}

// tlsOptionList lists the options of the TLS block, which every role has.
func tlsOptionList() []placed { return tlsSection.placed() }

// placed lists the options of the section, each with its theme.
func (s *optionSection) placed() []placed {
	out := make([]placed, 0, len(s.options))
	for i := range s.options {
		out = append(out, placed{Option: &s.options[i], Section: s.key})
	}

	return out
}

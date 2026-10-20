package servers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

// Keys of the protocol options in the configuration of a server, as the
// product reads them (the same as the page this screen replaces wrote).
const (
	cfgServerLogin          = "serverLogin"
	cfgBlockSize            = "blockSize"
	cfgNoFinalHash          = "noFinalHash"
	cfgCheckBlockHash       = "checkBlockHash"
	cfgMinTLSVersion        = "minTLSVersion"
	cfgCipherSuites         = "cipherSuites"
	cfgDisablePassiveMode   = "disablePassiveMode"
	cfgDisableActiveMode    = "disableActiveMode"
	cfgPassiveModeMinPort   = "passiveModeMinPort"
	cfgPassiveModeMaxPort   = "passiveModeMaxPort"
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
)

// cfgString reads a text or a number of the configuration as a string.
func cfgString(cfg map[string]any, key string) string {
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

func cfgBool(cfg map[string]any, key string) bool {
	b, _ := cfg[key].(bool)

	return b
}

// cfgList reads a list of texts of the configuration.
func cfgList(cfg map[string]any, key string) []string {
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

// optionRows lists the protocol options of a server as the details page
// shows them: the effect for the operator first, in the order of the form.
func optionRows(translate gui.Translator, proto protocol, cfg map[string]any) []kv {
	var rows []kv

	onOff := func(disabled bool) string {
		if disabled {
			return translate("servers.opt.disabled")
		}

		return translate("servers.opt.enabled")
	}
	text := func(key, value, dflt string) kv {
		if value == "" {
			return kv{Label: translate(key), Value: dflt}
		}

		return kv{Label: translate(key), Value: value, Mono: true}
	}

	switch proto.Family {
	case familyR66:
		checked := func(yes bool) string {
			if yes {
				return translate("servers.opt.checked")
			}

			return translate("servers.opt.unchecked")
		}

		rows = append(rows,
			text("servers.opt.r66_login", cfgString(cfg, cfgServerLogin), translate("servers.opt.default_login")),
			text("servers.opt.r66_block", cfgString(cfg, cfgBlockSize), translate("servers.opt.default_block")),
			kv{Label: translate("servers.opt.r66_final_hash"), Value: checked(!cfgBool(cfg, cfgNoFinalHash))},
			kv{Label: translate("servers.opt.r66_block_hash"), Value: checked(cfgBool(cfg, cfgCheckBlockHash))})
	case familyFTP:
		passive := onOff(cfgBool(cfg, cfgDisablePassiveMode))
		if !cfgBool(cfg, cfgDisablePassiveMode) {
			minPort, maxPort := cfgString(cfg, cfgPassiveModeMinPort), cfgString(cfg, cfgPassiveModeMaxPort)
			if minPort != "" || maxPort != "" {
				passive += " · " + translate("servers.opt.ports", "min", minPort, "max", maxPort)
			} else {
				passive += " · " + translate("servers.opt.any_port")
			}
		}

		rows = append(rows,
			kv{Label: translate("servers.opt.ftp_passive"), Value: passive},
			kv{Label: translate("servers.opt.ftp_active"), Value: onOff(cfgBool(cfg, cfgDisableActiveMode))})

		if proto.TLS {
			requirement := tlsRequirementLabel(translate, cfgString(cfg, cfgTLSRequirement))
			rows = append(rows, kv{Label: translate("servers.opt.tls_requirement"), Value: requirement})
		}
	case familyPeSIT:
		dflt := translate("servers.opt.default_value")

		rows = append(rows,
			kv{Label: translate("servers.opt.pesit_restart"), Value: onOff(cfgBool(cfg, cfgDisableRestart))},
			kv{Label: translate("servers.opt.pesit_checkpoints"), Value: onOff(cfgBool(cfg, cfgDisableCheckpoints))})

		if !cfgBool(cfg, cfgDisableCheckpoints) {
			rows = append(rows,
				text("servers.opt.pesit_checkpoint_size", cfgString(cfg, cfgCheckpointSize), dflt),
				text("servers.opt.pesit_checkpoint_window", cfgString(cfg, cfgCheckpointWindow), dflt))
		}

		rows = append(rows,
			text("servers.opt.pesit_max_message", cfgString(cfg, cfgMaxMessageSize), dflt),
			kv{
				Label: translate("servers.opt.pesit_mode"), Value: pesitModeLabel(translate, cfgString(cfg, cfgCompatibilityMode)),
			},
			kv{Label: translate("servers.opt.pesit_preconnection"), Value: onOff(cfgBool(cfg, cfgDisablePreConnection))})
	case familySFTP:
		list := func(key, cfgKey string) kv {
			if values := cfgList(cfg, cfgKey); len(values) > 0 {
				return kv{Label: translate(key), Value: strings.Join(values, ", "), Mono: true}
			}

			return kv{Label: translate(key), Value: translate("servers.opt.default_algos")}
		}

		rows = append(rows,
			list("servers.opt.sftp_kex", cfgKeyExchanges),
			list("servers.opt.sftp_ciphers", cfgCiphers),
			list("servers.opt.sftp_macs", cfgMACs))
	case familyAS2:
		rows = append(rows,
			text("servers.opt.as2_max_file_size", cfgString(cfg, cfgMaxFileSize), translate("servers.opt.default_value")),
			text("servers.opt.as2_mdn_algo", cfgString(cfg, cfgMDNSignatureAlgo), translate("servers.opt.mdn.none")))
	}

	if proto.TLS {
		rows = append(rows,
			kv{Label: translate("servers.opt.min_tls"), Value: tlsVersionLabel(cfgString(cfg, cfgMinTLSVersion))})

		suites := kv{Label: translate("servers.opt.tls_ciphers_short"), Value: translate("servers.opt.default_ciphers")}
		if chosen := cfgList(cfg, cfgCipherSuites); len(chosen) > 0 {
			labels := make([]string, 0, len(chosen))
			for _, name := range chosen {
				labels = append(labels, cipherLabel(name))
			}

			suites.Value, suites.Mono = strings.Join(labels, ", "), true
		}

		rows = append(rows, suites)
	}

	if len(rows) == 0 {
		rows = append(rows, kv{Label: translate("servers.detail.options_label"), Value: translate("servers.opt.none")})
	}

	return rows
}

func tlsVersionLabel(value string) string {
	for _, c := range tlsVersionChoices(value) {
		if c.Selected {
			return c.Label
		}
	}

	return tlsVersionLabelOf(protoutils.TLSv12)
}

func tlsVersionLabelOf(value string) string {
	for _, c := range tlsVersionChoices(value) {
		if c.Selected {
			return c.Label
		}
	}

	return value
}

func tlsRequirementLabel(translate gui.Translator, value string) string {
	if value == "" {
		value = string(ftp.TLSOptional)
	}

	for _, c := range tlsRequirementChoices(translate, value) {
		if c.Selected {
			return c.Label
		}
	}

	return value
}

func pesitModeLabel(translate gui.Translator, value string) string {
	if value == "" {
		value = pesit.CompatibilityModeStandard
	}

	for _, c := range pesitModeChoices(translate, value) {
		if c.Selected {
			return c.Label
		}
	}

	return value
}

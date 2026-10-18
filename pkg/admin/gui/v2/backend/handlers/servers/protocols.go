package servers

import (
	"crypto/tls"
	"errors"
	"slices"
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/as2"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ftp"
	httpconst "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/webdav"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

var errPortOutOfRange = errors.New("port out of range")

// Option families: the block of options a protocol shows in the form.
const (
	familyR66    = "r66"
	familyHTTP   = "http"
	familySFTP   = "sftp"
	familyFTP    = "ftp"
	familyPeSIT  = "pesit"
	familyWebDAV = "webdav"
	familyAS2    = "as2"
)

// protocol is one of the protocols a local server may use: the name the
// interface shows (the one the historic page shows, a datum rather than a
// label), the family of its options and whether it carries TLS options.
type protocol struct {
	Key    string
	Label  string
	Family string
	TLS    bool
}

//nolint:gochecknoglobals // reference data of the screen
var protocolTable = map[string]protocol{
	r66.R66:          {Key: r66.R66, Label: "R66", Family: familyR66},
	r66.R66TLS:       {Key: r66.R66TLS, Label: "R66-TLS", Family: familyR66, TLS: true},
	httpconst.HTTP:   {Key: httpconst.HTTP, Label: "HTTP", Family: familyHTTP},
	httpconst.HTTPS:  {Key: httpconst.HTTPS, Label: "HTTPS", Family: familyHTTP, TLS: true},
	sftp.SFTP:        {Key: sftp.SFTP, Label: "SFTP", Family: familySFTP},
	pesit.Pesit:      {Key: pesit.Pesit, Label: "PeSIT", Family: familyPeSIT},
	pesit.PesitTLS:   {Key: pesit.PesitTLS, Label: "PeSIT-TLS", Family: familyPeSIT, TLS: true},
	ftp.FTP:          {Key: ftp.FTP, Label: "FTP", Family: familyFTP},
	ftp.FTPS:         {Key: ftp.FTPS, Label: "FTPS", Family: familyFTP, TLS: true},
	webdav.Webdav:    {Key: webdav.Webdav, Label: "WebDAV", Family: familyWebDAV},
	webdav.WebdavTLS: {Key: webdav.WebdavTLS, Label: "WebDAV over HTTPS", Family: familyWebDAV, TLS: true},
	as2.AS2:          {Key: as2.AS2, Label: "AS2", Family: familyAS2},
	as2.AS2TLS:       {Key: as2.AS2TLS, Label: "AS2 over HTTPS", Family: familyAS2, TLS: true},
}

// protocolOf describes a protocol key; an unknown key keeps its key as label
// and shows no options.
func protocolOf(key string) protocol {
	if p, ok := protocolTable[key]; ok {
		return p
	}

	return protocol{Key: key, Label: key}
}

// protocolList lists the protocols registered in the gateway, in the order
// of their display names, so that the filters and the form follow the
// product and not this table.
func protocolList() []protocol {
	var list []protocol

	for key := range protocols.List() {
		list = append(list, protocolOf(key))
	}

	slices.SortFunc(list, func(a, b protocol) int { return strings.Compare(a.Label, b.Label) })

	return list
}

func isProtocol(key string) bool { return key != "" && protocols.Exists(key) }

// choice is an option of a select, selected or not.
type choice struct {
	Value    string
	Label    string
	Selected bool
}

func pick(pairs [][2]string, current string) []choice {
	out := make([]choice, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, choice{Value: p[0], Label: p[1], Selected: p[0] == current})
	}

	return out
}

// multi lists the chosen values first, in their order (the preference order
// of a dual list), then the other values.
func multi(values, chosen []string) []choice {
	out := make([]choice, 0, len(values))

	for _, v := range chosen {
		if slices.Contains(values, v) {
			out = append(out, choice{Value: v, Label: v, Selected: true})
		}
	}

	for _, v := range values {
		if !slices.Contains(chosen, v) {
			out = append(out, choice{Value: v, Label: v})
		}
	}

	return out
}

func tlsVersionChoices(current string) []choice {
	return pick([][2]string{
		{protoutils.TLSv10, "TLS 1.0"},
		{protoutils.TLSv11, "TLS 1.1"},
		{protoutils.TLSv12, "TLS 1.2"},
		{protoutils.TLSv13, "TLS 1.3"},
	}, current)
}

// tlsCipherChoices lists the cipher suites one can enable, from the product's
// table (crypto/tls): those negotiable in TLS 1.2 or earlier, the secure ones
// first, then the insecure ones flagged as such. The TLS 1.3 suites are
// always on and not configurable, so they are not offered.
func tlsCipherChoices(translate gui.Translator, chosen []string) []choice {
	var secure, insecure []string

	for name, suite := range protoutils.TLSCiphers {
		if !slices.Contains(suite.SupportedVersions, uint16(tls.VersionTLS12)) {
			continue
		}

		if suite.Insecure {
			insecure = append(insecure, name)
		} else {
			secure = append(secure, name)
		}
	}

	slices.Sort(secure)
	slices.Sort(insecure)

	out := multi(append(secure, insecure...), chosen)
	for i := range out {
		if protoutils.TLSCiphers[out[i].Value].Insecure {
			out[i].Label += " " + translate("servers.opt.tls_insecure")
		}
	}

	return out
}

// The SSH algorithms a server may use, from the SFTP module.
func sftpKeyExchanges() []string {
	names := make([]string, 0, len(sftp.ValidKeyExchanges))

	for _, algo := range sftp.ValidKeyExchanges {
		if algo.IsValid(true) {
			names = append(names, algo.Name)
		}
	}

	return names
}

func sftpCiphers() []string {
	names := make([]string, 0, len(sftp.ValidCiphers))

	for _, algo := range sftp.ValidCiphers {
		if algo.IsValid(true) {
			names = append(names, algo.Name)
		}
	}

	return names
}

func sftpMACs() []string {
	names := make([]string, 0, len(sftp.ValidMACs))

	for _, algo := range sftp.ValidMACs {
		if algo.IsValid(true) {
			names = append(names, algo.Name)
		}
	}

	return names
}

func sftpChoices(values, chosen []string) []choice { return multi(values, chosen) }

func pesitModeChoices(translate gui.Translator, current string) []choice {
	return pick([][2]string{
		{pesit.CompatibilityModeStandard, translate("servers.opt.pesit_modes.standard")},
		{pesit.CompatibilityModeNonStandard, translate("servers.opt.pesit_modes.non_standard")},
	}, current)
}

func tlsRequirementChoices(translate gui.Translator, current string) []choice {
	return pick([][2]string{
		{string(ftp.TLSOptional), translate("servers.opt.tls_requirements.optional")},
		{string(ftp.TLSMandatory), translate("servers.opt.tls_requirements.mandatory")},
		{string(ftp.TLSImplicit), translate("servers.opt.tls_requirements.implicit")},
	}, current)
}

func mdnAlgoChoices(translate gui.Translator, current string) []choice {
	pairs := [][2]string{{"", translate("servers.opt.mdn.none")}}
	for _, algo := range as2.SignatureAlgorithms() {
		pairs = append(pairs, [2]string{algo, algo})
	}

	return pick(pairs, current)
}

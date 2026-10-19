// Package protoform is what the screens of the local servers, the local
// clients and the remote partners share about protocols.
//
// The product has one protocol key per variant ("r66" and "r66-tls", "http"
// and "https"...): the key chooses the module, its configuration structures
// and its services, and it is what the database, the REST API, the command
// line and the configuration pushed by the manager carry. The web interface
// shows the protocol as a family (R66, HTTP, SFTP, FTP, PeSIT, WebDAV, AS2) and
// TLS as an attribute of the object (Driss, 2026-10-19: a "smart display" that
// leaves the model and the backend untouched); the stored key does not change.
//
// The package also holds the options of each family for each role (server,
// client, partner) as the product reads them, the part of a form that chooses
// the protocol and fills its options, and the rows of a details page.
package protoform

import (
	"cmp"
	"crypto/tls"
	"slices"
	"strconv"
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

// variant is one protocol key of the product: its family, whether it is the
// TLS variant, and the name the product shows for it (a datum rather than a
// label: the historic pages show it, the documentation names it).
type variant struct {
	family string
	tls    bool
	label  string
}

//nolint:gochecknoglobals // reference data of the product
var (
	variants = map[string]variant{
		r66.R66:          {FamilyR66, false, "R66"},
		r66.R66TLS:       {FamilyR66, true, "R66-TLS"},
		httpconst.HTTP:   {FamilyHTTP, false, "HTTP"},
		httpconst.HTTPS:  {FamilyHTTP, true, "HTTPS"},
		sftp.SFTP:        {FamilySFTP, false, "SFTP"},
		ftp.FTP:          {FamilyFTP, false, "FTP"},
		ftp.FTPS:         {FamilyFTP, true, "FTPS"},
		pesit.Pesit:      {FamilyPeSIT, false, "PeSIT"},
		pesit.PesitTLS:   {FamilyPeSIT, true, "PeSIT-TLS"},
		webdav.Webdav:    {FamilyWebDAV, false, "WebDAV"},
		webdav.WebdavTLS: {FamilyWebDAV, true, "WebDAV over HTTPS"},
		as2.AS2:          {FamilyAS2, false, "AS2"},
		as2.AS2TLS:       {FamilyAS2, true, "AS2 over HTTPS"},
	}

	familyLabels = map[string]string{
		FamilyR66: "R66", FamilyHTTP: "HTTP", FamilySFTP: "SFTP", FamilyFTP: "FTP",
		FamilyPeSIT: "PeSIT", FamilyWebDAV: "WebDAV", FamilyAS2: "AS2",
	}
)

// Family is a protocol as the interface shows it: the family and, when the
// product has one, its TLS variant.
type Family struct {
	Key   string // key of the family, which is the key of its plain variant
	Label string // "R66", "WebDAV"
	Plain string // protocol key without TLS, "" when the product only registers the TLS variant
	TLS   string // protocol key of the TLS variant, "" when the family has none (SFTP is secured by SSH)
}

// HasTLS tells whether the family exists with TLS.
func (f Family) HasTLS() bool { return f.TLS != "" }

// Protocol describes a protocol key of the product.
type Protocol struct {
	Key    string // "r66-tls"
	Label  string // "R66-TLS": the name of the product
	Family Family
	TLS    bool
}

// ProtocolOf describes a protocol key. An unknown key (a protocol the
// interface does not know yet) is its own family, without TLS variant.
func ProtocolOf(key string) *Protocol {
	v, ok := variants[key]
	if !ok {
		return &Protocol{Key: key, Label: key, Family: Family{Key: key, Label: key, Plain: key}}
	}

	return &Protocol{Key: key, Label: v.label, Family: familyOf(v.family), TLS: v.tls}
}

// familyOf builds a family from the protocols registered in the gateway: its
// plain and TLS variants exist only when their modules do.
func familyOf(key string) Family {
	f := Family{Key: key, Label: familyLabels[key]}
	if f.Label == "" {
		f.Label = key
	}

	for variantKey, v := range variants {
		if v.family != key || !protocols.Exists(variantKey) {
			continue
		}

		if v.tls {
			f.TLS = variantKey
		} else {
			f.Plain = variantKey
		}
	}

	if f.Plain == "" && f.TLS == "" && protocols.Exists(key) {
		f.Plain = key
	}

	return f
}

// Families lists the families of the protocols registered in the gateway,
// in the order of their labels, so that the filters and the forms follow
// the product and not this table.
func Families() []Family {
	seen := map[string]bool{}

	var list []Family

	for key := range protocols.List() {
		family := ProtocolOf(key).Family
		if seen[family.Key] {
			continue
		}

		seen[family.Key] = true
		list = append(list, family)
	}

	slices.SortFunc(list, func(a, b Family) int { return strings.Compare(a.Label, b.Label) })

	return list
}

// IsFamily tells whether a key names a registered family.
func IsFamily(key string) bool {
	return key != "" && slices.ContainsFunc(Families(), func(f Family) bool { return f.Key == key })
}

// FamilyKeyOf reads a family key, from the key of a family or from a protocol
// key of the product (the addresses of the historic filters); "" when
// nothing registered matches.
func FamilyKeyOf(key string) string {
	if IsFamily(key) {
		return key
	}

	if protocols.Exists(key) {
		return ProtocolOf(key).Family.Key
	}

	return ""
}

// Resolve gives the protocol key of a family with or without TLS: the TLS
// variant when asked and it exists, the plain variant otherwise. It fails on
// a family the gateway does not register.
func Resolve(family string, withTLS bool) (string, bool) {
	if !IsFamily(family) {
		return "", false
	}

	f := familyOf(family)

	switch {
	case withTLS && f.TLS != "":
		return f.TLS, true
	case f.Plain != "":
		return f.Plain, true
	default:
		return f.TLS, true
	}
}

// Describe names a protocol in a sentence: its family and, for a TLS
// variant, the mention of TLS with the name of the product ("R66, secured by
// TLS (R66-TLS)").
func Describe(translate gui.Translator, p *Protocol) string {
	if p.TLS {
		return translate("proto.described_tls", "family", p.Family.Label, "name", p.Label)
	}

	return p.Family.Label
}

// Choice is an option of a select, selected or not.
type Choice struct {
	Value    string
	Label    string
	Selected bool
}

func pick(pairs [][2]string, current string) []Choice {
	out := make([]Choice, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, Choice{Value: p[0], Label: p[1], Selected: p[0] == current})
	}

	return out
}

// multi lists the chosen values first, in their order (the preference order
// of a dual list), then the other values.
func multi(values, chosen []string) []Choice {
	out := make([]Choice, 0, len(values))

	for _, v := range chosen {
		if slices.Contains(values, v) {
			out = append(out, Choice{Value: v, Label: v, Selected: true})
		}
	}

	for _, v := range values {
		if !slices.Contains(chosen, v) {
			out = append(out, Choice{Value: v, Label: v})
		}
	}

	return out
}

// TLSVersionChoices lists the TLS versions a minimum version can be.
func TLSVersionChoices(current string) []Choice {
	return pick([][2]string{
		{protoutils.TLSv10, "TLS 1.0"},
		{protoutils.TLSv11, "TLS 1.1"},
		{protoutils.TLSv12, "TLS 1.2"},
		{protoutils.TLSv13, "TLS 1.3"},
	}, current)
}

// TLSCipherChoices lists the cipher suites one can enable, from the product's
// table (crypto/tls): those negotiable in TLS 1.2 or earlier, by decreasing
// decimal code, the insecure ones flagged as such. Operators name a suite by
// its code (Driss, 2026-10-19): the label reads "code - name", the value
// stays the name the product stores. The TLS 1.3 suites are always on and
// not configurable, so they are not offered.
func TLSCipherChoices(translate gui.Translator, chosen []string) []Choice {
	out := multi(tlsCipherNames(), chosen)
	for i := range out {
		out[i].Label = CipherLabel(out[i].Value)

		if protoutils.TLSCiphers[out[i].Value].Insecure {
			out[i].Label += " " + translate("proto.opt.tls_insecure")
		}
	}

	return out
}

func tlsCipherNames() []string {
	var names []string

	for name, suite := range protoutils.TLSCiphers {
		if slices.Contains(suite.SupportedVersions, uint16(tls.VersionTLS12)) {
			names = append(names, name)
		}
	}

	slices.SortFunc(names, func(a, b string) int {
		return cmp.Compare(protoutils.TLSCiphers[b].ID, protoutils.TLSCiphers[a].ID)
	})

	return names
}

// CipherLabel names a cipher suite as the operators do: its decimal code,
// then its name; a name unknown to the product is shown as it is.
func CipherLabel(name string) string {
	if suite, ok := protoutils.TLSCiphers[name]; ok {
		return strconv.FormatUint(uint64(suite.ID), 10) + " - " + name
	}

	return name
}

// The SSH algorithms a server or a client may use, from the SFTP module,
// which knows which ones are valid for a server and which for a client.
func sftpKeyExchanges(role Role) []string {
	names := make([]string, 0, len(sftp.ValidKeyExchanges))

	for _, algo := range sftp.ValidKeyExchanges {
		if algo.IsValid(role == RoleServer) {
			names = append(names, algo.Name)
		}
	}

	return names
}

func sftpCiphers(role Role) []string {
	names := make([]string, 0, len(sftp.ValidCiphers))

	for _, algo := range sftp.ValidCiphers {
		if algo.IsValid(role == RoleServer) {
			names = append(names, algo.Name)
		}
	}

	return names
}

func sftpMACs(role Role) []string {
	names := make([]string, 0, len(sftp.ValidMACs))

	for _, algo := range sftp.ValidMACs {
		if algo.IsValid(role == RoleServer) {
			names = append(names, algo.Name)
		}
	}

	return names
}

func pesitModeChoices(translate gui.Translator, current string) []Choice {
	return pick([][2]string{
		{pesit.CompatibilityModeStandard, translate("proto.opt.pesit_modes.standard")},
		{pesit.CompatibilityModeNonStandard, translate("proto.opt.pesit_modes.non_standard")},
	}, current)
}

func tlsRequirementChoices(translate gui.Translator, current string) []Choice {
	return pick([][2]string{
		{string(ftp.TLSOptional), translate("proto.opt.tls_requirements.optional")},
		{string(ftp.TLSMandatory), translate("proto.opt.tls_requirements.mandatory")},
		{string(ftp.TLSImplicit), translate("proto.opt.tls_requirements.implicit")},
	}, current)
}

func mdnAlgoChoices(translate gui.Translator, current string) []Choice {
	pairs := [][2]string{{"", translate("proto.opt.mdn.none")}}
	for _, algo := range as2.SignatureAlgorithms() {
		pairs = append(pairs, [2]string{algo, algo})
	}

	return pick(pairs, current)
}

// r66HashChoices lists the hash algorithms of the final check of an R66
// transfer, the product's default first.
func r66HashChoices(_ gui.Translator, current string) []Choice {
	pairs := make([][2]string, 0, len(r66HashAlgos))
	for _, algo := range r66HashAlgos {
		pairs = append(pairs, [2]string{algo, algo})
	}

	return pick(pairs, current)
}

package protoform

import (
	"net/url"
	"slices"
	"strings"

	"code.waarp.fr/lib/gui"
)

// The parameters of the protocol filters of a list: the families (repeated
// or comma separated; a protocol key of the product, as the historic
// addresses carry, counts as its family) and TLS (with, without or both).
const (
	FamilyParam = "protocol"
	TLSParam    = "tls"

	TLSYes = "yes"
	TLSNo  = "no"
)

// FilterParams are the parameters the protocol filters use.
//
//nolint:gochecknoglobals // reference data of the screens
var FilterParams = []string{FamilyParam, TLSParam}

// Filter is the protocol filters of a list: the families kept and TLS.
type Filter struct {
	Families []string
	TLS      string // TLSYes, TLSNo or "" for both
}

// ParseFilter reads the protocol filters of a query, dropping what is
// unknown.
func ParseFilter(v url.Values) *Filter {
	f := &Filter{}

	for _, raw := range v[FamilyParam] {
		for key := range strings.SplitSeq(raw, ",") {
			family := FamilyKeyOf(strings.TrimSpace(key))
			if family != "" && !slices.Contains(f.Families, family) {
				f.Families = append(f.Families, family)
			}
		}
	}

	if tls := v.Get(TLSParam); tls == TLSYes || tls == TLSNo {
		f.TLS = tls
	}

	return f
}

// Values encodes the filters, nothing when none is set.
func (f *Filter) Values() url.Values {
	v := url.Values{}

	if len(f.Families) > 0 {
		v.Set(FamilyParam, strings.Join(f.Families, ","))
	}

	if f.TLS != "" {
		v.Set(TLSParam, f.TLS)
	}

	return v
}

// Count is the number of active filters.
func (f *Filter) Count() int {
	count := len(f.Families)
	if f.TLS != "" {
		count++
	}

	return count
}

// Matches tells whether a protocol passes the filters.
func (f *Filter) Matches(p *Protocol) bool {
	if len(f.Families) > 0 && !slices.Contains(f.Families, p.Family.Key) {
		return false
	}

	return f.TLS == "" || (f.TLS == TLSYes) == p.TLS
}

// FilterBox is one family of the filters panel.
type FilterBox struct {
	Key     string
	Label   string
	Checked bool
}

// Boxes lists the families of the filters panel, the kept ones checked.
func (f *Filter) Boxes() []FilterBox {
	var boxes []FilterBox

	for _, family := range Families() {
		boxes = append(boxes,
			FilterBox{Key: family.Key, Label: family.Label, Checked: slices.Contains(f.Families, family.Key)})
	}

	return boxes
}

// TLSChoices is the "with or without TLS" select of the filters panel.
func (f *Filter) TLSChoices(translate gui.Translator) []Choice {
	return pick([][2]string{
		{"", translate("proto.filter_tls_all")},
		{TLSYes, translate("proto.filter_tls_yes")},
		{TLSNo, translate("proto.filter_tls_no")},
	}, f.TLS)
}

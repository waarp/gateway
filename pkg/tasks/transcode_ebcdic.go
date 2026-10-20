package tasks

import (
	_ "unsafe" // needed to patch the ebcdic library

	"github.com/indece-official/go-ebcdic"
	"golang.org/x/text/encoding"
)

const (
	ebcdic273  = ebcdic.EBCDIC273
	ebcdic500  = ebcdic.EBCDIC500
	ebcdic1141 = ebcdic.EBCDIC1141
	ebcdic1148 = ebcdic.EBCDIC1148

	ebcdicEuroChar = 0x9F
)

// unicodeToEbcdicMap is an exact copy of the unexported types of the same name
// defined in the go-ebcdic library. It MUST keep the exact same memory layout
// as the original for the go:linkname trick below to be safe.
type unicodeToEbcdicMap struct {
	HasEuroPatch bool
	EuroChar     byte
	Map          []byte
}

// The go-ebcdic library (v1.2.0) forgets to enable the Euro patch on the
// EBCDIC 1148 Unicode-to-EBCDIC conversion map, which means that the '€'
// character cannot be encoded. Since the maps are unexported, we use the
// "go:linkname" directive to access them and fix them at init time.

//nolint:gochecknoglobals //var is required here to patch the library before use
//go:linkname ebcdicMapsUnicodeToEBCDIC github.com/indece-official/go-ebcdic.mapsUnicodeToEBCDIC
var ebcdicMapsUnicodeToEBCDIC map[int]unicodeToEbcdicMap

//nolint:gochecknoinits //init is required here to patch the library before use
func init() {
	if m, ok := ebcdicMapsUnicodeToEBCDIC[ebcdic1148]; ok {
		m.HasEuroPatch = true
		m.EuroChar = ebcdicEuroChar
		ebcdicMapsUnicodeToEBCDIC[ebcdic1148] = m
	}
}

type ebcdicEncoding struct {
	encoder ebcdicEncoder
	decoder ebcdicDecoder
}

func newEBCDICEncoding(codePage int) ebcdicEncoding {
	return ebcdicEncoding{
		encoder: ebcdicEncoder{codePage: codePage},
		decoder: ebcdicDecoder{codePage: codePage},
	}
}

func (e ebcdicEncoding) NewDecoder() *encoding.Decoder {
	return &encoding.Decoder{Transformer: e.decoder}
}

func (e ebcdicEncoding) NewEncoder() *encoding.Encoder {
	return &encoding.Encoder{Transformer: e.encoder}
}

type ebcdicEncoder struct {
	codePage int
}

func (e ebcdicEncoder) Transform(dst, src []byte, _ bool) (nDst, nSrc int, err error) {
	res, err := ebcdic.Encode(string(src), e.codePage)
	copy(dst, res)

	return len(res), len(src), err
}

func (e ebcdicEncoder) Reset() {} // noop

type ebcdicDecoder struct {
	codePage int
}

func (e ebcdicDecoder) Transform(dst, src []byte, _ bool) (nDst, nSrc int, err error) {
	res, err := ebcdic.Decode(src, e.codePage)
	copy(dst, res)

	return len(res), len(src), err
}

func (e ebcdicDecoder) Reset() {} // noop

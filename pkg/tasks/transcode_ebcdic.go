package tasks

import (
	"fmt"
	"unicode/utf8"
	_ "unsafe" // needed to patch the ebcdic library

	"github.com/indece-official/go-ebcdic"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

const (
	ebcdic273  = ebcdic.EBCDIC273
	ebcdic500  = ebcdic.EBCDIC500
	ebcdic1141 = ebcdic.EBCDIC1141
	ebcdic1148 = ebcdic.EBCDIC1148

	ebcdicEuroChar = 0x9F
	ebcdicSubChar  = 0x3F
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

// unmappableRuneError is returned by the EBCDIC encoder when a rune cannot be
// represented in the target code page. It implements the (unexported)
// interface expected by encoding.ReplaceUnsupported, so wrapped encoders
// replace the rune with the EBCDIC substitute character instead.
type unmappableRuneError struct {
	r        rune
	codePage int
}

func (e *unmappableRuneError) Error() string {
	return fmt.Sprintf("ebcdic: rune %U cannot be encoded in code page %d", e.r, e.codePage)
}

func (e *unmappableRuneError) Replacement() byte { return ebcdicSubChar }

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

func (e ebcdicEncoder) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	// Read UTF-8 input rune per rune.
	for nSrc < len(src) {
		// Each rune is encoded as exactly one EBCDIC byte. So if we've reached
		// the end of the destination buffer, return ErrShortDst.
		if nDst == len(dst) {
			return nDst, nSrc, transform.ErrShortDst
		}

		// If the source ends in the middle of a rune and more input is coming,
		// return ErrShortSrc so that the caller provides the rest of the rune.
		if !atEOF && !utf8.FullRune(src[nSrc:]) {
			return nDst, nSrc, transform.ErrShortSrc
		}

		// Decode the next rune. A RuneError of size 1 means the input is not
		// valid UTF-8 (a real U+FFFD has a size of 3).
		r, size := utf8.DecodeRune(src[nSrc:])
		if r == utf8.RuneError && size == 1 {
			return nDst, nSrc, encoding.ErrInvalidUTF8
		}

		// Convert UTF-8 rune to EBCDIC byte.
		encoded, encErr := ebcdic.Encode(string(r), e.codePage)
		if encErr != nil {
			return nDst, nSrc, fmt.Errorf("%w", encErr)
		}

		// The library silently maps unknown runes to 0x00, so a 0x00 result
		// means the rune cannot be represented (unless the rune was NUL).
		if encoded[0] == 0x00 && r != 0 {
			return nDst, nSrc, &unmappableRuneError{r: r, codePage: e.codePage}
		}

		// Write EBCDIC byte to destination buffer. Increment nDst by 1, and
		// nSrc by the length of the rune.
		dst[nDst] = encoded[0]
		nDst++
		nSrc += size
	}

	return nDst, nSrc, nil
}

func (e ebcdicEncoder) Reset() {} // noop

type ebcdicDecoder struct {
	codePage int
}

func (e ebcdicDecoder) Transform(dst, src []byte, _ bool) (nDst, nSrc int, err error) {
	// Read EBCDIC input byte per byte.
	for _, b := range src {
		// Convert EBCDIC byte to UTF-8 rune.
		utf8Rune, decErr := ebcdic.Decode([]byte{b}, e.codePage)
		if decErr != nil {
			return nDst, nSrc, fmt.Errorf("%w", decErr)
		}

		// If the rune does not fit in the destination buffer, return ErrShortDst.
		if nDst+len(utf8Rune) > len(dst) {
			return nDst, nSrc, transform.ErrShortDst
		}

		// Copy UTF-8 rune to destination buffer. Increment nSrc by 1, and
		// nDst by the length of the rune.
		nDst += copy(dst[nDst:], utf8Rune)
		nSrc++
	}

	return nDst, nSrc, nil
}

func (e ebcdicDecoder) Reset() {} // noop

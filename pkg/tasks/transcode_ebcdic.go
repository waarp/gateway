package tasks

import (
	"github.com/indece-official/go-ebcdic"
	"golang.org/x/text/encoding"
)

const (
	ebcdic273  = ebcdic.EBCDIC273
	ebcdic500  = ebcdic.EBCDIC500
	ebcdic1141 = ebcdic.EBCDIC1141
	ebcdic1148 = ebcdic.EBCDIC1148
)

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

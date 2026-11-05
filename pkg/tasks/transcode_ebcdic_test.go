package tasks

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/indece-official/go-ebcdic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func TestEncodeEBCDIC(t *testing.T) {
	t.Parallel()

	srcBuf := strings.NewReader("This string special chars like § and Ö")
	expectedBytes := []byte{
		0xe3, 0x88, 0x89, 0xa2, 0x40, 0xa2, 0xa3, 0x99, 0x89, 0x95, 0x87, 0x40,
		0xa2, 0x97, 0x85, 0x83, 0x89, 0x81, 0x93, 0x40, 0x83, 0x88, 0x81, 0x99,
		0xa2, 0x40, 0x93, 0x89, 0x92, 0x85, 0x40, 0xb5, 0x40, 0x81, 0x95, 0x84,
		0x40, 0xec,
	}
	dstBuf := &bytes.Buffer{}

	src := transform.NewReader(srcBuf, unicode.UTF8.NewDecoder())
	dst := transform.NewWriter(dstBuf, ebcdicEncoder{codePage: ebcdic.EBCDIC037})

	_, err := io.Copy(dst, src)
	require.NoError(t, err)

	assert.Equal(t, expectedBytes, dstBuf.Bytes())
}

func TestDecodeEBCDIC(t *testing.T) {
	t.Parallel()

	srcBuf := bytes.NewReader([]byte{
		0xe3, 0x88, 0x89, 0xa2, 0x40, 0xa2, 0xa3, 0x99, 0x89, 0x95, 0x87, 0x40,
		0xa2, 0x97, 0x85, 0x83, 0x89, 0x81, 0x93, 0x40, 0x83, 0x88, 0x81, 0x99,
		0xa2, 0x40, 0x93, 0x89, 0x92, 0x85, 0x40, 0xb5, 0x40, 0x81, 0x95, 0x84,
		0x40, 0xec,
	})
	expectedString := "This string special chars like § and Ö"
	dstBuf := &strings.Builder{}

	src := transform.NewReader(srcBuf, ebcdicDecoder{codePage: ebcdic.EBCDIC037})
	dst := transform.NewWriter(dstBuf, unicode.UTF8.NewEncoder())

	_, err := io.Copy(dst, src)
	require.NoError(t, err)

	assert.Equal(t, expectedString, dstBuf.String())
}

func TestEBCDICEncoderTransform(t *testing.T) {
	t.Parallel()

	encoder := ebcdicEncoder{codePage: ebcdic.EBCDIC037}

	t.Run("short destination", func(t *testing.T) {
		t.Parallel()
		dst := make([]byte, 1)
		nDst, nSrc, err := encoder.Transform(dst, []byte("AB"), true)

		require.ErrorIs(t, err, transform.ErrShortDst)
		assert.Equal(t, 1, nDst)
		assert.Equal(t, 1, nSrc)
		assert.Equal(t, byte(0xc1), dst[0])
	})

	t.Run("empty destination", func(t *testing.T) {
		t.Parallel()
		nDst, nSrc, err := encoder.Transform(nil, []byte("A"), true)

		require.ErrorIs(t, err, transform.ErrShortDst)
		assert.Zero(t, nDst)
		assert.Zero(t, nSrc)
	})

	t.Run("incomplete source before EOF", func(t *testing.T) {
		t.Parallel()
		dst := make([]byte, 1)
		nDst, nSrc, err := encoder.Transform(dst, []byte{0xc3}, false)

		require.ErrorIs(t, err, transform.ErrShortSrc)
		assert.Zero(t, nDst)
		assert.Zero(t, nSrc)
	})

	t.Run("incomplete source at EOF", func(t *testing.T) {
		t.Parallel()
		dst := make([]byte, 1)
		nDst, nSrc, err := encoder.Transform(dst, []byte{0xc3}, true)

		require.ErrorIs(t, err, encoding.ErrInvalidUTF8)
		assert.Zero(t, nDst)
		assert.Zero(t, nSrc)
	})

	t.Run("invalid byte in source", func(t *testing.T) {
		t.Parallel()
		dst := make([]byte, 3)
		nDst, nSrc, err := encoder.Transform(dst, []byte("A\xffB"), true)

		require.ErrorIs(t, err, encoding.ErrInvalidUTF8)
		assert.Equal(t, 1, nDst)
		assert.Equal(t, 1, nSrc)
		assert.Equal(t, byte(0xc1), dst[0])
	})

	t.Run("replacement character", func(t *testing.T) {
		t.Parallel()
		dst := make([]byte, 1)
		nDst, nSrc, err := encoder.Transform(dst, []byte("\uFFFD"), true)

		var unmappable *unmappableRuneError
		require.ErrorAs(t, err, &unmappable)
		assert.Equal(t, '\uFFFD', unmappable.r)
		assert.Zero(t, nDst)
		assert.Zero(t, nSrc)
	})

	t.Run("NUL character", func(t *testing.T) {
		t.Parallel()
		dst := []byte{0xff}
		nDst, nSrc, err := encoder.Transform(dst, []byte{0x00}, true)

		require.NoError(t, err)
		assert.Equal(t, 1, nDst)
		assert.Equal(t, 1, nSrc)
		assert.Zero(t, dst[0])
	})

	t.Run("partial progress before incomplete source", func(t *testing.T) {
		t.Parallel()
		dst := make([]byte, 2)
		nDst, nSrc, err := encoder.Transform(dst, []byte{'A', 0xc3}, false)

		require.ErrorIs(t, err, transform.ErrShortSrc)
		assert.Equal(t, 1, nDst)
		assert.Equal(t, 1, nSrc)
		assert.Equal(t, byte(0xc1), dst[0])
	})

	t.Run("multibyte source before short destination", func(t *testing.T) {
		t.Parallel()
		dst := make([]byte, 1)
		nDst, nSrc, err := encoder.Transform(dst, []byte("ÖA"), true)

		require.ErrorIs(t, err, transform.ErrShortDst)
		assert.Equal(t, 1, nDst)
		assert.Equal(t, 2, nSrc)
		assert.Equal(t, byte(0xec), dst[0])
	})

	t.Run("unmappable rune", func(t *testing.T) {
		t.Parallel()
		dst := make([]byte, 2)
		nDst, nSrc, err := encoder.Transform(dst, []byte("A😀"), true)

		var unmappable *unmappableRuneError
		require.ErrorAs(t, err, &unmappable)
		assert.Equal(t, '😀', unmappable.r)
		assert.Equal(t, 1, nDst)
		assert.Equal(t, 1, nSrc)
		assert.Equal(t, byte(0xc1), dst[0])
	})

	t.Run("unknown code page", func(t *testing.T) {
		t.Parallel()
		invalidEncoder := ebcdicEncoder{codePage: -1}
		nDst, nSrc, err := invalidEncoder.Transform(make([]byte, 1), []byte("A"), true)

		require.Error(t, err)
		assert.Zero(t, nDst)
		assert.Zero(t, nSrc)
	})
}

func TestEBCDICDecoderTransform(t *testing.T) {
	t.Parallel()

	decoder := ebcdicDecoder{codePage: ebcdic.EBCDIC037}

	t.Run("first rune does not fit", func(t *testing.T) {
		t.Parallel()
		dst := make([]byte, 1)
		nDst, nSrc, err := decoder.Transform(dst, []byte{0xec}, true)

		require.ErrorIs(t, err, transform.ErrShortDst)
		assert.Zero(t, nDst)
		assert.Zero(t, nSrc)
	})

	t.Run("partial progress", func(t *testing.T) {
		t.Parallel()
		dst := make([]byte, 2)
		nDst, nSrc, err := decoder.Transform(dst, []byte{0xc1, 0xec}, true)

		require.ErrorIs(t, err, transform.ErrShortDst)
		assert.Equal(t, 1, nDst)
		assert.Equal(t, 1, nSrc)
		assert.Equal(t, byte('A'), dst[0])
	})

	t.Run("exact fit", func(t *testing.T) {
		t.Parallel()
		dst := make([]byte, 3)
		nDst, nSrc, err := decoder.Transform(dst, []byte{0xc1, 0xec}, true)

		require.NoError(t, err)
		assert.Equal(t, 3, nDst)
		assert.Equal(t, 2, nSrc)
		assert.Equal(t, "AÖ", string(dst))
	})

	t.Run("empty input and destination", func(t *testing.T) {
		t.Parallel()
		nDst, nSrc, err := decoder.Transform(nil, nil, true)

		require.NoError(t, err)
		assert.Zero(t, nDst)
		assert.Zero(t, nSrc)
	})

	t.Run("unknown code page", func(t *testing.T) {
		t.Parallel()
		invalidDecoder := ebcdicDecoder{codePage: -1}
		nDst, nSrc, err := invalidDecoder.Transform(make([]byte, 1), []byte{0xc1}, true)

		require.Error(t, err)
		assert.Zero(t, nDst)
		assert.Zero(t, nSrc)
	})
}

func TestEBCDICTransformsWithOneByteReads(t *testing.T) {
	t.Parallel()

	const text = "This string contains Ö and €"

	var encoded bytes.Buffer
	encoder := transform.NewWriter(&encoded, ebcdicEncoder{codePage: ebcdic.EBCDIC1148})
	_, err := io.Copy(encoder, iotest.OneByteReader(strings.NewReader(text)))
	require.NoError(t, err)
	require.NoError(t, encoder.Close())

	decoded, err := io.ReadAll(transform.NewReader(
		iotest.OneByteReader(bytes.NewReader(encoded.Bytes())),
		ebcdicDecoder{codePage: ebcdic.EBCDIC1148},
	))
	require.NoError(t, err)
	assert.Equal(t, text, string(decoded))
}

func TestEBCDICEuroPatch(t *testing.T) {
	t.Parallel()

	const text = "Price: 10€"

	for _, codePage := range []int{
		ebcdic.EBCDIC1140,
		ebcdic.EBCDIC1141,
		ebcdic.EBCDIC1148,
	} {
		t.Run(fmt.Sprintf("EBCDIC%d", codePage), func(t *testing.T) {
			t.Parallel()

			encoded, err := ebcdic.Encode(text, codePage)
			require.NoError(t, err)
			assert.Equal(t, byte(ebcdicEuroChar), encoded[len(encoded)-1])

			decoded, err := ebcdic.Decode(encoded, codePage)
			require.NoError(t, err)
			assert.Equal(t, text, decoded)
		})
	}
}

func TestEBCDICEuroPatchThroughTransformers(t *testing.T) {
	t.Parallel()

	const text = "Price: 10€"
	const codePage = ebcdic.EBCDIC1148

	var encoded bytes.Buffer
	encoder := transform.NewWriter(&encoded, ebcdicEncoder{codePage: codePage})
	_, err := io.WriteString(encoder, text)
	require.NoError(t, err)
	require.NoError(t, encoder.Close())
	assert.Equal(t, byte(ebcdicEuroChar), encoded.Bytes()[encoded.Len()-1])

	decoded, err := io.ReadAll(transform.NewReader(bytes.NewReader(encoded.Bytes()), ebcdicDecoder{codePage: codePage}))
	require.NoError(t, err)
	assert.Equal(t, text, string(decoded))
}

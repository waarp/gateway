package tasks

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/indece-official/go-ebcdic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func TestEncodeEBCDIC(t *testing.T) {
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

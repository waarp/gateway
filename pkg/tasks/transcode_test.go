package tasks

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// runTranscode converts content with the TRANSCODE task, in a file, and
// returns the content of the file afterwards.
func runTranscode(t *testing.T, from, to string, content []byte) ([]byte, error) {
	t.Helper()

	path := fs.JoinPath(t.TempDir(), "test.txt")
	require.NoError(t, fs.WriteFullFile(path, content))

	transCtx := &model.TransferContext{Transfer: &model.Transfer{LocalPath: path}}
	params := map[string]string{"fromCharset": from, "toCharset": to}

	runErr := (&transcodeTask{}).Run(t.Context(), params, nil, testhelpers.GetTestLogger(t), transCtx, nil)

	result, err := fs.ReadFullFile(path)
	require.NoError(t, err)

	return result, runErr
}

func TestTranscode(t *testing.T) {
	t.Parallel()

	// just a bunch of non-ASCII characters in a UTF-8 string
	const testString = "âàäçéêèîïôöûùüÿ"
	// this is the same string in Windows 1252 encoding
	testBytes := []byte{226, 224, 228, 231, 233, 234, 232, 238, 239, 244, 246, 251, 249, 252, 255}
	require.NotEqual(t, testString, string(testBytes)) // sanity check

	content, err := runTranscode(t, "Windows 1252", "UTF-8", testBytes)
	require.NoError(t, err)
	assert.Equal(t, testString, string(content))
}

func TestTranscodeEBCDIC(t *testing.T) {
	t.Parallel()

	// The expected bytes come from iconv (glibc 2.34).
	type testCase struct {
		name     string
		charset  string
		text     string
		expected []byte
	}

	testCases := []testCase{
		{
			"IBM 037", "IBM Code Page 037",
			"This string special chars like § and Ö",
			[]byte{
				0xe3, 0x88, 0x89, 0xa2, 0x40, 0xa2, 0xa3, 0x99, 0x89, 0x95, 0x87, 0x40,
				0xa2, 0x97, 0x85, 0x83, 0x89, 0x81, 0x93, 0x40, 0x83, 0x88, 0x81, 0x99,
				0xa2, 0x40, 0x93, 0x89, 0x92, 0x85, 0x40, 0xb5, 0x40, 0x81, 0x95, 0x84,
				0x40, 0xec,
			},
		},
		// go-ebcdic wrote these seven characters of the pages 500 and 1148
		// where the page 037 has them.
		{"IBM 500", "IBM Code Page 500", "[]!^¢¬|", []byte{0x4a, 0x5a, 0x4f, 0x5f, 0xb0, 0xba, 0xbb}},
		{"IBM 1148", "IBM Code Page 1148", "[]!^¢¬|", []byte{0x4a, 0x5a, 0x4f, 0x5f, 0xb0, 0xba, 0xbb}},
		{"Euro IBM 1140", "IBM Code Page 1140", "10€", []byte{0xf1, 0xf0, 0x9f}},
		{"Euro IBM 1141", "IBM Code Page 1141", "10€", []byte{0xf1, 0xf0, 0x9f}},
		{"Euro IBM 1148", "IBM Code Page 1148", "10€", []byte{0xf1, 0xf0, 0x9f}},
		{"NUL character", "IBM Code Page 037", "A\x00B", []byte{0xc1, 0x00, 0xc2}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := runTranscode(t, "UTF-8", tc.charset, []byte(tc.text))
			require.NoError(t, err)
			assert.Equal(t, tc.expected, encoded)

			decoded, err := runTranscode(t, tc.charset, "UTF-8", tc.expected)
			require.NoError(t, err)
			assert.Equal(t, tc.text, string(decoded))
		})
	}
}

func TestTranscodeFailure(t *testing.T) {
	t.Parallel()

	// A character that the target lacks makes the task fail, and leaves the
	// file as it was.
	for _, text := range []string{"A😀B", "A�B", "A\xffB"} {
		t.Run(fmt.Sprintf("%q", text), func(t *testing.T) {
			t.Parallel()

			content, err := runTranscode(t, "UTF-8", "IBM Code Page 037", []byte(text))
			require.Error(t, err)
			assert.Equal(t, text, string(content))
		})
	}
}

func TestTranscodeAliases(t *testing.T) {
	t.Parallel()

	content, err := runTranscode(t, "utf8", "IBM1148", []byte("10€"))
	require.NoError(t, err)
	assert.Equal(t, []byte{0xf1, 0xf0, 0x9f}, content)
}

func TestTranscodeValidate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		from, to string
		expected error
	}

	testCases := []testCase{
		{"No source", "", "UTF-8", ErrTranscodeNoSrcEncoding},
		{"No destination", "UTF-8", "", ErrTranscodeNoDstEncoding},
		{"Unknown source", "klingon", "UTF-8", ErrTranscodeInvalidEncoding},
		{"Unknown destination", "UTF-8", "klingon", ErrTranscodeInvalidEncoding},
		{"Identical", "UTF-8", "UTF-8", ErrTranscodeIdenticalEncodings},
		{"Identical through an alias", "utf8", "UTF-8", ErrTranscodeIdenticalEncodings},
		{"Deprecated character set", "X-User-Defined", "UTF-8", nil},
		{"New EBCDIC page", "UTF-8", "IBM Code Page 297", nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := (&transcodeTask{}).Validate(map[string]string{"fromCharset": tc.from, "toCharset": tc.to})
			if tc.expected == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expected)
			}
		})
	}
}

// TestTranscodeFormerNames checks that every name the task accepted before it
// relied on code.waarp.fr/lib/transcode is still accepted.
func TestTranscodeFormerNames(t *testing.T) {
	t.Parallel()

	names := []string{
		"UTF-8", "UTF-8 BOM", "UTF-16BE", "UTF-16LE", "UTF-16BE BOM", "UTF-16LE BOM",
		"UTF-32BE", "UTF-32LE", "UTF-32BE BOM", "UTF-32LE BOM",
		"IBM Code Page 273", "IBM Code Page 500", "IBM Code Page 1141", "IBM Code Page 1148",
	}

	for _, enc := range charmap.All {
		names = append(names, fmt.Sprint(enc))
	}

	require.Len(t, names, 60)

	for _, name := range names {
		pivot := "UTF-8"
		if name == pivot {
			pivot = "UTF-16LE"
		}

		require.NoError(t, (&transcodeTask{}).Validate(map[string]string{"fromCharset": name, "toCharset": pivot}), name)
	}
}

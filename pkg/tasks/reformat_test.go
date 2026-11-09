package tasks

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// runReformat lays content out with the REFORMAT task and the given
// arguments, and returns the content of the file afterwards.
func runReformat(t *testing.T, args map[string]string, content []byte) ([]byte, error) {
	t.Helper()

	path := fs.JoinPath(t.TempDir(), "records.txt")
	require.NoError(t, fs.WriteFullFile(path, content))

	transCtx := &model.TransferContext{Transfer: &model.Transfer{LocalPath: path}}
	runErr := (&reformatTask{}).Run(t.Context(), args, nil, testhelpers.GetTestLogger(t), transCtx, nil)

	result, err := fs.ReadFullFile(path)
	require.NoError(t, err)

	return result, runErr
}

func TestReformat(t *testing.T) {
	t.Parallel()

	ebcdic := map[string]string{"charset": "IBM Code Page 297"}

	testCases := []struct {
		name    string
		args    map[string]string
		in, out []byte
	}{
		{
			// The content is copied: 0x81 stands for no character in Windows
			// 1252, and decoding the file would fail on it.
			name: "CR LF to LF", args: map[string]string{
				"charset": "Windows 1252", "records": "lines", "outputRecords": "lines",
			},
			in: []byte("A\x81\r\nB\n"), out: []byte("A\x81\nB\n"),
		},
		{
			// An invalid UTF-8 byte goes through as it is.
			name: "UTF-8 lines to fixed-length records", args: map[string]string{
				"charset": "UTF-8", "records": "lines", "outputRecords": "fixed", "outputLength": "6",
			},
			in: []byte("AB\xffC\nD\n"), out: []byte("AB\xffC  D     "),
		},
		{
			// LF writes 0x0A on an EBCDIC side too: the PeSIT module cuts the
			// articles there.
			name: "EBCDIC RDW records to lines", args: withArgs(ebcdic, "records", "rdw", "outputRecords", "lines"),
			in:  []byte{0x00, 0x06, 0x00, 0x00, 0xc1, 0xff, 0x00, 0x05, 0x00, 0x00, 0xc2},
			out: []byte{0xc1, 0xff, 0x0a, 0xc2, 0x0a},
		},
		{
			name: "EBCDIC fixed-length records, spaces removed, to new lines",
			args: withArgs(ebcdic, "records", "fixed", "recordLength", "4", "trimPadding", "true",
				"outputRecords", "lines", "outputSeparator", "NEL"),
			in:  []byte{0xc1, 0x40, 0x40, 0x40, 0xc2, 0xc3, 0x40, 0x40},
			out: []byte{0xc1, 0x15, 0xc2, 0xc3, 0x15},
		},
		{
			name: "EBCDIC line feeds to records without separator",
			args: withArgs(ebcdic, "records", "lines", "outputRecords", "none"),
			in:   []byte{0xc1, 0xc2, 0x25, 0xc3, 0x25}, out: []byte{0xc1, 0xc2, 0xc3},
		},
		{
			name: "Lines to RDW records",
			args: withArgs(ebcdic, "records", "lines", "outputRecords", "rdw", "truncate", "true"),
			in:   []byte{0xc1, 0x25}, out: []byte{0x00, 0x05, 0x00, 0x00, 0xc1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, (&reformatTask{}).Validate(tc.args))

			out, err := runReformat(t, tc.args, tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.out, out)
		})
	}

	t.Run("A record too long", func(t *testing.T) {
		t.Parallel()

		args := map[string]string{"charset": "ISO 8859-1", "records": "lines", "outputRecords": "fixed", "outputLength": "2"}
		out, err := runReformat(t, args, []byte("ABC\n"))
		require.ErrorContains(t, err, "longer than 2 bytes")
		assert.Equal(t, []byte("ABC\n"), out, "the file stays as it was")
	})
}

func TestReformatValidate(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		args map[string]string
		err  error
	}{
		"No encoding": {map[string]string{"records": "lines", "outputRecords": "lines"}, ErrReformatNoEncoding},
		"Unknown encoding": {
			map[string]string{"charset": "klingon", "records": "lines", "outputRecords": "lines"},
			ErrTranscodeInvalidEncoding,
		},
		"No records":       {map[string]string{"charset": "UTF-8", "outputRecords": "lines"}, ErrTranscodeRecords},
		"Records none":     {map[string]string{"charset": "UTF-8", "records": "none", "outputRecords": "lines"}, ErrTranscodeRecords},
		"No output":        {map[string]string{"charset": "UTF-8", "records": "lines"}, ErrTranscodeRecords},
		"Output as source": {map[string]string{"charset": "UTF-8", "records": "lines", "outputRecords": "same"}, ErrTranscodeRecords},
		"Length not fixed": {map[string]string{"charset": "UTF-8", "records": "lines", "outputRecords": "lines", "outputLength": "8"}, ErrTranscodeRecords},
		"Separator not EBCDIC": {
			map[string]string{"charset": "UTF-8", "records": "lines", "outputRecords": "lines", "outputSeparator": "NEL"},
			ErrTranscodeRecords,
		},
		// Shift JIS gives no character boundary from a position in a record.
		"Multi-byte records cut": {
			map[string]string{
				"charset": "Shift JIS", "records": "lines", "outputRecords": "fixed", "outputLength": "8", "truncate": "true",
			},
			ErrTranscodeRecords,
		},
	} {
		require.ErrorIs(t, (&reformatTask{}).Validate(tc.args), tc.err, name)
	}

	require.NoError(t, (&reformatTask{}).Validate(map[string]string{
		"charset": "IBM1147", "records": "fixed", "recordLength": "80", "outputRecords": "rdw",
	}), "an alias of the encoding")
}

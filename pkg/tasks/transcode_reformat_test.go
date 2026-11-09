package tasks

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// runTranscodeReformat converts content with the TRANSCODE&REFORMAT task and
// the given arguments, and returns the content of the file afterwards.
func runTranscodeReformat(t *testing.T, args map[string]string, content []byte) ([]byte, error) {
	t.Helper()

	path := fs.JoinPath(t.TempDir(), "records.txt")
	require.NoError(t, fs.WriteFullFile(path, content))

	transCtx := &model.TransferContext{Transfer: &model.Transfer{LocalPath: path}}
	runErr := (&transcodeReformatTask{}).Run(t.Context(), args, nil, testhelpers.GetTestLogger(t), transCtx, nil)

	result, err := fs.ReadFullFile(path)
	require.NoError(t, err)

	return result, runErr
}

func withArgs(base map[string]string, extra ...string) map[string]string {
	args := maps.Clone(base)
	for i := 0; i+1 < len(extra); i += 2 {
		args[extra[i]] = extra[i+1]
	}

	return args
}

func TestTranscodeReformat(t *testing.T) {
	t.Parallel()

	toEBCDIC := map[string]string{"fromCharset": "ISO 8859-1", "toCharset": "IBM Code Page 297"}
	toASCII := map[string]string{"fromCharset": "IBM Code Page 297", "toCharset": "ISO 8859-1"}

	type testCase struct {
		name     string
		args     map[string]string
		in, out  []byte
		failures bool
	}

	testCases := []testCase{
		{
			// The case of the ticket: the PeSIT module of the gateway cuts the
			// articles at 0x0A, which a conversion of the whole file turns into
			// 0x25.
			name: "Whole file", args: toEBCDIC,
			in: []byte("AB\nCD"), out: []byte{0xc1, 0xc2, 0x25, 0xc3, 0xc4},
		},
		{
			name: "Lines, line ends kept", args: withArgs(toEBCDIC, "records", "lines"),
			in: []byte("AB\nCD\r\nEF"), out: []byte{0xc1, 0xc2, 0x0a, 0xc3, 0xc4, 0x0d, 0x0a, 0xc5, 0xc6},
		},
		{
			name: "EBCDIC lines to LF", args: withArgs(toASCII, "records", "lines", "outputRecords", "lines"),
			in:  []byte{0xc1, 0x0a, 0xc2, 0x0d, 0x0a, 0xc3, 0x25, 0xc4, 0x0d, 0x25, 0xc5, 0x15, 0xc6},
			out: []byte("A\nB\nC\nD\nE\nF"),
		},
		{
			name: "Lines to CRLF", args: withArgs(toASCII, "records", "lines", "outputRecords", "lines", "outputSeparator", "CRLF"),
			in: []byte{0xc1, 0x15, 0xc2, 0x15}, out: []byte("A\r\nB\r\n"),
		},
		{
			name: "Lines to EBCDIC LF and NL", args: withArgs(toEBCDIC, "records", "lines", "outputRecords", "lines", "outputSeparator", "NEL"),
			in: []byte("A\r\nB"), out: []byte{0xc1, 0x15, 0xc2},
		},
		{
			name: "Lines to EBCDIC LF", args: withArgs(toEBCDIC, "records", "lines", "outputRecords", "lines"),
			in: []byte("A\r\nB\n"), out: []byte{0xc1, 0x0a, 0xc2, 0x0a},
		},
		{
			name: "Lines without separator", args: withArgs(toEBCDIC, "records", "lines", "outputRecords", "none"),
			in: []byte("AB\nCD\n"), out: []byte{0xc1, 0xc2, 0xc3, 0xc4},
		},
		{
			name: "Fixed-length records, padding removed, to lines",
			args: withArgs(toASCII, "records", "fixed", "recordLength", "4", "trimPadding", "true", "outputRecords", "lines"),
			in:   []byte{0xc1, 0xc2, 0x40, 0x40, 0xc3, 0xc4, 0xc5, 0xc6}, out: []byte("AB\nCDEF\n"),
		},
		{
			name: "Fixed-length records kept",
			args: withArgs(toASCII, "records", "fixed", "recordLength", "4"),
			in:   []byte{0xc1, 0xc2, 0x40, 0x40, 0xc3, 0xc4, 0xc5, 0xc6}, out: []byte("AB  CDEF"),
		},
		{
			name: "Lines to fixed-length records",
			args: withArgs(toEBCDIC, "records", "lines", "outputRecords", "fixed", "outputLength", "4"),
			in:   []byte("AB\nCDEF"), out: []byte{0xc1, 0xc2, 0x40, 0x40, 0xc3, 0xc4, 0xc5, 0xc6},
		},
		{
			name: "Record too long", args: withArgs(toEBCDIC, "records", "lines", "outputRecords", "fixed", "outputLength", "2"),
			in: []byte("ABC"), failures: true,
		},
		{
			name: "Record too long, truncated",
			args: withArgs(toEBCDIC, "records", "lines", "outputRecords", "fixed", "outputLength", "2", "truncate", "true"),
			in:   []byte("ABC\nD"), out: []byte{0xc1, 0xc2, 0xc4, 0x40},
		},
		{
			name: "Lines to RDW", args: withArgs(toEBCDIC, "records", "lines", "outputRecords", "rdw"),
			in: []byte("AB\nC"), out: []byte{0x00, 0x06, 0x00, 0x00, 0xc1, 0xc2, 0x00, 0x05, 0x00, 0x00, 0xc3},
		},
		{
			name: "RDW to lines", args: withArgs(toASCII, "records", "rdw", "outputRecords", "lines"),
			in: []byte{0x00, 0x06, 0x00, 0x00, 0xc1, 0xc2, 0x00, 0x05, 0x00, 0x00, 0xc3}, out: []byte("AB\nC\n"),
		},
		{
			name: "Character the target lacks", args: map[string]string{"fromCharset": "UTF-8", "toCharset": "IBM Code Page 037"},
			in: []byte("A😀B"), failures: true,
		},
		{
			name: "Character the target lacks, substituted",
			args: map[string]string{"fromCharset": "UTF-8", "toCharset": "IBM Code Page 037", "substitute": "true"},
			in:   []byte("A😀B"), out: []byte{0xc1, 0x3f, 0xc2},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			content, err := runTranscodeReformat(t, tc.args, tc.in)
			if tc.failures {
				require.Error(t, err)
				assert.Equal(t, tc.in, content, "the file stays as it was")

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.out, content)
		})
	}
}

func TestTranscodeReformatValidate(t *testing.T) {
	t.Parallel()

	base := map[string]string{"fromCharset": "ISO 8859-1", "toCharset": "IBM Code Page 297"}

	for name, args := range map[string]map[string]string{
		"Unknown records":                    withArgs(base, "records", "pages"),
		"Fixed without length":               withArgs(base, "records", "fixed"),
		"Length too long":                    withArgs(base, "records", "fixed", "recordLength", "32761"),
		"Length without fixed records":       withArgs(base, "records", "lines", "recordLength", "80"),
		"Padding without fixed records":      withArgs(base, "records", "rdw", "trimPadding", "true"),
		"Unknown output":                     withArgs(base, "records", "lines", "outputRecords", "pages"),
		"Output records of a whole file":     withArgs(base, "outputRecords", "lines"),
		"Separator without lines":            withArgs(base, "records", "lines", "outputSeparator", "CRLF"),
		"Unknown separator":                  withArgs(base, "records", "lines", "outputRecords", "lines", "outputSeparator", "CR"),
		"EBCDIC new line to open systems":    {"fromCharset": "IBM Code Page 297", "toCharset": "ISO 8859-1", "records": "lines", "outputRecords": "lines", "outputSeparator": "NEL"},
		"Output length without fixed output": withArgs(base, "records", "lines", "outputRecords", "rdw", "outputLength", "80"),
		"Fixed output without length":        withArgs(base, "records", "lines", "outputRecords", "fixed"),
		"Truncate without bounded records":   withArgs(base, "records", "lines", "truncate", "true"),
		"Truncate a multi-byte encoding": {
			"fromCharset": "UTF-8", "toCharset": "Shift JIS", "records": "lines", "outputRecords": "fixed", "outputLength": "80", "truncate": "true",
		},
	} {
		err := (&transcodeReformatTask{}).Validate(args)
		require.ErrorIs(t, err, ErrTranscodeRecords, name)
	}

	for name, args := range map[string]map[string]string{
		"No records":                        base,
		"Whole file, as the source":         withArgs(base, "records", "none", "outputRecords", "same"),
		"Fixed records truncated as source": withArgs(base, "records", "fixed", "recordLength", "80", "truncate", "true"),
		"Lines to fixed":                    withArgs(base, "records", "lines", "outputRecords", "fixed", "outputLength", "80", "truncate", "true"),
	} {
		require.NoError(t, (&transcodeReformatTask{}).Validate(args), name)
	}
}

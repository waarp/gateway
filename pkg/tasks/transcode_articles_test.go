package tasks

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/articles"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// runWithInfo runs a task on content, with the given transfer information, and
// returns the file and the transfer information afterwards.
func runWithInfo(t *testing.T, task model.TaskRunner, args map[string]string, info map[string]any,
	content []byte,
) ([]byte, map[string]any, error) {
	t.Helper()

	path := fs.JoinPath(t.TempDir(), "articles.txt")
	require.NoError(t, fs.WriteFullFile(path, content))

	transCtx := &model.TransferContext{Transfer: &model.Transfer{LocalPath: path, TransferInfo: info}}
	runErr := task.Run(t.Context(), args, nil, testhelpers.GetTestLogger(t), transCtx, nil)

	result, err := fs.ReadFullFile(path)
	require.NoError(t, err)

	return result, transCtx.Transfer.TransferInfo, runErr
}

// lengthsIn are the article lengths of the transfer information, as a list.
func lengthsIn(t *testing.T, info map[string]any) []uint16 {
	t.Helper()

	lengths, err := transferLengths(info)
	require.NoError(t, err)

	var out []uint16
	for length := range lengths.All() {
		out = append(out, length)
	}

	return out
}

// The lengths of the articles come from the transfer information, where a
// PeSIT reception or a SETINFO task writes them.
func TestArticlesRead(t *testing.T) {
	t.Parallel()

	t.Run("Given articles turned into lines, without converting them", func(t *testing.T) {
		t.Parallel()

		// The lengths as a reception writes them, then as the database gives
		// them back.
		for name, lengths := range map[string]any{
			"in memory":  articles.NewLengths(2, 0, 3),
			"from JSON":  []any{2, 0, 3},
			"compressed": []any{map[string]any{"l": 2, "c": 1}, 0, 3},
		} {
			info := map[string]any{articles.LengthsKey: lengths, articles.FormatKey: articles.FormatVariable}
			out, info, err := runWithInfo(t, &reformatTask{}, map[string]string{
				"charset": "ISO 8859-1", "records": "articles", "outputRecords": "lines",
			}, info, []byte("ABCDE"))

			require.NoError(t, err, name)
			assert.Equal(t, "AB\n\nCDE\n", string(out), name)
			assert.Equal(t, articles.SeparatorLF, info[articles.SeparatorKey], name)
			assert.Equal(t, articles.FormatVariable, info[articles.FormatKey], name)
			assert.NotContains(t, info, articles.LengthsKey, "lines are cut at their line end")
		}
	})

	t.Run("Given lengths set by SETINFO", func(t *testing.T) {
		t.Parallel()

		// Nothing ties the lengths to a protocol: a SETINFO task placed before
		// gives them to a file of any origin.
		_, info, err := runWithInfo(t, &setInfoTask{}, map[string]string{
			articles.LengthsKey: "[2,0,3]",
		}, map[string]any{}, nil)
		require.NoError(t, err)

		out, info, err := runWithInfo(t, &reformatTask{}, map[string]string{
			"charset": "ISO 8859-1", "records": "articles", "outputRecords": "lines",
		}, info, []byte("ABCDE"))

		require.NoError(t, err)
		assert.Equal(t, "AB\n\nCDE\n", string(out))
	})

	t.Run("Given EBCDIC articles converted into lines", func(t *testing.T) {
		t.Parallel()

		out, info, err := runWithInfo(t, &transcodeReformatTask{}, map[string]string{
			"fromCharset": "IBM Code Page 297", "toCharset": "ISO 8859-1",
			"records": "articles", "outputRecords": "lines", "outputSeparator": "CRLF",
		}, map[string]any{articles.LengthsKey: articles.NewLengths(2, 1)}, []byte{0xc1, 0xc2, 0xc3})

		require.NoError(t, err)
		assert.Equal(t, "AB\r\nC\r\n", string(out))
		assert.Equal(t, articles.SeparatorCRLF, info[articles.SeparatorKey])
	})

	t.Run("Given articles laid out as the source", func(t *testing.T) {
		t.Parallel()

		toLatin := map[string]string{"fromCharset": "UTF-8", "toCharset": "ISO 8859-1", "records": "articles"}

		// "été" takes 5 bytes in UTF-8, 3 in ISO 8859-1: the lengths follow.
		out, info, err := runWithInfo(t, &transcodeReformatTask{}, toLatin,
			map[string]any{articles.LengthsKey: articles.NewLengths(5, 1), articles.FormatKey: articles.FormatVariable},
			[]byte("\xc3\xa9t\xc3\xa9x"))

		require.NoError(t, err)
		assert.Equal(t, "\xe9t\xe9x", string(out))
		assert.Equal(t, []uint16{3, 1}, lengthsIn(t, info))
		assert.Equal(t, articles.FormatVariable, info[articles.FormatKey], "the format of the articles kept")

		// Articles of one length that keep one length stay of a fixed length.
		_, info, err = runWithInfo(t, &transcodeReformatTask{}, toLatin,
			map[string]any{articles.LengthsKey: articles.NewLengths(3, 3), articles.FormatKey: articles.FormatFixed},
			[]byte("\xc3\xa9t\xc3\xa9x"))

		require.NoError(t, err)
		assert.Equal(t, []uint16{2, 2}, lengthsIn(t, info))
		assert.Equal(t, articles.FormatFixed, info[articles.FormatKey], "still one length: the format kept")

		// Articles of one length that no longer have one are of a variable
		// length: a PeSIT send of a fixed length would not follow them.
		_, info, err = runWithInfo(t, &transcodeReformatTask{}, toLatin,
			map[string]any{articles.LengthsKey: articles.NewLengths(3, 3), articles.FormatKey: articles.FormatFixed},
			[]byte("\xc3\xa9tabc"))

		require.NoError(t, err)
		assert.Equal(t, []uint16{2, 3}, lengthsIn(t, info))
		assert.Equal(t, articles.FormatVariable, info[articles.FormatKey], "no longer one length")

		// The task gives no format to articles that had none.
		_, info, err = runWithInfo(t, &transcodeReformatTask{}, toLatin,
			map[string]any{articles.LengthsKey: articles.NewLengths(3, 3)}, []byte("\xc3\xa9tabc"))

		require.NoError(t, err)
		assert.NotContains(t, info, articles.FormatKey, "no format given, none written")
	})

	t.Run("Given no lengths in the transfer info", func(t *testing.T) {
		t.Parallel()

		out, _, err := runWithInfo(t, &reformatTask{}, map[string]string{
			"charset": "ISO 8859-1", "records": "articles", "outputRecords": "lines",
		}, map[string]any{articles.SeparatorKey: articles.SeparatorLF}, []byte("ABCDE"))

		require.ErrorIs(t, err, ErrTranscodeRecords)
		require.ErrorContains(t, err, "gives no article lengths")
		assert.Equal(t, "ABCDE", string(out), "the file stays as it was")
	})

	t.Run("Given lengths that do not make the file", func(t *testing.T) {
		t.Parallel()

		out, _, err := runWithInfo(t, &reformatTask{}, map[string]string{
			"charset": "ISO 8859-1", "records": "articles", "outputRecords": "lines",
		}, map[string]any{articles.LengthsKey: articles.NewLengths(2, 2)}, []byte("ABCDE"))

		require.ErrorContains(t, err, "the data goes on beyond the lengths given")
		assert.Equal(t, "ABCDE", string(out), "the file stays as it was")
	})
}

// After a conversion by records, the article keys describe the converted file.
func TestArticlesWritten(t *testing.T) {
	t.Parallel()

	toEBCDIC := map[string]string{"fromCharset": "ISO 8859-1", "toCharset": "IBM Code Page 297", "records": "lines"}
	sent := func() map[string]any {
		return map[string]any{articles.SeparatorKey: articles.SeparatorLF, articles.FormatKey: articles.FormatVariable}
	}

	t.Run("Given lines written end to end", func(t *testing.T) {
		t.Parallel()

		// The file stays in EBCDIC, without line ends: a PeSIT send cuts it at
		// the lengths of its articles.
		out, info, err := runWithInfo(t, &transcodeReformatTask{}, withArgs(toEBCDIC, "outputRecords", "none"),
			sent(), []byte("AB\nC\n"))

		require.NoError(t, err)
		assert.Equal(t, []byte{0xc1, 0xc2, 0xc3}, out)
		assert.Equal(t, []uint16{2, 1}, lengthsIn(t, info))
		assert.Equal(t, articles.FormatVariable, info[articles.FormatKey])
		assert.NotContains(t, info, articles.SeparatorKey)
	})

	t.Run("Given lines written at a fixed length", func(t *testing.T) {
		t.Parallel()

		_, info, err := runWithInfo(t, &transcodeReformatTask{},
			withArgs(toEBCDIC, "outputRecords", "fixed", "outputLength", "4"), sent(), []byte("AB\nC\n"))

		require.NoError(t, err)
		assert.Equal(t, []uint16{4, 4}, lengthsIn(t, info))
		assert.Equal(t, articles.FormatFixed, info[articles.FormatKey])
		assert.NotContains(t, info, articles.SeparatorKey)
	})

	t.Run("Given RDW records written", func(t *testing.T) {
		t.Parallel()

		_, info, err := runWithInfo(t, &transcodeReformatTask{}, withArgs(toEBCDIC, "outputRecords", "rdw"),
			map[string]any{articles.SeparatorKey: articles.SeparatorLF, articles.LengthsKey: articles.NewLengths(3, 2)},
			[]byte("AB\nC\n"))

		require.NoError(t, err)
		assert.NotContains(t, info, articles.SeparatorKey)
		assert.NotContains(t, info, articles.LengthsKey)
	})

	t.Run("Given EBCDIC new lines written", func(t *testing.T) {
		t.Parallel()

		_, info, err := runWithInfo(t, &transcodeReformatTask{},
			withArgs(toEBCDIC, "outputRecords", "lines", "outputSeparator", "NEL"), sent(), []byte("AB\nC\n"))

		require.NoError(t, err)
		assert.NotContains(t, info, articles.SeparatorKey, "the PeSIT module cuts at LF or CR LF only")
	})

	t.Run("Given lines laid out as the source", func(t *testing.T) {
		t.Parallel()

		_, info, err := runWithInfo(t, &transcodeReformatTask{}, toEBCDIC, sent(), []byte("AB\nC\n"))

		require.NoError(t, err)
		assert.Equal(t, sent(), info, "each line keeps its line end: the keys stay")

		// Article lengths counted before a conversion of the same size still
		// describe the file.
		kept := sent()
		kept[articles.LengthsKey] = articles.NewLengths(3, 2)
		_, info, err = runWithInfo(t, &transcodeReformatTask{}, toEBCDIC, kept, []byte("AB\nC\n"))

		require.NoError(t, err)
		assert.Equal(t, []uint16{3, 2}, lengthsIn(t, info), "one byte for one byte: the lengths stay")

		// Counted before a conversion that changes the size of a character,
		// they would cut the file at the wrong places: they are left out.
		stale := sent()
		stale[articles.LengthsKey] = articles.NewLengths(4, 2)
		out, info, err := runWithInfo(t, &transcodeReformatTask{}, map[string]string{
			"fromCharset": "UTF-8", "toCharset": "ISO 8859-1", "records": "lines",
		}, stale, []byte("\xc3\xa9t\nC\n"))

		require.NoError(t, err)
		assert.Equal(t, "\xe9t\nC\n", string(out))
		assert.NotContains(t, info, articles.LengthsKey, "the lengths no longer describe the file")
		assert.Equal(t, articles.SeparatorLF, info[articles.SeparatorKey], "the line end stays")

		// The same when the characters grow.
		grown := sent()
		grown[articles.LengthsKey] = articles.NewLengths(3, 2)
		out, info, err = runWithInfo(t, &transcodeReformatTask{}, map[string]string{
			"fromCharset": "ISO 8859-1", "toCharset": "UTF-8", "records": "lines",
		}, grown, []byte("\xe9t\nC\n"))

		require.NoError(t, err)
		assert.Equal(t, "\xc3\xa9t\nC\n", string(out))
		assert.NotContains(t, info, articles.LengthsKey, "the lengths no longer describe the file")
	})

	t.Run("Given a record no article can hold", func(t *testing.T) {
		t.Parallel()

		long := strings.Repeat("A", 70000) + "\n"
		_, info, err := runWithInfo(t, &reformatTask{}, map[string]string{
			"charset": "ISO 8859-1", "records": "lines", "outputRecords": "none",
		}, sent(), []byte(long))

		require.NoError(t, err)
		assert.NotContains(t, info, articles.LengthsKey, "lengths beyond an article are left out")
	})

	t.Run("Given a whole file converted", func(t *testing.T) {
		t.Parallel()

		// TRANSCODE leaves the keys as they are; it only warns when the sizes
		// of the characters may change.
		info := map[string]any{articles.LengthsKey: articles.NewLengths(2, 1)}
		_, info, err := runWithInfo(t, &transcodeTask{}, map[string]string{
			"fromCharset": "UTF-8", "toCharset": "IBM Code Page 297",
		}, info, []byte("AB\nC\n"))

		require.NoError(t, err)
		assert.Equal(t, []uint16{2, 1}, lengthsIn(t, info))
	})

	t.Run("Given a whole file converted, without records", func(t *testing.T) {
		t.Parallel()

		// One byte for one byte: the lengths still describe the file.
		whole := map[string]string{"fromCharset": "ISO 8859-1", "toCharset": "IBM Code Page 297"}
		_, info, err := runWithInfo(t, &transcodeReformatTask{}, whole,
			map[string]any{articles.LengthsKey: articles.NewLengths(3, 1)}, []byte("\xe9tAB"))

		require.NoError(t, err)
		assert.Equal(t, []uint16{3, 1}, lengthsIn(t, info))

		// A character that changes size: the file goes from 4 to 5 bytes, a
		// PeSIT send cut at the lengths given would stop short. They are left
		// out, the format stays.
		out, info, err := runWithInfo(t, &transcodeReformatTask{},
			map[string]string{"fromCharset": "ISO 8859-1", "toCharset": "UTF-8"},
			map[string]any{articles.LengthsKey: articles.NewLengths(3, 1), articles.FormatKey: articles.FormatVariable},
			[]byte("\xe9tAB"))

		require.NoError(t, err)
		assert.Equal(t, "\xc3\xa9tAB", string(out))
		assert.NotContains(t, info, articles.LengthsKey, "the lengths no longer describe the file")
		assert.Equal(t, articles.FormatVariable, info[articles.FormatKey])
	})
}

// The value articles is checked as the others when the rule is saved; its
// lengths are only read when the task runs.
func TestArticlesValidate(t *testing.T) {
	t.Parallel()

	require.NoError(t, (&reformatTask{}).Validate(map[string]string{
		"charset": "IBM Code Page 297", "records": "articles", "outputRecords": "lines",
	}))
	require.NoError(t, (&transcodeReformatTask{}).Validate(map[string]string{
		"fromCharset": "IBM Code Page 297", "toCharset": "UTF-8", "records": "articles",
	}))
	require.ErrorIs(t, (&reformatTask{}).Validate(map[string]string{
		"charset": "UTF-8", "records": "articles", "recordLength": "80", "outputRecords": "lines",
	}), ErrTranscodeRecords, "a length applies to fixed-length records only")
}

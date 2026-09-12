package pesit

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordsOf collects the records handed by forEachRecord, as strings.
func recordsOf(t *testing.T, content string, separator []byte) []string {
	t.Helper()

	var records []string

	require.NoError(t, forEachRecord(strings.NewReader(content), separator,
		func(record []byte) error {
			records = append(records, string(record))

			return nil
		}))

	return records
}

func TestForEachRecord(t *testing.T) {
	t.Parallel()

	lf, crlf := []byte("\n"), []byte("\r\n")

	assert.Equal(t, []string{"one", "", "three"}, recordsOf(t, "one\n\nthree\n", lf),
		"an empty record is a record, and the separator ending the file adds none")
	assert.Equal(t, []string{"one", "last"}, recordsOf(t, "one\nlast", lf),
		"a last record without separator is handed as well")
	assert.Empty(t, recordsOf(t, "", lf), "an empty file has no record")
	assert.Equal(t, []string{"one", "two"}, recordsOf(t, "one\r\ntwo\r\n", crlf))
	assert.Equal(t, []string{"one\r", "two"}, recordsOf(t, "one\r\ntwo", lf),
		"with LF declared, a carriage return is data")
	assert.Equal(t, []string{"one\n", "two"}, recordsOf(t, "one\ntwo\r\n", crlf),
		"with CRLF declared, a bare line feed is data")

	long := strings.Repeat("x", 3*recordReadSize)
	assert.Equal(t, []string{long, "y"}, recordsOf(t, long+"\ny", lf),
		"a record longer than the read buffer is handed whole")

	boom := errors.New("boom") //nolint:err113 // test
	assert.ErrorIs(t, forEachRecord(strings.NewReader("a\nb\n"), lf,
		func([]byte) error { return boom }), boom)
}

func TestParseArticlesSeparator(t *testing.T) {
	t.Parallel()

	for name, expected := range map[string][]byte{"LF": {'\n'}, "lf ": {'\n'}, "CRLF": {'\r', '\n'}} {
		sep, err := parseArticlesSeparator(name)
		require.NoError(t, err)
		assert.Equal(t, expected, sep)
	}

	_, err := parseArticlesSeparator("TAB")
	assert.ErrorIs(t, err, errUnknownSeparator)
}

func TestLongestRecord(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "records.txt")
	require.NoError(t, os.WriteFile(path, []byte("ab\nabcdef\n\nabc"), 0o600))

	size, err := longestRecordOf(path, []byte("\n"))
	require.NoError(t, err)
	assert.Equal(t, uint16(6), size)

	empty := filepath.Join(dir, "empty.txt")
	require.NoError(t, os.WriteFile(empty, nil, 0o600))

	size, err = longestRecordOf(empty, []byte("\n"))
	require.NoError(t, err)
	assert.Equal(t, uint16(1), size, "an empty file is announced with a valid size")

	_, err = longestRecordOf(filepath.Join(dir, "missing.txt"), []byte("\n"))
	require.Error(t, err)

	huge := filepath.Join(dir, "huge.txt")
	require.NoError(t, os.WriteFile(huge, bytes.Repeat([]byte{'x'}, 70000), 0o600))

	_, err = longestRecordOf(huge, []byte("\n"))
	assert.ErrorIs(t, err, errRecordTooLongPI)

	// An open stream is read from its beginning and left where it was.
	stream := bytes.NewReader([]byte("ab\nabcdef\n"))
	_, err = stream.Seek(3, io.SeekStart)
	require.NoError(t, err)

	size, err = longestRecord(stream, []byte("\n"))
	require.NoError(t, err)
	assert.Equal(t, uint16(6), size)

	position, err := stream.Seek(0, io.SeekCurrent)
	require.NoError(t, err)
	assert.Equal(t, int64(3), position, "the stream is back at its position")
}

// fakeArticleSender records the articles it is asked to send.
type fakeArticleSender struct {
	manual   bool
	articles []*bytes.Buffer
	startErr error
}

func (f *fakeArticleSender) SetManualArticleHandling(manual bool) bool {
	f.manual = manual

	return true
}

func (f *fakeArticleSender) StartNextSendArticle() (io.Writer, error) {
	if f.startErr != nil {
		return nil, f.startErr
	}

	article := &bytes.Buffer{}
	f.articles = append(f.articles, article)

	return article, nil
}

func (f *fakeArticleSender) sent() []string {
	sent := make([]string, 0, len(f.articles))
	for _, article := range f.articles {
		sent = append(sent, article.String())
	}

	return sent
}

func TestSendDelimitedArticles(t *testing.T) {
	t.Parallel()

	t.Run("one article per record, the separator excluded", func(t *testing.T) {
		t.Parallel()

		sender := &fakeArticleSender{}
		require.NoError(t, sendDelimitedArticles(sender,
			strings.NewReader("first\n\nthird record\nlast"), []byte("\n"), 12))
		assert.True(t, sender.manual, "articles are handled one by one")
		assert.Equal(t, []string{"first", "", "third record", "last"}, sender.sent())
	})

	t.Run("a record longer than the article size is refused before being sent", func(t *testing.T) {
		t.Parallel()

		sender := &fakeArticleSender{}
		err := sendDelimitedArticles(sender, strings.NewReader("ok\ntoo long\n"), []byte("\n"), 5)
		require.ErrorIs(t, err, errRecordTooLong)
		assert.Equal(t, []string{"ok"}, sender.sent(), "the records before it were sent")
	})

	t.Run("a failure to start an article is reported", func(t *testing.T) {
		t.Parallel()

		boom := errors.New("boom") //nolint:err113 // test
		sender := &fakeArticleSender{startErr: boom}
		assert.ErrorIs(t, sendDelimitedArticles(sender, strings.NewReader("a\n"), []byte("\n"), 5), boom)
	})
}

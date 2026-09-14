package pesit

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

// recordReadSize is the buffer size used to read a delimited text file.
const recordReadSize = 64 * 1024

var (
	errUnknownSeparator = errors.New("unknown articles separator")
	errRecordTooLong    = errors.New("record longer than the article size")
	errRecordTooLongPI  = errors.New("record longer than what PeSIT can announce")
)

// articleSender is what a PeSIT transfer offers to send a file article by
// article. Both the client and the server transfers implement it.
type articleSender interface {
	SetManualArticleHandling(manual bool) bool
	StartNextSendArticle() (io.Writer, error)
}

// parseArticlesSeparator returns the byte sequence ending each record of a
// delimited text file, from its name in the transfer info.
func parseArticlesSeparator(name string) ([]byte, error) {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "LF":
		return []byte{'\n'}, nil
	case "CRLF":
		return []byte{'\r', '\n'}, nil
	default:
		return nil, fmt.Errorf("%w: %q (expected LF or CRLF)", errUnknownSeparator, name)
	}
}

// forEachRecord hands every record of a delimited text file to fn, without
// its separator. A last record without separator is handed as well; an
// empty file has no record.
func forEachRecord(file io.Reader, separator []byte, fn func(record []byte) error) error {
	reader := bufio.NewReaderSize(file, recordReadSize)

	for {
		record, err := reader.ReadBytes('\n')
		if len(record) > 0 {
			if fnErr := fn(bytes.TrimSuffix(record, separator)); fnErr != nil {
				return fnErr
			}
		}

		if errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to read the file: %w", err)
		}
	}
}

// longestRecord returns the length of the longest record of an open file,
// the separator excluded: the article size (PI 32) to announce when the
// configuration gives none. The file is read from its beginning and left at
// the position it had. At least 1, so that an empty file is announced with a
// valid size.
func longestRecord(file io.ReadSeeker, separator []byte) (uint16, error) {
	position, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, fmt.Errorf("failed to get the file position: %w", err)
	}

	if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil {
		return 0, fmt.Errorf("failed to rewind the file: %w", seekErr)
	}

	longest := 0

	if readErr := forEachRecord(file, separator, func(record []byte) error {
		longest = max(longest, len(record))

		return nil
	}); readErr != nil {
		return 0, readErr
	}

	if _, seekErr := file.Seek(position, io.SeekStart); seekErr != nil {
		return 0, fmt.Errorf("failed to restore the file position: %w", seekErr)
	}

	if longest > math.MaxUint16 {
		return 0, fmt.Errorf("%w: %d bytes", errRecordTooLongPI, longest)
	}

	return uint16(max(longest, 1)), nil
}

// longestRecordOf is longestRecord for a file that is not open yet.
func longestRecordOf(path string, separator []byte) (uint16, error) {
	file, err := fs.Open(path)
	if err != nil {
		return 0, fmt.Errorf("failed to open the file: %w", err)
	}

	defer file.Close()

	return longestRecord(file, separator)
}

// sendDelimitedArticles sends a delimited text file as one article per
// record, the separator excluded. Every record must fit in the article size
// announced (PI 32): the receiver would cut a longer one.
func sendDelimitedArticles(trans articleSender, file io.Reader, separator []byte,
	articleSize int,
) error {
	trans.SetManualArticleHandling(true)

	return forEachRecord(file, separator, func(record []byte) error {
		if len(record) > articleSize {
			return fmt.Errorf("%w: %d bytes for %d announced", errRecordTooLong,
				len(record), articleSize)
		}

		article, err := trans.StartNextSendArticle()
		if err != nil {
			return fmt.Errorf("failed to start the next article: %w", err)
		}

		if _, writeErr := article.Write(record); writeErr != nil {
			return fmt.Errorf("failed to send the article: %w", writeErr)
		}

		return nil
	})
}

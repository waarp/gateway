package pesit

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
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
	errBadLengths       = errors.New("invalid articles lengths")
)

// lengthsRun is a run of consecutive articles of the same L.
type lengthsRun struct {
	L uint16 `json:"l"`
	C uint64 `json:"c"`
}

//nolint:wrapcheck //wrapping adds nothing here
func (l *lengthsRun) MarshalJSON() ([]byte, error) {
	if l.C == 1 {
		return json.Marshal(l.L)
	}

	type lengths lengthsRun
	ls := lengths(*l)

	return json.Marshal(&ls)
}

//nolint:wrapcheck //wrapping adds nothing here
func (l *lengthsRun) UnmarshalJSON(b []byte) error {
	type lengths lengthsRun
	var ls lengths

	if err := json.Unmarshal(b, &ls); err != nil {
		var i uint16
		if json.Unmarshal(b, &i) != nil {
			return err
		}

		ls.L = i
		ls.C = 1
	}

	if ls.C == 0 {
		return fmt.Errorf("%w: count cannot be zero", errBadLengths)
	}

	l.L = ls.L
	l.C = ls.C

	return nil
}

// articleLengths is the length of each article of a file, stored
// run-length encoded: a file made of many articles of the same size takes a
// few bytes in the transfer info rather than one number per article.
type articleLengths []lengthsRun

// newArticleLengths returns the given lengths in their compact form.
func newArticleLengths(lengths ...uint16) *articleLengths {
	l := &articleLengths{}
	for _, length := range lengths {
		l.Add(length)
	}

	return l
}

// Add appends the length of one more article.
func (l *articleLengths) Add(length uint16) {
	l.addRun(length, 1)
}

func (l *articleLengths) addRun(length uint16, count uint64) {
	if last := len(*l) - 1; last >= 0 && (*l)[last].L == length {
		(*l)[last].C += count

		return
	}

	*l = append(*l, lengthsRun{L: length, C: count})
}

// Len returns the number of articles.
func (l *articleLengths) Len() int {
	if l == nil {
		return 0
	}

	var n uint64
	for _, run := range *l {
		n += run.C
	}

	return int(n)
}

// Max returns the length of the longest article, 0 when there is none.
func (l *articleLengths) Max() uint16 {
	var longest uint16

	if l != nil {
		for _, run := range *l {
			longest = max(longest, run.L)
		}
	}

	return longest
}

// All iterates over the length of each article, in order.
func (l *articleLengths) All() iter.Seq[uint16] {
	return func(yield func(uint16) bool) {
		if l == nil {
			return
		}

		for _, run := range *l {
			for range run.C {
				if !yield(run.L) {
					return
				}
			}
		}
	}
}

func (l *articleLengths) String() string {
	b, err := json.Marshal(l)
	if err != nil {
		panic(err)
	}

	return string(b)
}

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

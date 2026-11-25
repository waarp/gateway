package tasks

import (
	"cmp"
	"errors"
	"fmt"
	"iter"
	"math"

	"code.waarp.fr/lib/transcode"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/articles"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// The records of the TRANSCODE&REFORMAT task. "records" tells how the source
// file is cut: as a whole (none, the default), in lines, in records of a
// fixed length, in variable-length records preceded by their length (RDW,
// as z/OS writes them), or at the lengths of the articles that the transfer
// information gives (articles), where a PeSIT reception or a SETINFO
// task writes them.
// "outputRecords" tells how the result is laid out: as the source (same, the
// default: a line keeps its own line end, byte for byte), without anything
// between records, in lines, in fixed-length records or in RDW records. Only
// the content of the records is converted.
//
// A line end is a byte sequence of the file. On the side of open systems, it is
// LF or CR LF. On an EBCDIC side, LF and CR LF are also the bytes 0x0A and
// 0x0D 0x0A: the PeSIT module of the gateway cuts a text file into articles there,
// whatever its character set; a source also has its lines recognized at the line
// feed (0x25) and the new line (0x15) of EBCDIC, control bytes that text never holds.

// The values of the record arguments.
const (
	recordsNone  = "none"
	recordsSame  = "same"
	recordsLines = "lines"
	recordsFixed = "fixed"
	recordsRDW   = "rdw"

	recordsArticles = "articles"

	separatorLF   = "LF"
	separatorCRLF = "CRLF"
	separatorNEL  = "NEL"

	// The longest record a length may give: the longest variable-length
	// record of z/OS.
	maxRecordLength = 32760
)

// recordArgs are the record arguments of a task, read from its parameters.
type recordArgs struct {
	Records         string   `json:"records"`
	RecordLength    jsonInt  `json:"recordLength"`
	TrimPadding     jsonBool `json:"trimPadding"`
	OutputRecords   string   `json:"outputRecords"`
	OutputSeparator string   `json:"outputSeparator"`
	OutputLength    jsonInt  `json:"outputLength"`
	Truncate        jsonBool `json:"truncate"`

	// lengths are the lengths of the articles of the source, from the transfer
	// information, when it is cut at them; written collects the lengths of the
	// records written, and tooLong reports one that no article can hold.
	lengths *articles.Lengths
	written *articles.Lengths
	tooLong bool
}

// recordsError is an incoherent record argument, named in the message so that
// the web interface shows it under its field.
func recordsError(arg, problem string) error {
	return fmt.Errorf("%w: %q %s", ErrTranscodeRecords, arg, problem)
}

// inputFormat is how the source file is cut into records.
func (t *recordArgs) inputFormat(source *transcode.Entry) (transcode.RecordFormat, error) {
	records := cmp.Or(t.Records, recordsNone)

	if records != recordsFixed {
		switch {
		case t.RecordLength != 0:
			return transcode.RecordFormat{}, recordsError("recordLength", "only applies to fixed-length records")
		case bool(t.TrimPadding):
			return transcode.RecordFormat{}, recordsError("trimPadding", "only applies to fixed-length records")
		}
	}

	switch records {
	case recordsNone:
		return transcode.Stream(), nil
	case recordsLines:
		ends, err := lineEnds(source)
		if err != nil {
			return transcode.RecordFormat{}, err
		}

		return transcode.Separated(ends...), nil
	case recordsFixed:
		if t.RecordLength < 1 || t.RecordLength > maxRecordLength {
			return transcode.RecordFormat{}, recordsError("recordLength", "must be between 1 and 32760")
		}

		format := transcode.Fixed(int(t.RecordLength))
		if t.TrimPadding {
			format = format.TrimmingPad()
		}

		return format, nil
	case recordsRDW:
		return transcode.RDW(), nil
	case recordsArticles:
		return transcode.Lengths(lengthsOf(t.lengths)), nil
	default:
		return transcode.RecordFormat{}, recordsError("records", "must be none, lines, fixed, rdw or articles")
	}
}

// outputFormat is how the result is laid out.
func (t *recordArgs) outputFormat(target *transcode.Entry) (transcode.RecordFormat, error) {
	records := cmp.Or(t.Records, recordsNone)
	output := cmp.Or(t.OutputRecords, recordsSame)

	if err := t.checkOutput(records, output); err != nil {
		return transcode.RecordFormat{}, err
	}

	switch output {
	case recordsSame:
		return t.reporting(t.sameFormat(records)), nil
	case recordsNone:
		return t.reporting(transcode.Stream()), nil
	case recordsLines:
		end, err := lineEnd(target, cmp.Or(t.OutputSeparator, separatorLF))
		if err != nil {
			return transcode.RecordFormat{}, err
		}

		return t.reporting(transcode.Separated(end)), nil
	case recordsFixed:
		if t.OutputLength < 1 || t.OutputLength > maxRecordLength {
			return transcode.RecordFormat{}, recordsError("outputLength", "must be between 1 and 32760")
		}

		return t.reporting(t.truncating(transcode.Fixed(int(t.OutputLength)))), nil
	default: // recordsRDW, checkOutput refuses the other values
		return t.reporting(t.truncating(transcode.RDW())), nil
	}
}

// checkOutput refuses the output arguments that do not apply.
func (t *recordArgs) checkOutput(records, output string) error {
	switch {
	case output != recordsSame && output != recordsNone && output != recordsLines &&
		output != recordsFixed && output != recordsRDW:
		return recordsError("outputRecords", "must be same, none, lines, fixed or rdw")
	case records == recordsNone && output != recordsSame && output != recordsNone:
		return recordsError("outputRecords", "needs records in the source file")
	case t.OutputSeparator != "" && output != recordsLines:
		return recordsError("outputSeparator", "only applies to a result in lines")
	case t.OutputLength != 0 && output != recordsFixed:
		return recordsError("outputLength", "only applies to a result in fixed-length records")
	case bool(t.Truncate) && !truncatable(records, output):
		return recordsError("truncate", "only applies to a result in fixed-length or RDW records")
	}

	return nil
}

// truncatable says whether the records of the result have a bounded length.
func truncatable(records, output string) bool {
	if output == recordsSame {
		output = records
	}

	return output == recordsFixed || output == recordsRDW
}

// sameFormat lays the result out as the source: each line with its own line
// end, the records of a fixed length with the same length, RDW records as
// RDW records, articles end to end.
func (t *recordArgs) sameFormat(records string) transcode.RecordFormat {
	switch records {
	case recordsLines:
		return transcode.SameSeparators()
	case recordsFixed:
		return t.truncating(transcode.Fixed(int(t.RecordLength)))
	case recordsRDW:
		return t.truncating(transcode.RDW())
	default: // none, articles
		return transcode.Stream()
	}
}

// reporting makes the result collect the lengths of its records, when the
// task keeps them.
func (t *recordArgs) reporting(format transcode.RecordFormat) transcode.RecordFormat {
	if t.written == nil {
		return format
	}

	return format.Reporting(func(length int) {
		if length > math.MaxUint16 {
			t.tooLong = true

			return
		}

		t.written.Add(uint16(length))
	})
}

// lengthsOf is the sequence of the lengths of articles, as the library reads
// them; none when there are none.
func lengthsOf(lengths *articles.Lengths) iter.Seq[int] {
	return func(yield func(int) bool) {
		for length := range lengths.All() {
			if !yield(int(length)) {
				return
			}
		}
	}
}

// transferLengths are the lengths of the articles that the transfer
// information gives.
func transferLengths(info map[string]any) (*articles.Lengths, error) {
	lengths, err := utils.GetAs[*articles.Lengths](info, articles.LengthsKey)

	switch {
	case errors.Is(err, utils.ErrKeyNotFound), err == nil && lengths == nil:
		return nil, recordsError("records", "is articles, but the transfer info gives no article lengths")
	case err != nil:
		return nil, fmt.Errorf("%w: %w", ErrTranscodeRecords, err)
	}

	return lengths, nil
}

// describe writes in the transfer information the article keys that describe
// the result: the line end of lines, at which a PeSIT send cuts them;
// the lengths of records written end to end or at a fixed length, with their
// format. It removes the keys that no longer apply, and answers a problem
// met, if any. Lines laid out as the source keep their keys, but for article
// lengths counted before a conversion that may change them (resized).
func (t *recordArgs) describe(info map[string]any, resized bool) string {
	records := cmp.Or(t.Records, recordsNone)
	output := cmp.Or(t.OutputRecords, recordsSame)

	if output == recordsSame {
		if records == recordsLines {
			if _, has := info[articles.LengthsKey]; has && resized {
				delete(info, articles.LengthsKey)

				return "the conversion may change the lengths of the articles: those the transfer info gave are left out"
			}

			return ""
		}

		output = records
	}

	switch output {
	case recordsLines:
		delete(info, articles.LengthsKey)

		switch cmp.Or(t.OutputSeparator, separatorLF) {
		case separatorNEL: // the PeSIT module cuts at LF or CR LF only
			delete(info, articles.SeparatorKey)
			delete(info, articles.FormatKey)
		case separatorCRLF:
			info[articles.SeparatorKey], info[articles.FormatKey] = articles.SeparatorCRLF, articles.FormatVariable
		default:
			info[articles.SeparatorKey], info[articles.FormatKey] = articles.SeparatorLF, articles.FormatVariable
		}
	case recordsRDW: // a record descriptor word means nothing to PeSIT
		delete(info, articles.LengthsKey)
		delete(info, articles.SeparatorKey)
		delete(info, articles.FormatKey)
	default: // none, fixed, articles: the lengths written
		delete(info, articles.SeparatorKey)

		if t.tooLong {
			delete(info, articles.LengthsKey)

			return "a record is longer than an article can be (65,535 bytes): the article lengths are left out"
		}

		info[articles.LengthsKey] = t.written

		switch output {
		case recordsFixed:
			info[articles.FormatKey] = articles.FormatFixed
		case recordsNone:
			info[articles.FormatKey] = articles.FormatVariable
		case recordsArticles: // the format given stays, while the articles still have one length
			if info[articles.FormatKey] == articles.FormatFixed && !t.written.Uniform() {
				info[articles.FormatKey] = articles.FormatVariable
			}
		}
	}

	return ""
}

func (t *recordArgs) truncating(format transcode.RecordFormat) transcode.RecordFormat {
	if t.Truncate {
		return format.Truncating()
	}

	return format
}

// lineEnds are the line ends recognized in a source, as characters of its
// character set (the converter writes them in it), the longest first.
func lineEnds(source *transcode.Entry) ([]string, error) {
	if !source.EBCDIC {
		return []string{transcode.CRLF, transcode.LF}, nil
	}

	return ebcdicChars(source.Name, "\x0d\x0a", "\x0a", "\x0d"+transcode.EBCDICLF, transcode.EBCDICLF, transcode.NL)
}

// lineEnd is the line end written in the result.
func lineEnd(target *transcode.Entry, separator string) (string, error) {
	var bytes string

	switch separator {
	case separatorLF:
		bytes = "\x0a"
	case separatorCRLF:
		bytes = "\x0d\x0a"
	case separatorNEL:
		if !target.EBCDIC {
			return "", recordsError("outputSeparator", "NEL ends the lines of EBCDIC data only")
		}

		bytes = transcode.NL
	default:
		return "", recordsError("outputSeparator", "must be LF, CRLF or NEL")
	}

	if !target.EBCDIC {
		return bytes, nil // on the side of open systems, the bytes are the characters LF and CR
	}

	ends, err := ebcdicChars(target.Name, bytes)
	if err != nil {
		return "", err
	}

	return ends[0], nil
}

// ebcdicChars are the characters that byte sequences stand for in an EBCDIC
// page: the separators of a converter are characters, which it writes in the
// page, so that a byte of the file is given by the character it decodes to.
func ebcdicChars(name string, sequences ...string) ([]string, error) {
	page, err := transcodeRegistry().CodePage(name)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTranscodeRecords, err)
	}

	out := make([]string, 0, len(sequences))

	for _, sequence := range sequences {
		chars := make([]rune, 0, len(sequence))

		for i := range len(sequence) {
			char, ok := page.Char(sequence[i])
			if !ok {
				return nil, fmt.Errorf("%w: the byte 0x%02X of a line end stands for no character in %s",
					ErrTranscodeRecords, sequence[i], name)
			}

			chars = append(chars, char)
		}

		out = append(out, string(chars))
	}

	return out, nil
}

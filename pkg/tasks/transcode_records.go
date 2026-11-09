package tasks

import (
	"cmp"
	"fmt"

	"code.waarp.fr/lib/transcode"
)

// The records of the TRANSCODE&REFORMAT task. "records" tells how the source
// file is cut: as a whole (none, the default), in lines, in records of a
// fixed length or in variable-length records preceded by their length (RDW,
// as z/OS writes them). "outputRecords" tells how the result is laid out: as
// the source (same, the default: a line keeps its own line end, byte for
// byte), without anything between records, in lines, in fixed-length records
// or in RDW records. Only the content of the records is converted.
//
// A line end is a byte sequence of the file. On the side of open systems, it
// is LF or CR LF. On an EBCDIC side, LF and CR LF are also the bytes 0x0A and
// 0x0D 0x0A (Driss, 2026-11-05): the PeSIT module of the gateway cuts a text
// file into articles there, whatever its character set; a source also has
// its lines recognized at the line feed (0x25) and the new line (0x15) of
// EBCDIC, control bytes that text never holds.

// The values of the record arguments.
const (
	recordsNone  = "none"
	recordsSame  = "same"
	recordsLines = "lines"
	recordsFixed = "fixed"
	recordsRDW   = "rdw"

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
	default:
		return transcode.RecordFormat{}, recordsError("records", "must be none, lines, fixed or rdw")
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
		return t.sameFormat(records), nil
	case recordsNone:
		return transcode.Stream(), nil
	case recordsLines:
		end, err := lineEnd(target, cmp.Or(t.OutputSeparator, separatorLF))
		if err != nil {
			return transcode.RecordFormat{}, err
		}

		return transcode.Separated(end), nil
	case recordsFixed:
		if t.OutputLength < 1 || t.OutputLength > maxRecordLength {
			return transcode.RecordFormat{}, recordsError("outputLength", "must be between 1 and 32760")
		}

		return t.truncating(transcode.Fixed(int(t.OutputLength))), nil
	default: // recordsRDW, checkOutput refuses the other values
		return t.truncating(transcode.RDW()), nil
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
// RDW records.
func (t *recordArgs) sameFormat(records string) transcode.RecordFormat {
	switch records {
	case recordsLines:
		return transcode.SameSeparators()
	case recordsFixed:
		return t.truncating(transcode.Fixed(int(t.RecordLength)))
	case recordsRDW:
		return t.truncating(transcode.RDW())
	default:
		return transcode.Stream()
	}
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

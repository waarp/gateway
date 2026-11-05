package tasks

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"code.waarp.fr/lib/transcode"
	"golang.org/x/text/transform"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// transcodeRegistry holds the character sets of the TRANSCODE task, those of
// code.waarp.fr/lib/transcode. It is built once, on first use.
//
//nolint:gochecknoglobals //the registry is built once and only read
var transcodeRegistry = sync.OnceValue(transcode.NewRegistry)

// TranscodeRegistry returns the registry of the character sets that the
// TRANSCODE task accepts. Its list gives the Unicode encodings, the
// single-byte code pages (EBCDIC ones by number), then the multi-byte
// encodings; a deprecated one is still accepted, but is not to be offered for
// a new task.
func TranscodeRegistry() *transcode.Registry {
	return transcodeRegistry()
}

var (
	ErrTranscodeNoSrcEncoding      = errors.New("missing source encoding")
	ErrTranscodeNoDstEncoding      = errors.New("missing destination encoding")
	ErrTranscodeInvalidEncoding    = errors.New("invalid encoding")
	ErrTranscodeIdenticalEncodings = errors.New("source and destination encodings are identical")
	ErrTranscodeRecords            = errors.New("invalid records")
)

// transcodeTask converts the transfer file from a character set to another,
// as a whole, or record by record when its arguments describe the records of
// the file and of the result (transcode_records.go). Without them, it behaves
// as before records were added.
type transcodeTask struct {
	FromCharset     string   `json:"fromCharset"`
	ToCharset       string   `json:"toCharset"`
	Records         string   `json:"records"`
	RecordLength    jsonInt  `json:"recordLength"`
	TrimPadding     jsonBool `json:"trimPadding"`
	OutputRecords   string   `json:"outputRecords"`
	OutputSeparator string   `json:"outputSeparator"`
	OutputLength    jsonInt  `json:"outputLength"`
	Truncate        jsonBool `json:"truncate"`
	Substitute      jsonBool `json:"substitute"`

	conversion transcode.Conversion
}

// transcodeCharset returns the character set called name, which may be one of
// its aliases ("utf8", "IBM1147", "CP037").
func transcodeCharset(name string) (transcode.Entry, error) {
	entry, ok := transcodeRegistry().Lookup(name)
	if !ok || entry.Kind == transcode.KindTable {
		return transcode.Entry{}, fmt.Errorf("%w %q", ErrTranscodeInvalidEncoding, name)
	}

	return entry, nil
}

func (t *transcodeTask) parseParams(params map[string]string) error {
	if utils.JSONConvert(params, t) != nil {
		return fmt.Errorf("failed to parse transcode task params: %w", ErrBadTaskArguments)
	}

	if t.FromCharset == "" {
		return ErrTranscodeNoSrcEncoding
	}

	if t.ToCharset == "" {
		return ErrTranscodeNoDstEncoding
	}

	from, err := transcodeCharset(t.FromCharset)
	if err != nil {
		return fmt.Errorf("source encoding: %w", err)
	}

	to, err := transcodeCharset(t.ToCharset)
	if err != nil {
		return fmt.Errorf("destination encoding: %w", err)
	}

	if from.Name == to.Name {
		return ErrTranscodeIdenticalEncodings
	}

	conversion := transcode.Conversion{Source: from.Name, Target: to.Name, Substitute: bool(t.Substitute)}

	if conversion.Input, err = t.inputFormat(&from); err != nil {
		return err
	}

	if conversion.Output, err = t.outputFormat(&to); err != nil {
		return err
	}

	// The library checks the rest: a separator or a padding that a character
	// set cannot write, a record that a multi-byte encoding cannot cut.
	if _, err = transcodeRegistry().NewConverter(&conversion); err != nil {
		return fmt.Errorf("%w: %w", ErrTranscodeRecords, err)
	}

	t.conversion = conversion

	return nil
}

func (t *transcodeTask) Validate(params map[string]string) error {
	return t.parseParams(params)
}

func (t *transcodeTask) Run(_ context.Context, params map[string]string, _ *database.DB,
	logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	if err := t.parseParams(params); err != nil {
		return err
	}

	tempFilename := transCtx.Transfer.LocalPath + ".tmp"
	if err := t.transcode(logger, transCtx.Transfer.LocalPath, tempFilename); err != nil {
		return err
	}

	if err := fs.Remove(transCtx.Transfer.LocalPath); err != nil {
		logger.Errorf("Failed to delete source file: %v", err)

		return fmt.Errorf("failed to delete source file: %w", err)
	}

	if err := fs.MoveFile(tempFilename, transCtx.Transfer.LocalPath); err != nil {
		logger.Errorf("Failed to rename temporary file: %v", err)

		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

func (t *transcodeTask) transcode(logger *log.Logger, srcFilepath, dstFilepath string) error {
	converter, convErr := transcodeRegistry().NewConverter(&t.conversion)
	if convErr != nil {
		return fmt.Errorf("failed to prepare the conversion: %w", convErr)
	}

	srcFile, opErr := fs.Open(srcFilepath)
	if opErr != nil {
		logger.Errorf("Failed to open source file: %v", opErr)

		return fmt.Errorf("failed to open source file: %w", opErr)
	}
	defer srcFile.Close() //nolint:errcheck //close never returns an error on read-only files

	dstFile, opErr := fs.Create(dstFilepath)
	if opErr != nil {
		logger.Errorf("Failed to create destination file: %v", opErr)

		return fmt.Errorf("failed to create destination file: %w", opErr)
	}
	defer dstFile.Close() //nolint:errcheck //error is checked bellow, this is just in case of error

	// Without records, the converter is the decoder of the source then the
	// encoder of the target, as with golang.org/x/text.
	if _, err := io.Copy(dstFile, transform.NewReader(srcFile, converter)); err != nil {
		return fmt.Errorf("failed to transcode file: %w", err)
	}

	if err := dstFile.Close(); err != nil {
		logger.Errorf("Failed to close destination file: %v", err)

		return fmt.Errorf("failed to close destination file: %w", err)
	}

	return nil
}

package tasks

import (
	"context"
	"fmt"

	"code.waarp.fr/lib/transcode"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// transcodeReformatTask is TRANSCODE&REFORMAT: it converts the transfer file
// from a character set to another and lays its records out again, in a single
// reading of the file (transcode_records.go). TRANSCODE is this task without
// records (Paolo, 2026-11-06).
type transcodeReformatTask struct {
	recordArgs

	FromCharset string   `json:"fromCharset"`
	ToCharset   string   `json:"toCharset"`
	Substitute  jsonBool `json:"substitute"`

	conversion transcode.Conversion
}

func (t *transcodeReformatTask) parseParams(params map[string]string) error {
	*t = transcodeReformatTask{}
	if utils.JSONConvert(params, t) != nil {
		return fmt.Errorf("failed to parse transcode task params: %w", ErrBadTaskArguments)
	}

	return t.prepare()
}

// prepare checks the two character sets, which must differ, and builds the
// conversion.
func (t *transcodeReformatTask) prepare() error {
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

	return t.build(&from, &to)
}

// build makes the conversion between two character sets, with the records of
// the task.
func (t *transcodeReformatTask) build(from, to *transcode.Entry) error {
	conversion := transcode.Conversion{Source: from.Name, Target: to.Name, Substitute: bool(t.Substitute)}

	var err error
	if conversion.Input, err = t.inputFormat(from); err != nil {
		return err
	}

	if conversion.Output, err = t.outputFormat(to); err != nil {
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

func (t *transcodeReformatTask) Validate(params map[string]string) error {
	return t.parseParams(params)
}

func (t *transcodeReformatTask) Run(_ context.Context, params map[string]string, _ *database.DB,
	logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	if err := t.parseParams(params); err != nil {
		return err
	}

	return convertFile(logger, transCtx, &t.conversion)
}

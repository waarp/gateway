package tasks

import (
	"cmp"
	"context"
	"fmt"

	"code.waarp.fr/lib/transcode"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/articles"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// transcodeReformatTask is TRANSCODE&REFORMAT: it converts the transfer file
// from a character set to another and lays its records out again, in a single
// reading of the file (transcode_records.go). TRANSCODE is this task without
// records.
type transcodeReformatTask struct {
	recordArgs

	FromCharset string   `json:"fromCharset"`
	ToCharset   string   `json:"toCharset"`
	Substitute  jsonBool `json:"substitute"`

	conversion transcode.Conversion
	from, to   transcode.Entry

	// leavesArticles keeps the article keys of the transfer information as
	// they are after a conversion of the whole file: TRANSCODE, which is older
	// than these keys, only warns.
	leavesArticles bool
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
	t.from, t.to = *from, *to
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

	return t.convert(logger, transCtx)
}

// convert converts the transfer file, cut at the lengths of the articles
// that the transfer information gives when the task says so, then keeps the
// article keys there true.
func (t *transcodeReformatTask) convert(logger *log.Logger, transCtx *model.TransferContext) error {
	if cmp.Or(t.Records, recordsNone) != recordsNone {
		if t.Records == recordsArticles {
			lengths, err := transferLengths(transCtx.Transfer.TransferInfo)
			if err != nil {
				return err
			}

			t.lengths = lengths
		}

		t.written, t.tooLong = &articles.Lengths{}, false

		from, to := t.from, t.to
		if err := t.build(&from, &to); err != nil {
			return err
		}
	}

	if err := convertFile(logger, transCtx, &t.conversion); err != nil {
		return err
	}

	t.keepArticles(logger, transCtx)

	return nil
}

// keepArticles keeps the article keys of the transfer information true: the
// records of the task rewrite them; a conversion of the whole file leaves
// them, but for the lengths of the articles when it may change them, which
// it leaves out with a warning (TRANSCODE only warns).
func (t *transcodeReformatTask) keepArticles(logger *log.Logger, transCtx *model.TransferContext) {
	info := transCtx.Transfer.TransferInfo

	if cmp.Or(t.Records, recordsNone) == recordsNone {
		if _, has := info[articles.LengthsKey]; has && t.resizes() {
			done := "they are left as they are"
			if !t.leavesArticles {
				delete(info, articles.LengthsKey)

				done = "they are left out"
			}

			logger.Warningf("The transfer info gives the lengths of the articles of the file, which a conversion "+
				"of the whole file from %s to %s may change: %s; TRANSCODE&REFORMAT with records %q keeps them true",
				t.from.Name, t.to.Name, done, recordsArticles)
		}

		return
	}

	if info == nil {
		info = map[string]any{}
		transCtx.Transfer.TransferInfo = info
	}

	if problem := t.describe(info, t.resizes()); problem != "" {
		logger.Warningf("Article keys of the transfer info: %s", problem)
	}
}

// resizes tells whether the conversion may change the number of bytes of a
// character: lengths counted in bytes before it may be wrong after it.
func (t *transcodeReformatTask) resizes() bool {
	return t.from.Kind != transcode.KindSingleByte || t.to.Kind != transcode.KindSingleByte
}

package tasks

import (
	"context"
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var ErrReformatNoEncoding = errors.New("missing encoding")

// reformatTask is REFORMAT: it lays the records of the transfer file out
// again, without converting them. Their content is copied byte for byte, never
// decoded nor checked; the character set of the file only gives the bytes of
// the line ends and of the padding (0x25 and 0x40 in EBCDIC). It is
// TRANSCODE&REFORMAT with the same character set on both sides
// (transcode_reformat.go).
type reformatTask struct {
	recordArgs

	Charset string `json:"charset"`

	task transcodeReformatTask
}

func (t *reformatTask) parseParams(params map[string]string) error {
	*t = reformatTask{}
	if utils.JSONConvert(params, t) != nil {
		return fmt.Errorf("failed to parse reformat task params: %w", ErrBadTaskArguments)
	}

	if t.Charset == "" {
		return ErrReformatNoEncoding
	}

	charset, err := transcodeCharset(t.Charset)
	if err != nil {
		return fmt.Errorf("encoding: %w", err)
	}

	// The records of the file and of the result are both required: the task
	// has nothing else to do.
	switch t.Records {
	case recordsLines, recordsFixed, recordsRDW:
	default:
		return recordsError("records", "must be lines, fixed or rdw")
	}

	switch t.OutputRecords {
	case recordsNone, recordsLines, recordsFixed, recordsRDW:
	default:
		return recordsError("outputRecords", "must be none, lines, fixed or rdw")
	}

	t.task = transcodeReformatTask{recordArgs: t.recordArgs, FromCharset: charset.Name, ToCharset: charset.Name}

	return t.task.build(&charset, &charset)
}

func (t *reformatTask) Validate(params map[string]string) error {
	return t.parseParams(params)
}

func (t *reformatTask) Run(_ context.Context, params map[string]string, _ *database.DB,
	logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	if err := t.parseParams(params); err != nil {
		return err
	}

	return convertFile(logger, transCtx, &t.task.conversion)
}

package tasks

import (
	"context"
	"errors"
	"fmt"
	"path"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrExtractNoOutput  = errors.New("no output directory specified")
	ErrExtractOutputDir = errors.New("output path is not a directory")
)

type extractTask struct {
	Archive   string `json:"archive"`
	OutputDir string `json:"outputDir"`
}

func (e *extractTask) parseParams(params map[string]string) error {
	*e = extractTask{}
	if err := utils.JSONConvert(params, e); err != nil {
		return fmt.Errorf("failed to parse the extract task parameters: %w", err)
	}

	if e.OutputDir == "" {
		return ErrExtractNoOutput
	}

	return nil
}

func (e *extractTask) Validate(args map[string]string) error {
	return e.parseParams(args)
}

func (e *extractTask) Run(_ context.Context, params map[string]string, _ *database.DB,
	logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	if err := e.parseParams(params); err != nil {
		logger.Errorf("%v", err)

		return err
	}

	if e.Archive == "" {
		e.Archive = transCtx.Transfer.LocalPath
	}

	if err := e.extractArchive(); err != nil {
		logger.Errorf("Failed to extract archive: %v", err)

		return err
	}

	return nil
}

//nolint:dupl //simpler to keep archive & extract separate
func (e *extractTask) extractArchive() error {
	outputInfo, statErr := fs.Stat(e.OutputDir)

	switch {
	case errors.Is(statErr, fs.ErrNotExist):
		if err := fs.MkdirAll(e.OutputDir); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	case statErr != nil:
		return fmt.Errorf("failed to retrieve output dir info: %w", statErr)
	case !outputInfo.IsDir():
		return fmt.Errorf("%s: %w", e.OutputDir, ErrExtractOutputDir)
	default:
	}

	for ext, decompress := range ExtractExtensions.Iter() {
		if hasExtension(e.Archive, ext) {
			return decompress(e)
		}
	}

	return fmt.Errorf("%w %q", ErrArchiveUnknownType, path.Ext(e.Archive))
}

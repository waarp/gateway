package tasks

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrArchiveNoFiles     = errors.New("no archive files specified")
	ErrArchiveNoOutput    = errors.New("no archive output path specified")
	ErrArchiveUnknownType = errors.New("unknown archive file extension")
)

type archiveTask struct {
	Files            string `json:"files"`
	CompressionLevel string `json:"compressionLevel"`
	OutputPath       string `json:"outputPath"`

	files []string
	level int
}

func (a *archiveTask) parseParams(params map[string]string) error {
	*a = archiveTask{}
	if err := utils.JSONConvert(params, a); err != nil {
		return fmt.Errorf("failed to parse the archive task parameters: %w", err)
	}

	if a.Files == "" {
		return ErrArchiveNoFiles
	}

	if a.CompressionLevel == "" {
		a.level = defaultCompressionLevel
	} else {
		var err error
		if a.level, err = strconv.Atoi(a.CompressionLevel); err != nil {
			return fmt.Errorf("failed to parse the compression level: %w", err)
		}

		if !isValidCompressionLevel(a.level) {
			return fmt.Errorf("%w %d", ErrInvalidCompressionLevel, a.level)
		}
	}

	if a.OutputPath == "" {
		return ErrArchiveNoOutput
	}

	for _, pattern := range strings.Split(a.Files, ",") {
		files, err := fs.Glob(strings.TrimSpace(pattern))
		if err != nil {
			return fmt.Errorf("invalid input file path %q: %w", pattern, err)
		}

		a.files = append(a.files, files...)
	}

	return nil
}

func (a *archiveTask) Validate(params map[string]string) error {
	return a.parseParams(params)
}

func (a *archiveTask) Run(_ context.Context, params map[string]string,
	_ *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := a.parseParams(params); err != nil {
		logger.Errorf("%v", err)

		return err
	}

	if err := a.makeArchive(); err != nil {
		logger.Errorf("Failed to create archive: %v", err)

		return err
	}

	transCtx.Transfer.LocalPath = a.OutputPath

	return nil
}

//nolint:dupl //simpler to keep archive & extract separate
func (a *archiveTask) makeArchive() error {
	switch {
	case hasExtension(a.OutputPath, extensionZip):
		return a.makeZipArchive()
	case hasExtension(a.OutputPath, extensionTar):
		return a.makeTarArchive(noCompressor)
	case hasExtension(a.OutputPath, extensionTarGz):
		return a.makeTarArchive(gzipCompressor)
	case hasExtension(a.OutputPath, extensionTarBz2):
		return a.makeTarArchive(bzip2Compressor)
	case hasExtension(a.OutputPath, extensionTarXz):
		return a.makeTarArchive(xzCompressor)
	case hasExtension(a.OutputPath, extensionTarZlib):
		return a.makeTarArchive(zlibCompressor)
	case hasExtension(a.OutputPath, extensionTarZstd):
		return a.makeTarArchive(zstdCompressor)
	default:
		return fmt.Errorf("%w %q", ErrArchiveUnknownType, path.Ext(a.OutputPath))
	}
}

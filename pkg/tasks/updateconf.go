package tasks

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	updateconfFwFilename        = "fw.json"
	updateconfGetRemoteFilename = "get-file.list"
)

var ErrUpdateconfEmptyFile = errors.New("zip file is empty")

type updateconfTask struct {
	ZipFile string `json:"zipFile"`
}

func (u *updateconfTask) Validate(params map[string]string) error {
	*u = updateconfTask{}

	if err := utils.JSONConvert(params, u); err != nil {
		return fmt.Errorf("failed to parse updateconf task params: %w", err)
	}

	return nil
}

func (u *updateconfTask) Run(_ context.Context, params map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	if err := u.Validate(params); err != nil {
		return err
	}

	if u.ZipFile == "" {
		u.ZipFile = transCtx.Transfer.LocalPath
	}

	if err := u.run(db, logger); err != nil {
		return err
	}

	logger.Debugf("Successfully imported config file %q", u.ZipFile)

	return nil
}

func (u *updateconfTask) run(db *database.DB, logger *log.Logger) error {
	file, fErr := fs.Open(u.ZipFile)
	if fErr != nil {
		return fmt.Errorf("failed to open zip file %q: %w", u.ZipFile, fErr)
	}

	defer file.Close()

	arch, zErr := u.openZip(file)
	if zErr != nil {
		return zErr
	}

	// Import config
	if err := u.importConfig(db, logger, arch); err != nil {
		return err
	}

	// Move filewatcher file
	if err := u.copyFile(arch, updateconfFwFilename); err != nil {
		return err
	}

	// Move get-remote file
	return u.copyFile(arch, updateconfGetRemoteFilename)
}

func (u *updateconfTask) openZip(file fs.File) (*zip.Reader, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat zip file: %w", err)
	}

	arch, err := zip.NewReader(file, info.Size())
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate zip reader: %w", err)
	}

	if len(arch.File) == 0 {
		return nil, ErrUpdateconfEmptyFile
	}

	return arch, nil
}

func (u *updateconfTask) importConfig(db *database.DB, logger *log.Logger, arch *zip.Reader) error {
	filename := conf.GlobalConfig.GatewayName + ".json"

	rc, err := arch.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open config file %q: %w", filename, err)
	}

	defer rc.Close()

	file := &updateconfFile{rc, filename}

	if err = backup.Import(db, logger, file, []string{"all"}, false, false); err != nil {
		return fmt.Errorf("failed to import data: %w", err)
	}

	return nil
}

func (*updateconfTask) copyFile(arch *zip.Reader, name string) error {
	file, err := arch.Open(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to open source file %q: %w", name, err)
	}

	confDir := os.Getenv(conf.ConfigDirEnvVar)
	filepath := fs.JoinPath(confDir, name)

	if err = fs.WriteFileFromReader(filepath, file); err != nil {
		return fmt.Errorf("failed to copy file %q: %w", name, err)
	}

	return nil
}

type updateconfFile struct {
	io.ReadCloser

	filename string
}

func (u *updateconfFile) Name() string { return u.filename }

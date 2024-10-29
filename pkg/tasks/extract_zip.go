package tasks

import (
	"archive/zip"
	"fmt"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

func (e *extractTask) extractZip() error {
	archInfo, statErr := fs.Stat(e.Archive)
	if statErr != nil {
		return fmt.Errorf("failed to retrieve archive info: %w", statErr)
	}

	archiveFile, opErr := fs.Open(e.Archive)
	if opErr != nil {
		return fmt.Errorf("failed to open archive file: %w", opErr)
	}
	defer archiveFile.Close() //nolint:errcheck //no errors on close for read files

	zipReader, zipErr := zip.NewReader(archiveFile, archInfo.Size())
	if zipErr != nil {
		return fmt.Errorf("failed to instantiate zip reader: %w", zipErr)
	}

	for _, file := range zipReader.File {
		if err := unzipFile(e.OutputDir, file); err != nil {
			return err
		}
	}

	return nil
}

func unzipFile(outputDir string, file *zip.File) error {
	outputPath := fs.JoinPath(outputDir, file.Name)

	if file.Mode().IsDir() {
		if err := fs.MkdirAll(outputPath); err != nil {
			return fmt.Errorf("%w", err)
		}

		return nil
	}

	parentDir := path.Dir(outputPath)

	if err := fs.MkdirAll(parentDir); err != nil {
		return fmt.Errorf("%w", err)
	}

	outFile, creatErr := fs.Create(outputPath)
	if creatErr != nil {
		return fmt.Errorf("%w", creatErr)
	}
	defer outFile.Close() //nolint:errcheck //error is checked elsewhere, this is just for safety

	zipFile, openErr := file.Open()
	if openErr != nil {
		return fmt.Errorf("failed to open zip file %q: %w", file.Name, openErr)
	}
	defer zipFile.Close() //nolint:errcheck //no errors on close for read files

	if _, err := io.CopyN(outFile, zipFile, int64(file.UncompressedSize64)); err != nil {
		return fmt.Errorf("failed to copy file %q: %w", file.Name, err)
	}

	if err := outFile.Close(); err != nil {
		return fmt.Errorf("failed to close output file %q: %w", outputPath, err)
	}

	return nil
}

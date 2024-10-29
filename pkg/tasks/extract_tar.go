package tasks

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

func (e *extractTask) extractTar(mkDecompressor func(fs.File) (io.Reader, error)) error {
	archiveFile, opErr := fs.Open(e.Archive)
	if opErr != nil {
		return fmt.Errorf("failed to open archive file: %w", opErr)
	}
	defer archiveFile.Close() //nolint:errcheck //no errors on close for read files

	decompressor, decErr := mkDecompressor(archiveFile)
	if decErr != nil {
		return fmt.Errorf("failed to instantiate decompressor: %w", decErr)
	}

	tarReader := tar.NewReader(decompressor)

	for {
		file, nextErr := tarReader.Next()
		if errors.Is(nextErr, io.EOF) {
			return nil
		} else if nextErr != nil {
			return fmt.Errorf("failed to read tar file: %w", nextErr)
		}

		if err := untarFile(e.OutputDir, file, tarReader); err != nil {
			return err
		}
	}
}

func untarFile(outputDir string, header *tar.Header, tarFile *tar.Reader) error {
	outputPath := fs.JoinPath(outputDir, header.Name)

	if header.Typeflag == tar.TypeDir {
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
	defer outFile.Close() //nolint:errcheck //error is checked bellow, this is just for safety

	if _, err := io.Copy(outFile, tarFile); err != nil {
		return fmt.Errorf("failed to copy file %q: %w", header.Name, err)
	}

	if err := outFile.Close(); err != nil {
		return fmt.Errorf("failed to close output file %q: %w", outputPath, err)
	}

	return nil
}

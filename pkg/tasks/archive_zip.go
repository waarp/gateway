package tasks

import (
	"archive/zip"
	"fmt"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

func (a *archiveTask) makeZipArchive() error {
	archiveFile, creatErr := fs.Create(a.OutputPath)
	if creatErr != nil {
		return fmt.Errorf("failed to create archive file: %w", creatErr)
	}

	defer archiveFile.Close() //nolint:errcheck //error is checked bellow, this is just for safety

	zipWriter := zip.NewWriter(archiveFile)
	defer zipWriter.Close() //nolint:errcheck //error is checked bellow, this is just for safety

	zipWriter.RegisterCompressor(zip.Deflate, getDeflateCompressor(a.level))

	for _, filepath := range a.files {
		if err := zipFile(zipWriter, filepath, ""); err != nil {
			return err
		}
	}

	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close zip writer: %w", err)
	}

	if err := archiveFile.Close(); err != nil {
		return fmt.Errorf("failed to close archive file: %w", err)
	}

	return nil
}

//nolint:dupl //simpler to keep ZIP and TAR separate
func zipFile(zipWriter *zip.Writer, filepath, baseInZip string) error {
	info, statErr := fs.Stat(filepath)
	if statErr != nil {
		return fmt.Errorf("failed to retrieve file info: %w", statErr)
	}

	header, headErr := zip.FileInfoHeader(info)
	if headErr != nil {
		return fmt.Errorf("failed to create file info header: %w", headErr)
	}

	header.Method = zip.Deflate
	header.Name = path.Join(baseInZip, header.Name)

	if info.IsDir() {
		header.Name += "/"
	}

	headerWriter, creatErr := zipWriter.CreateHeader(header)
	if creatErr != nil {
		return fmt.Errorf("failed to create archive header: %w", creatErr)
	}

	if info.IsDir() {
		entries, listErr := fs.List(filepath)
		if listErr != nil {
			return fmt.Errorf("failed to list directory: %w", listErr)
		}

		for _, entry := range entries {
			childPath := fs.JoinPath(filepath, entry.Name())
			if err := zipFile(zipWriter, childPath, header.Name); err != nil {
				return err
			}
		}
	} else {
		file, openErr := fs.Open(filepath)
		if openErr != nil {
			return fmt.Errorf("failed to open file: %w", openErr)
		}

		defer file.Close() //nolint:errcheck //no errors on close for read files

		if _, err := io.Copy(headerWriter, file); err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}
	}

	return nil
}

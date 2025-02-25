package tasks

import (
	"archive/tar"
	"fmt"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

type compressorFunc func(fs.File, int) (io.WriteCloser, error)

func (a *archiveTask) makeTarArchive(mkCompressor compressorFunc) error {
	archiveFile, creatErr := fs.Create(a.OutputPath)
	if creatErr != nil {
		return fmt.Errorf("failed to create archive file: %w", creatErr)
	}
	defer archiveFile.Close() //nolint:errcheck //error is checked bellow, this is just for safety

	compressor, compErr := mkCompressor(archiveFile, a.level)
	if compErr != nil {
		return fmt.Errorf("failed to instantiate compressor: %w", compErr)
	}
	defer compressor.Close() //nolint:errcheck //error is checked bellow, this is just for safety

	tarWriter := tar.NewWriter(compressor)
	defer tarWriter.Close() //nolint:errcheck //error is checked bellow, this is just for safety

	for _, filepath := range a.files {
		if err := tarFile(tarWriter, filepath, ""); err != nil {
			return err
		}
	}

	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	if err := compressor.Close(); err != nil {
		return fmt.Errorf("failed to close compressor: %w", err)
	}

	if err := archiveFile.Close(); err != nil {
		return fmt.Errorf("failed to close archive file: %w", err)
	}

	return nil
}

//nolint:dupl //simpler to keep ZIP and TAR separate
func tarFile(tarWriter *tar.Writer, filepath, baseInTar string) error {
	info, statErr := fs.Stat(filepath)
	if statErr != nil {
		return fmt.Errorf("failed to retrieve file info: %w", statErr)
	}

	header, headErr := tar.FileInfoHeader(info, "")
	if headErr != nil {
		return fmt.Errorf("failed to create file info header: %w", headErr)
	}

	//nolint:gosec //false positive -> we are not extracting, we are archiving
	header.Name = path.Join(baseInTar, header.Name)

	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write archive header: %w", err)
	}

	if info.IsDir() {
		entries, listErr := fs.List(filepath)
		if listErr != nil {
			return fmt.Errorf("failed to list directory: %w", listErr)
		}

		for _, entry := range entries {
			childPath := fs.JoinPath(filepath, entry.Name())
			if err := tarFile(tarWriter, childPath, header.Name); err != nil {
				return err
			}
		}
	} else {
		file, openErr := fs.Open(filepath)
		if openErr != nil {
			return fmt.Errorf("failed to open file: %w", openErr)
		}

		defer file.Close() //nolint:errcheck //no errors on close for read files

		if _, err := io.Copy(tarWriter, file); err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}
	}

	return nil
}

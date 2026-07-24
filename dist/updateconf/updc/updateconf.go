package updc

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

var (
	errNoArguments  = errors.New("updateconf needs at least 1 parameter")
	errFileNotFound = errors.New("file not found in archive")
	errNoConfDir    = errors.New("configuration directory not found or invalid")
)

//nolint:gochecknoglobals //global vars needed here for Waarp Transfer
var (
	ExeName = "waarp-gatewayd"
	DirName = "waarp-gateway"
	AppName = "Gateway"
)

func Do(args []string) error {
	if len(args) == 0 {
		return errNoArguments
	}

	fmt.Println("Start of updateconf")

	archFile := args[0]
	instance := getConfFilename(archFile)

	archReader, opErr := zip.OpenReader(archFile)
	if opErr != nil {
		return fmt.Errorf("failed to open archive: %w", opErr)
	}

	defer archReader.Close() //nolint:errcheck //ignore error

	// Import
	if err := importConf(&archReader.Reader, instance); err != nil {
		return fmt.Errorf("failed to import configuration: %w", err)
	}

	// Additional files
	// Filewatcher
	if err := moveToConf(&archReader.Reader, "fw.json"); err != nil {
		return fmt.Errorf("failed to write filewatcher configuration file: %w", err)
	}

	// Get-Remote
	if err := moveToConf(&archReader.Reader, "get-file.list"); err != nil {
		return fmt.Errorf("failed to write get-remote configuration file: %w", err)
	}

	return nil
}

func getConfFilename(archfile string) string {
	archName := filepath.Base(archfile)
	separator := "-"
	part := strings.Split(archName, separator)

	// Remove last part of file (from last '-')
	if len(part) < 2 { //nolint:mnd //this would be a constant used only once
		return ""
	}

	part = part[:len(part)-1]
	builder := strings.Builder{}

	for i, s := range part {
		builder.WriteString(s)

		if i < len(part)-1 {
			builder.WriteString(separator)
		} else {
			builder.WriteString(".json")
		}
	}

	return builder.String()
}

func importConf(arch *zip.Reader, file string) error {
	rc, err := getFileFromArch(arch, file)
	if err != nil {
		return err
	}

	defer func() { _ = rc.Close() }() //nolint:errcheck //no need to check error

	return execImport(rc)
}

func getFileFromArch(arch *zip.Reader, file string) (io.ReadCloser, error) {
	for _, f := range arch.File {
		if f.Name == file {
			confFile, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("cannot read zip file: %w", err)
			}

			return confFile, nil
		}
	}

	return nil, fmt.Errorf("%w: %q", errFileNotFound, file)
}

func execImport(confReader io.Reader) error {
	envConfFile := os.Getenv("WAARP_CONFIG_FILE")

	params := []string{"import", "-v"}
	if envConfFile != "" {
		params = append(params, "-c", envConfFile)
	}

	cmd := exec.CommandContext(context.Background(), ExeName, params...)
	cmd.Stdin = confReader

	out, err := cmd.CombinedOutput()
	if err != nil {
		if out != nil {
			//nolint:err113 //dynamical error is better here for readability
			return fmt.Errorf("subprocess returned an error: %s", string(out))
		}

		return fmt.Errorf("subprocess returned an error: %w", err)
	}

	fmt.Print(string(out))

	return nil
}

func moveToConf(arch *zip.Reader, file string) error {
	envConfDir := os.Getenv("WAARP_CONFIG_DIR")
	confDir, dirErr := getConfDir(envConfDir, "etc", path.Join("/etc", DirName))
	if dirErr != nil {
		return dirErr
	}

	src, err := getFileFromArch(arch, file)
	if errors.Is(err, errFileNotFound) {
		return nil
	} else if err != nil {
		return err
	}

	dstPath := filepath.Clean(filepath.Join(confDir, file))
	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("cannot open file %q: %w", dstPath, err)
	}

	_, err = io.Copy(dst, src)
	if err != nil {
		return fmt.Errorf("cannot write to file %q: %w", dstPath, err)
	}

	return nil
}

func getConfDir(dirs ...string) (string, error) {
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("%s exists but is not a directory: %w", dir, errNoConfDir)
			}

			return dir, nil
		}
	}

	return "", fmt.Errorf("no %s directory found: %w", AppName, errNoConfDir)
}

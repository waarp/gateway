package updc

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	minArgs       = 2
	exitErrorCode = 2
)

var (
	errFileNotFound = errors.New("file not found in archive")
	errNoConfDir    = errors.New("configuration directory not found or invalid")
)

//nolint:gochecknoglobals //global vars needed here for Waarp Transfer
var (
	ExeName = "waarp-gatewayd"
	DirName = "waarp-gateway"
	AppName = "Gateway"
)

//nolint:forbidigo //prints are needed here
func Do() {
	if len(os.Args) < minArgs {
		fmt.Println("updateconf needs at least 1 parameter")
		os.Exit(exitErrorCode)
	}

	fmt.Println("Start of updateconf")

	archFile := os.Args[1]
	instance := getConfFilename(archFile)

	archReader, opErr := zip.OpenReader(archFile)
	if opErr != nil {
		fmt.Println("Cannot open archive:", opErr)
		os.Exit(exitErrorCode)
	}

	// Import
	if err := importConf(&archReader.Reader, instance); err != nil {
		fmt.Println("Cannot import configuration:", err)

		_ = archReader.Close() //nolint:errcheck //ignore error

		os.Exit(exitErrorCode)
	}

	// Additional files
	if err := moveToConf(&archReader.Reader, "get-file.list"); err != nil {
		fmt.Println("Cannot write configuration file:", err)

		_ = archReader.Close() //nolint:errcheck //ignore error
	}

	_ = archReader.Close() //nolint:errcheck //ignore error

	fmt.Println("End of process updateconf")
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

	err = execImport(rc)
	if err != nil {
		return err
	}

	return nil
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
			fmt.Print(string(out))
		}

		return fmt.Errorf("cannot read subprocess output: %w", err)
	}

	fmt.Print(string(out))

	return nil
}

func moveToConf(arch *zip.Reader, file string) error {
	envConfDir := os.Getenv("WAARP_CONFIG_DIR")
	confDir, dirErr := getConfDir(envConfDir, "etc/", "/etc/"+DirName)
	if dirErr != nil {
		return dirErr
	}

	src, err := getFileFromArch(arch, file)
	if errors.Is(err, errFileNotFound) {
		return nil
	} else if err != nil {
		return err
	}

	dst, err := os.Create(filepath.Clean(confDir + file))
	if err != nil {
		return fmt.Errorf("cannot open file %q: %w", confDir+file, err)
	}

	_, err = io.Copy(dst, src)
	if err != nil {
		return fmt.Errorf("cannot write to file %q: %w", confDir+file, err)
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

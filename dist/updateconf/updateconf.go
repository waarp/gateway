package main

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
	"time"
)

const (
	minArgs       = 2
	exitErrorCode = 2
	execTimeout   = 2 * time.Second
)

var (
	errFileNotFound = errors.New("file not found")
	errNoConfDir    = errors.New("configuration directory not found or invalid")
)

//nolint:forbidigo // this is allowed in the main function
func main() {
	if len(os.Args) < minArgs {
		fmt.Printf("updateconf needs at least 1 parameter")
		os.Exit(exitErrorCode)
	}

	archFile := os.Args[1]
	instance := getConfFilename(archFile)

	archReader, err := zip.OpenReader(archFile)
	if err != nil {
		fmt.Printf("Cannot open archive: %s\n", err.Error())
		os.Exit(exitErrorCode)
	}

	// Import
	if err := importConf(&archReader.Reader, instance); err != nil {
		fmt.Printf("Cannot import configuration: %s\n", err.Error())

		_ = archReader.Close() //nolint: errcheck // ignore error

		os.Exit(exitErrorCode)
	}

	// Additional files
	if err := moveToConf(&archReader.Reader, "get-file.list"); err != nil {
		fmt.Printf("Cannot write configuration file: %s\n", err.Error())

		_ = archReader.Close() //nolint: errcheck // ignore error
	}

	_ = archReader.Close() //nolint: errcheck // ignore error
}

func getConfFilename(archfile string) string {
	archName := filepath.Base(archfile)
	separator := "-"
	part := strings.Split(archName, separator)

	// Remove last part of file (from last '-')
	if len(part) < 2 { //nolint: gomnd // this would be a constant used only once
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

	defer func() { _ = rc.Close() }() //nolint: errcheck // no need to check error

	err = execImport(rc)
	if err != nil {
		return err
	}

	return nil
}

func getFileFromArch(arch *zip.Reader, file string) (io.ReadCloser, error) {
	for _, f := range arch.File {
		if f.Name == file {
			conf, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("cannot read zip file: %w", err)
			}

			return conf, nil
		}
	}

	return nil, fmt.Errorf("file %s is not in the archive: %w", file, errFileNotFound)
}

func execImport(confReader io.Reader) error {
	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "waarp-gatewayd", "import", "-v")

	writer, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("cannot open subprocess stdin: %w", err)
	}

	go func() {
		defer writer.Close() //nolint:errcheck // nothing to handle the error

		_, err = io.Copy(writer, confReader)
		if err != nil {
			//nolint:forbidigo // with this design, this is sadly the only way to handle the error
			fmt.Printf("cannot import configuration: %s\n", err.Error())
		}
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cannot read subprocess output: %w", err)
	}

	fmt.Print(string(out)) //nolint:forbidigo // output must be written for the Gateway uses it

	return nil
}

func moveToConf(arch *zip.Reader, files ...string) error {
	confDir, err := getConfDir("etc/", "/etc/waarp-gateway/")
	if err != nil {
		return err
	}

	for _, f := range files {
		src, err := getFileFromArch(arch, f)
		if err != nil {
			return err
		}

		dst, err := os.Create(confDir + f)
		if err != nil {
			return fmt.Errorf("cannot open file %q: %w", confDir+f, err)
		}

		_, err = io.Copy(dst, src)
		if err != nil {
			return fmt.Errorf("cannot write to file %q: %w", confDir+f, err)
		}
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

	return "", fmt.Errorf("no gateway directory found: %w", errNoConfDir)
}

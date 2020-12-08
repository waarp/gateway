package main

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("updateconf needs at least 1 parameter")
		os.Exit(2)
	}
	archFile := os.Args[1]
	instance := getConfFilename(archFile)

	archReader, err := zip.OpenReader(archFile)
	if err != nil {
		fmt.Printf("Cannot open archive: %s\n", err.Error())
		os.Exit(2)
	}
	defer archReader.Close()

	// Import
	if err := importConf(archReader.Reader, instance); err != nil {
		fmt.Printf("Cannot import configuration: %s\n", err.Error())
		os.Exit(2)
	}

	// Additional files
	if err := moveToConf(archReader.Reader, ""); err != nil {
		fmt.Printf("Cannot write configuration file: %s\n", err.Error())
		os.Exit(2)
	}
	os.Exit(0)
}

func getConfFilename(archfile string) string {
	archName := filepath.Base(archfile)
	separator := "-"
	part := strings.Split(archName, separator)
	// Remove last part of file (from last '-')
	if len(part) < 2 {
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

func importConf(arch zip.Reader, file string) error {
	rc, err := getFileFromArch(arch, file)
	if err != nil {
		return err
	}
	defer rc.Close()

	err = execImport(rc)
	if err != nil {
		return err
	}
	return nil
}

func getFileFromArch(arch zip.Reader, file string) (io.ReadCloser, error) {
	for _, f := range arch.File {
		if f.Name == file {
			conf, err := f.Open()
			if err != nil {
				return nil, err
			}
			return conf, nil
		}
	}
	return nil, fmt.Errorf("file %s is not in the archive", file)
}

func execImport(confReader io.Reader) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "waarp-gatewayd", "import", "-v", "-s", "-")
	writer, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer writer.Close()
		_, err = io.Copy(writer, confReader)
		if err != nil {
			fmt.Printf("cannot import configuration: %s\n", err.Error())
		}
	}()

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func moveToConf(arch zip.Reader, files ...string) error {
	confDir, err := getConfDir("etc/", "/etc/waarp-gateway/")
	if err != nil {
		return err
	}

	for _, f := range files {
		src, err := getFileFromArch(arch, f)
		if err != nil {
			return err
		}
		// TODO
		dst, err := os.Create(confDir + f)
		if err != nil {
			return err
		}
		_, err = io.Copy(dst, src)
		if err != nil {
			return err
		}
	}
	return nil
}

func getConfDir(dirs ...string) (string, error) {
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("%s exists but is not a directory", dir)
			}
			return dir, nil
		}
	}
	return "", fmt.Errorf("no gateway directory found")
}

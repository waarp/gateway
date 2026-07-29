package wg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jessevdk/go-flags"
)

const (
	updateconfAPIPath = "/api/updateconf"

	updateconfTargetsHeader = "Targets"
	updateconfResetHeader   = "Reset"
	updateconfDryRunHeader  = "Dry-Run"
	updateconfRestartHeader = "Restart"

	contentTypeHeader = "Content-Type"
	acceptHeader      = "Accept"

	contentTypeJSON = "application/json"
	contentTypeYAML = "application/yaml"
)

// fileContentType returns the MIME type to use for the given file, based on
// its extension. Files with a ".yaml" or ".yml" extension are considered to
// be YAML, all others are considered to be JSON.
func fileContentType(filename flags.Filename) string {
	switch strings.ToLower(filepath.Ext(string(filename))) {
	case ".yaml", ".yml":
		return contentTypeYAML
	default:
		return contentTypeJSON
	}
}

//nolint:lll //struct tags can be long for flags
type UpdateconfImport struct {
	File    flags.Filename `required:"yes" short:"s" long:"source" description:"The configuration file to import, in either JSON or YAML format."`
	Target  []string       `short:"t" long:"target" default:"all" choice:"rules" choice:"servers" choice:"partners" choice:"clients" choice:"users" choice:"clouds" choice:"snmp" choice:"authorities" choice:"keys" choice:"email" choice:"filewatchers" choice:"all" description:"Limit the import to a subset of data. Can be repeated to import multiple subsets."`
	Dry     bool           `short:"d" long:"dry-run" description:"Do not make any changes, but simulate the import."`
	Reset   bool           `short:"r" long:"reset" description:"Empty the database before importing the elements from the file."`
	Restart bool           `long:"restart" description:"Restart the imported servers, clients and filewatchers once the import is complete."`
}

func (u *UpdateconfImport) Execute([]string) error { return u.execute(stdOutput) }

func (u *UpdateconfImport) execute(w io.Writer) error {
	file, openErr := os.Open(string(u.File))
	if openErr != nil {
		return fmt.Errorf("failed to open the configuration file: %w", openErr)
	}
	defer file.Close() //nolint:errcheck,gosec // nothing to do with the error

	addr.Path = updateconfAPIPath

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, addr.String(), file)
	if reqErr != nil {
		return fmt.Errorf("cannot prepare request: %w", reqErr)
	}

	user := addr.User.Username()
	passwd, _ := addr.User.Password()
	req.SetBasicAuth(user, passwd)

	req.Header.Set(contentTypeHeader, fileContentType(u.File))

	for _, target := range u.Target {
		req.Header.Add(updateconfTargetsHeader, target)
	}

	if u.Dry {
		req.Header.Set(updateconfDryRunHeader, "true")
	}

	if u.Reset {
		req.Header.Set(updateconfResetHeader, "true")
	}

	if u.Restart {
		req.Header.Set(updateconfRestartHeader, "true")
	}

	resp, doErr := getHTTPClient(insecure).Do(req)
	if doErr != nil {
		return fmt.Errorf("an error occurred while sending the HTTP request: %w", doErr)
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

	switch resp.StatusCode {
	case http.StatusCreated:
		if err := displayResponseMessage(w, resp); err != nil {
			return err
		}

		fmt.Fprintln(w, "The configuration was successfully imported.")

		return nil
	default:
		return fmt.Errorf("unexpected response (%s): %w", resp.Status, getResponseErrorMessage(resp))
	}
}

//nolint:lll //struct tags can be long for flags
type UpdateconfExport struct {
	File   flags.Filename `short:"f" long:"file" description:"The destination file. If none is given, the exported configuration will be written to the standard output."`
	Target []string       `short:"t" long:"target" default:"all" choice:"rules" choice:"servers" choice:"partners" choice:"clients" choice:"users" choice:"clouds" choice:"snmp" choice:"authorities" choice:"keys" choice:"email" choice:"filewatchers" choice:"all" description:"Limit the export to a subset of data. Can be repeated to export multiple subsets."`
}

func (u *UpdateconfExport) Execute([]string) error { return u.execute(stdOutput) }

func (u *UpdateconfExport) execute(w io.Writer) error {
	addr.Path = updateconfAPIPath

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, addr.String(), http.NoBody)
	if reqErr != nil {
		return fmt.Errorf("cannot prepare request: %w", reqErr)
	}

	user := addr.User.Username()
	passwd, _ := addr.User.Password()
	req.SetBasicAuth(user, passwd)

	for _, target := range u.Target {
		req.Header.Add(updateconfTargetsHeader, target)
	}

	acceptType := contentTypeJSON
	if u.File != "" {
		acceptType = fileContentType(u.File)
	}

	req.Header.Set(acceptHeader, acceptType)

	resp, doErr := getHTTPClient(insecure).Do(req)
	if doErr != nil {
		return fmt.Errorf("an error occurred while sending the HTTP request: %w", doErr)
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

	switch resp.StatusCode {
	case http.StatusOK:
		return u.writeExport(w, resp.Body)
	default:
		return fmt.Errorf("unexpected response (%s): %w", resp.Status, getResponseErrorMessage(resp))
	}
}

func (u *UpdateconfExport) writeExport(w io.Writer, body io.Reader) error {
	if u.File == "" {
		if _, err := io.Copy(w, body); err != nil {
			return fmt.Errorf("failed to write the exported configuration: %w", err)
		}

		return nil
	}

	dest, openErr := os.OpenFile(string(u.File), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if openErr != nil {
		return fmt.Errorf("failed to open the destination file: %w", openErr)
	}

	defer dest.Close() //nolint:errcheck,gosec // nothing to do with the error

	if _, err := io.Copy(dest, body); err != nil {
		return fmt.Errorf("failed to write the exported configuration: %w", err)
	}

	fmt.Fprintln(w, "The configuration was successfully exported.")

	return nil
}

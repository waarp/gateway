// Package config encapsulates 3rd party libraries to abstract config file
// management.
package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/jessevdk/go-flags"
)

// Parser is the central point to load config files in a struct and
// Write configuration files.
type Parser struct {
	parser *flags.Parser
}

// NewParser creates a new Parser associated to a configuration struct.
// `data` accepts tags for configuration
// (see https://godoc.org/github.com/jessevdk/go-flags#IniParse).
func NewParser(data interface{}) (*Parser, error) {
	var options flags.Options = flags.Default | flags.IgnoreUnknown
	parser := flags.NewNamedParser(path.Base(os.Args[0]), options)

	_, err := parser.AddGroup("global", "", data)
	if err != nil {
		return nil, fmt.Errorf("cannot add a global group to the parser: %w", err)
	}

	p := &Parser{
		parser: parser,
	}

	_, err = p.parser.ParseArgs([]string{})
	if err != nil {
		return nil, fmt.Errorf("cannot initialize parser: %w", err)
	}

	return p, nil
}

// Parse gets a Reader interface to an ini structure parsed to populate the
// config struct.
func (p Parser) Parse(r io.Reader) error {
	if err := flags.NewIniParser(p.parser).Parse(r); err != nil {
		return fmt.Errorf("cannot parse configuration: %w", err)
	}

	return nil
}

// ParseFile tries to read the configuration file filename and parses its
// content in the configuration object.
func (p Parser) ParseFile(filename string) error {
	content, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return fmt.Errorf("cannot parse configuration file: %w", err)
	}

	return p.Parse(bytes.NewReader(content))
}

// Write writes the configuration object in INI format to the writer w.
func (p Parser) Write(w io.Writer) {
	var options flags.IniOptions = flags.IniIncludeComments | flags.IniIncludeDefaults | flags.IniCommentDefaults

	flags.NewIniParser(p.parser).Write(w, options)
}

// WriteFile tries to write the configuration to the file filename.
func (p Parser) WriteFile(filename string) error {
	var buf bytes.Buffer

	p.Write(&buf)

	err := os.WriteFile(filename, buf.Bytes(), 0o600)
	if err != nil {
		return fmt.Errorf("cannot write configuration file: %w", err)
	}

	return nil
}

// UpdateFile updates the configuration file filename by adding new instructions
// and removing those that do not exist anymore.
func (p Parser) UpdateFile(filename string) error {
	if err := p.ParseFile(filename); err != nil {
		return fmt.Errorf("cannot update configuration file: %w", err)
	}

	return p.WriteFile(filename)
}

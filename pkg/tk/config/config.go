// Package config encapsulates 3rd party libraries to abstract config file
// management
package config

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"

	flags "github.com/jessevdk/go-flags"
)

// Parser is the central point to load config files in a struct and
// Write configuration files.
type Parser struct {
	parser *flags.Parser
}

// NewParser creates a new Parser associated to a configuration struct
// data accepts tags for configuration
// (see https://godoc.org/github.com/jessevdk/go-flags#IniParse)
func NewParser(data interface{}) *Parser {
	var options flags.Options = flags.Default | flags.IgnoreUnknown
	parser := flags.NewNamedParser(path.Base(os.Args[0]), options)
	// FIXME error should not be discarded
	_, _ = parser.AddGroup("global", "", data)

	p := &Parser{
		parser: parser,
	}
	// FIXME error should not be discarded
	_, _ = p.parser.ParseArgs([]string{})
	return p
}

// Parse gets a Reader interface to an ini structure parsed to populate the
// config struct
func (p Parser) Parse(r io.Reader) error {
	if err := flags.NewIniParser(p.parser).Parse(r); err != nil {
		return err
	}
	return nil
}

// ParseFile tries to read the configuration file filename and parses its
// its content in the configuration object
func (p Parser) ParseFile(filename string) error {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	return p.Parse(bytes.NewReader(content))
}

// Write writes the configuration object in INI format to the writer w
func (p Parser) Write(w io.Writer) {
	var options flags.IniOptions = flags.IniIncludeComments | flags.IniIncludeDefaults | flags.IniCommentDefaults
	flags.NewIniParser(p.parser).Write(w, options)
}

// WriteFile tries to write the configuration to the file filename
func (p Parser) WriteFile(filename string) error {
	var buf bytes.Buffer
	p.Write(&buf)
	content := buf.Bytes()

	return ioutil.WriteFile(filename, content, 0600)
}

// UpdateFile updates the configuration file filename by adding new instructions
// and removing those that do not exist anymore
func (p Parser) UpdateFile(filename string) error {
	if err := p.ParseFile(filename); err != nil {
		return err
	}
	return p.WriteFile(filename)
}

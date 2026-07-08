package conf

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/config"
)

// ConfigOverride is a struct defining a list of settings local to a gateway instance
// (or node) which can be used to override settings defined at the cluster level.
type ConfigOverride struct {
	overrideLock    sync.RWMutex
	filename        string
	ListenAddresses *addressOverride `group:"Address Indirection"`
}

// NewOverride returns a new correctly initialized, instance of ConfigOverride.
func NewOverride(filename string) *ConfigOverride {
	return &ConfigOverride{
		filename:        filename,
		ListenAddresses: &addressOverride{addressMap: map[string]string{}},
	}
}

func (o *ConfigOverride) parse() error {
	return o.ListenAddresses.Parse()
}

func (o *ConfigOverride) writeFile() error {
	file, fErr := os.OpenFile(o.filename, os.O_TRUNC|os.O_RDWR|os.O_CREATE, 0o600)
	if fErr != nil {
		return fmt.Errorf("failed to open config override file: %w", fErr)
	}

	if err := o.writeTo(file); err != nil {
		_ = file.Close() //nolint:errcheck //the write error takes precedence

		return fmt.Errorf("failed to write the config override file: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close the config override file: %w", err)
	}

	return nil
}

func (o *ConfigOverride) writeTo(w io.Writer) error {
	parser, err := config.NewParser(o)
	if err != nil {
		return fmt.Errorf("failed to initialize the config parser: %w", err)
	}

	parser.Write(w)

	return nil
}

const iniExtension = ".ini"

func CreateOverride(configFile, nodeID string) error {
	if nodeID == "" {
		return nil
	}

	overrideFile := filepath.Join(filepath.Dir(configFile), nodeID+iniExtension)
	o := NewOverride(overrideFile)

	p, pErr := config.NewParser(o)
	if pErr != nil {
		return fmt.Errorf("failed to initialize the config parser: %w", pErr)
	}

	if err := p.WriteFile(overrideFile); err != nil {
		return fmt.Errorf("failed to write the config override file: %w", err)
	}

	return nil
}

func LoadOverride(configPath, nodeID string) (*ConfigOverride, error) {
	overrideFile := filepath.Join(filepath.Dir(configPath), nodeID+iniExtension)
	o := NewOverride(overrideFile)

	p, pErr := config.NewParser(o)
	if pErr != nil {
		return nil, fmt.Errorf("failed to initialize the config parser: %w", pErr)
	}

	if err := p.ParseFile(overrideFile); err != nil {
		return nil, fmt.Errorf("failed to parse the config override file: %w", err)
	}

	//nolint:gocritic //it's ok not to check the error before returning here
	return o, o.parse()
}

func UpdateOverride(configFile, nodeID string) error {
	if nodeID == "" {
		return nil
	}

	overrideFile := filepath.Join(filepath.Dir(configFile), nodeID+iniExtension)
	o := NewOverride(overrideFile)

	parser, pErr := config.NewParser(o)
	if pErr != nil {
		return fmt.Errorf("failed to initialize the config parser: %w", pErr)
	}

	if err := parser.ParseFile(overrideFile); err != nil {
		return fmt.Errorf("failed to parse the config override file: %w", err)
	}

	if err := parser.WriteFile(overrideFile); err != nil {
		return fmt.Errorf("failed to write the config override file: %w", err)
	}

	return nil
}

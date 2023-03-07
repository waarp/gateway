package conf

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

// LocalOverrides is a global instance of configOverride containing the local
// configuration overrides defined for this particular gateway node.
//
//nolint:gochecknoglobals //global var is required here for simplicity
var LocalOverrides *configOverride

// configOverride is a struct defining a list of settings local to a gateway instance
// (or node) which can be used to configOverride settings defined at the cluster level.
type configOverride struct {
	overrideLock    sync.RWMutex
	filename        string
	ListenAddresses *addressOverride `group:"Address Indirection"`
}

// newOverride returns a new correctly initialized, instance of configOverride.
func newOverride(filename string) *configOverride {
	return &configOverride{
		filename:        filename,
		ListenAddresses: &addressOverride{addressMap: map[string]string{}},
	}
}

// InitTestOverrides is a test helper function to quickly initiate the
// LocalOverrides global variable. This function should only be used in tests.
func InitTestOverrides(c convey.C) {
	ovrdFile := testhelpers.TempFile(c, "test_addr_override_*.ini")
	LocalOverrides = newOverride(ovrdFile)
	c.So(LocalOverrides.ListenAddresses.parse(), convey.ShouldBeNil)
}

func (o *configOverride) parse() error {
	return o.ListenAddresses.parse()
}

func (o *configOverride) writeFile() error {
	file, err := os.OpenFile(o.filename, os.O_TRUNC|os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open config override file: %w", err)
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

func (o *configOverride) writeTo(w io.Writer) error {
	parser, err := config.NewParser(o)
	if err != nil {
		return fmt.Errorf("failed to initialize the config parser: %w", err)
	}

	parser.Write(w)

	return nil
}

func CreateOverride(configFile, nodeID string) error {
	if nodeID == "" {
		return nil
	}

	overrideFile := filepath.Join(filepath.Dir(configFile), nodeID+".ini")
	o := newOverride(overrideFile)

	p, err := config.NewParser(o)
	if err != nil {
		return fmt.Errorf("failed to initialize the config parser: %w", err)
	}

	if err := p.WriteFile(overrideFile); err != nil {
		return fmt.Errorf("failed to write the config override file: %w", err)
	}

	return nil
}

func LoadOverride(configPath, nodeID string) (*configOverride, error) {
	overrideFile := filepath.Join(filepath.Dir(configPath), nodeID+".ini")
	o := newOverride(overrideFile)

	p, err := config.NewParser(o)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize the config parser: %w", err)
	}

	if err := p.ParseFile(overrideFile); err != nil {
		return nil, fmt.Errorf("failed to parse the config override file: %w", err)
	}

	return o, o.parse()
}

func UpdateOverride(configFile, nodeID string) error {
	if nodeID == "" {
		return nil
	}

	overrideFile := filepath.Join(filepath.Dir(configFile), nodeID+".ini")
	o := newOverride(overrideFile)

	parser, err := config.NewParser(o)
	if err != nil {
		return fmt.Errorf("failed to initialize the config parser: %w", err)
	}

	if err := parser.ParseFile(overrideFile); err != nil {
		return fmt.Errorf("failed to parse the config override file: %w", err)
	}

	if err := parser.WriteFile(overrideFile); err != nil {
		return fmt.Errorf("failed to write the config override file: %w", err)
	}

	return nil
}

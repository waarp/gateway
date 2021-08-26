package conf

import (
	"io"
	"os"
	"path/filepath"
	"sync"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/config"
)

// LocalOverrides is a global instance of configOverride containing the local
// configuration overrides defined for this particular gateway node.
var LocalOverrides *configOverride

var overrideLock sync.RWMutex

// configOverride is a struct defining a list of settings local to a gateway instance
// (or node) which can be used to configOverride settings defined at the cluster level.
type configOverride struct {
	filename        string
	ListenAddresses *addressOverride `group:"Address Indirection"`
}

// newOverride returns a new correctly initialised, instance of configOverride.
func newOverride() *configOverride {
	return &configOverride{
		ListenAddresses: &addressOverride{},
	}
}

// InitTestOverrides is a test helper function to quickly initiate the
// LocalOverrides global variable. This function should only be used in tests.
func InitTestOverrides(c convey.C) {
	ovrdFile := testhelpers.TempFile(c, "test_addr_override_*.ini")
	LocalOverrides = newOverride()
	LocalOverrides.filename = ovrdFile
	c.So(LocalOverrides.ListenAddresses.parse(), convey.ShouldBeNil)
}

func (o *configOverride) writeFile() error {
	file, err := os.OpenFile(o.filename, os.O_TRUNC|os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	o.writeTo(file)
	return file.Close()
}

func (o *configOverride) writeTo(w io.Writer) {
	parser := config.NewParser(o)
	parser.Write(w)
}

func createOverride(configFile, nodeID string) error {
	if nodeID == "" {
		return nil
	}
	overrideFile := filepath.Join(filepath.Dir(configFile), nodeID+".ini")

	o := newOverride()
	p := config.NewParser(o)
	return p.WriteFile(overrideFile)
}

func loadOverride(configPath, nodeID string) (*configOverride, error) {
	if nodeID == "" {
		return nil, nil
	}
	overrideFile := filepath.Join(filepath.Dir(configPath), nodeID+".ini")

	o := newOverride()
	p := config.NewParser(o)
	if err := p.ParseFile(overrideFile); err != nil {
		return nil, err
	}
	return o, nil
}

func updateOverride(configFile, nodeID string) error {
	if nodeID == "" {
		return nil
	}
	overrideFile := filepath.Join(filepath.Dir(configFile), nodeID+".ini")

	o := newOverride()
	parser := config.NewParser(o)
	if err := parser.ParseFile(overrideFile); err != nil {
		return err
	}
	return parser.WriteFile(overrideFile)
}

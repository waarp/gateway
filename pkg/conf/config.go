package conf

import (
	"os"

	"code.waarp.fr/waarp/gateway-ng/pkg/tk/config"
)

// ServerConfig holds the server configuration options
type ServerConfig struct {

	// Needed for test purpose (to be removed later)
	Foo string `ini-name:"Foo" default:"default-foo" description:"foo desc"`
	Bar string `ini-name:"Bar" default:"default-bar"`
	Baz string `ini-name:"Baz" default:"default-baz"`
}

// LoadServerConfig creates a configuration object.
// It tries to read configuration files from common places to populate the
// configuration object (paths are relative to cwd):
// - gatewayd.ini
// - etc/gatewayd.ini
// - /etc/waarp/gatewayd.ini
func LoadServerConfig(userConfig string) (*ServerConfig, error) {
	fileList := []string{
		"gatewayd.ini",
		"etc/gatewayd.ini",
		"/etc/waarp/gatewayd.ini",
	}
	c := &ServerConfig{}
	p := config.NewParser(c)

	if userConfig != "" {
		if err := p.ParseFile(userConfig); err != nil {
			return nil, err
		}
	} else {
		for _, file := range fileList {
			err := p.ParseFile(file)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			break
		}
	}

	return c, nil
}

// UpdateServerConfig updates a configuration file to a newer version of the
// configuration struct.
func UpdateServerConfig(configFile string) error {
	c := &ServerConfig{}
	p := config.NewParser(c)
	if err := p.ParseFile(configFile); err != nil {
		return err
	}
	return p.WriteFile(configFile)
}

// CreateServerConfig creates a new configuration file at the given location
func CreateServerConfig(configFile string) error {
	c := &ServerConfig{}
	p := config.NewParser(c)
	return p.WriteFile(configFile)
}

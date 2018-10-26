package conf

import (
	"os"

	"code.waarp.fr/waarp/gateway-ng/pkg/tk/config"
)

// ServerConfig holds the server configuration options
type ServerConfig struct {
	Log struct {
		Level string `ini-name:"Level" default:"INFO" description:"All messages with a severity above this level will be logged. Possible values are DEBUG, INFO, WARNING, ERROR and CRITICAL."` 
		LogTo string `ini-name:"LogTo" default:"stdout" description:"The path to the file where the logs must be written. Special values 'stdout' and 'syslog' log respectively to the standard outpout and to the syslog daemon"` 
		SyslogFacility string `ini-name:"SyslogFacility" default:"local0" description:"If LogTo is set on 'syslog', the logs will be written to this facility."` 
	} `group:"log"`
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

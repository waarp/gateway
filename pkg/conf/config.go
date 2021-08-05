// Package conf contains the structure representing the gateway's
// configuration as defined in the configuration file.
package conf

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/config"
)

// GlobalConfig is a global instance of GatewayConfig containing the configuration
// of the gateway instance.
var GlobalConfig GatewayConfig

// GatewayConfig regroups the gateway's global configuration with its local,
// instance-specific, settings overrides.
type GatewayConfig struct {
	ServerConf     ServerConfig
	LocalOverrides Override
}

// ServerConfig holds the server configuration options
type ServerConfig struct {
	GatewayName string           `ini-name:"GatewayName" default:"waarp-gateway" description:"The name given to identify this gateway instance. If the the database is shared between multiple gateways, this name MUST be unique across these gateways."`
	Paths       PathsConfig      `group:"paths"`
	Log         LogConfig        `group:"log"`
	Admin       AdminConfig      `group:"admin"`
	Database    DatabaseConfig   `group:"database"`
	Controller  ControllerConfig `group:"controller"`
}

// PathsConfig holds the server paths
type PathsConfig struct {
	GatewayHome   string `ini-name:"GatewayHome" description:"The root directory of the gateway. By default, it is the working directory of the process."`
	InDirectory   string `ini-name:"InDirectory" default:"in" description:"The directory for all incoming files."`
	OutDirectory  string `ini-name:"OutDirectory" default:"out" description:"The directory for all outgoing files."`
	WorkDirectory string `ini-name:"WorkDirectory" default:"work" description:"The directory for all running transfer files."`
}

// LogConfig holds the server logging options
type LogConfig struct {
	Level          string `ini-name:"Level" default:"INFO" description:"All messages with a severity above this level will be logged. Possible values are DEBUG, INFO, WARNING, ERROR and CRITICAL."`
	LogTo          string `ini-name:"LogTo" default:"stdout" description:"The path to the file where the logs must be written. Special values 'stdout' and 'syslog' log respectively to the standard output and to the syslog daemon"`
	SyslogFacility string `ini-name:"SyslogFacility" default:"local0" description:"If LogTo is set on 'syslog', the logs will be written to this facility."`
}

// AdminConfig holds the server administration options
type AdminConfig struct {
	Host    string `ini-name:"Host" default:"localhost" description:"The address used by the admin interface."`
	Port    uint16 `ini-name:"Port" default:"8080" description:"The port used by the admin interface. If the port is 0, a free port will automatically be chosen."`
	TLSCert string `ini-name:"TLSCert" description:"Path of the TLS certificate for the admin interface."`
	TLSKey  string `ini-name:"TLSKey" description:"Path of the key of the TLS certificate."`
}

// DatabaseConfig holds the server database options
type DatabaseConfig struct {
	Type          string `ini-name:"Type" default:"sqlite" description:"Name of the RDBMS used for the gateway database. Possible values: sqlite, mysql, postgresql"`
	Address       string `ini-name:"Address" default:"waarp-gateway.db" description:"Address of the database"`
	Name          string `ini-name:"Name" description:"The name of the database"`
	User          string `ini-name:"User" description:"The name of the gateway database user"`
	Password      string `ini-name:"Password" description:"The password of the gateway database user"`
	TLSCert       string `ini-name:"TLSCert" description:"Path of the database TLS certificate file."`
	TLSKey        string `ini-name:"TLSKey" description:"Path of the key of the TLS certificate file."`
	AESPassphrase string `ini-name:"AESPassphrase" default:"passphrase.aes" description:"The path to the file containing the passphrase used to encrypt account passwords using AES"`
}

// ControllerConfig holds the transfer controller options
type ControllerConfig struct {
	Delay           time.Duration `ini-name:"Delay" default:"5s" description:"The frequency at which the database will be probed for new transfers"`
	MaxTransfersIn  uint64        `ini-name:"MaxTransferIn" description:"The maximum number of concurrent incoming transfers allowed on the gateway (0 = unlimited)."`
	MaxTransfersOut uint64        `ini-name:"MaxTransferOut" description:"The maximum number of concurrent outgoing transfers allowed on the gateway (0 = unlimited)."`
}

func normalizePaths(config *ServerConfig) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to retrieve current working directory: %s", err)
	}

	if config.Paths.GatewayHome != "" {
		if filepath.IsAbs(config.Paths.GatewayHome) {
			config.Paths.GatewayHome = filepath.Clean(config.Paths.GatewayHome)
		} else {
			config.Paths.GatewayHome = filepath.Join(wd, config.Paths.GatewayHome)
		}
	} else {
		config.Paths.GatewayHome = wd
	}
	if config.Paths.InDirectory != "" {
		if filepath.IsAbs(config.Paths.InDirectory) {
			config.Paths.InDirectory = filepath.Clean(config.Paths.InDirectory)
		} else {
			config.Paths.InDirectory = filepath.Join(config.Paths.GatewayHome, config.Paths.InDirectory)
		}
	} else {
		config.Paths.InDirectory = config.Paths.GatewayHome
	}
	if config.Paths.OutDirectory != "" {
		if filepath.IsAbs(config.Paths.OutDirectory) {
			config.Paths.OutDirectory = filepath.Clean(config.Paths.OutDirectory)
		} else {
			config.Paths.OutDirectory = filepath.Join(config.Paths.GatewayHome, config.Paths.OutDirectory)
		}
	} else {
		config.Paths.OutDirectory = config.Paths.GatewayHome
	}
	if config.Paths.WorkDirectory != "" {
		if filepath.IsAbs(config.Paths.WorkDirectory) {
			config.Paths.WorkDirectory = filepath.Clean(config.Paths.WorkDirectory)
		} else {
			config.Paths.WorkDirectory = filepath.Join(config.Paths.GatewayHome, config.Paths.WorkDirectory)
		}
	} else {
		config.Paths.WorkDirectory = config.Paths.GatewayHome
	}

	return nil
}

func loadServerConfig(userConfig string) (*ServerConfig, string, error) {
	c := &ServerConfig{}
	p := config.NewParser(c)

	if userConfig != "" {
		if err := p.ParseFile(userConfig); err != nil {
			return nil, "", err
		}
	} else {
		for _, file := range fileList {
			err := p.ParseFile(file)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, "", err
			}
			userConfig = file
			break
		}
	}

	if err := normalizePaths(c); err != nil {
		return nil, "", err
	}
	return c, userConfig, nil
}

// LoadServerConfig creates a configuration object.
// It tries to read configuration files from common places to populate the
// configuration object (paths are relative to cwd):
// - gatewayd.ini
// - etc/gatewayd.ini
// - /etc/waarp/gatewayd.ini
func LoadServerConfig(userConfig string) (*ServerConfig, error) {
	c, _, err := loadServerConfig(userConfig)
	if err != nil {
		return nil, err
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

func updateOverride(configFile, nodeID string) error {
	if nodeID == "" {
		return nil
	}
	overrideFile := filepath.Join(filepath.Dir(configFile), nodeID+".ini")
	oRead := NewOverride(overrideFile)
	pRead := config.NewParser(&oRead)
	if err := pRead.ParseFile(overrideFile); err != nil {
		return err
	}

	oWrite := oRead.makeWrite()
	pWrite := config.NewParser(oWrite)
	return pWrite.WriteFile(overrideFile)
}

func createOverride(configFile, nodeID string) error {
	if nodeID == "" {
		return nil
	}
	o := &overrideWrite{}
	p := config.NewParser(o)
	overrideFile := filepath.Join(filepath.Dir(configFile), nodeID+".ini")
	return p.WriteFile(overrideFile)
}

func loadOverride(configPath, nodeID string) (*Override, error) {
	overrideConfig := &Override{}
	p := config.NewParser(overrideConfig)
	overrideFile := filepath.Join(filepath.Dir(configPath), nodeID+".ini")
	if err := p.ParseFile(overrideFile); err != nil {
		return nil, err
	}
	return overrideConfig, nil
}

// LoadGatewayConfig loads the given configuration file, along with the local
// override file associated with the given node ID, and stores both in the global
// GlobalConfig variable.
func LoadGatewayConfig(configFile, nodeID string) error {
	serverConfig, configPath, err := loadServerConfig(configFile)
	if err != nil {
		return err
	}

	overrideConfig, err := loadOverride(configPath, nodeID)
	if err != nil {
		return err
	}

	GlobalConfig.ServerConf = *serverConfig
	GlobalConfig.LocalOverrides = *overrideConfig
	return nil
}

// UpdateGatewayConfig updates both the gateway configuration file, and the
// settings override file associated with the given node ID to their latest versions.
func UpdateGatewayConfig(configFile, nodeID string) error {
	if err := UpdateServerConfig(configFile); err != nil {
		return err
	}

	return updateOverride(configFile, nodeID)
}

// CreateGatewayConfig creates a new configuration file at the given location,
// along with a new settings override file for the given node ID.
func CreateGatewayConfig(configFile, nodeID string) error {
	if err := CreateServerConfig(configFile); err != nil {
		return err
	}

	return createOverride(configFile, nodeID)
}

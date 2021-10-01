// Package conf contains the structure representing the gateway's
// configuration as defined in the configuration file.
package conf

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/config"
)

// ServerConfig holds the server configuration options.
//nolint:lll // cannot split struct tags
type ServerConfig struct {
	GatewayName string           `ini-name:"GatewayName" default:"waarp-gateway" description:"The name given to identify this gateway instance. If the the database is shared between multiple gateways, this name MUST be unique across these gateways."`
	Paths       PathsConfig      `group:"paths"`
	Log         LogConfig        `group:"log"`
	Admin       AdminConfig      `group:"admin"`
	Database    DatabaseConfig   `group:"database"`
	Controller  ControllerConfig `group:"controller"`
}

// PathsConfig holds the server paths.
//nolint:lll // cannot split struct tags
type PathsConfig struct {
	GatewayHome   string `ini-name:"GatewayHome" description:"The root directory of the gateway. By default, it is the working directory of the process."`
	InDirectory   string `ini-name:"InDirectory" default:"in" description:"DEPRECATED, use DefaultInDir instead"`      // DEPRECATED
	OutDirectory  string `ini-name:"OutDirectory" default:"out" description:"DEPRECATED, use DefaultOutDir instead"`   // DEPRECATED
	WorkDirectory string `ini-name:"WorkDirectory" default:"work" description:"DEPRECATED, use DefaultTmpDir instead"` // DEPRECATED
	DefaultInDir  string `ini-name:"DefaultInDir" default:"in" description:"The directory for all incoming files."`
	DefaultOutDir string `ini-name:"DefaultOutDir" default:"out" description:"The directory for all outgoing files."`
	DefaultTmpDir string `ini-name:"DefaultTmpDir" default:"tmp" description:"The directory for all running transfer files."`
}

// LogConfig holds the server logging options.
//nolint:lll // cannot split struct tags
type LogConfig struct {
	Level          string `ini-name:"Level" default:"INFO" description:"All messages with a severity above this level will be logged. Possible values are DEBUG, INFO, WARNING, ERROR and CRITICAL."`
	LogTo          string `ini-name:"LogTo" default:"stdout" description:"The path to the file where the logs must be written. Special values 'stdout' and 'syslog' log respectively to the standard output and to the syslog daemon"`
	SyslogFacility string `ini-name:"SyslogFacility" default:"local0" description:"If LogTo is set on 'syslog', the logs will be written to this facility."`
}

// AdminConfig holds the server administration options.
//nolint:lll // cannot split struct tags
type AdminConfig struct {
	Host    string `ini-name:"Host" default:"localhost" description:"The address used by the admin interface."`
	Port    uint16 `ini-name:"Port" default:"8080" description:"The port used by the admin interface. If the port is 0, a free port will automatically be chosen."`
	TLSCert string `ini-name:"TLSCert" description:"Path of the TLS certificate for the admin interface."`
	TLSKey  string `ini-name:"TLSKey" description:"Path of the key of the TLS certificate."`
}

// DatabaseConfig holds the server database options.
//nolint:lll // cannot split struct tags
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

// ControllerConfig holds the transfer controller options.
//nolint:lll // cannot split struct tags
type ControllerConfig struct {
	Delay           time.Duration `ini-name:"Delay" default:"5s" description:"The frequency at which the database will be probed for new transfers"`
	MaxTransfersIn  uint64        `ini-name:"MaxTransferIn" description:"The maximum number of concurrent incoming transfers allowed on the gateway (0 = unlimited)."`
	MaxTransfersOut uint64        `ini-name:"MaxTransferOut" description:"The maximum number of concurrent outgoing transfers allowed on the gateway (0 = unlimited)."`
}

func normalizePaths(configFile *ServerConfig, logger *log.Logger) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to retrieve current working directory: %w", err)
	}

	if configFile.Paths.GatewayHome != "" {
		if filepath.IsAbs(configFile.Paths.GatewayHome) {
			configFile.Paths.GatewayHome = filepath.Clean(configFile.Paths.GatewayHome)
		} else {
			configFile.Paths.GatewayHome = filepath.Join(wd, configFile.Paths.GatewayHome)
		}
	} else {
		configFile.Paths.GatewayHome = wd
	}

	if configFile.Paths.InDirectory != "" {
		logger.Warning("Option 'InDirectory' is deprecated, use 'DefaultInDir' instead")

		configFile.Paths.DefaultInDir = configFile.Paths.InDirectory
	}

	if configFile.Paths.OutDirectory != "" {
		logger.Warning("Option 'OutDirectory' is deprecated, use 'DefaultOutDir' instead")

		configFile.Paths.DefaultOutDir = configFile.Paths.OutDirectory
	}

	if configFile.Paths.WorkDirectory != "" {
		logger.Warning("Option 'WorkDirectory' is deprecated, use 'DefaultTmpDir' instead")

		configFile.Paths.DefaultTmpDir = configFile.Paths.WorkDirectory
	}

	return nil
}

// LoadServerConfig creates a configuration object.
// It tries to read configuration files from common places to populate the
// configuration object (paths are relative to cwd):
// - gatewayd.ini,
// - etc/gatewayd.ini,
// - /etc/waarp/gatewayd.ini.
func LoadServerConfig(userConfig string) (*ServerConfig, error) {
	c := &ServerConfig{}

	p, err := config.NewParser(c)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize a parser for the server configuration: %w", err)
	}

	if userConfig != "" {
		if err := p.ParseFile(userConfig); err != nil {
			return nil, fmt.Errorf("cannot parse configuration file %q: %w", userConfig, err)
		}
	} else {
		if err := loadDefaultConfig(p); err != nil {
			return nil, err
		}
	}

	if err := log.InitBackend(c.Log.Level, c.Log.LogTo, c.Log.SyslogFacility); err != nil {
		return nil, fmt.Errorf("failed to initialize log backend: %w", err)
	}

	logger := log.NewLogger("Config file")
	if err := normalizePaths(c, logger); err != nil {
		return nil, err
	}

	return c, nil
}

func loadDefaultConfig(p *config.Parser) error {
	for _, file := range getDefaultConfFiles() {
		err := p.ParseFile(file)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return fmt.Errorf("cannot parse configuration file %q: %w", file, err)
		}

		break
	}

	return nil
}

// UpdateServerConfig updates a configuration file to a newer version of the
// configuration struct.
func UpdateServerConfig(configFile string) error {
	c := &ServerConfig{}

	p, err := config.NewParser(c)
	if err != nil {
		return fmt.Errorf("cannot initialize a parser for the server configuration: %w", err)
	}

	if err := p.ParseFile(configFile); err != nil {
		return fmt.Errorf("cannot parse configuration file %q: %w", configFile, err)
	}

	if err := p.WriteFile(configFile); err != nil {
		return fmt.Errorf("cannot update server configuration file %q: %w", configFile, err)
	}

	return nil
}

// CreateServerConfig creates a new configuration file at the given location.
func CreateServerConfig(configFile string) error {
	c := &ServerConfig{}

	p, err := config.NewParser(c)
	if err != nil {
		return fmt.Errorf("cannot initialize a parser for the server configuration: %w", err)
	}

	if err := p.WriteFile(configFile); err != nil {
		return fmt.Errorf("cannot create server configuration file %q: %w", configFile, err)
	}

	return nil
}

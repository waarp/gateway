// Package conf contains the structure representing the gateway's
// configuration as defined in the configuration file.
package conf

import (
	"errors"
	"fmt"
	gofs "io/fs"
	"os"
	"path/filepath"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/config"
)

const (
	ConfigFileEnvVar = "WAARP_CONFIG_FILE"
	ConfigDirEnvVar  = "WAARP_CONFIG_DIR"
)

// ServerConfig holds the server configuration options
//
//nolint:lll // cannot split struct tags
type ServerConfig struct {
	GatewayName string `ini-name:"GatewayName" default:"waarp-gateway" description:"The name given to identify this gateway instance. If the database is shared between multiple gateways, this name MUST be unique across these gateways."`

	NodeID    string          `no-ini:"yes" no-flag:"yes" hidden:"yes"`
	Overrides *ConfigOverride `no-ini:"yes" no-flag:"yes" hidden:"yes"`

	Paths      PathsConfig      `group:"paths"`
	Log        LogConfig        `group:"log"`
	Admin      AdminConfig      `group:"admin"`
	Database   DatabaseConfig   `group:"database"`
	Controller ControllerConfig `group:"controller"`
	Security   SecurityConfig   `group:"security"`
}

// PathsConfig holds the server paths.
//
//nolint:lll // cannot split struct tags
type PathsConfig struct {
	GatewayHome   string   `ini-name:"GatewayHome" description:"The root directory of the gateway. By default, it is the working directory of the process."`
	DefaultInDir  string   `ini-name:"DefaultInDir" default:"in" description:"The default directory for incoming files."`
	DefaultOutDir string   `ini-name:"DefaultOutDir" default:"out" description:"The default directory for outgoing files."`
	DefaultTmpDir string   `ini-name:"DefaultTmpDir" default:"tmp" description:"The default directory for running transfer files."`
	FilePerms     FileMode `ini-name:"FilePermissions" default:"0640" description:"The UNIX permissions of the files created by the gateway." `
	DirPerms      FileMode `ini-name:"DirectoryPermissions" default:"0750" description:"The UNIX permissions of the directories created by the gateway." `

	// Deprecated fields.
	InDirectory   string `ini-name:"InDirectory" hidden:"yes"`   // Deprecated: replaced by DefaultInDir
	OutDirectory  string `ini-name:"OutDirectory" hidden:"yes"`  // Deprecated: replaced by DefaultOutDir
	WorkDirectory string `ini-name:"WorkDirectory" hidden:"yes"` // Deprecated: replaced by DefaultTmpDir
}

// LogConfig holds the server logging options.
//
//nolint:lll // cannot split struct tags
type LogConfig struct {
	Level          string `ini-name:"Level" default:"INFO" description:"All messages with a severity above this level will be logged. Possible values are DEBUG, INFO, WARNING, ERROR and CRITICAL."`
	LogTo          string `ini-name:"LogTo" default:"stdout" description:"The path to the file where the logs must be written. Special values 'stdout', 'stderr' and 'syslog' log respectively to the standard output, standard error output, and to the syslog daemon"`
	SyslogFacility string `ini-name:"SyslogFacility" default:"local0" description:"If LogTo is set on 'syslog', the logs will be written to this facility."`
}

// AdminConfig holds the server administration options.
//
//nolint:lll // cannot split struct tags
type AdminConfig struct {
	Host          string `ini-name:"Host" default:"0.0.0.0" description:"The address used by the admin interface."`
	Port          uint16 `ini-name:"Port" default:"8080" description:"The port used by the admin interface. If the port is 0, a random free port will automatically be chosen."`
	DisableWebUI  bool   `ini-name:"DisableWebUI" description:"If set to true, the web admin interface will be disabled."`
	TLSCert       string `ini-name:"TLSCert" description:"Path of the TLS certificate for the admin interface."`
	TLSKey        string `ini-name:"TLSKey" description:"Path of the key of the TLS certificate."`
	TLSPassphrase string `ini-name:"TLSPassphrase" description:"RFC 1423 style passphrase for the TLS key (if the key is encrypted)."`
}

// DatabaseConfig holds the server database options.
//
//nolint:lll // cannot split struct tags
type DatabaseConfig struct {
	Type          string `ini-name:"Type" default:"sqlite" description:"Type of the RDBMS used for the gateway database. Possible values: sqlite, mysql, postgresql"`
	Address       string `ini-name:"Address" default:"waarp-gateway.db" description:"Address of the database"`
	Name          string `ini-name:"Name" description:"The name of the database"`
	User          string `ini-name:"User" description:"The name of the gateway database user"`
	Password      string `ini-name:"Password" description:"The password of the gateway database user"`
	TLSCert       string `ini-name:"TLSCert" description:"Path of the database TLS certificate file."`
	TLSKey        string `ini-name:"TLSKey" description:"Path of the key of the TLS certificate file."`
	AESPassphrase string `ini-name:"AESPassphrase" default:"passphrase.aes" description:"The path to the file containing the passphrase used to encrypt account passwords using AES"`

	// Deprecated fields
	String string `ini-name:"String" hidden:"true"` // Deprecated: misspell of "Name"
}

// ControllerConfig holds the transfer controller options.
//
//nolint:lll // cannot split struct tags
type ControllerConfig struct {
	Delay           time.Duration `ini-name:"Delay" default:"5s" description:"The frequency at which the database will be probed for new transfers"`
	MaxTransfersIn  uint64        `ini-name:"MaxTransfersIn" description:"The maximum number of concurrent incoming transfers allowed on the gateway (0 = unlimited)."`
	MaxTransfersOut uint64        `ini-name:"MaxTransfersOut" description:"The maximum number of concurrent outgoing transfers allowed on the gateway (0 = unlimited)."`

	// Deprecated fields
	MaxTransferIn  uint64 `ini-name:"MaxTransferIn" hidden:"yes"`  // Deprecated: misspell of "MaxTransfersIn"
	MaxTransferOut uint64 `ini-name:"MaxTransferOut" hidden:"yes"` // Deprecated: misspell of "MaxTransfersOut"
}

// SecurityConfig holds the options protecting the authentication against
// repeated attempts.
//
//nolint:lll // cannot split struct tags
type SecurityConfig struct {
	MaxAuthFailures        uint32        `ini-name:"MaxAuthFailures" default:"5" description:"The number of failed authentications, for the same account from the same address, after which new attempts are temporarily refused (0 = no limit)."`
	MaxAuthFailuresPerHost uint32        `ini-name:"MaxAuthFailuresPerHost" default:"25" description:"The number of failed authentications from the same address, whatever the account, after which new attempts from this address are temporarily refused (0 = no limit)."`
	AuthFailureWindow      time.Duration `ini-name:"AuthFailureWindow" default:"5m" description:"The period after which failed authentications are forgotten if no new failure occurs."`
	AuthLockDuration       time.Duration `ini-name:"AuthLockDuration" default:"1m" description:"How long new attempts are refused once the maximum number of failures is reached. The duration doubles at each new lock."`
	AuthMaxLockDuration    time.Duration `ini-name:"AuthMaxLockDuration" default:"30m" description:"The maximum duration of a lock."`
}

func normalizePaths(configFile *ServerConfig) error {
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

	return nil
}

func handleDeprecatedOpts(configFile *ServerConfig, logger *log.Logger) {
	const (
		defaultInDir  = "in"
		defaultOutDir = "out"
		defaultTmpDir = "tmp"
	)

	if configFile.Paths.InDirectory != "" {
		logger.Warning("Option 'paths.InDirectory' is deprecated, use 'paths.DefaultInDir' instead")

		if configFile.Paths.DefaultInDir == "" || configFile.Paths.DefaultInDir == defaultInDir {
			configFile.Paths.DefaultInDir = configFile.Paths.InDirectory
		}
	}

	if configFile.Paths.OutDirectory != "" {
		logger.Warning("Option 'paths.OutDirectory' is deprecated, use 'paths.DefaultOutDir' instead")

		if configFile.Paths.DefaultOutDir == "" || configFile.Paths.DefaultOutDir == defaultOutDir {
			configFile.Paths.DefaultOutDir = configFile.Paths.OutDirectory
		}
	}

	if configFile.Paths.WorkDirectory != "" {
		logger.Warning("Option 'paths.WorkDirectory' is deprecated, use 'paths.DefaultTmpDir' instead")

		if configFile.Paths.DefaultTmpDir == "" || configFile.Paths.DefaultTmpDir == defaultTmpDir {
			configFile.Paths.DefaultTmpDir = configFile.Paths.WorkDirectory
		}
	}

	if configFile.Database.String != "" {
		logger.Warning("Option 'database.String' is deprecated, use 'Database.Name' instead")

		if configFile.Database.Name == "" {
			configFile.Database.Name = configFile.Database.String
		}
	}

	if configFile.Controller.MaxTransferIn != 0 {
		logger.Warning("Option 'controller.MaxTransferIn' is deprecated, use 'controller.MaxTransfersIn' instead")

		if configFile.Controller.MaxTransfersIn == 0 {
			configFile.Controller.MaxTransfersIn = configFile.Controller.MaxTransferIn
		}
	}

	if configFile.Controller.MaxTransferOut != 0 {
		logger.Warning("Option 'controller.MaxTransferOut' is deprecated, use 'controller.MaxTransfersOut' instead")

		if configFile.Controller.MaxTransfersOut == 0 {
			configFile.Controller.MaxTransfersOut = configFile.Controller.MaxTransferOut
		}
	}
}

var ErrNoConfigFile = errors.New("no config file found")

// defaultConfFiles returns the default locations of the configuration file. It
// is a variable so that the tests do not depend on a configuration installed
// on the machine.
//
//nolint:gochecknoglobals //overridden by the tests
var defaultConfFiles = getDefaultConfFiles

// ParseServerConfig parses and returns the ServerConfig contained in the given
// user config file. If no user file is given, the file will be taken from the
// OS's default config file locations.
func ParseServerConfig(userConfig string) (*ServerConfig, error) {
	c := &ServerConfig{}

	p, pErr := config.NewParser(c)
	if pErr != nil {
		return nil, fmt.Errorf("failed to initialize the config parser: %w", pErr)
	}

	if userConfig != "" {
		if err := p.ParseFile(userConfig); err != nil {
			return nil, fmt.Errorf("failed to parse the config file: %w", err)
		}
	} else {
		for _, file := range defaultConfFiles() {
			if err := p.ParseFile(file); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}

				return nil, fmt.Errorf("failed to parse the config file: %w", err)
			}

			userConfig = file

			break
		}

		if userConfig == "" {
			return nil, ErrNoConfigFile
		}
	}

	fs.FilePerms = fs.FileMode(c.Paths.FilePerms)
	fs.DirPerms = fs.FileMode(c.Paths.DirPerms)

	if err := logging.SetLogBackend(c.Log.Level, c.Log.LogTo, c.Log.SyslogFacility,
		"waarp-gateway"); err != nil {
		return nil, fmt.Errorf("failed to initialize log backend: %w", err)
	}

	handleDeprecatedOpts(c, logging.NewLogger("Config file"))

	if err := normalizePaths(c); err != nil {
		return nil, err
	}

	return c, SetConfEnvVars(userConfig)
}

func SetConfEnvVars(userConfig string) error {
	if !filepath.IsAbs(userConfig) {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to retrieve current working directory: %w", err)
		}

		userConfig = filepath.Join(wd, userConfig)
	}

	os.Setenv(ConfigFileEnvVar, userConfig)
	os.Setenv(ConfigDirEnvVar, filepath.Dir(userConfig))

	return nil
}

func loadServerConfig(userConfig string) (*ServerConfig, string, error) {
	c, pErr := ParseServerConfig(userConfig)
	if pErr != nil {
		return nil, "", pErr
	}

	return c, userConfig, nil
}

// LoadServerConfig creates a configuration object.
// It tries to read configuration files from common places to populate the
// configuration object (paths are relative to cwd):
// - gatewayd.ini
// - etc/gatewayd.ini
// - /etc/waarp/gatewayd.ini.
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

	p, pErr := config.NewParser(c)
	if pErr != nil {
		return fmt.Errorf("cannot initialize a parser for the server configuration: %w", pErr)
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

	p, pErr := config.NewParser(c)
	if pErr != nil {
		return fmt.Errorf("cannot initialize a parser for the server configuration: %w", pErr)
	}

	if err := p.WriteFile(configFile); err != nil {
		return fmt.Errorf("cannot create server configuration file %q: %w", configFile, err)
	}

	return nil
}

// LoadGatewayConfig loads the given configuration file, along with the local
// configOverride file associated with the given node ID, and stores both in the global
// GlobalConfig variable.
func LoadGatewayConfig(configFile, nodeID string) (*ServerConfig, error) {
	serverConfig, configPath, err := loadServerConfig(configFile)
	if err != nil {
		return nil, err
	}

	if nodeID != "" {
		serverConfig.Overrides, err = LoadOverride(configPath, nodeID)
		if err != nil {
			return nil, err
		}
	}

	serverConfig.NodeID = nodeID

	return serverConfig, nil
}

// UpdateGatewayConfig updates both the gateway configuration file, and the
// settings configOverride file associated with the given node ID to their latest versions.
func UpdateGatewayConfig(configFile, nodeID string) error {
	if err := UpdateServerConfig(configFile); err != nil {
		return err
	}

	return UpdateOverride(configFile, nodeID)
}

// CreateGatewayConfig creates a new configuration file at the given location,
// along with a new settings configOverride file for the given node ID.
func CreateGatewayConfig(configFile, nodeID string) error {
	if err := CreateServerConfig(configFile); err != nil {
		return err
	}

	return CreateOverride(configFile, nodeID)
}

type FileMode gofs.FileMode

func (f FileMode) MarshalFlag() (string, error) {
	return fmt.Sprintf("0%o", f), nil
}

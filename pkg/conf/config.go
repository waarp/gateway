package conf

import (
	"os"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/config"
)

// ServerConfig holds the server configuration options
type ServerConfig struct {
	GatewayName string           `ini-name:"GatewayName" default:"waarp-gateway" description:"The name given to identify this gateway instance. If the the database is shared between multiple gateways, this name MUST be unique across these gateways."`
	Log         LogConfig        `group:"log"`
	Admin       AdminConfig      `group:"admin"`
	Database    DatabaseConfig   `group:"database"`
	Controller  ControllerConfig `group:"controller"`
}

// LogConfig holds the server logging options
type LogConfig struct {
	Level          string `ini-name:"Level" default:"INFO" description:"All messages with a severity above this level will be logged. Possible values are DEBUG, INFO, WARNING, ERROR and CRITICAL."`
	LogTo          string `ini-name:"LogTo" default:"stdout" description:"The path to the file where the logs must be written. Special values 'stdout' and 'syslog' log respectively to the standard outpout and to the syslog daemon"`
	SyslogFacility string `ini-name:"SyslogFacility" default:"local0" description:"If LogTo is set on 'syslog', the logs will be written to this facility."`
}

// AdminConfig holds the server administration options
type AdminConfig struct {
	Address string `ini-name:"Address" default:":8080" description:"The IP address + TCP port used by the admin interface."`
	TLSCert string `ini-name:"TLSCert" default:"" description:"Path of the TLS certificate for the admin interface."`
	TLSKey  string `ini-name:"TLSKey" default:"" description:"Path of the key of the TLS certificate."`
}

// DatabaseConfig holds the server database options
type DatabaseConfig struct {
	Type          string `ini-name:"Type" default:"" description:"Name of the RDBMS used for the gateway database"`
	Address       string `ini-name:"Address" default:"localhost" description:"Address of the database"`
	Port          uint16 `ini-name:"Port" default:"" description:"Port of the database"`
	Name          string `ini-name:"Name" default:"waarp_gatewayd" description:"The name of the database"`
	User          string `ini-name:"User" default:"waarp_gatewayd" description:"The name of the gateway database user"`
	Password      string `ini-name:"Password" default:"" description:"The password of the gateway database user"`
	AESPassphrase string `ini-name:"AESPassphrase" default:"aes_passphrase" description:"The path to the file containing the passphrase used to encrypt account passwords using AES"`
}

// ControllerConfig holds the transfer controller options
type ControllerConfig struct {
	Delay time.Duration `ini-name:"Delay" default:"5s" description:"The frequency at which the database will be probed for new transfers"`
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

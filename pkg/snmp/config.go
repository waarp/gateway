// Package snmp contains the code for the SNMP subsystem.
package snmp

import (
	"fmt"
	"net"

	"github.com/gosnmp/gosnmp"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	Version2  = "SNMPv2"
	Version2c = "SNMPv2c"
	Version3  = "SNMPv3"
)

const (
	V3SecurityNoAuthNoPriv = "noAuthNoPriv"
	V3SecurityAuthNoPriv   = "authNoPriv"
	V3SecurityAuthPriv     = "authPriv"
)

const (
	AuthMD5    = "MD5"
	AuthSHA    = "SHA"
	AuthSHA224 = "SHA-224"
	AuthSHA256 = "SHA-256"
	AuthSHA384 = "SHA-384"
	AuthSHA512 = "SHA-512"
)

const (
	PrivDES     = "DES"
	PrivAES     = "AES"
	PrivAES192  = "AES-192"
	PrivAES256  = "AES-256"
	PrivAES192C = "AES-192C"
	PrivAES256C = "AES-256C"
)

type ServerConfig struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	LocalUDPAddress string `xorm:"local_udp_address"`
	Community       string `xorm:"community"`

	SNMPv3Only           bool                `xorm:"v3_only"`
	SNMPv3Username       string              `xorm:"v3_username"`
	SNMPv3AuthProtocol   string              `xorm:"v3_auth_protocol"`
	SNMPv3AuthPassphrase database.SecretText `xorm:"v3_auth_passphrase"`
	SNMPv3PrivProtocol   string              `xorm:"v3_priv_protocol"`
	SNMPv3PrivPassphrase database.SecretText `xorm:"v3_priv_passphrase"`
}

func (*ServerConfig) TableName() string   { return "snmp_server_conf" }
func (*ServerConfig) Appellation() string { return "SNMP server config" }
func (s *ServerConfig) GetID() int64      { return s.ID }

func (s *ServerConfig) BeforeWrite(db database.Access) error {
	s.Owner = conf.GlobalConfig.GatewayName

	if s.Community == "" {
		s.Community = "public"
	}

	if _, err := net.ResolveUDPAddr("udp", s.LocalUDPAddress); err != nil {
		return database.NewValidationError("invalid UDP address %q: %w", s.LocalUDPAddress, err)
	}

	if proto := getAuthProtocol(s.SNMPv3AuthProtocol); proto == 0 {
		return database.NewValidationError("invalid authentication protocol %q", s.SNMPv3AuthProtocol)
	} else if proto == gosnmp.NoAuth {
		s.SNMPv3AuthPassphrase = ""
	}

	if proto := getPrivProtocol(s.SNMPv3PrivProtocol); proto == 0 {
		return database.NewValidationError("invalid privacy protocol %q", s.SNMPv3PrivProtocol)
	} else if proto == gosnmp.NoPriv {
		s.SNMPv3PrivPassphrase = ""
	}

	if n, err := db.Count(s).Where("id<>? AND owner=?", s.ID, s.Owner).Run(); err != nil {
		return fmt.Errorf("failed to check existing SNMP server config: %w", err)
	} else if n > 0 {
		return database.NewValidationError("this agent already has an SNMP server config")
	}

	return nil
}

type MonitorConfig struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Name  string `xorm:"name"`
	Owner string `xorm:"owner"`

	Version    string `xorm:"snmp_version"`
	UDPAddress string `xorm:"udp_address"`
	Community  string `xorm:"community"`
	UseInforms bool   `xorm:"use_informs"`

	// SNMPv3 settings
	ContextName     string `xorm:"snmp_v3_context_name"`
	ContextEngineID string `xorm:"snmp_v3_context_engine_id"`
	SNMPv3Security  string `xorm:"snmp_v3_security"`

	AuthEngineID   string              `xorm:"snmp_v3_auth_engine_id"`
	AuthUsername   string              `xorm:"snmp_v3_auth_username"`
	AuthProtocol   string              `xorm:"snmp_v3_auth_protocol"`
	AuthPassphrase database.SecretText `xorm:"snmp_v3_auth_passphrase"`

	PrivProtocol   string              `xorm:"snmp_v3_priv_protocol"`
	PrivPassphrase database.SecretText `xorm:"snmp_v3_priv_passphrase"`
}

func (m *MonitorConfig) GetID() int64      { return m.ID }
func (*MonitorConfig) TableName() string   { return "snmp_monitors" }
func (*MonitorConfig) Appellation() string { return "SNMP monitor" }

func (m *MonitorConfig) BeforeWrite(db database.Access) error {
	m.Owner = conf.GlobalConfig.GatewayName

	if m.Name == "" {
		return database.NewValidationError("missing SNMP monitor name")
	}

	if vers, err := toSNMPVersion(m.Version); err != nil {
		return database.WrapAsValidationError(err)
	} else if errV3 := m.checkV3Settings(vers); errV3 != nil {
		return errV3
	}

	if _, err := net.ResolveUDPAddr("udp", m.UDPAddress); err != nil {
		return database.NewValidationError("invalid UDP address %q: %w", m.UDPAddress, err)
	}

	if m.Community == "" && m.Version != Version3 {
		m.Community = "public"
	}

	if n, err := db.Count(m).Where("id<>? AND name=? AND owner=?", m.ID, m.Name,
		m.Owner).Run(); err != nil {
		return fmt.Errorf("failed to check existing SNMP monitors: %w", err)
	} else if n > 0 {
		return database.NewValidationError("an SNMP monitor named %q already exists", m.Name)
	}

	return nil
}

func (m *MonitorConfig) checkV3Settings(version gosnmp.SnmpVersion) error {
	if version != gosnmp.Version3 {
		// If not SNMPv3, all v3 settings are not applicable
		m.ContextEngineID = ""
		m.ContextName = ""
		m.SNMPv3Security = ""
		m.AuthUsername = ""
	} else {
		if m.AuthUsername == "" {
			return database.NewValidationError("missing authentication username " +
				"(a username is required for SNMPv3 even if no authentication is used)")
		}

		if m.SNMPv3Security == "" {
			m.SNMPv3Security = V3SecurityNoAuthNoPriv
		}
	}

	if v3Flags := getSNMPv3MsgFlags(m.SNMPv3Security); v3Flags == unknownV3MsgFlag {
		return database.NewValidationError("invalid SNMPv3 message flags %q", m.SNMPv3Security)
	}

	if m.SNMPv3Security == V3SecurityAuthPriv {
		privProtocol := getPrivProtocol(m.PrivProtocol)
		if privProtocol == 0 {
			return database.NewValidationError("invalid privacy protocol %q", m.PrivProtocol)
		}

		if privProtocol != gosnmp.NoPriv && m.PrivPassphrase == "" {
			return database.NewValidationError("missing privacy passphrase")
		}
	} else {
		m.PrivProtocol = ""
		m.PrivPassphrase = ""
	}

	if m.SNMPv3Security == V3SecurityAuthNoPriv || m.SNMPv3Security == V3SecurityAuthPriv {
		if !m.UseInforms && m.AuthEngineID == "" {
			return database.NewValidationError("missing authentication engine ID")
		}

		authProtocol := getAuthProtocol(m.AuthProtocol)
		if authProtocol == 0 {
			return database.NewValidationError("invalid authentication protocol %q", m.AuthProtocol)
		}

		if authProtocol != gosnmp.NoAuth && m.AuthPassphrase == "" {
			return database.NewValidationError("missing authentication passphrase")
		}
	} else {
		m.AuthEngineID = ""
		m.AuthProtocol = ""
		m.AuthPassphrase = ""
	}

	return nil
}

func (m *MonitorConfig) snmpV3MsgFlags() gosnmp.SnmpV3MsgFlags {
	return getSNMPv3MsgFlags(m.SNMPv3Security)
}

package backup

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
)

func exportSNMPConfig(logger *log.Logger, db database.ReadAccess) (*file.SNMPConfig, error) {
	var serverConfig snmp.ServerConfig
	if err := db.Get(&serverConfig, "owner=?", conf.GlobalConfig.GatewayName).
		Run(); err != nil && !database.IsNotFound(err) {
		return nil, fmt.Errorf("failed to retrieve SNMP server config: %w", err)
	}

	var monitors model.Slice[*snmp.MonitorConfig]
	if err := db.Select(&monitors).Owner().Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve SNMP monitors: %w", err)
	}

	config := &file.SNMPConfig{}

	if serverConfig.ID != 0 {
		logger.Info("Exported SNMP server config")
		config.Server = &file.SNMPServer{
			LocalUDPAddress:     serverConfig.LocalUDPAddress,
			Community:           serverConfig.Community,
			V3Only:              serverConfig.SNMPv3Only,
			V3Username:          serverConfig.SNMPv3Username,
			V3AuthProtocol:      serverConfig.SNMPv3AuthProtocol,
			V3AuthPassphrase:    serverConfig.SNMPv3AuthPassphrase.String(),
			V3PrivacyProtocol:   serverConfig.SNMPv3PrivProtocol,
			V3PrivacyPassphrase: serverConfig.SNMPv3PrivPassphrase.String(),
		}
	}

	if len(monitors) > 0 {
		config.Monitors = make([]*file.SNMPMonitor, len(monitors))
	}

	for i, monitor := range monitors {
		config.Monitors[i] = &file.SNMPMonitor{
			Name:                monitor.Name,
			SNMPVersion:         monitor.Version,
			UDPAddress:          monitor.UDPAddress,
			Community:           monitor.Community,
			UseInforms:          monitor.UseInforms,
			V3ContextName:       monitor.ContextName,
			V3ContextEngineID:   monitor.ContextEngineID,
			V3Security:          monitor.SNMPv3Security,
			V3AuthEngineID:      monitor.AuthEngineID,
			V3AuthUsername:      monitor.AuthUsername,
			V3AuthProtocol:      monitor.AuthProtocol,
			V3AuthPassphrase:    monitor.AuthPassphrase.String(),
			V3PrivacyProtocol:   monitor.PrivProtocol,
			V3PrivacyPassphrase: monitor.PrivPassphrase.String(),
		}
	}

	return config, nil
}

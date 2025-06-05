package backup

import (
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
)

func importSNMPConfig(logger *log.Logger, db database.Access, config *file.SNMPConfig,
	reset bool,
) error {
	if reset {
		if err := db.DeleteAll(&snmp.MonitorConfig{}).Owner().Run(); err != nil {
			return fmt.Errorf("failed to delete old SNMP monitor configs: %w", err)
		}

		if err := db.DeleteAll(&snmp.ServerConfig{}).Owner().Run(); err != nil {
			return fmt.Errorf("failed to delete old SNMP server configs: %w", err)
		}
	}

	if config == nil {
		return nil
	}

	if err := importSNMPServer(logger, db, config.Server); err != nil {
		return err
	}

	return importSNMPMonitors(logger, db, config.Monitors)
}

func importSNMPServer(logger *log.Logger, db database.Access, server *file.SNMPServer) error {
	if server == nil {
		return nil
	}

	var dbServer snmp.ServerConfig
	if err := db.Get(&dbServer, "owner=?", conf.GlobalConfig.GatewayName).
		Run(); err != nil && !database.IsNotFound(err) {
		return fmt.Errorf("failed to retrieve SNMP server config: %w", err)
	}

	dbServer.LocalUDPAddress = server.LocalUDPAddress
	dbServer.Community = server.Community
	dbServer.SNMPv3Only = server.V3Only
	dbServer.SNMPv3Username = server.V3Username
	dbServer.SNMPv3AuthProtocol = server.V3AuthProtocol
	dbServer.SNMPv3AuthPassphrase = database.SecretText(server.V3AuthPassphrase)
	dbServer.SNMPv3PrivProtocol = server.V3PrivacyProtocol
	dbServer.SNMPv3PrivPassphrase = database.SecretText(server.V3PrivacyPassphrase)

	var dbErr error

	if dbServer.ID == 0 {
		logger.Info("Import new SNMP server config")
		dbErr = db.Insert(&dbServer).Run()
	} else {
		logger.Info("Update existing SNMP server config")
		dbErr = db.Update(&dbServer).Run()
	}

	if dbErr != nil {
		return fmt.Errorf("failed to import SNMP server config: %w", dbErr)
	}

	return nil
}

func importSNMPMonitors(logger *log.Logger, db database.Access, monitors []*file.SNMPMonitor) error {
	for _, monitor := range monitors {
		var dbMonitor snmp.MonitorConfig
		if err := db.Get(&dbMonitor, "name=?", monitor.Name).Owner().
			Run(); err != nil && !database.IsNotFound(err) {
			return fmt.Errorf("failed to retrieve SNMP monitor %q: %w", monitor.Name, err)
		}

		dbMonitor.Name = monitor.Name
		dbMonitor.Version = monitor.SNMPVersion
		dbMonitor.UDPAddress = monitor.UDPAddress
		dbMonitor.Community = monitor.Community
		dbMonitor.UseInforms = monitor.UseInforms
		dbMonitor.ContextName = monitor.V3ContextName
		dbMonitor.ContextEngineID = monitor.V3ContextEngineID
		dbMonitor.SNMPv3Security = monitor.V3Security
		dbMonitor.AuthEngineID = monitor.V3AuthEngineID
		dbMonitor.AuthUsername = monitor.V3AuthUsername
		dbMonitor.AuthProtocol = monitor.V3AuthProtocol
		dbMonitor.AuthPassphrase = database.SecretText(monitor.V3AuthPassphrase)
		dbMonitor.PrivProtocol = monitor.V3PrivacyProtocol
		dbMonitor.PrivPassphrase = database.SecretText(monitor.V3PrivacyPassphrase)

		var dbErr error

		if dbMonitor.ID == 0 {
			logger.Infof("Import new SNMP monitor %q", dbMonitor.Name)
			dbErr = db.Insert(&dbMonitor).Run()
		} else {
			logger.Infof("Update existing SNMP monitor %q", dbMonitor.Name)
			dbErr = db.Update(&dbMonitor).Run()
		}

		if dbErr != nil {
			return fmt.Errorf("failed to import SNMP monitor %q: %w", dbMonitor.Name, dbErr)
		}
	}

	return nil
}

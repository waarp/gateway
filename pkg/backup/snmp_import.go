package backup

import (
	"context"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
)

func importSNMPConfig(logger *log.Logger, db database.Access, config *file.SNMPConfig,
	reset, restart bool,
) error {
	if reset {
		if err := db.DeleteAll(&snmp.MonitorConfig{}).Run(); err != nil {
			return fmt.Errorf("failed to delete old SNMP monitor configs: %w", err)
		}

		if err := db.DeleteAll(&snmp.ServerConfig{}).Run(); err != nil {
			return fmt.Errorf("failed to delete old SNMP server configs: %w", err)
		}
	}

	if config == nil {
		return nil
	}

	if restart && snmp.GlobalService == nil {
		snmp.GlobalService = &snmp.Service{DB: db.AsDB()}
		if err := snmp.GlobalService.Start(); err != nil {
			return fmt.Errorf("failed to start SNMP server config: %w", err)
		}
	}

	if err := importSNMPServer(logger, db, config.Server, restart); err != nil {
		return err
	}

	return importSNMPMonitors(logger, db, config.Monitors, restart)
}

func importSNMPServer(logger *log.Logger, db database.Access, server *file.SNMPServer,
	restart bool,
) error {
	if server == nil {
		return nil
	}

	var dbServer snmp.ServerConfig
	if err := db.Get(&dbServer, "").Run(); err != nil && !database.IsNotFound(err) {
		return fmt.Errorf("failed to retrieve SNMP server config: %w", err)
	}

	dbServer.LocalUDPAddress = server.LocalUDPAddress
	dbServer.Community = server.Community
	dbServer.SNMPv3Only = server.V3Only
	dbServer.SNMPv3Username = server.V3Username
	dbServer.SNMPv3AuthProtocol = server.V3AuthProtocol
	dbServer.SNMPv3AuthPassphrase = server.V3AuthPassphrase
	dbServer.SNMPv3PrivProtocol = server.V3PrivacyProtocol
	dbServer.SNMPv3PrivPassphrase = server.V3PrivacyPassphrase

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

	if restart {
		if err := snmp.GlobalService.ReloadServerConf(context.Background()); err != nil {
			return fmt.Errorf("failed to reload SNMP server config: %w", err)
		}
	}

	return nil
}

func importSNMPMonitors(logger *log.Logger, db database.Access, monitors []*file.SNMPMonitor,
	restart bool,
) error {
	for _, monitor := range monitors {
		var dbMonitor snmp.MonitorConfig
		if err := db.Get(&dbMonitor, "name=?", monitor.Name).
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
		dbMonitor.AuthPassphrase = monitor.V3AuthPassphrase
		dbMonitor.PrivProtocol = monitor.V3PrivacyProtocol
		dbMonitor.PrivPassphrase = monitor.V3PrivacyPassphrase

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

		if restart {
			if err := snmp.GlobalService.ReloadMonitorsConf(); err != nil {
				return fmt.Errorf("failed to reload SNMP server config: %w", err)
			}
		}
	}

	return nil
}

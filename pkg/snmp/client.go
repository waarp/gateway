package snmp

import (
	"errors"
	"fmt"
	"math/big"
	"slices"
	"time"

	"github.com/gosnmp/gosnmp"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:gochecknoglobals //made vars to speed up tests
var (
	trapsTimeout = 5 * time.Second
	trapsRetries = 3
)

func connect(logger *log.Logger, config *MonitorConfig) (*gosnmp.GoSNMP, error) {
	host, port, addrErr := utils.SplitHostPort[uint16](config.UDPAddress)
	if addrErr != nil {
		return nil, fmt.Errorf("failed to parse the target address %q: %w", config.UDPAddress, addrErr)
	}

	version, vErr := toSNMPVersion(config.Version)
	if vErr != nil {
		return nil, vErr
	}

	snmpLogger := clientLog(logger)

	client := &gosnmp.GoSNMP{
		Version:   version,
		Target:    host,
		Port:      port,
		Community: config.Community,
		Transport: "udp",
		Logger:    snmpLogger,
		Timeout:   trapsTimeout,
		Retries:   trapsRetries,
	}

	if version == gosnmp.Version3 {
		if config.AuthEngineID != "" {
			var authEngineIDInt big.Int
			if _, ok := authEngineIDInt.SetString(config.AuthEngineID, 0); ok {
				config.AuthEngineID = string(authEngineIDInt.Bytes())
			}
		}

		client.MsgFlags = config.snmpV3MsgFlags()
		client.ContextName = config.ContextName
		client.ContextEngineID = config.ContextEngineID
		client.SecurityModel = gosnmp.UserSecurityModel
		client.SecurityParameters = &gosnmp.UsmSecurityParameters{
			AuthoritativeEngineID:    config.AuthEngineID,
			UserName:                 config.AuthUsername,
			AuthenticationProtocol:   getAuthProtocol(config.AuthProtocol),
			PrivacyProtocol:          getPrivProtocol(config.PrivProtocol),
			AuthenticationPassphrase: config.AuthPassphrase,
			PrivacyPassphrase:        config.PrivPassphrase,
			Logger:                   snmpLogger,
		}
	}

	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to %q: %w", config.UDPAddress, err)
	}

	return client, nil
}

func (s *Service) sendTrap(variables []gosnmp.SnmpPDU) error {
	s.monConfLock.RLock()
	defer s.monConfLock.RUnlock()

	type connCtx struct {
		isInform bool
		client   *gosnmp.GoSNMP
	}

	connCtxs := make([]*connCtx, 0, len(s.monitors))

	var errs []error

	for _, monitor := range s.monitors {
		client, err := connect(s.Logger, monitor)
		if err != nil {
			s.Logger.Errorf("Failed to connect to SNMP monitor %q at %q: %v",
				monitor.Name, monitor.UDPAddress, err)

			errs = append(errs, fmt.Errorf("failed to connect to SNMP monitor %q at %q: %w",
				monitor.Name, monitor.UDPAddress, err))
		} else {
			connCtxs = append(connCtxs, &connCtx{
				isInform: monitor.UseInforms,
				client:   client,
			})
		}
	}

	defer func() {
		for _, ctx := range connCtxs {
			if err := ctx.client.Conn.Close(); err != nil {
				s.Logger.Warningf("Failed to close SNMP connection to %q: %v",
					ctx.client.Conn.RemoteAddr().String(), err)
			}
		}
	}()

	for _, ctx := range connCtxs {
		trap := gosnmp.SnmpTrap{Variables: variables, IsInform: ctx.isInform}
		if _, err := ctx.client.SendTrap(trap); err != nil {
			s.Logger.Errorf("Failed to send SNMP trap: %v", err)

			errs = append(errs, fmt.Errorf("failed to send SNMP trap: %w", err))
		}
	}

	return errors.Join(errs...)
}

//nolint:funlen //function is fine as is
func (s *Service) sendTransferError(trans *model.NormalizedTransferView) error {
	variables := []gosnmp.SnmpPDU{
		{
			Value: s.sysUpTime(),
			Name:  SNMPSysUpTimeOID,
			Type:  gosnmp.TimeTicks,
		}, {
			Value: TransferErrorNotifOID,
			Name:  SNMPTrapOID,
			Type:  gosnmp.ObjectIdentifier,
		}, {
			Value: s.DB.Config.GatewayName,
			Name:  InstanceNameOID,
			Type:  gosnmp.OctetString,
		}, {
			Value: utils.FormatInt(trans.ID),
			Name:  TeObjectTransOID,
			Type:  gosnmp.OctetString,
		}, {
			Value: trans.RemoteTransferID,
			Name:  TeObjectRemoteOID,
			Type:  gosnmp.OctetString,
		}, {
			Value: trans.Rule,
			Name:  TeObjectRuleNameOID,
			Type:  gosnmp.OctetString,
		}, {
			Value: trans.Account,
			Name:  TeObjectRequesterOID,
			Type:  gosnmp.OctetString,
		}, {
			Value: trans.Agent,
			Name:  TeObjectRequestedOID,
			Type:  gosnmp.OctetString,
		}, {
			Value: trans.LocalPath,
			Name:  TeObjectFilenameOID,
			Type:  gosnmp.OctetString,
		}, {
			Value: time.Now().Local().Format(time.RFC3339Nano),
			Name:  TeObjectDateOID,
			Type:  gosnmp.OctetString,
		}, {
			Value: trans.ErrCode.String(),
			Name:  TeObjectErrCodeOID,
			Type:  gosnmp.OctetString,
		}, {
			Value: trans.ErrDetails,
			Name:  TeObjectErrDetailsOID,
			Type:  gosnmp.OctetString,
		},
	}

	if trans.Client != "" {
		//nolint:mnd //magic number needed here
		variables = slices.Insert(variables, 5, gosnmp.SnmpPDU{
			Value: trans.Client,
			Name:  TeObjectClientNameOID,
			Type:  gosnmp.OctetString,
		})
	}

	return s.sendTrap(variables)
}

func (s *Service) sendServiceError(service string, sErr error) error {
	variables := []gosnmp.SnmpPDU{
		{
			Value: s.sysUpTime(),
			Name:  SNMPSysUpTimeOID,
			Type:  gosnmp.TimeTicks,
		}, {
			Value: ServiceErrorNotifOID,
			Name:  SNMPTrapOID,
			Type:  gosnmp.ObjectIdentifier,
		}, {
			Value: s.DB.Config.GatewayName,
			Name:  InstanceNameOID,
			Type:  gosnmp.OctetString,
		}, {
			Value: service,
			Name:  SeObjectNameOID,
			Type:  gosnmp.OctetString,
		}, {
			Value: sErr.Error(),
			Name:  SeObjectErrorOID,
			Type:  gosnmp.OctetString,
		},
	}

	return s.sendTrap(variables)
}

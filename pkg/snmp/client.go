package snmp

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"code.waarp.fr/lib/log"
	"github.com/gosnmp/gosnmp"
	"golang.org/x/exp/slices"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	trapsTimeout = 5 * time.Second
	trapsRetries = 3
)

func connect(logger *log.Logger, conf *MonitorConfig) (*gosnmp.GoSNMP, error) {
	host, port, addrErr := utils.SplitHostPort(conf.UDPAddress)
	if addrErr != nil {
		return nil, fmt.Errorf("failed to parse the target address %q: %w", conf.UDPAddress, addrErr)
	}

	version, vErr := toSNMPVersion(conf.Version)
	if vErr != nil {
		return nil, vErr
	}

	snmpLogger := clientLog(logger)

	client := &gosnmp.GoSNMP{
		Version:   version,
		Target:    host,
		Port:      port,
		Community: conf.Community,
		Transport: "udp",
		Logger:    snmpLogger,
		Timeout:   trapsTimeout,
		Retries:   trapsRetries,
	}

	if version == gosnmp.Version3 {
		if conf.AuthEngineID != "" {
			var authEngineIDInt big.Int
			if _, ok := authEngineIDInt.SetString(conf.AuthEngineID, 0); ok {
				conf.AuthEngineID = string(authEngineIDInt.Bytes())
			}
		}

		client.MsgFlags = conf.snmpV3MsgFlags()
		client.ContextName = conf.ContextName
		client.ContextEngineID = conf.ContextEngineID
		client.SecurityModel = gosnmp.UserSecurityModel
		client.SecurityParameters = &gosnmp.UsmSecurityParameters{
			AuthoritativeEngineID:    conf.AuthEngineID,
			UserName:                 conf.AuthUsername,
			AuthenticationProtocol:   getAuthProtocol(conf.AuthProtocol),
			PrivacyProtocol:          getPrivProtocol(conf.PrivProtocol),
			AuthenticationPassphrase: string(conf.AuthPassphrase),
			PrivacyPassphrase:        string(conf.PrivPassphrase),
			Logger:                   snmpLogger,
		}
	}

	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to %q: %w", conf.UDPAddress, err)
	}

	return client, nil
}

func (s *Service) sendTrap(trap *gosnmp.SnmpTrap) error {
	s.monConfLock.RLock()
	defer s.monConfLock.RUnlock()

	clients := make([]*gosnmp.GoSNMP, 0, len(s.monitors))

	for _, monitor := range s.monitors {
		client, err := connect(s.Logger, monitor)
		if err != nil {
			s.Logger.Error("Failed to connect to SNMP monitor %q at %q: %v",
				monitor.Name, monitor.UDPAddress, err)
		}

		clients = append(clients, client)
	}

	defer func() {
		for _, client := range clients {
			if err := client.Conn.Close(); err != nil {
				s.Logger.Error("Failed to close SNMP connection to %q: %v",
					client.Conn.RemoteAddr().String(), err)
			}
		}
	}()

	var errs []error

	for _, client := range clients {
		if _, err := client.SendTrap(*trap); err != nil {
			s.Logger.Error("Failed to send SNMP trap: %v", err)

			return fmt.Errorf("failed to send SNMP trap: %w", err)
		}
	}

	return errors.Join(errs...)
}

//nolint:funlen //function is fine as is
func (s *Service) sendTransferError(trans *model.NormalizedTransferView) error {
	trap := gosnmp.SnmpTrap{
		Variables: []gosnmp.SnmpPDU{
			{
				Value: s.sysUpTime(),
				Name:  SNMPSysUpTimeOID,
				Type:  gosnmp.TimeTicks,
			}, {
				Value: TransferErrorNotifOID,
				Name:  SNMPTrapOID,
				Type:  gosnmp.ObjectIdentifier,
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
				Value: trans.LocalPath.String(),
				Name:  TeObjectFilenameOID,
				Type:  gosnmp.OctetString,
			}, {
				Value: trans.Stop.Format(time.RFC3339Nano),
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
		},
		IsInform: false,
	}

	if trans.Client != "" {
		//nolint:mnd //magic number needed here
		trap.Variables = slices.Insert(trap.Variables, 5, gosnmp.SnmpPDU{
			Value: trans.Client,
			Name:  TeObjectClientNameOID,
			Type:  gosnmp.OctetString,
		})
	}

	return s.sendTrap(&trap)
}

func (s *Service) sendServiceError(service string, sErr error) error {
	trap := gosnmp.SnmpTrap{
		Variables: []gosnmp.SnmpPDU{
			{
				Value: s.sysUpTime(),
				Name:  SNMPSysUpTimeOID,
				Type:  gosnmp.TimeTicks,
			}, {
				Value: ServiceErrorNotifOID,
				Name:  SNMPTrapOID,
				Type:  gosnmp.ObjectIdentifier,
			}, {
				Value: service,
				Name:  SeObjectNameOID,
				Type:  gosnmp.OctetString,
			}, {
				Value: sErr.Error(),
				Name:  SeObjectErrorOID,
				Type:  gosnmp.OctetString,
			},
		},
		IsInform: false,
	}

	return s.sendTrap(&trap)
}

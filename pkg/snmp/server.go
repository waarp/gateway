package snmp

import (
	"fmt"

	"github.com/gosnmp/gosnmp"
	snmplib "github.com/slayercat/GoSNMPServer"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func (s *Service) listen(conf *ServerConfig) error {
	handler := snmplib.MasterAgent{
		SecurityConfig: snmplib.SecurityConfig{
			SnmpV3Only:               conf.SNMPv3Only,
			AuthoritativeEngineBoots: 1,
			Users: []gosnmp.UsmSecurityParameters{{
				UserName:                 conf.SNMPv3Username,
				AuthenticationProtocol:   getAuthProtocol(conf.SNMPv3AuthProtocol),
				PrivacyProtocol:          getPrivProtocol(conf.SNMPv3PrivProtocol),
				AuthenticationPassphrase: string(conf.SNMPv3AuthPassphrase),
				PrivacyPassphrase:        string(conf.SNMPv3PrivPassphrase),
			}},
		},
		SubAgents: []*snmplib.SubAgent{{
			CommunityIDs: []string{conf.Community},
			OIDs:         getServerPDUs(),
			Logger:       serverLog(s.Logger),
		}},
		Logger: serverLog(s.Logger),
	}

	s.server = snmplib.NewSNMPServer(handler)

	if err := s.server.ListenUDP("udp", conf.LocalUDPAddress); err != nil {
		return fmt.Errorf("cannot listen on %q: %w", conf.LocalUDPAddress, err)
	}

	go func() {
		if err := s.server.ServeForever(); err != nil {
			s.state.Set(utils.StateError, err.Error())
		}
	}()

	return nil
}

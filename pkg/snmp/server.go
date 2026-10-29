package snmp

import (
	"errors"
	"fmt"

	"github.com/gosnmp/gosnmp"
	snmplib "github.com/slayercat/GoSNMPServer"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var ErrNoAnalytics = errors.New("analytics service is not running")

func (s *Service) listen(conf *ServerConfig) error {
	handler := snmplib.MasterAgent{
		SecurityConfig: snmplib.SecurityConfig{
			SnmpV3Only:               conf.SNMPv3Only,
			AuthoritativeEngineBoots: 1,
			Users: []gosnmp.UsmSecurityParameters{{
				UserName:                 conf.SNMPv3Username,
				AuthenticationProtocol:   getAuthProtocol(conf.SNMPv3AuthProtocol),
				PrivacyProtocol:          getPrivProtocol(conf.SNMPv3PrivProtocol),
				AuthenticationPassphrase: conf.SNMPv3AuthPassphrase,
				PrivacyPassphrase:        conf.SNMPv3PrivPassphrase,
			}},
		},
		SubAgents: []*snmplib.SubAgent{{
			CommunityIDs: []string{conf.Community},
			OIDs:         getServerPDUs(),
			Logger:       serverLog(s.Logger),
		}},
		Logger: serverLog(s.Logger),
	}

	server := snmplib.NewSNMPServer(handler)
	if err := server.ListenUDP("udp", conf.LocalUDPAddress); err != nil {
		return fmt.Errorf("cannot listen on %q: %w", conf.LocalUDPAddress, err)
	}

	s.server.Store(server)

	s.serverWG.Go(func() {
		if err := server.ServeForever(); err != nil {
			if s.state.IsRunning() {
				s.state.Set(utils.StateError, err.Error())
			}
		}
	})

	return nil
}

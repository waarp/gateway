//go:build snmp_server

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
				UserName:                 conf.Username,
				AuthenticationProtocol:   getAuthProtocol(conf.AuthProtocol),
				PrivacyProtocol:          getPrivProtocol(conf.PrivProtocol),
				AuthenticationPassphrase: conf.AuthPassphrase,
				PrivacyPassphrase:        conf.PrivPassphrase,
			}},
		},
		SubAgents: []*snmplib.SubAgent{
			// TODO: implement this
		},
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

		s.server.Shutdown()
	}()

	return nil
}

package file

func (s *SNMPServer) IsZero() bool {
	return s.LocalUDPAddress == "" &&
		s.Community == "" &&
		!s.V3Only &&
		s.V3Username == "" &&
		s.V3AuthProtocol == "" &&
		s.V3AuthPassphrase == "" &&
		s.V3PrivacyProtocol == "" &&
		s.V3PrivacyPassphrase == ""
}

func (s SNMPConfig) IsZero() bool {
	return len(s.Monitors) == 0 && s.Server.IsZero()
}

func (e EmailConfig) IsZero() bool {
	return len(e.Credentials) == 0 && len(e.Templates) == 0
}

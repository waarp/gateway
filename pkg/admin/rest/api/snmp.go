package api

type GetSnmpMonitorRespObject struct {
	Name            string `json:"name" yaml:"name"`
	Version         string `json:"version" yaml:"version"`
	UDPAddress      string `json:"udpAddress" yaml:"udpAddress"`
	Community       string `json:"community,omitempty" yaml:"community,omitempty"`
	UseInforms      bool   `json:"useInforms" yaml:"useInforms"`
	ContextName     string `json:"contextName,omitempty" yaml:"contextName,omitempty"`
	ContextEngineID string `json:"contextEngineID,omitempty" yaml:"contextEngineID,omitempty"`
	SNMPv3Security  string `json:"snmpv3Security,omitempty" yaml:"snmpv3Security,omitempty"` //nolint:tagliatelle //SNMP
	AuthEngineID    string `json:"authEngineID,omitempty" yaml:"authEngineID,omitempty"`
	AuthUsername    string `json:"authUsername,omitempty" yaml:"authUsername,omitempty"`
	AuthProtocol    string `json:"authProtocol,omitempty" yaml:"authProtocol,omitempty"`
	AuthPassphrase  string `json:"authPassphrase,omitempty" yaml:"authPassphrase,omitempty"`
	PrivProtocol    string `json:"privProtocol,omitempty" yaml:"privProtocol,omitempty"`
	PrivPassphrase  string `json:"privPassphrase,omitempty" yaml:"privPassphrase,omitempty"`
}

type PostSnmpMonitorReqObject struct {
	Name            string `json:"name" yaml:"name"`
	Version         string `json:"version" yaml:"version"`
	UDPAddress      string `json:"udpAddress" yaml:"udpAddress"`
	Community       string `json:"community,omitempty" yaml:"community,omitempty"`
	UseInforms      bool   `json:"useInforms" yaml:"useInforms"`
	ContextName     string `json:"contextName,omitempty" yaml:"contextName,omitempty"`
	ContextEngineID string `json:"contextEngineID,omitempty" yaml:"contextEngineID,omitempty"`
	SNMPv3Security  string `json:"snmpv3Security,omitempty" yaml:"snmpv3Security,omitempty"` //nolint:tagliatelle //SNMP
	AuthEngineID    string `json:"authEngineID,omitempty" yaml:"authEngineID,omitempty"`
	AuthUsername    string `json:"authUsername,omitempty" yaml:"authUsername,omitempty"`
	AuthProtocol    string `json:"authProtocol,omitempty" yaml:"authProtocol,omitempty"`
	AuthPassphrase  string `json:"authPassphrase,omitempty" yaml:"authPassphrase,omitempty"`
	PrivProtocol    string `json:"privProtocol,omitempty" yaml:"privProtocol,omitempty"`
	PrivPassphrase  string `json:"privPassphrase,omitempty" yaml:"privPassphrase,omitempty"`
}

type PatchSnmpMonitorReqObject struct {
	Name            Nullable[string] `json:"name" yaml:"name"`
	Version         Nullable[string] `json:"version" yaml:"version"`
	UDPAddress      Nullable[string] `json:"udpAddress" yaml:"udpAddress"`
	Community       Nullable[string] `json:"community,omitzero" yaml:"community,omitempty"`
	UseInforms      Nullable[bool]   `json:"useInforms" yaml:"useInforms"`
	ContextName     Nullable[string] `json:"contextName,omitzero" yaml:"contextName,omitempty"`
	ContextEngineID Nullable[string] `json:"contextEngineID,omitzero" yaml:"contextEngineID,omitempty"`
	SNMPv3Security  Nullable[string] `json:"snmpv3Security,omitzero" yaml:"snmpv3Security,omitempty"` //nolint:tagliatelle,lll //SNMP
	AuthEngineID    Nullable[string] `json:"authEngineID,omitzero" yaml:"authEngineID,omitempty"`
	AuthUsername    Nullable[string] `json:"authUsername,omitzero" yaml:"authUsername,omitempty"`
	AuthProtocol    Nullable[string] `json:"authProtocol,omitzero" yaml:"authProtocol,omitempty"`
	AuthPassphrase  Nullable[string] `json:"authPassphrase,omitzero" yaml:"authPassphrase,omitempty"`
	PrivProtocol    Nullable[string] `json:"privProtocol,omitzero" yaml:"privProtocol,omitempty"`
	PrivPassphrase  Nullable[string] `json:"privPassphrase,omitzero" yaml:"privPassphrase,omitempty"`
}

type GetSnmpServiceRespObject struct {
	LocalUDPAddress  string `json:"localUDPAddress" yaml:"localUDPAddress"`
	Community        string `json:"community,omitempty" yaml:"community,omitempty"`
	V3Only           bool   `json:"v3Only" yaml:"v3Only"`
	V3Username       string `json:"v3Username,omitempty" yaml:"v3Username,omitempty"`
	V3AuthProtocol   string `json:"v3AuthProtocol,omitempty" yaml:"v3AuthProtocol,omitempty"`
	V3AuthPassphrase string `json:"v3AuthPassphrase,omitempty" yaml:"v3AuthPassphrase,omitempty"`
	V3PrivProtocol   string `json:"v3PrivProtocol,omitempty" yaml:"v3PrivProtocol,omitempty"`
	V3PrivPassphrase string `json:"v3PrivPassphrase,omitempty" yaml:"v3PrivPassphrase,omitempty"`
}

type PostSnmpServiceReqObject struct {
	LocalUDPAddress  string `json:"localUDPAddress" yaml:"localUDPAddress"`
	Community        string `json:"community,omitempty" yaml:"community,omitempty"`
	V3Only           bool   `json:"v3Only" yaml:"v3Only"`
	V3Username       string `json:"v3Username,omitempty" yaml:"v3Username,omitempty"`
	V3AuthProtocol   string `json:"v3AuthProtocol,omitempty" yaml:"v3AuthProtocol,omitempty"`
	V3AuthPassphrase string `json:"v3AuthPassphrase,omitempty" yaml:"v3AuthPassphrase,omitempty"`
	V3PrivProtocol   string `json:"v3PrivProtocol,omitempty" yaml:"v3PrivProtocol,omitempty"`
	V3PrivPassphrase string `json:"v3PrivPassphrase,omitempty" yaml:"v3PrivPassphrase,omitempty"`
}

type PatchSnmpServiceReqObject struct {
	LocalUDPAddress  Nullable[string] `json:"localUDPAddress" yaml:"localUDPAddress"`
	Community        Nullable[string] `json:"community,omitzero" yaml:"community,omitempty"`
	V3Only           Nullable[bool]   `json:"v3Only" yaml:"v3Only"`
	V3Username       Nullable[string] `json:"v3Username,omitzero" yaml:"v3Username,omitempty"`
	V3AuthProtocol   Nullable[string] `json:"v3AuthProtocol,omitzero" yaml:"v3AuthProtocol,omitempty"`
	V3AuthPassphrase Nullable[string] `json:"v3AuthPassphrase,omitzero" yaml:"v3AuthPassphrase,omitempty"`
	V3PrivProtocol   Nullable[string] `json:"v3PrivProtocol,omitzero" yaml:"v3PrivProtocol,omitempty"`
	V3PrivPassphrase Nullable[string] `json:"v3PrivPassphrase,omitzero" yaml:"v3PrivPassphrase,omitempty"`
}

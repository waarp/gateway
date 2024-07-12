package api

type GetSnmpMonitorRespObject struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	UDPAddress      string `json:"udpAddress"`
	Community       string `json:"community,omitempty"`
	UseInforms      bool   `json:"useInforms"`
	ContextName     string `json:"contextName,omitempty"`
	ContextEngineID string `json:"contextEngineID,omitempty"`
	SNMPv3Security  string `json:"snmpv3Security,omitempty"`
	AuthEngineID    string `json:"authEngineID,omitempty"`
	AuthUsername    string `json:"authUsername,omitempty"`
	AuthProtocol    string `json:"authProtocol,omitempty"`
	AuthPassphrase  string `json:"authPassphrase,omitempty"`
	PrivProtocol    string `json:"privProtocol,omitempty"`
	PrivPassphrase  string `json:"privPassphrase,omitempty"`
}

type PostSnmpMonitorReqObject struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	UDPAddress      string `json:"udpAddress"`
	Community       string `json:"community,omitempty"`
	UseInforms      bool   `json:"useInforms"`
	ContextName     string `json:"contextName,omitempty"`
	ContextEngineID string `json:"contextEngineID,omitempty"`
	SNMPv3Security  string `json:"snmpv3Security,omitempty"`
	AuthEngineID    string `json:"authEngineID,omitempty"`
	AuthUsername    string `json:"authUsername,omitempty"`
	AuthProtocol    string `json:"authProtocol,omitempty"`
	AuthPassphrase  string `json:"authPassphrase,omitempty"`
	PrivProtocol    string `json:"privProtocol,omitempty"`
	PrivPassphrase  string `json:"privPassphrase,omitempty"`
}

type PatchSnmpMonitorReqObject struct {
	Name            Nullable[string] `json:"name"`
	Version         Nullable[string] `json:"version"`
	UDPAddress      Nullable[string] `json:"udpAddress"`
	Community       Nullable[string] `json:"community,omitempty"`
	UseInforms      Nullable[bool]   `json:"useInforms"`
	ContextName     Nullable[string] `json:"contextName,omitempty"`
	ContextEngineID Nullable[string] `json:"contextEngineID,omitempty"`
	SNMPv3Security  Nullable[string] `json:"snmpv3Security,omitempty"`
	AuthEngineID    Nullable[string] `json:"authEngineID,omitempty"`
	AuthUsername    Nullable[string] `json:"authUsername,omitempty"`
	AuthProtocol    Nullable[string] `json:"authProtocol,omitempty"`
	AuthPassphrase  Nullable[string] `json:"authPassphrase,omitempty"`
	PrivProtocol    Nullable[string] `json:"privProtocol,omitempty"`
	PrivPassphrase  Nullable[string] `json:"privPassphrase,omitempty"`
}

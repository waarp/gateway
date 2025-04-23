package wg

import (
	"fmt"
	"io"
	"net/http"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

func displaySnmpMonitor(w io.Writer, monitor *api.GetSnmpMonitorRespObject) {
	Style1.Printf(w, "SNMP monitor %q", monitor.Name)
	Style22.PrintL(w, "SNMP version", monitor.Version)
	Style22.PrintL(w, "UDP address", monitor.UDPAddress)
	Style22.Option(w, "Community", monitor.Community)
	Style22.PrintL(w, "Notification type", snmpNotifType(monitor.UseInforms))
	Style22.Option(w, "SNMPv3 security", monitor.SNMPv3Security)
	Style22.Option(w, "SNMPv3 context name", monitor.ContextName)
	Style22.Option(w, "SNMPv3 context engine ID", monitor.ContextEngineID)
	Style22.Option(w, "SNMPv3 auth engine ID", monitor.AuthEngineID)
	Style22.Option(w, "SNMPv3 username", monitor.AuthUsername)
	Style22.Option(w, "SNMPv3 authentication protocol", monitor.AuthProtocol)
	Style22.Option(w, "SNMPv3 authentication passphrase", monitor.AuthPassphrase)
	Style22.Option(w, "SNMPv3 privacy protocol", monitor.PrivProtocol)
	Style22.Option(w, "SNMPv3 privacy passphrase", monitor.PrivPassphrase)
}

func snmpNotifType(useInforms bool) string {
	return ifElse(useInforms, "INFORM", "TRAP")
}

//nolint:lll // tags can be long for flags
type SnmpMonitorAdd struct {
	Name            string `required:"yes" short:"n" long:"name" description:"The SNMP monitor's name" json:"name,omitempty"`
	UDPAddress      string `required:"yes" short:"a" long:"address" description:"The SNMP monitor's address" json:"udpAddress,omitempty"`
	Version         string `required:"yes" short:"v" long:"version" description:"The SNMP monitor's version" choice:"SNMPv2" choice:"SNMPv3" json:"version,omitempty"`
	NotifType       string `short:"t" long:"notif-type" choice:"trap" choice:"inform" default:"trap" description:"Specifies which type of notification should be sent to this monitor. Defaults to traps." json:"-"`
	Community       string `short:"c" long:"community" description:"The SNMP monitor's community string. Defaults to 'public'" json:"community,omitempty"`
	ContextName     string `long:"context-name" description:"The SNMPv3 context name." json:"contextName,omitempty"`
	ContextEngineID string `long:"context-engine-id" description:"The SNMPv3 context engine ID" json:"contextEngineID,omitempty"`
	SNMPv3Security  string `long:"snmpv3-sec" choice:"noAuthNoPriv" choice:"authNoPriv" choice:"authPriv" description:"The SNMPv3 security level" json:"snmpv3Security,omitempty"`
	AuthEngineID    string `long:"auth-engine-id" description:"The SNMPv3 authentication engine ID" json:"authEngineID,omitempty"`
	AuthUsername    string `long:"auth-username" description:"The SNMPv3 authentication username" json:"authUsername,omitempty"`
	AuthProtocol    string `long:"auth-protocol" description:"The SNMPv3 authentication protocol" choice:"MD5" choice:"SHA" choice:"SHA224" choice:"SHA256" choice:"SHA384" choice:"SHA512" json:"authProtocol,omitempty"`
	AuthPassphrase  string `long:"auth-passphrase" description:"The SNMPv3 authentication passphrase" json:"authPassphrase,omitempty"`
	PrivProtocol    string `long:"priv-protocol" description:"The SNMPv3 privacy protocol" choice:"DES" choice:"AES" choice:"AES192" choice:"AES192C" choice:"AES256" choice:"AES256C" json:"privProtocol,omitempty"`
	PrivPassphrase  string `long:"priv-passphrase" description:"The SNMPv3 privacy passphrase" json:"privPassphrase,omitempty"`

	UseInforms bool `json:"useInforms"`
}

func (s *SnmpMonitorAdd) Execute([]string) error { return s.execute(stdOutput) }
func (s *SnmpMonitorAdd) execute(w io.Writer) error {
	addr.Path = "/api/snmp/monitors"

	if s.NotifType == "inform" {
		s.UseInforms = true
	}

	if _, err := add(w, s); err != nil {
		return err
	}

	fmt.Fprintf(w, "The SNMP monitor %q was successfully added.\n", s.Name)

	return nil
}

//nolint:lll // tags can be long for flags
type SnmpMonitorList struct {
	ListOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" choice:"address+" choice:"address-" default:"name+" `
}

func (s *SnmpMonitorList) Execute([]string) error { return s.execute(stdOutput) }
func (s *SnmpMonitorList) execute(w io.Writer) error {
	addr.Path = "/api/snmp/monitors"

	listURL(&s.ListOptions, s.SortBy)

	respBody := map[string][]*api.GetSnmpMonitorRespObject{}
	if err := list(&respBody); err != nil {
		return err
	}

	if monitors := respBody["monitors"]; len(monitors) > 0 {
		Style0.Printf(w, "=== SNMP monitors ===")

		for _, monitor := range monitors {
			displaySnmpMonitor(w, monitor)
		}
	} else {
		fmt.Fprintln(w, "No SNMP monitor found.")
	}

	return nil
}

type SnmpMonitorGet struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The SNMP monitor's name"`
	} `positional-args:"yes"`
}

func (s *SnmpMonitorGet) Execute([]string) error { return s.execute(stdOutput) }
func (s *SnmpMonitorGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/snmp/monitors", s.Args.Name)

	var monitor api.GetSnmpMonitorRespObject
	if err := get(&monitor); err != nil {
		return err
	}

	displaySnmpMonitor(w, &monitor)

	return nil
}

//nolint:lll // tags can be long for flags
type SnmpMonitorUpdate struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The SNMP monitor's name"`
	} `positional-args:"yes" json:"-"`

	Name            *string `short:"n" long:"name" description:"The SNMP monitor's name" json:"name,omitempty"`
	UDPAddress      *string `short:"a" long:"address" description:"The SNMP monitor's address" json:"udpAddress,omitempty"`
	Version         *string `short:"v" long:"version" description:"The SNMP monitor's version" choice:"SNMPv2" choice:"SNMPv3" json:"version,omitempty"`
	NotifType       *string `long:"notif-type" choice:"trap" choice:"inform" default:"trap" description:"Specifies which type of notification should be sent to this monitor. Defaults to traps." json:"-"`
	Community       *string `short:"c" long:"community" description:"The SNMP monitor's community string." json:"community,omitempty"`
	ContextName     *string `long:"context-name" description:"The SNMPv3 context name." json:"contextName,omitempty"`
	ContextEngineID *string `long:"context-engine-id" description:"The SNMPv3 context engine ID" json:"contextEngineID,omitempty"`
	SNMPv3Security  *string `long:"snmpv3-sec" choice:"noAuthNoPriv" choice:"authNoPriv" choice:"authPriv" description:"The SNMPv3 security level" json:"snmpv3Security,omitempty"`
	AuthEngineID    *string `long:"auth-engine-id" description:"The SNMPv3 authentication engine ID" json:"authEngineID,omitempty"`
	AuthUsername    *string `long:"auth-username" description:"The SNMPv3 authentication username" json:"authUsername,omitempty"`
	AuthProtocol    *string `long:"auth-protocol" description:"The SNMPv3 authentication protocol" choice:"MD5" choice:"SHA" choice:"SHA224" choice:"SHA256" choice:"SHA384" choice:"SHA512" json:"authProtocol,omitempty"`
	AuthPassphrase  *string `long:"auth-passphrase" description:"The SNMPv3 authentication passphrase" json:"authPassphrase,omitempty"`
	PrivProtocol    *string `long:"priv-protocol" description:"The SNMPv3 privacy protocol" choice:"DES" choice:"AES" choice:"AES192" choice:"AES192C" choice:"AES256" choice:"AES256C" json:"privProtocol,omitempty"`
	PrivPassphrase  *string `long:"priv-passphrase" description:"The SNMPv3 privacy passphrase" json:"privPassphrase,omitempty"`

	UseInforms *bool `json:"useInforms,omitempty"`
}

func (s *SnmpMonitorUpdate) Execute([]string) error { return s.execute(stdOutput) }
func (s *SnmpMonitorUpdate) execute(w io.Writer) error {
	addr.Path = path.Join("/api/snmp/monitors", s.Args.Name)

	if s.NotifType != nil {
		s.UseInforms = new(bool)
		if *s.NotifType == "inform" {
			*s.UseInforms = true
		}
	}

	if err := update(w, s); err != nil {
		return err
	}

	displayName := s.Args.Name
	if s.Name != nil && *s.Name != "" {
		displayName = *s.Name
	}

	fmt.Fprintf(w, "The SNMP monitor %q was successfully updated.\n", displayName)

	return nil
}

type SnmpMonitorDelete struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The SNMP monitor's name"`
	} `positional-args:"yes"`
}

func (s *SnmpMonitorDelete) Execute([]string) error { return s.execute(stdOutput) }
func (s *SnmpMonitorDelete) execute(w io.Writer) error {
	addr.Path = path.Join("/api/snmp/monitors", s.Args.Name)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The SNMP monitor %q was successfully deleted.\n", s.Args.Name)

	return nil
}

func displaySnmpServerConfig(w io.Writer, monitor *api.GetSnmpServiceRespObject) {
	Style1.Printf(w, "SNMP server configuration")
	Style22.PrintL(w, "Local UDP address", monitor.LocalUDPAddress)
	Style22.Option(w, "Community", monitor.Community)
	Style22.PrintL(w, "Accepted SNMP versions", ifElse(monitor.V3Only, "SNMPv3",
		"SNMPv2c & SNMPv3"))
	Style22.Option(w, "SNMPv3 username", monitor.V3Username)
	Style22.Option(w, "SNMPv3 authentication protocol", monitor.V3AuthProtocol)
	Style22.Option(w, "SNMPv3 authentication passphrase", monitor.V3AuthPassphrase)
	Style22.Option(w, "SNMPv3 privacy protocol", monitor.V3PrivProtocol)
	Style22.Option(w, "SNMPv3 privacy passphrase", monitor.V3PrivPassphrase)
}

//nolint:lll //tags can be long for flags
type SnmpServerSet struct {
	LocalUDPAddress  string `short:"a" long:"udp-address" description:"The SNMP server's local UDP address" json:"localUDPAddress,omitempty"`
	Community        string `short:"c" long:"community" description:"The SNMP server's community string" default:"public" json:"community,omitempty"`
	V3Only           bool   `long:"v3-only" description:"Set the server to only accept SNMPv3" json:"v3Only"`
	V3Username       string `long:"auth-username" description:"The SNMPv3 authentication username" json:"v3Username,omitempty"`
	V3AuthProtocol   string `long:"auth-protocol" description:"The SNMPv3 authentication protocol" choice:"MD5" choice:"SHA" choice:"SHA224" choice:"SHA256" choice:"SHA384" choice:"SHA512" json:"v3AuthProtocol,omitempty"`
	V3AuthPassphrase string `long:"auth-passphrase" description:"The SNMPv3 authentication passphrase" json:"v3AuthPassphrase,omitempty"`
	V3PrivProtocol   string `long:"priv-protocol" description:"The SNMPv3 privacy protocol" choice:"DES" choice:"AES" choice:"AES192" choice:"AES192C" choice:"AES256" choice:"AES256C" json:"v3PrivProtocol,omitempty"`
	V3PrivPassphrase string `long:"priv-passphrase" description:"The SNMPv3 privacy passphrase" json:"v3PrivPassphrase,omitempty"`
}

func (s *SnmpServerSet) Execute([]string) error { return s.execute(stdOutput) }
func (s *SnmpServerSet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/snmp/server")

	if err := updateMethod(w, s, http.MethodPut); err != nil {
		return err
	}

	fmt.Fprintln(w, "The SNMP server config was successfully updated.")

	return nil
}

type SnmpServerGet struct{}

func (s *SnmpServerGet) Execute([]string) error { return s.execute(stdOutput) }
func (s *SnmpServerGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/snmp/server")

	var serverConf api.GetSnmpServiceRespObject
	if err := get(&serverConf); err != nil {
		return err
	}

	displaySnmpServerConfig(w, &serverConf)

	return nil
}

type SnmpServerDelete struct{}

func (s *SnmpServerDelete) Execute([]string) error { return s.execute(stdOutput) }
func (s *SnmpServerDelete) execute(w io.Writer) error {
	addr.Path = path.Join("/api/snmp/server")

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintln(w, "The SNMP server config was successfully deleted.")

	return nil
}

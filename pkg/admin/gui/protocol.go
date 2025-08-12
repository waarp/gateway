package gui

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

type Protocols struct {
	R66      string
	R66TLS   string
	SFTP     string
	HTTP     string
	HTTPS    string
	FTP      string
	FTPS     string
	PeSIT    string
	PeSITTLS string
}

func supportedProtocolInternal(protocol string) []string {
	supportedProtocolsInternal := map[string][]string{
		"r66":       {"password"},
		"r66-tls":   {"password", "trusted_tls_certificate", "r66_legacy_certificate"},
		"http":      {"password"},
		"https":     {"password", "trusted_tls_certificate"},
		"sftp":      {"password", "ssh_public_key"},
		"pesit":     {"password"},
		"pesit-tls": {"password", "trusted_tls_certificate"},
	}

	return supportedProtocolsInternal[protocol]
}

func supportedProtocolExternal(protocol string) []string {
	supportedProtocolsExternal := map[string][]string{
		"r66":       {"password"},
		"r66-tls":   {"password", "tls_certificate", "r66_legacy_certificate"},
		"http":      {"password"},
		"https":     {"password", "tls_certificate"},
		"sftp":      {"password", "ssh_private_key"},
		"pesit":     {"password", "pesit_pre-connection_auth"},
		"pesit-tls": {"tls_certificate", "pesit_pre-connection_auth"},
	}

	return supportedProtocolsExternal[protocol]
}

//nolint:gochecknoglobals // Constant
var (
	TLSVersions            = []string{protoutils.TLSv10, protoutils.TLSv11, protoutils.TLSv12, protoutils.TLSv13}
	CompatibilityModePeSIT = []string{pesit.CompatibilityModeStandard, pesit.CompatibilityModeNonStandard}
	TLSRequirement         = []string{string(ftp.TLSOptional), string(ftp.TLSMandatory), string(ftp.TLSImplicit)}
)

func protocolsFilter(r *http.Request, filter *FiltersPagination) (*FiltersPagination, []string) {
	var filterProtocol []string
	urlParams := r.URL.Query()

	if filter.Protocols.R66 = urlParams.Get("filterProtocolR66"); filter.Protocols.R66 == "true" {
		filterProtocol = append(filterProtocol, "r66")
	}

	if filter.Protocols.R66TLS = urlParams.Get("filterProtocolR66-TLS"); filter.Protocols.R66TLS == "true" {
		filterProtocol = append(filterProtocol, "r66-tls")
	}

	if filter.Protocols.SFTP = urlParams.Get("filterProtocolSFTP"); filter.Protocols.SFTP == "true" {
		filterProtocol = append(filterProtocol, "sftp")
	}

	if filter.Protocols.HTTP = urlParams.Get("filterProtocolHTTP"); filter.Protocols.HTTP == "true" {
		filterProtocol = append(filterProtocol, "http")
	}

	if filter.Protocols.HTTPS = urlParams.Get("filterProtocolHTTPS"); filter.Protocols.HTTPS == "true" {
		filterProtocol = append(filterProtocol, "https")
	}

	if filter.Protocols.FTP = urlParams.Get("filterProtocolFTP"); filter.Protocols.FTP == "true" {
		filterProtocol = append(filterProtocol, "ftp")
	}

	if filter.Protocols.FTPS = urlParams.Get("filterProtocolFTPS"); filter.Protocols.FTPS == "true" {
		filterProtocol = append(filterProtocol, "ftps")
	}

	if filter.Protocols.PeSIT = urlParams.Get("filterProtocolPeSIT"); filter.Protocols.PeSIT == "true" {
		filterProtocol = append(filterProtocol, "pesit")
	}

	if filter.Protocols.PeSITTLS = urlParams.Get("filterProtocolPeSIT-TLS"); filter.Protocols.PeSITTLS == "true" {
		filterProtocol = append(filterProtocol, "pesit-tls")
	}

	return filter, filterProtocol
}

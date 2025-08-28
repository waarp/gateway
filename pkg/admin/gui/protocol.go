package gui

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ftp"
	httpconst "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
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

func applyProtocolsFilter(filter *Filters) []string {
	var filterProtocol []string
	if filter.Protocols.R66 == True {
		filterProtocol = append(filterProtocol, r66.R66)
	}

	if filter.Protocols.R66TLS == True {
		filterProtocol = append(filterProtocol, r66.R66TLS)
	}

	if filter.Protocols.SFTP == True {
		filterProtocol = append(filterProtocol, sftp.SFTP)
	}

	if filter.Protocols.HTTP == True {
		filterProtocol = append(filterProtocol, httpconst.HTTP)
	}

	if filter.Protocols.HTTPS == True {
		filterProtocol = append(filterProtocol, httpconst.HTTPS)
	}

	if filter.Protocols.FTP == True {
		filterProtocol = append(filterProtocol, ftp.FTP)
	}

	if filter.Protocols.FTPS == True {
		filterProtocol = append(filterProtocol, ftp.FTPS)
	}

	if filter.Protocols.PeSIT == True {
		filterProtocol = append(filterProtocol, pesit.Pesit)
	}

	if filter.Protocols.PeSITTLS == True {
		filterProtocol = append(filterProtocol, pesit.PesitTLS)
	}

	return filterProtocol
}

func checkProtocolsFilter(r *http.Request, isApply bool, filter *Filters) (*Filters, []string) {
	urlParams := r.URL.Query()
	hasProtoParams := urlParams.Has("filterProtocolR66") ||
		urlParams.Has("filterProtocolR66-TLS") || urlParams.Has("filterProtocolSFTP") ||
		urlParams.Has("filterProtocolHTTP") || urlParams.Has("filterProtocolHTTPS") ||
		urlParams.Has("filterProtocolFTP") || urlParams.Has("filterProtocolFTPS") ||
		urlParams.Has("filterProtocolPeSIT") || urlParams.Has("filterProtocolPeSIT-TLS")

	if isApply || hasProtoParams {
		return protocolsFilter(r, filter)
	}
	filterProtocol := applyProtocolsFilter(filter)

	return filter, filterProtocol
}

func supportedProtocolInternal(protocol string) []string {
	supportedProtocolsInternal := map[string][]string{
		r66.R66:         {auth.Password},
		r66.R66TLS:      {auth.Password, auth.TLSTrustedCertificate, r66.AuthLegacyCertificate},
		httpconst.HTTP:  {auth.Password},
		httpconst.HTTPS: {auth.Password, auth.TLSTrustedCertificate},
		sftp.SFTP:       {auth.Password, sftp.AuthSSHPublicKey},
		pesit.Pesit:     {auth.Password},
		pesit.PesitTLS:  {auth.Password, auth.TLSTrustedCertificate},
	}

	return supportedProtocolsInternal[protocol]
}

func supportedProtocolExternal(protocol string) []string {
	supportedProtocolsExternal := map[string][]string{
		r66.R66:         {auth.Password},
		r66.R66TLS:      {auth.Password, auth.TLSCertificate, r66.AuthLegacyCertificate},
		httpconst.HTTP:  {auth.Password},
		httpconst.HTTPS: {auth.Password, auth.TLSCertificate},
		sftp.SFTP:       {auth.Password, sftp.AuthSSHPrivateKey},
		pesit.Pesit:     {auth.Password, pesit.PreConnectionAuth},
		pesit.PesitTLS:  {auth.TLSCertificate, pesit.PreConnectionAuth},
	}

	return supportedProtocolsExternal[protocol]
}

func getProtocolsList() []string {
	var protocolsList []string
	for protocol := range protocols.List {
		protocolsList = append(protocolsList, protocol)
	}

	return protocolsList
}

//nolint:gochecknoglobals // Constant
var (
	TLSVersions            = []string{protoutils.TLSv10, protoutils.TLSv11, protoutils.TLSv12, protoutils.TLSv13}
	CompatibilityModePeSIT = []string{pesit.CompatibilityModeStandard, pesit.CompatibilityModeNonStandard}
	TLSRequirement         = []string{string(ftp.TLSOptional), string(ftp.TLSMandatory), string(ftp.TLSImplicit)}
	ProtocolsList          = getProtocolsList()
)

func protocolsFilter(r *http.Request, filter *Filters) (*Filters, []string) {
	var filterProtocol []string
	urlParams := r.URL.Query()

	if filter.Protocols.R66 = urlParams.Get("filterProtocolR66"); filter.Protocols.R66 == True {
		filterProtocol = append(filterProtocol, r66.R66)
	}

	if filter.Protocols.R66TLS = urlParams.Get("filterProtocolR66-TLS"); filter.Protocols.R66TLS == True {
		filterProtocol = append(filterProtocol, r66.R66TLS)
	}

	if filter.Protocols.SFTP = urlParams.Get("filterProtocolSFTP"); filter.Protocols.SFTP == True {
		filterProtocol = append(filterProtocol, sftp.SFTP)
	}

	if filter.Protocols.HTTP = urlParams.Get("filterProtocolHTTP"); filter.Protocols.HTTP == True {
		filterProtocol = append(filterProtocol, httpconst.HTTP)
	}

	if filter.Protocols.HTTPS = urlParams.Get("filterProtocolHTTPS"); filter.Protocols.HTTPS == True {
		filterProtocol = append(filterProtocol, httpconst.HTTPS)
	}

	if filter.Protocols.FTP = urlParams.Get("filterProtocolFTP"); filter.Protocols.FTP == True {
		filterProtocol = append(filterProtocol, ftp.FTP)
	}

	if filter.Protocols.FTPS = urlParams.Get("filterProtocolFTPS"); filter.Protocols.FTPS == True {
		filterProtocol = append(filterProtocol, ftp.FTPS)
	}

	if filter.Protocols.PeSIT = urlParams.Get("filterProtocolPeSIT"); filter.Protocols.PeSIT == True {
		filterProtocol = append(filterProtocol, pesit.Pesit)
	}

	if filter.Protocols.PeSITTLS = urlParams.Get("filterProtocolPeSIT-TLS"); filter.Protocols.PeSITTLS == True {
		filterProtocol = append(filterProtocol, pesit.PesitTLS)
	}

	return filter, filterProtocol
}

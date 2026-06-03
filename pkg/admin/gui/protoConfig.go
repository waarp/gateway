package gui

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/as2"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ftp"
	httpconst "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/webdav"
)

// formList returns the values submitted for the given multi-valued form field
// (ex: "cipherSuites[]"). The empty values are ignored, since the placeholder
// option of an unset dropdown is submitted as an empty value.
func formList(r *http.Request, field string) []string {
	if r.Form == nil {
		if err := r.ParseForm(); err != nil {
			return nil
		}
	}

	values := make([]string, 0, len(r.Form[field]))

	for _, value := range r.Form[field] {
		if value != "" {
			values = append(values, value)
		}
	}

	return values
}

//nolint:dupl // is for partner protoConfig
func protoConfigR66Partner(r *http.Request, protocol string) map[string]any {
	r66ProtoConfig := make(map[string]any)

	if serverLogin := r.FormValue("protoConfigR66serverLogin"); serverLogin != "" {
		r66ProtoConfig["serverLogin"] = serverLogin
	}

	if blockSize := r.FormValue("protoConfigR66blockSize"); blockSize != "" {
		size, err := internal.ParseUint[uint32](blockSize)
		if err != nil {
			return nil
		}
		r66ProtoConfig["blockSize"] = size
	}

	r66ProtoConfig["noFinalHash"] = r.FormValue("noFinalHash") == True

	r66ProtoConfig["checkBlockHash"] = r.FormValue("checkBlockHash") == True

	if protocol == r66.R66TLS {
		if minTLSVersion := r.FormValue("protoConfigR66MinTLSVersion"); minTLSVersion != "" {
			r66ProtoConfig["minTLSVersion"] = minTLSVersion
		}

		if cipherSuites := formList(r, "protoConfigR66CipherSuites[]"); len(cipherSuites) > 0 {
			r66ProtoConfig["cipherSuites"] = cipherSuites
		}
	}

	return r66ProtoConfig
}

//nolint:dupl // is for server protoConfig
func protoConfigR66Server(r *http.Request, protocol string) map[string]any {
	r66ProtoConfig := make(map[string]any)

	if serverLogin := r.FormValue("protoConfigR66serverLogin"); serverLogin != "" {
		r66ProtoConfig["serverLogin"] = serverLogin
	}

	if blockSize := r.FormValue("protoConfigR66blockSize"); blockSize != "" {
		size, err := internal.ParseUint[uint32](blockSize)
		if err != nil {
			return nil
		}
		r66ProtoConfig["blockSize"] = size
	}

	r66ProtoConfig["noFinalHash"] = r.FormValue("noFinalHash") == True

	r66ProtoConfig["checkBlockHash"] = r.FormValue("checkBlockHash") == True

	if protocol == r66.R66TLS {
		if minTLSVersion := r.FormValue("protoConfigR66MinTLSVersion"); minTLSVersion != "" {
			r66ProtoConfig["minTLSVersion"] = minTLSVersion
		}

		if cipherSuites := formList(r, "protoConfigR66CipherSuites[]"); len(cipherSuites) > 0 {
			r66ProtoConfig["cipherSuites"] = cipherSuites
		}
	}

	return r66ProtoConfig
}

func protoConfigR66Client(r *http.Request, protocol string) map[string]any {
	r66ProtoConfig := make(map[string]any)

	if blockSize := r.FormValue("protoConfigR66blockSize"); blockSize != "" {
		size, err := internal.ParseUint[uint32](blockSize)
		if err != nil {
			return nil
		}
		r66ProtoConfig["blockSize"] = size
	}

	r66ProtoConfig["noFinalHash"] = r.FormValue("noFinalHash") == True

	r66ProtoConfig["checkBlockHash"] = r.FormValue("checkBlockHash") == True

	if protocol == r66.R66TLS {
		if minTLSVersion := r.FormValue("protoConfigR66MinTLSVersion"); minTLSVersion != "" {
			r66ProtoConfig["minTLSVersion"] = minTLSVersion
		}

		if cipherSuites := formList(r, "protoConfigR66CipherSuites[]"); len(cipherSuites) > 0 {
			r66ProtoConfig["cipherSuites"] = cipherSuites
		}
	}

	return r66ProtoConfig
}

func protoConfigHTTPpartner(r *http.Request, protocol string) map[string]any {
	httpProtoConfig := make(map[string]any)

	if protocol == httpconst.HTTPS {
		if minTLSVersion := r.FormValue("protoConfigHTTPSMinTLSVersion"); minTLSVersion != "" {
			httpProtoConfig["minTLSVersion"] = minTLSVersion
		}

		if cipherSuites := formList(r, "protoConfigHTTPSCipherSuites[]"); len(cipherSuites) > 0 {
			httpProtoConfig["cipherSuites"] = cipherSuites
		}
	}

	return httpProtoConfig
}

func protoConfigHTTPserver(r *http.Request, protocol string) map[string]any {
	httpProtoConfig := make(map[string]any)

	if protocol == httpconst.HTTPS {
		if minTLSVersion := r.FormValue("protoConfigHTTPSMinTLSVersion"); minTLSVersion != "" {
			httpProtoConfig["minTLSVersion"] = minTLSVersion
		}

		if cipherSuites := formList(r, "protoConfigHTTPSCipherSuites[]"); len(cipherSuites) > 0 {
			httpProtoConfig["cipherSuites"] = cipherSuites
		}
	}

	return httpProtoConfig
}

func protoConfigHTTPclient(r *http.Request, protocol string) map[string]any {
	httpProtoConfig := make(map[string]any)

	if protocol == httpconst.HTTPS {
		if minTLSVersion := r.FormValue("protoConfigHTTPSMinTLSVersion"); minTLSVersion != "" {
			httpProtoConfig["minTLSVersion"] = minTLSVersion
		}

		if cipherSuites := formList(r, "protoConfigHTTPSCipherSuites[]"); len(cipherSuites) > 0 {
			httpProtoConfig["cipherSuites"] = cipherSuites
		}
	}

	return httpProtoConfig
}

func protoConfigSFTPpartner(r *http.Request) map[string]any {
	sftpProtoConfig := make(map[string]any)

	if keyExchanges := r.Form["keyExchanges[]"]; len(keyExchanges) > 0 {
		sftpProtoConfig["keyExchanges"] = keyExchanges
	}

	if ciphers := r.Form["ciphers[]"]; len(ciphers) > 0 {
		sftpProtoConfig["ciphers"] = ciphers
	}

	if macs := r.Form["macs[]"]; len(macs) > 0 {
		sftpProtoConfig["macs"] = macs
	}

	sftpProtoConfig["useStat"] = r.FormValue("useStat") == True

	sftpProtoConfig["disableClientConcurrentReads"] = r.FormValue("disableClientConcurrentReads") == True

	return sftpProtoConfig
}

func protoConfigSFTPServer(r *http.Request) map[string]any {
	sftpProtoConfig := make(map[string]any)

	if keyExchanges := r.Form["keyExchanges[]"]; len(keyExchanges) > 0 {
		sftpProtoConfig["keyExchanges"] = keyExchanges
	}

	if ciphers := r.Form["ciphers[]"]; len(ciphers) > 0 {
		sftpProtoConfig["ciphers"] = ciphers
	}

	if macs := r.Form["macs[]"]; len(macs) > 0 {
		sftpProtoConfig["macs"] = macs
	}

	return sftpProtoConfig
}

func protoConfigSFTPClient(r *http.Request) map[string]any {
	sftpProtoConfig := make(map[string]any)

	if keyExchanges := r.Form["keyExchanges[]"]; len(keyExchanges) > 0 {
		sftpProtoConfig["keyExchanges"] = keyExchanges
	}

	if ciphers := r.Form["ciphers[]"]; len(ciphers) > 0 {
		sftpProtoConfig["ciphers"] = ciphers
	}

	if macs := r.Form["macs[]"]; len(macs) > 0 {
		sftpProtoConfig["macs"] = macs
	}

	return sftpProtoConfig
}

func protoConfigFTPpartner(r *http.Request, protocol string) map[string]any {
	ftpProtoConfig := make(map[string]any)

	ftpProtoConfig["disableActiveMode"] = r.FormValue("disableActiveMode") == True

	ftpProtoConfig["disableEPSV"] = r.FormValue("disableEPSV") == True

	if protocol == ftp.FTPS {
		ftpProtoConfig["useImplicitTLS"] = r.FormValue("useImplicitTLS") == True
		if minTLSVersion := r.FormValue("protoConfigFTPSMinTLSVersion"); minTLSVersion != "" {
			ftpProtoConfig["minTLSVersion"] = minTLSVersion
		}

		if cipherSuites := formList(r, "protoConfigFTPSCipherSuites[]"); len(cipherSuites) > 0 {
			ftpProtoConfig["cipherSuites"] = cipherSuites
		}

		ftpProtoConfig["disableTLSSessionReuse"] = r.FormValue("disableTLSSessionReuse") == True
	}

	return ftpProtoConfig
}

func protoConfigFTPServer(r *http.Request, protocol string) map[string]any {
	ftpProtoConfig := make(map[string]any)

	ftpProtoConfig["disablePassiveMode"] = r.FormValue("disablePassiveMode") == True

	ftpProtoConfig["disableActiveMode"] = r.FormValue("disableActiveMode") == True

	if passiveModeMinPort := r.FormValue("passiveModeMinPort"); passiveModeMinPort != "" {
		size, err := internal.ParseUint[uint16](passiveModeMinPort)
		if err != nil {
			return nil
		}
		ftpProtoConfig["passiveModeMinPort"] = size
	}

	if passiveModeMaxPort := r.FormValue("passiveModeMaxPort"); passiveModeMaxPort != "" {
		size, err := internal.ParseUint[uint16](passiveModeMaxPort)
		if err != nil {
			return nil
		}
		ftpProtoConfig["passiveModeMaxPort"] = size
	}

	if protocol == ftp.FTPS {
		if tlsRequirement := r.FormValue("tlsRequirement"); tlsRequirement != "" {
			ftpProtoConfig["tlsRequirement"] = tlsRequirement
		}

		if minTLSVersion := r.FormValue("protoConfigFTPSMinTLSVersion"); minTLSVersion != "" {
			ftpProtoConfig["minTLSVersion"] = minTLSVersion
		}

		if cipherSuites := formList(r, "protoConfigFTPSCipherSuites[]"); len(cipherSuites) > 0 {
			ftpProtoConfig["cipherSuites"] = cipherSuites
		}
	}

	return ftpProtoConfig
}

func protoConfigFTPClient(r *http.Request, protocol string) map[string]any {
	ftpProtoConfig := make(map[string]any)

	ftpProtoConfig["enableActiveMode"] = r.FormValue("enableActiveMode") == True

	if activeModeAddress := r.FormValue("activeModeAddress"); activeModeAddress != "" {
		ftpProtoConfig["activeModeAddress"] = activeModeAddress
	}

	if activeModeMinPort := r.FormValue("activeModeMinPort"); activeModeMinPort != "" {
		size, err := internal.ParseUint[uint16](activeModeMinPort)
		if err != nil {
			return nil
		}
		ftpProtoConfig["activeModeMinPort"] = size
	}

	if activeModeMaxPort := r.FormValue("activeModeMaxPort"); activeModeMaxPort != "" {
		size, err := internal.ParseUint[uint16](activeModeMaxPort)
		if err != nil {
			return nil
		}
		ftpProtoConfig["activeModeMaxPort"] = size
	}

	if protocol == ftp.FTPS {
		if minTLSVersion := r.FormValue("protoConfigFTPSMinTLSVersion"); minTLSVersion != "" {
			ftpProtoConfig["minTLSVersion"] = minTLSVersion
		}

		if cipherSuites := formList(r, "protoConfigFTPSCipherSuites[]"); len(cipherSuites) > 0 {
			ftpProtoConfig["cipherSuites"] = cipherSuites
		}
	}

	return ftpProtoConfig
}

//nolint:gocyclo,cyclop,funlen // no split method
func protoConfigPeSITPartner(r *http.Request, protocol string) map[string]any {
	pesitProtoConfig := make(map[string]any)

	if login := r.FormValue("protoConfigPeSITlogin"); login != "" {
		pesitProtoConfig["login"] = login
	}

	pesitProtoConfig["disableRestart"] = r.FormValue("disableRestart") == True

	pesitProtoConfig["disableCheckpoints"] = r.FormValue("disableCheckpoints") == True

	if checkpointSize := r.FormValue("protoConfigPeSITcheckpointSize"); checkpointSize != "" {
		size, err := internal.ParseUint[uint32](checkpointSize)
		if err != nil {
			return nil
		}
		pesitProtoConfig["checkpointSize"] = size
	}

	if checkpointWindow := r.FormValue("protoConfigPeSITcheckpointWindow"); checkpointWindow != "" {
		size, err := internal.ParseUint[uint32](checkpointWindow)
		if err != nil {
			return nil
		}
		pesitProtoConfig["checkpointWindow"] = size
	}

	pesitProtoConfig["useNSDU"] = r.FormValue("useNSDU") == True

	if compatibilityMode := r.FormValue("protoConfigPeSITcompatibilityMode"); compatibilityMode != "" {
		pesitProtoConfig["compatibilityMode"] = compatibilityMode
	}

	if maxMessageSize := r.FormValue("protoConfigPeSITmaxMessageSize"); maxMessageSize != "" {
		size, err := internal.ParseUint[uint32](maxMessageSize)
		if err != nil {
			return nil
		}
		pesitProtoConfig["maxMessageSize"] = size
	}

	pesitProtoConfig["disablePreConnection"] = r.FormValue("disablePreConnection") == True
	pesitProtoConfig["expectsAck"] = r.FormValue("expectsAck") == True

	if protocol == pesit.PesitTLS {
		if minTLSVersion := r.FormValue("protoConfigPeSITMinTLSVersion"); minTLSVersion != "" {
			pesitProtoConfig["minTLSVersion"] = minTLSVersion
		}

		if cipherSuites := formList(r, "protoConfigPeSITCipherSuites[]"); len(cipherSuites) > 0 {
			pesitProtoConfig["cipherSuites"] = cipherSuites
		}
	}

	return pesitProtoConfig
}

//nolint:gocyclo,cyclop,funlen // no split method
func protoConfigPeSITServer(r *http.Request, protocol string) map[string]any {
	pesitProtoConfig := make(map[string]any)

	pesitProtoConfig["disableRestart"] = r.FormValue("disableRestart") == True

	pesitProtoConfig["disableCheckpoints"] = r.FormValue("disableCheckpoints") == True

	if checkpointSize := r.FormValue("protoConfigPeSITcheckpointSize"); checkpointSize != "" {
		size, err := internal.ParseUint[uint32](checkpointSize)
		if err != nil {
			return nil
		}
		pesitProtoConfig["checkpointSize"] = size
	}

	if checkpointWindow := r.FormValue("protoConfigPeSITcheckpointWindow"); checkpointWindow != "" {
		size, err := internal.ParseUint[uint32](checkpointWindow)
		if err != nil {
			return nil
		}
		pesitProtoConfig["checkpointWindow"] = size
	}

	if compatibilityMode := r.FormValue("protoConfigPeSITcompatibilityMode"); compatibilityMode != "" {
		pesitProtoConfig["compatibilityMode"] = compatibilityMode
	}

	if maxMessageSize := r.FormValue("protoConfigPeSITmaxMessageSize"); maxMessageSize != "" {
		size, err := internal.ParseUint[uint32](maxMessageSize)
		if err != nil {
			return nil
		}
		pesitProtoConfig["maxMessageSize"] = size
	}

	pesitProtoConfig["disablePreConnection"] = r.FormValue("disablePreConnection") == True

	if protocol == pesit.PesitTLS {
		if minTLSVersion := r.FormValue("protoConfigPeSITMinTLSVersion"); minTLSVersion != "" {
			pesitProtoConfig["minTLSVersion"] = minTLSVersion
		}

		if cipherSuites := formList(r, "protoConfigPeSITCipherSuites[]"); len(cipherSuites) > 0 {
			pesitProtoConfig["cipherSuites"] = cipherSuites
		}
	}

	return pesitProtoConfig
}

func protoConfigPeSITClient(r *http.Request, protocol string) map[string]any {
	pesitProtoConfig := make(map[string]any)

	pesitProtoConfig["disableRestart"] = r.FormValue("disableRestart") == True

	pesitProtoConfig["disableCheckpoints"] = r.FormValue("disableCheckpoints") == True

	if checkpointSize := r.FormValue("protoConfigPeSITcheckpointSize"); checkpointSize != "" {
		size, err := internal.ParseUint[uint32](checkpointSize)
		if err != nil {
			return nil
		}
		pesitProtoConfig["checkpointSize"] = size
	}

	if checkpointWindow := r.FormValue("protoConfigPeSITcheckpointWindow"); checkpointWindow != "" {
		size, err := internal.ParseUint[uint32](checkpointWindow)
		if err != nil {
			return nil
		}
		pesitProtoConfig["checkpointWindow"] = size
	}

	if protocol == pesit.PesitTLS {
		if minTLSVersion := r.FormValue("protoConfigPeSITMinTLSVersion"); minTLSVersion != "" {
			pesitProtoConfig["minTLSVersion"] = minTLSVersion
		}

		if cipherSuites := formList(r, "protoConfigPeSITCipherSuites[]"); len(cipherSuites) > 0 {
			pesitProtoConfig["cipherSuites"] = cipherSuites
		}
	}

	return pesitProtoConfig
}

func protoConfigWebdavPartner(r *http.Request, protocol string) map[string]any {
	conf := make(map[string]any)

	if protocol == webdav.WebdavTLS {
		if minTLSVersion := r.FormValue("protoConfigWebdavMinTLSVersion"); minTLSVersion != "" {
			conf["minTLSVersion"] = minTLSVersion
		}

		if cipherSuites := formList(r, "protoConfigWebdavCipherSuites[]"); len(cipherSuites) > 0 {
			conf["cipherSuites"] = cipherSuites
		}
	}

	return conf
}

func protoConfigWebdavServer(r *http.Request, protocol string) map[string]any {
	conf := make(map[string]any)

	if protocol == webdav.WebdavTLS {
		if minTLSVersion := r.FormValue("protoConfigWebdavMinTLSVersion"); minTLSVersion != "" {
			conf["minTLSVersion"] = minTLSVersion
		}

		if cipherSuites := formList(r, "protoConfigWebdavCipherSuites[]"); len(cipherSuites) > 0 {
			conf["cipherSuites"] = cipherSuites
		}
	}

	return conf
}

func protoConfigWebdavClient(r *http.Request, protocol string) map[string]any {
	conf := make(map[string]any)

	if protocol == webdav.WebdavTLS {
		if minTLSVersion := r.FormValue("protoConfigWebdavMinTLSVersion"); minTLSVersion != "" {
			conf["minTLSVersion"] = minTLSVersion
		}

		if cipherSuites := formList(r, "protoConfigWebdavCipherSuites[]"); len(cipherSuites) > 0 {
			conf["cipherSuites"] = cipherSuites
		}
	}

	return conf
}

func protoConfigAS2Partner(r *http.Request, protocol string) map[string]any {
	conf := make(map[string]any)

	if signAlgo := r.FormValue("protoConfigAS2SignAlgo"); signAlgo != "" {
		conf["signatureAlgorithm"] = signAlgo
	}

	if encrAlgo := r.FormValue("protoConfigAS2EncryptAlgo"); encrAlgo != "" {
		conf["encryptionAlgorithm"] = encrAlgo
	}

	if asyncURL := r.FormValue("protoConfigAS2AsyncMDNURL"); asyncURL != "" {
		conf["asyncMDNAddress"] = asyncURL
	}

	conf["handleAsyncMDN"] = r.FormValue("protoConfigAS2HandleAsyncMDN") == True

	if protocol == as2.AS2TLS {
		if minTLSVersion := r.FormValue("protoConfigAS2MinTLSVersion"); minTLSVersion != "" {
			conf["minTLSVersion"] = minTLSVersion
		}

		if cipherSuites := formList(r, "protoConfigAS2CipherSuites[]"); len(cipherSuites) > 0 {
			conf["cipherSuites"] = cipherSuites
		}
	}

	return conf
}

func protoConfigAS2Server(r *http.Request, protocol string) map[string]any {
	conf := make(map[string]any)

	if fileLimit := r.FormValue("protoConfigAS2MaxFileSize"); fileLimit != "" {
		size, err := internal.ParseInt[int64](fileLimit)
		if err != nil {
			return nil
		}

		conf["maxFileSize"] = size
	}

	if signAlgo := r.FormValue("protoConfigAS2MDNSignAlgo"); signAlgo != "" {
		conf["mdnSignatureAlgorithm"] = signAlgo
	}

	if protocol == as2.AS2TLS {
		if minTLSVersion := r.FormValue("protoConfigAS2MinTLSVersion"); minTLSVersion != "" {
			conf["minTLSVersion"] = minTLSVersion
		}

		if cipherSuites := formList(r, "protoConfigAS2CipherSuites[]"); len(cipherSuites) > 0 {
			conf["cipherSuites"] = cipherSuites
		}
	}

	return conf
}

func protoConfigAS2Client(r *http.Request, protocol string) map[string]any {
	conf := make(map[string]any)

	if fileLimit := r.FormValue("protoConfigAS2MaxFileSize"); fileLimit != "" {
		size, err := internal.ParseInt[int64](fileLimit)
		if err != nil {
			return nil
		}

		conf["maxFileSize"] = size
	}

	if protocol == as2.AS2TLS {
		if minTLSVersion := r.FormValue("protoConfigAS2MinTLSVersion"); minTLSVersion != "" {
			conf["minTLSVersion"] = minTLSVersion
		}

		if cipherSuites := formList(r, "protoConfigAS2CipherSuites[]"); len(cipherSuites) > 0 {
			conf["cipherSuites"] = cipherSuites
		}
	}

	return conf
}

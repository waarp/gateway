package gui

import (
	"net/http"
	"strconv"
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

func protoConfigR66(r *http.Request) map[string]any {
	r66ProtoConfig := make(map[string]any)

	if serverLogin := r.FormValue("protoConfigR66serverLogin"); serverLogin != "" {
		r66ProtoConfig["serverLogin"] = serverLogin
	}

	if blockSize := r.FormValue("protoConfigR66blockSize"); blockSize != "" {
		size, err := strconv.Atoi(blockSize)
		if err != nil {
			return nil
		}
		r66ProtoConfig["blockSize"] = uint32(size)
	}

	if noFinalHash := r.FormValue("noFinalHash"); noFinalHash == "true" {
		r66ProtoConfig["noFinalHash"] = true
	} else {
		r66ProtoConfig["noFinalHash"] = false
	}

	if checkBlockHash := r.FormValue("checkBlockHash"); checkBlockHash == "true" {
		r66ProtoConfig["checkBlockHash"] = true
	} else {
		r66ProtoConfig["checkBlockHash"] = false
	}

	return r66ProtoConfig
}

func protoConfigSFTP(r *http.Request) map[string]any {
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

	if useStat := r.FormValue("useStat"); useStat == "true" {
		sftpProtoConfig["useStat"] = true
	} else {
		sftpProtoConfig["useStat"] = false
	}

	if dCCR := r.FormValue("disableClientConcurrentReads"); dCCR == "true" {
		sftpProtoConfig["disableClientConcurrentReads"] = true
	} else {
		sftpProtoConfig["disableClientConcurrentReads"] = false
	}

	return sftpProtoConfig
}

func protoConfigFTP(r *http.Request, protocol string) map[string]any {
	ftpProtoConfig := make(map[string]any)

	if disableActiveMode := r.FormValue("disableActiveMode"); disableActiveMode == "true" {
		ftpProtoConfig["disableActiveMode"] = true
	} else {
		ftpProtoConfig["disableActiveMode"] = false
	}

	if disableEPSV := r.FormValue("disableEPSV"); disableEPSV == "true" {
		ftpProtoConfig["disableEPSV"] = true
	} else {
		ftpProtoConfig["disableEPSV"] = false
	}

	if protocol == "ftps" { //nolint:nestif // call ftps
		if useImplicitTLS := r.FormValue("useImplicitTLS"); useImplicitTLS == "true" {
			ftpProtoConfig["useImplicitTLS"] = true
		} else {
			ftpProtoConfig["useImplicitTLS"] = false
		}

		if minTLSVersion := r.FormValue("protoConfigFTPSminTLSVersion"); minTLSVersion != "" {
			ftpProtoConfig["minTLSVersion"] = minTLSVersion
		}

		if disableTLSSessionReuse := r.FormValue("disableTLSSessionReuse"); disableTLSSessionReuse == "true" {
			ftpProtoConfig["disableTLSSessionReuse"] = true
		} else {
			ftpProtoConfig["disableTLSSessionReuse"] = false
		}
	}

	return ftpProtoConfig
}

//nolint:gocyclo,cyclop,funlen // no split method
func protoConfigPeSIT(r *http.Request, protocol string) map[string]any {
	pesitProtoConfig := make(map[string]any)

	if login := r.FormValue("protoConfigPeSITlogin"); login != "" {
		pesitProtoConfig["login"] = login
	}

	if disableRestart := r.FormValue("disableRestart"); disableRestart == "true" {
		pesitProtoConfig["disableRestart"] = true
	} else {
		pesitProtoConfig["disableRestart"] = false
	}

	if disableCheckpoints := r.FormValue("disableCheckpoints"); disableCheckpoints == "true" {
		pesitProtoConfig["disableCheckpoints"] = true
	} else {
		pesitProtoConfig["disableCheckpoints"] = false
	}

	if checkpointSize := r.FormValue("protoConfigPeSITcheckpointSize"); checkpointSize != "" {
		size, err := strconv.Atoi(checkpointSize)
		if err != nil {
			return nil
		}
		pesitProtoConfig["checkpointSize"] = uint32(size)
	}

	if checkpointWindow := r.FormValue("protoConfigPeSITcheckpointWindow"); checkpointWindow != "" {
		size, err := strconv.Atoi(checkpointWindow)
		if err != nil {
			return nil
		}
		pesitProtoConfig["checkpointWindow"] = uint32(size)
	}

	if useNSDU := r.FormValue("useNSDU"); useNSDU == "true" {
		pesitProtoConfig["useNSDU"] = true
	} else {
		pesitProtoConfig["useNSDU"] = false
	}

	if compatibilityMode := r.FormValue("protoConfigPeSITcompatibilityMode"); compatibilityMode != "" {
		pesitProtoConfig["compatibilityMode"] = compatibilityMode
	}

	if maxMessageSize := r.FormValue("protoConfigPeSITmaxMessageSize"); maxMessageSize != "" {
		size, err := strconv.Atoi(maxMessageSize)
		if err != nil {
			return nil
		}
		pesitProtoConfig["maxMessageSize"] = uint32(size)
	}

	if disablePreConnection := r.FormValue("disablePreConnection"); disablePreConnection == "true" {
		pesitProtoConfig["disablePreConnection"] = true
	} else {
		pesitProtoConfig["disablePreConnection"] = false
	}

	if protocol == "pesit-tls" {
		if minTLSVersion := r.FormValue("protoConfigPeSIT-TLSminTLSVersion"); minTLSVersion != "" {
			pesitProtoConfig["minTLSVersion"] = minTLSVersion
		}
	}

	return pesitProtoConfig
}

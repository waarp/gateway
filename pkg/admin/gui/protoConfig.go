package gui

import (
	"net/http"
	"strconv"

	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ftp"
	httpconst "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
)

//nolint:dupl // is for partner protoConfig
func protoConfigR66Partner(r *http.Request, protocol string) map[string]any {
	r66ProtoConfig := make(map[string]any)

	if serverLogin := r.FormValue("protoConfigR66serverLogin"); serverLogin != "" {
		r66ProtoConfig["serverLogin"] = serverLogin
	}

	if blockSize := r.FormValue("protoConfigR66blockSize"); blockSize != "" {
		size, err := strconv.ParseUint(blockSize, 10, 64)
		if err != nil {
			return nil
		}
		r66ProtoConfig["blockSize"] = uint32(size)
	}

	r66ProtoConfig["noFinalHash"] = r.FormValue("noFinalHash") == True

	r66ProtoConfig["checkBlockHash"] = r.FormValue("checkBlockHash") == True

	if protocol == r66.R66TLS {
		if minTLSVersion := r.FormValue("protoConfigR66-tlsMinTLSVersion"); minTLSVersion != "" {
			r66ProtoConfig["minTLSVersion"] = minTLSVersion
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
		size, err := strconv.ParseUint(blockSize, 10, 64)
		if err != nil {
			return nil
		}
		r66ProtoConfig["blockSize"] = uint32(size)
	}

	r66ProtoConfig["noFinalHash"] = r.FormValue("noFinalHash") == True

	r66ProtoConfig["checkBlockHash"] = r.FormValue("checkBlockHash") == True

	if protocol == r66.R66TLS {
		if minTLSVersion := r.FormValue("protoConfigR66-tlsMinTLSVersion"); minTLSVersion != "" {
			r66ProtoConfig["minTLSVersion"] = minTLSVersion
		}
	}

	return r66ProtoConfig
}

func protoConfigR66Client(r *http.Request, protocol string) map[string]any {
	r66ProtoConfig := make(map[string]any)

	if blockSize := r.FormValue("protoConfigR66blockSize"); blockSize != "" {
		size, err := strconv.ParseUint(blockSize, 10, 64)
		if err != nil {
			return nil
		}
		r66ProtoConfig["blockSize"] = uint32(size)
	}

	r66ProtoConfig["noFinalHash"] = r.FormValue("noFinalHash") == True

	r66ProtoConfig["checkBlockHash"] = r.FormValue("checkBlockHash") == True

	if protocol == r66.R66TLS {
		if minTLSVersion := r.FormValue("protoConfigR66-tlsMinTLSVersion"); minTLSVersion != "" {
			r66ProtoConfig["minTLSVersion"] = minTLSVersion
		}
	}

	return r66ProtoConfig
}

func protoConfigHTTPpartner(r *http.Request, protocol string) map[string]any {
	httpProtoConfig := make(map[string]any)

	if protocol == httpconst.HTTPS {
		if minTLSVersion := r.FormValue("protoConfigHttpsMinTLSVersion"); minTLSVersion != "" {
			httpProtoConfig["minTLSVersion"] = minTLSVersion
		}
	}

	return httpProtoConfig
}

func protoConfigHTTPserver(r *http.Request, protocol string) map[string]any {
	httpProtoConfig := make(map[string]any)

	if protocol == httpconst.HTTPS {
		if minTLSVersion := r.FormValue("protoConfigHttpsMinTLSVersion"); minTLSVersion != "" {
			httpProtoConfig["minTLSVersion"] = minTLSVersion
		}
	}

	return httpProtoConfig
}

func protoConfigHTTPclient(r *http.Request, protocol string) map[string]any {
	httpProtoConfig := make(map[string]any)

	if protocol == httpconst.HTTPS {
		if minTLSVersion := r.FormValue("protoConfigHttpsMinTLSVersion"); minTLSVersion != "" {
			httpProtoConfig["minTLSVersion"] = minTLSVersion
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
		if minTLSVersion := r.FormValue("protoConfigFTPSminTLSVersion"); minTLSVersion != "" {
			ftpProtoConfig["minTLSVersion"] = minTLSVersion
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
		size, err := strconv.ParseUint(passiveModeMinPort, 10, 64)
		if err != nil {
			return nil
		}
		ftpProtoConfig["passiveModeMinPort"] = uint32(size)
	}

	if passiveModeMaxPort := r.FormValue("passiveModeMaxPort"); passiveModeMaxPort != "" {
		size, err := strconv.ParseUint(passiveModeMaxPort, 10, 64)
		if err != nil {
			return nil
		}
		ftpProtoConfig["passiveModeMaxPort"] = uint32(size)
	}

	if protocol == ftp.FTPS {
		if tlsRequirement := r.FormValue("tlsRequirement"); tlsRequirement != "" {
			ftpProtoConfig["tlsRequirement"] = tlsRequirement
		}

		if minTLSVersion := r.FormValue("protoConfigFTPSminTLSVersion"); minTLSVersion != "" {
			ftpProtoConfig["minTLSVersion"] = minTLSVersion
		}
	}

	return ftpProtoConfig
}

func protoConfigFTPClient(r *http.Request, protocol string) map[string]any {
	ftpProtoConfig := make(map[string]any)

	ftpProtoConfig["enablePassiveMode"] = r.FormValue("enablePassiveMode") == True

	if activeModeAddress := r.FormValue("activeModeAddress"); activeModeAddress != "" {
		ftpProtoConfig["activeModeAddress"] = activeModeAddress
	}

	if activeModeMinPort := r.FormValue("activeModeMinPort"); activeModeMinPort != "" {
		size, err := strconv.ParseUint(activeModeMinPort, 10, 64)
		if err != nil {
			return nil
		}
		ftpProtoConfig["activeModeMinPort"] = uint32(size)
	}

	if activeModeMaxPort := r.FormValue("activeModeMaxPort"); activeModeMaxPort != "" {
		size, err := strconv.ParseUint(activeModeMaxPort, 10, 64)
		if err != nil {
			return nil
		}
		ftpProtoConfig["activeModeMaxPort"] = uint32(size)
	}

	if protocol == ftp.FTPS {
		if minTLSVersion := r.FormValue("protoConfigFTPSminTLSVersion"); minTLSVersion != "" {
			ftpProtoConfig["minTLSVersion"] = minTLSVersion
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
		size, err := strconv.ParseUint(checkpointSize, 10, 64)
		if err != nil {
			return nil
		}
		pesitProtoConfig["checkpointSize"] = uint32(size)
	}

	if checkpointWindow := r.FormValue("protoConfigPeSITcheckpointWindow"); checkpointWindow != "" {
		size, err := strconv.ParseUint(checkpointWindow, 10, 64)
		if err != nil {
			return nil
		}
		pesitProtoConfig["checkpointWindow"] = uint32(size)
	}

	pesitProtoConfig["useNSDU"] = r.FormValue("useNSDU") == True

	if compatibilityMode := r.FormValue("protoConfigPeSITcompatibilityMode"); compatibilityMode != "" {
		pesitProtoConfig["compatibilityMode"] = compatibilityMode
	}

	if maxMessageSize := r.FormValue("protoConfigPeSITmaxMessageSize"); maxMessageSize != "" {
		size, err := strconv.ParseUint(maxMessageSize, 10, 64)
		if err != nil {
			return nil
		}
		pesitProtoConfig["maxMessageSize"] = uint32(size)
	}

	pesitProtoConfig["disablePreConnection"] = r.FormValue("disablePreConnection") == True

	if protocol == pesit.PesitTLS {
		if minTLSVersion := r.FormValue("protoConfigFTPSminTLSVersion"); minTLSVersion != "" {
			pesitProtoConfig["minTLSVersion"] = minTLSVersion
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
		size, err := strconv.ParseUint(checkpointSize, 10, 64)
		if err != nil {
			return nil
		}
		pesitProtoConfig["checkpointSize"] = uint32(size)
	}

	if checkpointWindow := r.FormValue("protoConfigPeSITcheckpointWindow"); checkpointWindow != "" {
		size, err := strconv.ParseUint(checkpointWindow, 10, 64)
		if err != nil {
			return nil
		}
		pesitProtoConfig["checkpointWindow"] = uint32(size)
	}

	if maxMessageSize := r.FormValue("protoConfigPeSITmaxMessageSize"); maxMessageSize != "" {
		size, err := strconv.ParseUint(maxMessageSize, 10, 64)
		if err != nil {
			return nil
		}
		pesitProtoConfig["maxMessageSize"] = uint32(size)
	}

	pesitProtoConfig["disablePreConnection"] = r.FormValue("disablePreConnection") == True

	if protocol == pesit.PesitTLS {
		if minTLSVersion := r.FormValue("protoConfigFTPSminTLSVersion"); minTLSVersion != "" {
			pesitProtoConfig["minTLSVersion"] = minTLSVersion
		}
	}

	return pesitProtoConfig
}

func protoConfigPeSITClient(r *http.Request) map[string]any {
	pesitProtoConfig := make(map[string]any)

	pesitProtoConfig["disableRestart"] = r.FormValue("disableRestart") == True

	pesitProtoConfig["disableCheckpoints"] = r.FormValue("disableCheckpoints") == True

	if checkpointSize := r.FormValue("protoConfigPeSITcheckpointSize"); checkpointSize != "" {
		size, err := strconv.ParseUint(checkpointSize, 10, 64)
		if err != nil {
			return nil
		}
		pesitProtoConfig["checkpointSize"] = uint32(size)
	}

	if checkpointWindow := r.FormValue("protoConfigPeSITcheckpointWindow"); checkpointWindow != "" {
		size, err := strconv.ParseUint(checkpointWindow, 10, 64)
		if err != nil {
			return nil
		}
		pesitProtoConfig["checkpointWindow"] = uint32(size)
	}

	return pesitProtoConfig
}

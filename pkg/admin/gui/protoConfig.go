package gui

import (
	"net/http"
	"strconv"
)

//nolint:dupl // is for partner protoConfig
func protoConfigR66Partner(r *http.Request) map[string]any {
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

//nolint:dupl // is for server protoConfig
func protoConfigR66Server(r *http.Request) map[string]any {
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

func protoConfigR66Client(r *http.Request) map[string]any {
	r66ProtoConfig := make(map[string]any)

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

func protoConfigFTPServer(r *http.Request, protocol string) map[string]any {
	ftpProtoConfig := make(map[string]any)

	if disablePassiveMode := r.FormValue("disablePassiveMode"); disablePassiveMode == "true" {
		ftpProtoConfig["disablePassiveMode"] = true
	} else {
		ftpProtoConfig["disablePassiveMode"] = false
	}

	if disableActiveMode := r.FormValue("disableActiveMode"); disableActiveMode == "true" {
		ftpProtoConfig["disableActiveMode"] = true
	} else {
		ftpProtoConfig["disableActiveMode"] = false
	}

	if passiveModeMinPort := r.FormValue("passiveModeMinPort"); passiveModeMinPort != "" {
		size, err := strconv.Atoi(passiveModeMinPort)
		if err != nil {
			return nil
		}
		ftpProtoConfig["passiveModeMinPort"] = uint32(size)
	}

	if passiveModeMaxPort := r.FormValue("passiveModeMaxPort"); passiveModeMaxPort != "" {
		size, err := strconv.Atoi(passiveModeMaxPort)
		if err != nil {
			return nil
		}
		ftpProtoConfig["passiveModeMaxPort"] = uint32(size)
	}

	if protocol == "ftps" { //nolint:nestif // call ftps
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

	if enablePassiveMode := r.FormValue("enablePassiveMode"); enablePassiveMode == "true" {
		ftpProtoConfig["enablePassiveMode"] = true
	} else {
		ftpProtoConfig["enablePassiveMode"] = false
	}

	if activeModeAddress := r.FormValue("activeModeAddress"); activeModeAddress != "" {
		ftpProtoConfig["activeModeAddress"] = activeModeAddress
	}

	if activeModeMinPort := r.FormValue("activeModeMinPort"); activeModeMinPort != "" {
		size, err := strconv.Atoi(activeModeMinPort)
		if err != nil {
			return nil
		}
		ftpProtoConfig["activeModeMinPort"] = uint32(size)
	}

	if activeModeMaxPort := r.FormValue("activeModeMaxPort"); activeModeMaxPort != "" {
		size, err := strconv.Atoi(activeModeMaxPort)
		if err != nil {
			return nil
		}
		ftpProtoConfig["activeModeMaxPort"] = uint32(size)
	}

	if protocol == "ftps" {
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
		if minTLSVersion := r.FormValue("protoConfigFTPSminTLSVersion"); minTLSVersion != "" {
			pesitProtoConfig["minTLSVersion"] = minTLSVersion
		}
	}

	return pesitProtoConfig
}

//nolint:gocyclo,cyclop,funlen // no split method
func protoConfigPeSITServer(r *http.Request, protocol string) map[string]any {
	pesitProtoConfig := make(map[string]any)

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
		if minTLSVersion := r.FormValue("protoConfigFTPSminTLSVersion"); minTLSVersion != "" {
			pesitProtoConfig["minTLSVersion"] = minTLSVersion
		}
	}

	return pesitProtoConfig
}

func protoConfigPeSITClient(r *http.Request) map[string]any {
	pesitProtoConfig := make(map[string]any)

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

	return pesitProtoConfig
}

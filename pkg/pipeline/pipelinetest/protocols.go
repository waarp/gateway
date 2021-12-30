package pipelinetest

import "code.waarp.fr/apps/gateway/gateway/pkg/model/config"

type features struct {
	transID, ruleName, size bool
}

//nolint:gochecknoglobals // global var is necessary here
var protocols = map[string]features{
	"sftp":                {transID: false, ruleName: false, size: false},
	config.ProtocolR66:    {transID: true, ruleName: true, size: true},
	config.ProtocolR66TLS: {transID: true, ruleName: true, size: true},
	"http":                {transID: true, ruleName: true, size: true},
	"https":               {transID: true, ruleName: true, size: true},
}

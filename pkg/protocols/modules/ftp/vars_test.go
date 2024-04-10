package ftp

import "code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"

func init() {
	pipelinetest.Protocols[FTP] = pipelinetest.ProtoFeatures{
		MakeClient:        Module{}.NewClient,
		MakeServer:        Module{}.NewServer,
		MakeServerConfig:  Module{}.MakeServerConfig,
		MakePartnerConfig: Module{}.MakePartnerConfig,
		MakeClientConfig:  Module{}.MakeClientConfig,
		TransID:           false,
		RuleName:          false,
	}

	pipelinetest.Protocols[FTPS] = pipelinetest.ProtoFeatures{
		MakeClient:        ModuleFTPS{}.NewClient,
		MakeServer:        ModuleFTPS{}.NewServer,
		MakeServerConfig:  ModuleFTPS{}.MakeServerConfig,
		MakePartnerConfig: ModuleFTPS{}.MakePartnerConfig,
		MakeClientConfig:  ModuleFTPS{}.MakeClientConfig,
		TransID:           false,
		RuleName:          false,
	}
}

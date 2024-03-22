package pipelinetest

import (
	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

type ProtoFeatures struct {
	ClientConstr            clientConstructor
	ServiceConstr           serviceConstructor
	TransID, RuleName, Size bool
}

type (
	serviceConstructor func(db *database.DB, logger *log.Logger) proto.Service
	clientConstructor  func(*model.Client) (pipeline.Client, error)
)

//nolint:gochecknoglobals //global var is required here for more flexibility
var Protocols = map[string]ProtoFeatures{}

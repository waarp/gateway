package httptransport

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func getMinTLSVersion(ctx *model.TransferContext) protoutils.TLSVersion {
	const minTLSJsonKey = "minTLSVersion"
	var minVersion protoutils.TLSVersion = protoutils.DefaultTLSVersion

	if minStr, err := utils.GetAs[string](ctx.Client.ProtoConfig, minTLSJsonKey); err == nil {
		//nolint:errcheck //error is guaranteed to be nil here
		minVersion, _ = protoutils.TLSVersionFromString(minStr)
	}

	if minStr, err := utils.GetAs[string](ctx.RemoteAgent.ProtoConfig, minTLSJsonKey); err == nil {
		//nolint:errcheck //error is guaranteed to be nil here
		minVersion, _ = protoutils.TLSVersionFromString(minStr)
	}

	return minVersion
}

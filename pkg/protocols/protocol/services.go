// Package protocol defines the components which must be implemented by a
// protocol module in order to be usable by the gateway. These components are
// typically implemented as part of a module which is then registered in
// the protocols package.
package protocol

import (
	"context"

	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type StartStopper interface {
	Start() error
	Stop(ctx context.Context) error
	State() (utils.StateCode, string)
}

type Server interface {
	StartStopper
}

type Client interface {
	StartStopper
	InitTransfer(pip *pipeline.Pipeline) (TransferClient, *pipeline.Error)
}

package tasks

import (
	"net/url"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type (
	leaf   = utils.Leaf
	branch = utils.Branch
)

//nolint:dupl // factorizing would add complexity
func makeOutDir(transCtx *model.TransferContext) (*url.URL, error) {
	if transCtx.Transfer.IsServer() {
		return utils.GetPath("",
			leaf(transCtx.Rule.LocalDir),
			leaf(transCtx.LocalAgent.SendDir),
			branch(transCtx.LocalAgent.RootDir),
			leaf(transCtx.Paths.DefaultOutDir),
			branch(transCtx.Paths.GatewayHome))
	} else {
		return utils.GetPath("",
			leaf(transCtx.Rule.LocalDir),
			leaf(transCtx.Paths.DefaultOutDir),
			branch(transCtx.Paths.GatewayHome))
	}
}

//nolint:dupl // factorizing would add complexity
func makeInDir(transCtx *model.TransferContext) (*url.URL, error) {
	if transCtx.Transfer.IsServer() {
		return utils.GetPath("",
			leaf(transCtx.Rule.LocalDir),
			leaf(transCtx.LocalAgent.ReceiveDir),
			branch(transCtx.LocalAgent.RootDir),
			leaf(transCtx.Paths.DefaultInDir),
			branch(transCtx.Paths.GatewayHome))
	} else {
		return utils.GetPath("",
			leaf(transCtx.Rule.LocalDir),
			leaf(transCtx.Paths.DefaultInDir),
			branch(transCtx.Paths.GatewayHome))
	}
}

func makeTmpDir(transCtx *model.TransferContext) (*url.URL, error) {
	if transCtx.Transfer.IsServer() {
		return utils.GetPath("",
			leaf(transCtx.Rule.TmpLocalRcvDir),
			leaf(transCtx.Rule.LocalDir),
			leaf(transCtx.LocalAgent.TmpReceiveDir),
			leaf(transCtx.LocalAgent.ReceiveDir),
			branch(transCtx.LocalAgent.RootDir),
			leaf(transCtx.Paths.DefaultTmpDir),
			leaf(transCtx.Paths.DefaultInDir),
			branch(transCtx.Paths.GatewayHome))
	} else {
		return utils.GetPath("",
			leaf(transCtx.Rule.TmpLocalRcvDir),
			leaf(transCtx.Rule.LocalDir),
			leaf(transCtx.Paths.DefaultTmpDir),
			leaf(transCtx.Paths.DefaultInDir),
			branch(transCtx.Paths.GatewayHome))
	}
}

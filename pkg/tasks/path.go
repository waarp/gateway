package tasks

import (
	"net/url"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

//nolint:dupl // factorizing would add complexity
func makeOutDir(transCtx *model.TransferContext) (*url.URL, error) {
	if transCtx.Transfer.IsServer() {
		return utils.GetPath("",
			utils.Leaf(transCtx.Rule.LocalDir),
			utils.Leaf(transCtx.LocalAgent.SendDir),
			utils.Branch(transCtx.LocalAgent.RootDir),
			utils.Leaf(transCtx.Paths.DefaultOutDir),
			utils.Branch(transCtx.Paths.GatewayHome))
	} else {
		return utils.GetPath("",
			utils.Leaf(transCtx.Rule.LocalDir),
			utils.Leaf(transCtx.Paths.DefaultOutDir),
			utils.Branch(transCtx.Paths.GatewayHome))
	}
}

//nolint:dupl // factorizing would add complexity
func makeInDir(transCtx *model.TransferContext) (*url.URL, error) {
	if transCtx.Transfer.IsServer() {
		return utils.GetPath("",
			utils.Leaf(transCtx.Rule.LocalDir),
			utils.Leaf(transCtx.LocalAgent.ReceiveDir),
			utils.Branch(transCtx.LocalAgent.RootDir),
			utils.Leaf(transCtx.Paths.DefaultInDir),
			utils.Branch(transCtx.Paths.GatewayHome))
	} else {
		return utils.GetPath("",
			utils.Leaf(transCtx.Rule.LocalDir),
			utils.Leaf(transCtx.Paths.DefaultInDir),
			utils.Branch(transCtx.Paths.GatewayHome))
	}
}

func makeTmpDir(transCtx *model.TransferContext) (*url.URL, error) {
	if transCtx.Transfer.IsServer() {
		return utils.GetPath("",
			utils.Leaf(transCtx.Rule.TmpLocalRcvDir),
			utils.Leaf(transCtx.Rule.LocalDir),
			utils.Leaf(transCtx.LocalAgent.TmpReceiveDir),
			utils.Leaf(transCtx.LocalAgent.ReceiveDir),
			utils.Branch(transCtx.LocalAgent.RootDir),
			utils.Leaf(transCtx.Paths.DefaultTmpDir),
			utils.Leaf(transCtx.Paths.DefaultInDir),
			utils.Branch(transCtx.Paths.GatewayHome))
	} else {
		return utils.GetPath("",
			utils.Leaf(transCtx.Rule.TmpLocalRcvDir),
			utils.Leaf(transCtx.Rule.LocalDir),
			utils.Leaf(transCtx.Paths.DefaultTmpDir),
			utils.Leaf(transCtx.Paths.DefaultInDir),
			utils.Branch(transCtx.Paths.GatewayHome))
	}
}

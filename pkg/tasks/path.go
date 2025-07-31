package tasks

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:wrapcheck // wrapping errors adds nothing here
func makeOutDir(transCtx *model.TransferContext, _ string) (string, error) {
	leaf, branch := utils.Leaf, utils.Branch

	if transCtx.Transfer.IsServer() {
		return utils.GetPath("",
			leaf(transCtx.Rule.LocalDir),
			leaf(transCtx.LocalAgent.SendDir),
			branch(transCtx.LocalAgent.RootDir),
			leaf(transCtx.Paths.DefaultOutDir),
			branch(transCtx.Paths.GatewayHome))
	}

	return utils.GetPath("",
		leaf(transCtx.Rule.LocalDir),
		leaf(transCtx.Paths.DefaultOutDir),
		branch(transCtx.Paths.GatewayHome))
}

//nolint:wrapcheck // wrapping errors adds nothing here
func makeInDir(transCtx *model.TransferContext, _ string) (string, error) {
	leaf, branch := utils.Leaf, utils.Branch

	if transCtx.Transfer.IsServer() {
		return utils.GetPath("",
			leaf(transCtx.Rule.LocalDir),
			leaf(transCtx.LocalAgent.ReceiveDir),
			branch(transCtx.LocalAgent.RootDir),
			leaf(transCtx.Paths.DefaultInDir),
			branch(transCtx.Paths.GatewayHome))
	}

	return utils.GetPath("",
		leaf(transCtx.Rule.LocalDir),
		leaf(transCtx.Paths.DefaultInDir),
		branch(transCtx.Paths.GatewayHome))
}

//nolint:wrapcheck // wrapping errors adds nothing here
func makeTmpDir(transCtx *model.TransferContext, _ string) (string, error) {
	leaf, branch := utils.Leaf, utils.Branch

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
	}

	return utils.GetPath("",
		leaf(transCtx.Rule.TmpLocalRcvDir),
		leaf(transCtx.Rule.LocalDir),
		leaf(transCtx.Paths.DefaultTmpDir),
		leaf(transCtx.Paths.DefaultInDir),
		branch(transCtx.Paths.GatewayHome))
}

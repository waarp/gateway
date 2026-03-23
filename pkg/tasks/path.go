package tasks

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:wrapcheck // wrapping errors adds nothing here
func makeOutDir(transCtx *model.TransferContext, _ string) (string, error) {
	ruleDir := transCtx.Rule.LocalDir
	if !transCtx.Rule.IsSend {
		ruleDir = ""
	}

	leaf, branch := utils.Leaf, utils.Branch

	if transCtx.Transfer.IsServer() {
		return utils.GetPath("",
			leaf(ruleDir),
			leaf(transCtx.LocalAgent.SendDir),
			branch(transCtx.LocalAgent.RootDir),
			leaf(transCtx.Paths.DefaultOutDir),
			branch(transCtx.Paths.GatewayHome))
	}

	return utils.GetPath("",
		leaf(ruleDir),
		leaf(transCtx.Paths.DefaultOutDir),
		branch(transCtx.Paths.GatewayHome))
}

//nolint:wrapcheck // wrapping errors adds nothing here
func makeInDir(transCtx *model.TransferContext, _ string) (string, error) {
	ruleDir := transCtx.Rule.LocalDir
	if transCtx.Rule.IsSend {
		ruleDir = ""
	}

	leaf, branch := utils.Leaf, utils.Branch

	if transCtx.Transfer.IsServer() {
		return utils.GetPath("",
			leaf(ruleDir),
			leaf(transCtx.LocalAgent.ReceiveDir),
			branch(transCtx.LocalAgent.RootDir),
			leaf(transCtx.Paths.DefaultInDir),
			branch(transCtx.Paths.GatewayHome))
	}

	return utils.GetPath("",
		leaf(ruleDir),
		leaf(transCtx.Paths.DefaultInDir),
		branch(transCtx.Paths.GatewayHome))
}

//nolint:wrapcheck // wrapping errors adds nothing here
func makeTmpDir(transCtx *model.TransferContext, _ string) (string, error) {
	tmpRuleDir := transCtx.Rule.TmpLocalRcvDir
	ruleDir := transCtx.Rule.LocalDir
	if transCtx.Rule.IsSend {
		tmpRuleDir, ruleDir = "", ""
	}

	leaf, branch := utils.Leaf, utils.Branch

	if transCtx.Transfer.IsServer() {
		return utils.GetPath("",
			leaf(tmpRuleDir),
			leaf(ruleDir),
			leaf(transCtx.LocalAgent.TmpReceiveDir),
			leaf(transCtx.LocalAgent.ReceiveDir),
			branch(transCtx.LocalAgent.RootDir),
			leaf(transCtx.Paths.DefaultTmpDir),
			leaf(transCtx.Paths.DefaultInDir),
			branch(transCtx.Paths.GatewayHome))
	}

	return utils.GetPath("",
		leaf(tmpRuleDir),
		leaf(ruleDir),
		leaf(transCtx.Paths.DefaultTmpDir),
		leaf(transCtx.Paths.DefaultInDir),
		branch(transCtx.Paths.GatewayHome))
}

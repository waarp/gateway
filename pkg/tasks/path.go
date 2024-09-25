package tasks

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:dupl // factorizing would add complexity
func makeOutDir(transCtx *model.TransferContext) (*types.FSPath, error) {
	leaf, branch := utils.Leaf, utils.Branch

	var (
		backend, path string
		err           error
	)

	if transCtx.Transfer.IsServer() {
		backend, path, err = utils.GetPath("",
			leaf(transCtx.Rule.LocalDir),
			leaf(transCtx.LocalAgent.SendDir),
			branch(transCtx.LocalAgent.RootDir),
			leaf(transCtx.Paths.DefaultOutDir),
			branch(transCtx.Paths.GatewayHome))
	} else {
		backend, path, err = utils.GetPath("",
			leaf(transCtx.Rule.LocalDir),
			leaf(transCtx.Paths.DefaultOutDir),
			branch(transCtx.Paths.GatewayHome))
	}

	return &types.FSPath{Backend: backend, Path: path}, err
}

//nolint:dupl // factorizing would add complexity
func makeInDir(transCtx *model.TransferContext) (*types.FSPath, error) {
	leaf, branch := utils.Leaf, utils.Branch

	var (
		backend, path string
		err           error
	)

	if transCtx.Transfer.IsServer() {
		backend, path, err = utils.GetPath("",
			leaf(transCtx.Rule.LocalDir),
			leaf(transCtx.LocalAgent.ReceiveDir),
			branch(transCtx.LocalAgent.RootDir),
			leaf(transCtx.Paths.DefaultInDir),
			branch(transCtx.Paths.GatewayHome))
	} else {
		backend, path, err = utils.GetPath("",
			leaf(transCtx.Rule.LocalDir),
			leaf(transCtx.Paths.DefaultInDir),
			branch(transCtx.Paths.GatewayHome))
	}

	return &types.FSPath{Backend: backend, Path: path}, err
}

func makeTmpDir(transCtx *model.TransferContext) (*types.FSPath, error) {
	leaf, branch := utils.Leaf, utils.Branch

	var (
		backend, path string
		err           error
	)

	if transCtx.Transfer.IsServer() {
		backend, path, err = utils.GetPath("",
			leaf(transCtx.Rule.TmpLocalRcvDir),
			leaf(transCtx.Rule.LocalDir),
			leaf(transCtx.LocalAgent.TmpReceiveDir),
			leaf(transCtx.LocalAgent.ReceiveDir),
			branch(transCtx.LocalAgent.RootDir),
			leaf(transCtx.Paths.DefaultTmpDir),
			leaf(transCtx.Paths.DefaultInDir),
			branch(transCtx.Paths.GatewayHome))
	} else {
		backend, path, err = utils.GetPath("",
			leaf(transCtx.Rule.TmpLocalRcvDir),
			leaf(transCtx.Rule.LocalDir),
			leaf(transCtx.Paths.DefaultTmpDir),
			leaf(transCtx.Paths.DefaultInDir),
			branch(transCtx.Paths.GatewayHome))
	}

	return &types.FSPath{Backend: backend, Path: path}, err
}

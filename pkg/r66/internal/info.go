package internal

import (
	"path"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-r66/r66"
)

// UpdateServerInfo updates the server pipeline transfer's attributes with the
// ones given.
func UpdateServerInfo(info *r66.UpdateInfo, pip *pipeline.Pipeline) error {
	if err := UpdateInfo(info, pip); err != nil {
		return ToR66Error(err)
	}
	return nil
}

// UpdateInfo updates the pipeline transfer's attributes with the ones given.
func UpdateInfo(info *r66.UpdateInfo, pip *pipeline.Pipeline) *types.TransferError {
	if pip.TransCtx.Transfer.Step >= types.StepData {
		return nil //cannot update transfer info after data transfer started
	}

	var cols []string
	if info.Filename != "" {
		pip.TransCtx.Transfer.LocalPath = path.Base(info.Filename)
		pip.TransCtx.Transfer.RemotePath = path.Base(info.Filename)
		pip.RebuildFilepaths()
		cols = append(cols, "local_path", "remote_path")
	}

	if info.FileSize >= 0 {
		pip.TransCtx.Transfer.Filesize = info.FileSize
		cols = append(cols, "filesize")
	}

	if len(cols) > 0 {
		if err := pip.UpdateTrans(cols...); err != nil {
			return err
		}
	}

	return nil
}

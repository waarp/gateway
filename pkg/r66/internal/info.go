package internal

import (
	"path"

	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

// UpdateInfo updates the pipeline transfer's attributes with the ones given.
func UpdateInfo(info *r66.UpdateInfo, pip *pipeline.Pipeline) *types.TransferError {
	if pip.TransCtx.Transfer.Step >= types.StepData {
		return nil // cannot update transfer info after data transfer started
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

// Package internal regroups utilities for the R66 protocol module.
package internal

import (
	"encoding/json"
	"strings"

	"code.waarp.fr/lib/log"
	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

const (
	// FollowID defines the name of the transfer info value containing the R66
	// follow ID.
	FollowID = "__followID__"

	// UserContent defines the name of the transfer info value containing the R66
	// user content.
	UserContent = "__userContent__"
)

// UpdateFileInfo updates the pipeline file info with the ones given.
func UpdateFileInfo(info *r66.UpdateInfo, pip *pipeline.Pipeline) error {
	if pip.TransCtx.Transfer.Step >= types.StepData {
		return nil // cannot update file info after data transfer started
	}

	if info.Filename != "" {
		newFile := strings.TrimLeft(info.Filename, "/")
		newFile = strings.TrimPrefix(newFile, pip.TransCtx.Rule.RemoteDir)
		newFile = strings.TrimLeft(newFile, "/")

		if err := pip.RebuildFilepaths(newFile); err != nil {
			return err
		}
	}

	if info.FileSize >= 0 {
		pip.TransCtx.Transfer.Filesize = info.FileSize
	}

	if err := pip.UpdateTrans(); err != nil {
		return err
	}

	/*
		if info.FileInfo != nil && info.FileInfo.SystemData.FollowID != 0 {
			pip.TransCtx.FileInfo[FollowID] = info.FileInfo.SystemData.FollowID
			if err := pip.TransCtx.Transfer.SetFileInfo(pip.DB, pip.TransCtx.FileInfo); err != nil {
				pip.Logger.Errorf("Failed to update transfer info: %s", err)

				return types.NewTransferError(types.TeInternal, "database error")
			}
		}
	*/

	return nil
}

// UpdateTransferInfo updates the pipeline transfer info with the ones given.
func UpdateTransferInfo(userContent string, pip *pipeline.Pipeline) *pipeline.Error {
	if pip.TransCtx.Transfer.Step >= types.StepData {
		return nil // cannot update transfer info after data transfer started
	}

	if uContent := []byte(userContent); json.Valid(uContent) {
		if err := json.Unmarshal(uContent, &pip.TransCtx.TransInfo); err != nil {
			pip.Logger.Error("Failed to unmarshall transfer info: %s", err)

			return pipeline.NewErrorWith(types.TeInternal, "failed to parse transfer info", err)
		}
	} else {
		pip.TransCtx.TransInfo[UserContent] = userContent
	}

	if err := pip.TransCtx.Transfer.SetTransferInfo(pip.DB, pip.TransCtx.TransInfo); err != nil {
		pip.Logger.Error("Failed to update transfer info: %s", err)

		return pipeline.NewError(types.TeInternal, "database error")
	}

	return nil
}

/*
// MakeFileInfo fills the given r66.TransferData instance with file information
// relating to the given transfer pipeline.
func MakeFileInfo(pip *pipeline.Pipeline, info *r66.SystemData) *types.TransferError {
	var fID float64

	if follow, ok := pip.TransCtx.FileInfo[FollowID]; ok {
		if fID, ok = follow.(float64); !ok {
			pip.Logger.Errorf("Invalid type '%T' for R66 follow ID", follow)

			return types.NewTransferError(types.TeInternal, "failed to make file info")
		}
	}

	info.FollowID = int(fID)

	return nil
}
*/

// MakeTransferInfo fills the given r66.TransferData instance with transfer information
// relating to the given transfer pipeline.
func MakeTransferInfo(logger *log.Logger, transCtx *model.TransferContext,
	info *r66.TransferData,
) error {
	var fID float64

	if follow, ok := transCtx.TransInfo[FollowID]; ok {
		if fID, ok = follow.(float64); !ok {
			logger.Error("Invalid type '%T' for R66 follow ID", follow)

			return pipeline.NewError(types.TeInternal, "failed to make file info")
		}
	}

	info.SystemData.FollowID = int(fID)

	userContent, err := MakeUserContent(logger, transCtx.TransInfo)
	if err != nil {
		return err
	}

	info.UserContent = userContent

	return nil
}

// MakeUserContent returns a string containing the marshaled transfer infos.
func MakeUserContent(logger *log.Logger, transInfo map[string]any) (string, *pipeline.Error) {
	var userContent string

	if cont, ok := transInfo[UserContent]; ok {
		if userContent, ok = cont.(string); !ok {
			logger.Error("Invalid type '%T' for R66 user content", cont)

			return "", pipeline.NewError(types.TeInternal, "failed to make transfer info")
		}
	} else {
		cont, err := json.Marshal(transInfo)
		if err != nil {
			logger.Error("Failed to marshal transfer info: %s", err)

			return "", pipeline.NewErrorWith(types.TeInternal, "failed to make transfer info", err)
		}

		userContent = string(cont)
	}

	return userContent, nil
}

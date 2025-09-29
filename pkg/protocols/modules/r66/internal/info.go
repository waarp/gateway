// Package internal regroups utilities for the R66 protocol module.
package internal

import (
	"encoding/json"
	"maps"
	"strings"

	"code.waarp.fr/lib/log"
	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

// UserContent defines the name of the transfer info value containing the R66
// user content.
const UserContent = "__userContent__"

// UpdateFileInfo updates the pipeline file info with the ones given.
func UpdateFileInfo(info *r66.UpdateInfo, pip *pipeline.Pipeline) *pipeline.Error {
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

	if info.FileInfo != nil {
		if info.FileInfo.SystemData.FollowID > 0 {
			pip.TransCtx.TransInfo[model.FollowID] = info.FileInfo.SystemData.FollowID
		}

		if info.FileInfo.UserContent != "" {
			if err := UpdateTransferInfo(info.FileInfo.UserContent, pip); err != nil {
				return err
			}
		}
	}

	return pip.UpdateTrans()
}

// UpdateTransferInfo updates the pipeline transfer info with the ones given.
func UpdateTransferInfo(userContent string, pip *pipeline.Pipeline) *pipeline.Error {
	if pip.TransCtx.Transfer.Step >= types.StepData {
		return nil // cannot update transfer info after data transfer started
	}

	info := map[string]any{}
	if !pip.IsServer() {
		info = pip.TransCtx.TransInfo
	}

	if uContent := []byte(userContent); json.Valid(uContent) {
		if err := json.Unmarshal(uContent, &info); err != nil {
			pip.Logger.Errorf("Failed to unmarshall transfer info: %s", err)

			return pipeline.NewErrorWith(types.TeInternal, "failed to parse transfer info", err)
		}
	} else {
		pip.TransCtx.TransInfo[UserContent] = userContent
	}

	if pip.IsServer() {
		maps.Copy(info, pip.TransCtx.TransInfo)
		pip.TransCtx.TransInfo = info
	}

	if err := pip.TransCtx.Transfer.SetTransferInfo(pip.DB, pip.TransCtx.TransInfo); err != nil {
		pip.Logger.Errorf("Failed to update transfer info: %v", err)

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
func MakeTransferInfo(pip *pipeline.Pipeline, info *r66.TransferData) error {
	if err := makeTransferInfo(pip.Logger, pip.TransCtx, info); err != nil {
		pip.SetError(err.Code(), err.Details())

		return err
	}

	return nil
}

func makeTransferInfo(logger *log.Logger, transCtx *model.TransferContext,
	info *r66.TransferData,
) *pipeline.Error {
	follow, hasFollow := transCtx.TransInfo[model.FollowID]
	if !hasFollow {
		return nil
	}

	switch fID := follow.(type) {
	case json.Number:
		fID64, err := fID.Int64()
		if err != nil {
			logger.Errorf("Could not parse the R66 follow ID: %v", err)

			return pipeline.NewError(types.TeInternal, "failed to parse follow ID")
		}

		info.SystemData.FollowID = int(fID64)
	case int:
		info.SystemData.FollowID = fID
	}

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
			logger.Errorf("Invalid type '%T' for R66 user content", cont)

			return "", pipeline.NewError(types.TeInternal, "failed to make transfer info")
		}
	} else {
		userContentMap := maps.Clone(transInfo)
		delete(userContentMap, model.FollowID)

		cont, err := json.Marshal(userContentMap)
		if err != nil {
			logger.Errorf("Failed to marshal transfer info: %v", err)

			return "", pipeline.NewErrorWith(types.TeInternal, "failed to make transfer info", err)
		}

		userContent = string(cont)
	}

	return userContent, nil
}

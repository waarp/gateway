package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func jsonTransToDbHist(trans *file.Transfer) *model.HistoryEntry {
	return &model.HistoryEntry{
		ID:               trans.ID,
		RemoteTransferID: trans.RemoteID,
		IsServer:         trans.IsServer,
		IsSend:           trans.IsSend,
		Rule:             trans.Rule,
		Client:           trans.Client,
		Account:          trans.Requester,
		Agent:            trans.Requested,
		Protocol:         trans.Protocol,
		SrcFilename:      trans.SrcFilename,
		DestFilename:     trans.DestFilename,
		LocalPath:        trans.LocalFilepath,
		RemotePath:       trans.RemoteFilepath,
		Filesize:         trans.Filesize,
		Start:            trans.Start,
		Stop:             trans.Stop,
		Status:           trans.Status,
		Step:             trans.Step,
		Progress:         trans.Progress,
		TaskNumber:       trans.TaskNumber,
		ErrCode:          trans.ErrorCode,
		ErrDetails:       trans.ErrorMsg,
	}
}

var ErrInvalidJSONInput = errors.New("invalid JSON input")

func importHistory(ses *database.Session, r io.Reader) (int64, error) {
	decoder := json.NewDecoder(r)

	if tok, tokErr := decoder.Token(); tokErr != nil {
		return 0, fmt.Errorf("failed to parse JSON input: %w", tokErr)
	} else if tok != json.Delim('[') {
		return 0, fmt.Errorf("%w: expected array start, got '%v'", ErrInvalidJSONInput, tok)
	}

	var maxID int64

	for decoder.More() {
		var trans file.Transfer
		if err := decoder.Decode(&trans); err != nil {
			return 0, fmt.Errorf("failed to parse JSON history entry: %w", err)
		}

		if trans.ID > maxID {
			maxID = trans.ID
		}

		h := jsonTransToDbHist(&trans)
		if err := ses.Insert(h).Run(); err != nil {
			return 0, fmt.Errorf("failed to insert history entry %d: %w", trans.ID, err)
		}

		if err := h.SetTransferInfo(ses, trans.TransferInfo); err != nil {
			return 0, fmt.Errorf("failed to insert info of transfer %d: %w", trans.ID, err)
		}
	}

	if tok, err := decoder.Token(); err != nil {
		return 0, fmt.Errorf("failed to parse JSON input: %w", err)
	} else if tok != json.Delim(']') {
		return 0, fmt.Errorf("%w: expected array end, got '%v'", ErrInvalidJSONInput, tok)
	}

	return maxID, nil
}

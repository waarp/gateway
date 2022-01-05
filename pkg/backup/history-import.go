package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func jsonTransToDbHist(trans *file.Transfer) *model.HistoryEntry {
	return &model.HistoryEntry{
		ID:               trans.ID,
		RemoteTransferID: trans.RemoteID,
		IsServer:         trans.IsServer,
		IsSend:           trans.IsSend,
		Rule:             trans.Rule,
		Account:          trans.Requester,
		Agent:            trans.Requested,
		Protocol:         trans.Protocol,
		LocalPath:        trans.LocalFilepath,
		RemotePath:       trans.RemoteFilepath,
		Filesize:         trans.Filesize,
		Start:            trans.Start,
		Stop:             trans.Stop,
		Status:           trans.Status,
		Step:             trans.Step,
		Progress:         trans.Progress,
		TaskNumber:       trans.TaskNumber,
		Error:            types.TransferError{Code: trans.ErrorCode, Details: trans.ErrorMsg},
	}
}

var ErrInvalidJSONInput = errors.New("invalid JSON input")

func importHistory(db database.Access, r io.Reader) error {
	decoder := json.NewDecoder(r)

	if tok, err := decoder.Token(); err != nil {
		return fmt.Errorf("failed to parse JSON input: %w", err)
	} else if tok != json.Delim('[') {
		return fmt.Errorf("%w: expected array start, got '%v'", ErrInvalidJSONInput, tok)
	}

	for decoder.More() {
		var trans file.Transfer
		if err := decoder.Decode(&trans); err != nil {
			return fmt.Errorf("failed to parse JSON history entry: %w", err)
		}

		h := jsonTransToDbHist(&trans)

		if err := db.Insert(h).Run(); err != nil {
			return err
		}

		if err := h.SetTransferInfo(db, trans.TransferInfo); err != nil {
			return err
		}
	}

	if tok, err := decoder.Token(); err != nil {
		return fmt.Errorf("failed to parse JSON input: %w", err)
	} else if tok != json.Delim(']') {
		return fmt.Errorf("%w: expected array end, got '%v'", ErrInvalidJSONInput, tok)
	}

	return nil
}

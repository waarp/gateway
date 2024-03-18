package backup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func dbHistToFileTrans(hist *model.HistoryEntry, db database.ReadAccess) (*file.Transfer, error) {
	info, err := hist.GetTransferInfo(db)
	if err != nil {
		return nil, err
	}

	return &file.Transfer{
		ID:             hist.ID,
		RemoteID:       hist.RemoteTransferID,
		Rule:           hist.Rule,
		IsSend:         hist.IsSend,
		IsServer:       hist.IsServer,
		Requester:      hist.Account,
		Requested:      hist.Agent,
		Protocol:       hist.Protocol,
		SrcFilename:    hist.SrcFilename,
		DestFilename:   hist.DestFilename,
		LocalFilepath:  hist.LocalPath.String(),
		RemoteFilepath: hist.RemotePath,
		Filesize:       hist.Filesize,
		Start:          hist.Start,
		Stop:           hist.Stop,
		Status:         hist.Status,
		Step:           hist.Step,
		Progress:       hist.Progress,
		TaskNumber:     hist.TaskNumber,
		ErrorCode:      hist.Error.Code,
		ErrorMsg:       hist.Error.Details,
		TransferInfo:   info,
	}, nil
}

func encodeHistEntry(db database.ReadAccess, hist *model.HistoryEntry, w io.Writer) error {
	trans, err := dbHistToFileTrans(hist, db)
	if err != nil {
		return err
	}

	jTrans, err := json.Marshal(trans)
	if err != nil {
		return fmt.Errorf("failed to marshal history entry: %w", err)
	}

	buf := &bytes.Buffer{}
	if err := json.Indent(buf, jTrans, "  ", "  "); err != nil {
		return fmt.Errorf("failed to indent the JSON history entry: %w", err)
	}

	if _, err := w.Write(bytes.TrimSpace(buf.Bytes())); err != nil {
		return fmt.Errorf("failed to write the JSON to output: %w", err)
	}

	return nil
}

func ExportHistory(db database.Access, w io.Writer, olderThan time.Time) error {
	const sliceSize = 20

	for i := 0; ; i += sliceSize {
		var transfers model.HistoryEntries
		query := db.Select(&transfers).OrderBy("id", true).Limit(sliceSize, i)

		if !olderThan.IsZero() {
			query.Where("start <= ?", olderThan.UTC().Truncate(time.Microsecond).
				Format(time.RFC3339Nano))
		}

		if err := query.Run(); err != nil {
			return err
		}

		if len(transfers) == 0 {
			return nil
		}

		fmt.Fprintln(w, "[")

		defer fmt.Fprintln(w, "\n]")

		for i, hist := range transfers {
			if i == 0 {
				fmt.Fprint(w, "  ")
			} else {
				fmt.Fprint(w, ",\n  ")
			}

			if err := encodeHistEntry(db, hist, w); err != nil {
				return err
			}
		}
	}
}

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
		LocalFilepath:  hist.LocalPath,
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

func encodeHistEntry(db database.ReadAccess, rows *database.Iterator, w io.Writer) error {
	var hist model.HistoryEntry
	if err := rows.Scan(&hist); err != nil {
		return fmt.Errorf("failed to parse history entry: %w", err)
	}

	trans, err := dbHistToFileTrans(&hist, db)
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
	query := db.Iterate(&model.HistoryEntry{})

	if !olderThan.IsZero() {
		query.Where("start <= ?", olderThan.UTC().Truncate(time.Microsecond).
			Format(time.RFC3339Nano))
	}

	rows, err := query.Run()
	if err != nil {
		return err
	}

	defer rows.Close()
	defer fmt.Fprintln(w, "\n]")

	fmt.Fprintln(w, "[")

	if !rows.Next() {
		return nil
	}

	fmt.Fprint(w, "  ")

	if err := encodeHistEntry(db, rows, w); err != nil {
		return err
	}

	for rows.Next() {
		fmt.Fprint(w, ",\n  ")

		if err := encodeHistEntry(db, rows, w); err != nil {
			return err
		}
	}

	return nil
}

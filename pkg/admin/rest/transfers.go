package rest

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// restTransferToDB transforms the JSON transfer into its database equivalent.
func restTransferToDB(jTrans *api.InTransfer, db *database.DB, logger *log.Logger) (*model.Transfer, error) {
	rule, account, client, err := getTransInfo(db, jTrans)
	if err != nil {
		return nil, err
	}

	srcFile := jTrans.File
	destFile := jTrans.Output

	if srcFile == "" && jTrans.SourcePath != "" {
		logger.Warning("JSON field 'sourcePath' is deprecated, use 'file' instead")

		srcFile = utils.DenormalizePath(jTrans.SourcePath)
	}

	if destFile == "" {
		if jTrans.DestPath != "" {
			logger.Warning("JSON field 'destPath' is deprecated, use  'output' instead")

			destFile = utils.DenormalizePath(jTrans.DestPath)
		} else {
			destFile = filepath.Base(srcFile)
		}
	}

	start := jTrans.Start

	if jTrans.StartDate.Valid {
		logger.Warning("JSON field 'startDate' is deprecated, use 'start' instead")

		if start.IsZero() {
			start = jTrans.StartDate.Value
		}
	}

	return &model.Transfer{
		RuleID:          rule.ID,
		ClientID:        utils.NewNullInt64(client.ID),
		RemoteAccountID: utils.NewNullInt64(account.ID),
		SrcFilename:     srcFile,
		DestFilename:    destFile,
		Filesize:        model.UnknownSize,
		Start:           start,
	}, nil
}

// DBTransferToREST transforms the given database transfer into its JSON equivalent.
func DBTransferToREST(db *database.DB, trans *model.NormalizedTransferView) (*api.OutTransfer, error) {
	src := path.Base(trans.RemotePath)
	dst := trans.LocalPath.OSPath()

	if trans.IsSend {
		dst = path.Base(trans.RemotePath)
		src = trans.LocalPath.OSPath()
	}

	var stop api.Nullable[time.Time]
	if !trans.Stop.IsZero() {
		stop = api.AsNullable(trans.Stop)
	}

	info, iErr := trans.GetTransferInfo(db)
	if iErr != nil {
		return nil, fmt.Errorf("failed to retrieve transfer info: %w", iErr)
	}

	return &api.OutTransfer{
		ID:             trans.ID,
		RemoteID:       trans.RemoteTransferID,
		Rule:           trans.Rule,
		IsServer:       trans.IsServer,
		IsSend:         trans.IsSend,
		Requested:      trans.Agent,
		Requester:      trans.Account,
		Protocol:       trans.Protocol,
		SrcFilename:    trans.SrcFilename,
		DestFilename:   trans.DestFilename,
		LocalFilepath:  trans.LocalPath.OSPath(),
		RemoteFilepath: trans.RemotePath,
		Filesize:       trans.Filesize,
		Start:          trans.Start,
		Stop:           stop,
		Status:         trans.Status,
		Step:           trans.Step.String(),
		Progress:       trans.Progress,
		TaskNumber:     trans.TaskNumber,
		ErrorCode:      trans.ErrCode.String(),
		ErrorMsg:       trans.ErrDetails,
		TransferInfo:   info,
		TrueFilepath:   trans.LocalPath.OSPath(),
		SourcePath:     src,
		DestPath:       dst,
		StartDate:      trans.Start,
	}, nil
}

// DBTransfersToREST transforms the given list of database transfers into its
// JSON equivalent.
func DBTransfersToREST(db *database.DB, models []*model.NormalizedTransferView) ([]*api.OutTransfer, error) {
	jsonArray := make([]*api.OutTransfer, len(models))

	for i, trans := range models {
		jsonObj, err := DBTransferToREST(db, trans)
		if err != nil {
			return nil, err
		}

		jsonArray[i] = jsonObj
	}

	return jsonArray, nil
}

//nolint:dupl // dupicated code is about a different type
func getDBTrans(r *http.Request, db *database.DB) (*model.Transfer, error) {
	val := mux.Vars(r)["transfer"]

	id, parsErr := strconv.ParseUint(val, 10, 64) //nolint:gomnd // useless to add a constant for that
	if parsErr != nil || id == 0 {
		return nil, notFound("'%s' is not a valid transfer ID", val)
	}

	var transfer model.Transfer
	if err := db.Get(&transfer, "id=? AND owner=?", id, conf.GlobalConfig.GatewayName).
		Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFound("transfer %v not found", id)
		}

		return nil, fmt.Errorf("failed to retrieve transfer %d: %w", id, err)
	}

	return &transfer, nil
}

//nolint:dupl // dupicated code is about a different type
func getDBTransView(r *http.Request, db *database.DB) (*model.NormalizedTransferView, error) {
	val := mux.Vars(r)["transfer"]

	id, parsErr := strconv.ParseUint(val, 10, 64) //nolint:gomnd // useless to add a constant for that
	if parsErr != nil || id == 0 {
		return nil, notFound("'%s' is not a valid transfer ID", val)
	}

	var transfer model.NormalizedTransferView
	if err := db.Get(&transfer, "id=? AND owner=?", id, conf.GlobalConfig.GatewayName).
		Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFound("transfer %v not found", id)
		}

		return nil, fmt.Errorf("failed to retrieve transfer %d: %w", id, err)
	}

	return &transfer, nil
}

func addTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var jsonTrans api.InTransfer
		if err := readJSON(r, &jsonTrans); handleError(w, logger, err) {
			return
		}

		trans, err := restTransferToDB(&jsonTrans, db, logger)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Insert(trans).Run(); handleError(w, logger, err) {
			return
		}

		if err := trans.SetTransferInfo(db, jsonTrans.TransferInfo); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, utils.FormatInt(trans.ID)))
		w.WriteHeader(http.StatusCreated)
	}
}

func getTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := getDBTransView(r, db)
		if handleError(w, logger, err) {
			return
		}

		json, err := DBTransferToREST(db, result)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, json))
	}
}

func listTransfers(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var transfers model.NormalizedTransfers

		query, queryErr := parseTransferListQuery(r, db, &transfers)
		if handleError(w, logger, queryErr) {
			return
		}

		if err := query.Where("owner=?", conf.GlobalConfig.GatewayName).
			Run(); handleError(w, logger, err) {
			return
		}

		json, convErr := DBTransfersToREST(db, transfers)
		if handleError(w, logger, convErr) {
			return
		}

		resp := map[string][]*api.OutTransfer{"transfers": json}
		handleError(w, logger, writeJSON(w, resp))
	}
}

func pauseTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		trans, tErr := getDBTrans(r, db)
		if handleError(w, logger, tErr) {
			return
		}

		switch trans.Status {
		case types.StatusPlanned:
			trans.Status = types.StatusPaused
			if err := db.Update(trans).Cols("status").Run(); handleError(w, logger, err) {
				return
			}

			w.WriteHeader(http.StatusAccepted)

			return
		case types.StatusRunning:
			pip := pipeline.List.Get(trans.ID)
			if pip == nil {
				handleError(w, logger, internal("pipeline for transfer %d not found", trans.ID))

				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), time.Second)
			defer cancel()

			if err := pip.Pause(ctx); err != nil {
				handleError(w, logger, err)

				return
			}

			w.WriteHeader(http.StatusAccepted)

			return
		default:
			handleError(w, logger, badRequest("cannot pause an already "+
				"interrupted transfer"))

			return
		}
	}
}

func cancelTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		trans, tErr := getDBTrans(r, db)
		if handleError(w, logger, tErr) {
			return
		}

		if trans.Status == types.StatusRunning {
			pip := pipeline.List.Get(trans.ID)
			if pip == nil {
				handleError(w, logger, internal("pipeline for transfer %d not found", trans.ID))

				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), time.Second)
			defer cancel()

			if err := pip.Cancel(ctx); err != nil {
				handleError(w, logger, err)

				return
			}
		} else {
			trans.Status = types.StatusCancelled
			if err := trans.MoveToHistory(db, logger, time.Time{}); handleError(w, logger, err) {
				return
			}
		}

		r.URL.Path = "/api/history"
		w.Header().Set("Location", location(r.URL, utils.FormatInt(trans.ID)))
		w.WriteHeader(http.StatusAccepted)
	}
}

func resumeTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbTransView, getErr := getDBTransView(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if !dbTransView.IsTransfer {
			handleError(w, logger, badRequest("cannot resume completed transfers"))

			return
		}

		if dbTransView.IsServer {
			handleError(w, logger, badRequest("only the client can restart a transfer"))

			return
		}

		if dbTransView.Status != types.StatusPaused && dbTransView.Status != types.StatusInterrupted &&
			dbTransView.Status != types.StatusError {
			handleError(w, logger, badRequest("cannot resume an already running transfer"))

			return
		}

		var dbHist model.Transfer
		if err := db.Get(&dbHist, "id=?", dbTransView.ID).Run(); handleError(w, logger, err) {
			return
		}

		dbHist.Status = types.StatusPlanned
		dbHist.ErrCode = types.TeOk
		dbHist.ErrDetails = ""

		if err := db.Update(&dbHist).Cols("status", "error_code", "error_details").
			Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func retryTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbTransView, getErr := getDBTransView(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if dbTransView.IsTransfer {
			handleError(w, logger, badRequest("cannot retry non-ended transfer"))

			return
		}

		var dbHist model.HistoryEntry
		if err := db.Get(&dbHist, "id=?", dbTransView.ID).Run(); handleError(w, logger, err) {
			return
		}

		date := time.Now()

		if dateStr := r.FormValue("date"); dateStr != "" {
			var err error
			if date, err = time.Parse(time.RFC3339Nano, dateStr); handleError(w, logger, err) {
				return
			}
		}

		trans, restartErr := dbHist.Restart(db, date)
		if handleError(w, logger, restartErr) {
			return
		}

		if err := db.Insert(trans).Run(); handleError(w, logger, err) {
			return
		}

		r.URL.Path = "/api/transfers"
		w.Header().Set("Location", location(r.URL, utils.FormatInt(trans.ID)))
		w.WriteHeader(http.StatusCreated)
	}
}

func cancelDBTransfer(db *database.DB, logger *log.Logger, w http.ResponseWriter,
	status ...types.TransferStatus,
) bool {
	statuses := make([]interface{}, len(status))

	for i := range status {
		statuses[i] = status[i]
	}

	tErr := db.Transaction(func(ses *database.Session) error {
		for i := 0; ; i += 20 {
			var transfers model.Transfers
			if err := ses.Select(&transfers).Limit(0, i).Run(); err != nil {
				logger.Error("Failed to retrieve transfers: %v", err)

				return fmt.Errorf("failed to retrieve transfers: %w", err)
			}

			if len(transfers) == 0 {
				break
			}

			for _, trans := range transfers {
				trans.Status = types.StatusCancelled

				if err := trans.CopyToHistory(ses, logger, time.Time{}); err != nil {
					return fmt.Errorf("failed to move transfer %d to history: %w", trans.ID, err)
				}
			}
		}

		if err := ses.DeleteAll(&model.Transfer{}).In("status", statuses...).Run(); err != nil {
			return fmt.Errorf("failed to cancel transfers: %w", err)
		}

		return nil
	})

	return !handleError(w, logger, tErr)
}

func cancelRunningTransfers(r *http.Request) bool {
	const cancelTimeout = 2 * time.Second

	ctx, cancel := context.WithTimeout(r.Context(), cancelTimeout)
	defer cancel()

	if err := pipeline.List.CancelAll(ctx); err != nil {
		return false
	}

	return true
}

//nolint:gocognit //there is no way to further simplify this function
func cancelTransfers(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch target := r.FormValue("target"); target {
		case "":
			handleError(w, logger, badRequest("missing 'target' parameter"))

			return
		case "error":
			if !cancelDBTransfer(db, logger, w, types.StatusError) {
				return
			}
		case "planned":
			if !cancelDBTransfer(db, logger, w, types.StatusPlanned) {
				return
			}
		case "paused":
			if !cancelDBTransfer(db, logger, w, types.StatusPaused) {
				return
			}
		case "interrupted":
			if !cancelDBTransfer(db, logger, w, types.StatusInterrupted) {
				return
			}
		case "running":
			if !cancelRunningTransfers(r) {
				return
			}
		case "all":
			if !cancelDBTransfer(db, logger, w, types.StatusError, types.StatusPlanned,
				types.StatusPaused, types.StatusInterrupted) {
				return
			}

			if !cancelRunningTransfers(r) {
				return
			}
		default:
			handleError(w, logger, badRequest("unknown cancel target '%s'", target))

			return
		}

		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, "Transfers canceled successfully")
	}
}

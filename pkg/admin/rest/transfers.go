package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	api "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"github.com/gorilla/mux"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// transToDB transforms the JSON transfer into its database equivalent.
func transToDB(trans *api.InTransfer, db *database.DB) (*model.Transfer, error) {
	ruleID, accountID, agentID, err := getTransIDs(db, trans)
	if err != nil {
		return nil, err
	}
	return &model.Transfer{
		RuleID:     ruleID,
		IsServer:   false,
		AgentID:    agentID,
		AccountID:  accountID,
		SourceFile: trans.SourcePath,
		DestFile:   trans.DestPath,
		Start:      trans.Start,
	}, nil
}

// FromTransfer transforms the given database transfer into its JSON equivalent.
func FromTransfer(db *database.DB, trans *model.Transfer) (*api.OutTransfer, error) {
	rule, requester, requested, err := getTransNames(db, trans)
	if err != nil {
		return nil, err
	}

	return &api.OutTransfer{
		ID:           trans.ID,
		RemoteID:     trans.RemoteTransferID,
		Rule:         rule.Name,
		IsServer:     trans.IsServer,
		IsSend:       rule.IsSend,
		Requested:    requested,
		Requester:    requester,
		TrueFilepath: trans.TrueFilepath,
		SourcePath:   trans.SourceFile,
		DestPath:     trans.DestFile,
		Start:        trans.Start.Local(),
		Status:       trans.Status,
		Step:         trans.Step.String(),
		Progress:     trans.Progress,
		TaskNumber:   trans.TaskNumber,
		ErrorCode:    trans.Error.Code.String(),
		ErrorMsg:     trans.Error.Details,
	}, nil
}

// FromTransfers transforms the given list of database transfers into its
// JSON equivalent.
func FromTransfers(db *database.DB, models []model.Transfer) ([]api.OutTransfer, error) {
	jsonArray := make([]api.OutTransfer, len(models))
	for i, t := range models {
		trans := t
		jsonObj, err := FromTransfer(db, &trans)
		if err != nil {
			return nil, err
		}
		jsonArray[i] = *jsonObj
	}
	return jsonArray, nil
}

func getTrans(r *http.Request, db *database.DB) (*model.Transfer, error) {
	val := mux.Vars(r)["transfer"]
	id, err := strconv.ParseUint(val, 10, 64)
	if err != nil || id == 0 {
		return nil, notFound("'%s' is not a valid transfer ID", val)
	}
	var transfer model.Transfer
	if err := db.Get(&transfer, "id=?", id).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFound("transfer %v not found", id)
		}
		return nil, err
	}
	return &transfer, nil
}

func addTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var jsonTrans api.InTransfer
		if err := readJSON(r, &jsonTrans); handleError(w, logger, err) {
			return
		}

		trans, err := transToDB(&jsonTrans, db)
		if handleError(w, logger, err) {
			return
		}
		if err := db.Insert(trans).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, fmt.Sprint(trans.ID)))
		w.WriteHeader(http.StatusCreated)
	}
}

func getTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := getTrans(r, db)
		if handleError(w, logger, err) {
			return
		}

		json, err := FromTransfer(db, result)
		if handleError(w, logger, err) {
			return
		}

		err = writeJSON(w, json)
		handleError(w, logger, err)
	}
}

func listTransfers(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var transfers model.Transfers
		query, err := parseTransferListQuery(r, db, &transfers)
		if handleError(w, logger, err) {
			return
		}

		if err := query.Run(); handleError(w, logger, err) {
			return
		}

		json, err := FromTransfers(db, transfers)
		if handleError(w, logger, err) {
			return
		}

		resp := map[string][]api.OutTransfer{"transfers": json}
		err = writeJSON(w, resp)
		handleError(w, logger, err)
	}
}

func pauseTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		check, err := getTrans(r, db)
		if handleError(w, logger, err) {
			return
		}

		if check.Status != types.StatusPlanned && check.Status != types.StatusRunning {
			err := badRequest("cannot pause an already interrupted transfer")
			handleError(w, logger, err)
			return
		}

		switch check.Status {
		case types.StatusPlanned:
			check.Status = types.StatusPaused
			if err := db.Update(check).Cols("status").Run(); handleError(w, logger, err) {
				return
			}
		case types.StatusRunning:
			pipeline.Signals.SendSignal(check.ID, model.SignalPause)
		default:
			err := badRequest("cannot pause an already interrupted transfer")
			handleError(w, logger, err)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func cancelTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		check, err := getTrans(r, db)
		if handleError(w, logger, err) {
			return
		}

		if check.Status != types.StatusRunning {
			check.Status = types.StatusCancelled
			if err := pipeline.ToHistory(db, logger, check, time.Time{}); handleError(w, logger, err) {
				return
			}
		} else {
			pipeline.Signals.SendSignal(check.ID, model.SignalCancel)
		}

		r.URL.Path = "/api/history"
		w.Header().Set("Location", location(r.URL, fmt.Sprint(check.ID)))
		w.WriteHeader(http.StatusAccepted)
	}
}

func resumeTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		check, err := getTrans(r, db)
		if handleError(w, logger, err) {
			return
		}

		if check.IsServer {
			handleError(w, logger, badRequest("only the client can restart a transfer"))
			return
		}

		if check.Status != types.StatusPaused && check.Status != types.StatusInterrupted &&
			check.Status != types.StatusError {
			handleError(w, logger, badRequest("cannot resume an already running transfer"))
			return
		}

		check.Status = types.StatusPlanned
		check.Error = types.TransferError{}
		if err := db.Update(check).Cols("status", "error_code", "error_details").
			Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

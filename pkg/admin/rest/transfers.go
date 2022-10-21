package rest

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/proto"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

// transToDB transforms the JSON transfer into its database equivalent.
func transToDB(trans *api.InTransfer, db *database.DB, logger *log.Logger) (*model.Transfer, error) {
	ruleID, accountID, agentID, err := getTransIDs(db, trans)
	if err != nil {
		return nil, err
	}

	file := trans.File
	out := str(trans.Output)

	if file == "" && trans.SourcePath != "" {
		logger.Warning("JSON field 'sourcePath' is deprecated, use 'file' instead")

		file = utils.DenormalizePath(trans.SourcePath)
	}

	if out == "" {
		if trans.DestPath != "" {
			logger.Warning("JSON field 'destPath' is deprecated, use  'output' instead")

			out = utils.DenormalizePath(trans.DestPath)
		} else {
			out = filepath.Base(file)
		}
	}

	start := trans.Start

	if !trans.StartDate.IsZero() {
		logger.Warning("JSON field 'startDate' is deprecated, use 'start' instead")

		if start.IsZero() {
			start = trans.StartDate
		}
	}

	locPath := file
	remPath := strings.TrimPrefix(out, "/")

	if !*trans.IsSend {
		locPath = out
		remPath = strings.TrimPrefix(file, "/")
	}

	return &model.Transfer{
		RuleID:     ruleID,
		IsServer:   false,
		AgentID:    agentID,
		AccountID:  accountID,
		LocalPath:  locPath,
		RemotePath: remPath,
		Filesize:   model.UnknownSize,
		Start:      start,
	}, nil
}

// FromTransfer transforms the given database transfer into its JSON equivalent.
func FromTransfer(db *database.DB, trans *model.Transfer) (*api.OutTransfer, error) {
	rule, requester, requested, protocol, err := getTransNames(db, trans)
	if err != nil {
		return nil, err
	}

	src := path.Base(trans.RemotePath)
	dst := filepath.Base(trans.LocalPath)

	if rule.IsSend {
		dst = path.Base(trans.RemotePath)
		src = filepath.Base(trans.LocalPath)
	}

	info, iErr := trans.GetTransferInfo(db)
	if iErr != nil {
		return nil, iErr
	}

	return &api.OutTransfer{
		ID:             trans.ID,
		RemoteID:       trans.RemoteTransferID,
		Rule:           rule.Name,
		IsServer:       trans.IsServer,
		IsSend:         rule.IsSend,
		Requested:      requested,
		Requester:      requester,
		Protocol:       protocol,
		LocalFilepath:  trans.LocalPath,
		RemoteFilepath: trans.RemotePath,
		Filesize:       trans.Filesize,
		Start:          trans.Start.Local(),
		Status:         trans.Status,
		Step:           trans.Step.String(),
		Progress:       trans.Progress,
		TaskNumber:     trans.TaskNumber,
		ErrorCode:      trans.Error.Code.String(),
		ErrorMsg:       trans.Error.Details,
		TransferInfo:   info,
		TrueFilepath:   trans.LocalPath,
		SourcePath:     src,
		DestPath:       dst,
		StartDate:      trans.Start.Local(),
	}, nil
}

// FromTransfers transforms the given list of database transfers into its
// JSON equivalent.
func FromTransfers(db *database.DB, models []model.Transfer) ([]api.OutTransfer, error) {
	jsonArray := make([]api.OutTransfer, len(models))

	for i := range models {
		trans := &models[i]

		jsonObj, err := FromTransfer(db, trans)
		if err != nil {
			return nil, err
		}

		jsonArray[i] = *jsonObj
	}

	return jsonArray, nil
}

//nolint:dupl // dupicated code is about a different type
func getTrans(r *http.Request, db *database.DB) (*model.Transfer, error) {
	val := mux.Vars(r)["transfer"]

	id, err := strconv.ParseUint(val, 10, 64) //nolint:gomnd // useless to add a constant for that
	if err != nil || id == 0 {
		return nil, notFound("'%s' is not a valid transfer ID", val)
	}

	var transfer model.Transfer
	if err := db.Get(&transfer, "id=? AND owner=?", id, conf.GlobalConfig.GatewayName).
		Run(); err != nil {
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

		trans, err := transToDB(&jsonTrans, db, logger)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Insert(trans).Run(); handleError(w, logger, err) {
			return
		}

		if err := trans.SetTransferInfo(db, jsonTrans.TransferInfo); handleError(w, logger, err) {
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

		if err := query.Where("owner=?", conf.GlobalConfig.GatewayName).
			Run(); handleError(w, logger, err) {
			return
		}

		json, err := FromTransfers(db, transfers)
		if handleError(w, logger, err) {
			return
		}

		resp := map[string][]api.OutTransfer{"transfers": json}
		handleError(w, logger, writeJSON(w, resp))
	}
}

func pauseTransfer(protoServices map[uint64]proto.Service) handler {
	return func(logger *log.Logger, db *database.DB) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			trans, tErr := getTrans(r, db)
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
				pips, err := getPipelineMap(protoServices, trans)
				if handleError(w, logger, err) {
					return
				}

				ctx, cancel := context.WithTimeout(r.Context(), time.Second)
				defer cancel()

				ok, err := pips.Pause(ctx, trans.ID)
				if !ok {
					handleError(w, logger, internal("could not find a "+
						"corresponding pipeline for transfer %d", trans.ID))

					return
				}

				if err != nil {
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
}

func cancelTransfer(protoServices map[uint64]proto.Service) handler {
	return func(logger *log.Logger, db *database.DB) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			trans, tErr := getTrans(r, db)
			if handleError(w, logger, tErr) {
				return
			}

			if trans.Status == types.StatusRunning {
				pips, err := getPipelineMap(protoServices, trans)
				if handleError(w, logger, err) {
					return
				}

				ctx, cancel := context.WithTimeout(r.Context(), time.Second)
				defer cancel()

				ok, err := pips.Cancel(ctx, trans.ID)
				if !ok {
					logger.Warning("Could not find a corresponding pipeline "+
						"for transfer %d", trans.ID)

					trans.Status = types.StatusCancelled
					if err := trans.ToHistory(db, logger, time.Time{}); handleError(w, logger, err) {
						return
					}
				}

				if err != nil {
					handleError(w, logger, err)

					return
				}
			} else {
				trans.Status = types.StatusCancelled
				if err := trans.ToHistory(db, logger, time.Time{}); handleError(w, logger, err) {
					return
				}
			}

			r.URL.Path = "/api/history"
			w.Header().Set("Location", location(r.URL, fmt.Sprint(trans.ID)))
			w.WriteHeader(http.StatusAccepted)
		}
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

package rest

import (
	"fmt"
	"net/http"
	"strconv"

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
		Rule:         rule,
		IsServer:     trans.IsServer,
		Requested:    requested,
		Requester:    requester,
		TrueFilepath: trans.TrueFilepath,
		SourcePath:   trans.SourceFile,
		DestPath:     trans.DestFile,
		Start:        trans.Start,
		Status:       trans.Status,
		Step:         trans.Step,
		Progress:     trans.Progress,
		TaskNumber:   trans.TaskNumber,
		ErrorCode:    trans.Error.Code,
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
	transfer := &model.Transfer{ID: id}
	if err := db.Get(transfer); err != nil {
		if err == database.ErrNotFound {
			return nil, notFound("transfer %v not found", id)
		}
		return nil, err
	}
	return transfer, nil
}

func addTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			jsonTrans := &api.InTransfer{}
			if err := readJSON(r, jsonTrans); err != nil {
				return err
			}

			trans, err := transToDB(jsonTrans, db)
			if err != nil {
				return err
			}
			if err := db.Create(trans); err != nil {
				return err
			}

			w.Header().Set("Location", location(r.URL, fmt.Sprint(trans.ID)))
			w.WriteHeader(http.StatusCreated)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func getTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			result, err := getTrans(r, db)
			if err != nil {
				return err
			}

			json, err := FromTransfer(db, result)
			if err != nil {
				return err
			}
			return writeJSON(w, json)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func listTransfers(logger *log.Logger, db *database.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			filters, err := parseTransferListQuery(r)
			if err != nil {
				return err
			}

			var transfers []model.Transfer
			if err := db.Select(&transfers, filters); err != nil {
				return fmt.Errorf("query failed: %s", err.Error())
			}

			json, err := FromTransfers(db, transfers)
			if err != nil {
				return err
			}

			resp := map[string][]api.OutTransfer{"transfers": json}
			return writeJSON(w, resp)
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func pauseTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			check, err := getTrans(r, db)
			if err != nil {
				return err
			}

			if check.Status != types.StatusPlanned && check.Status != types.StatusRunning {
				return badRequest("cannot pause an already interrupted transfer")
			}

			switch check.Status {
			case types.StatusPlanned:
				check.Status = types.StatusPaused
				if err := db.Update(check); err != nil {
					return err
				}
			case types.StatusRunning:
				pipeline.Signals.SendSignal(check.ID, model.SignalPause)
			default:
				return badRequest("cannot pause an already interrupted transfer")
			}

			w.WriteHeader(http.StatusAccepted)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func cancelTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			check, err := getTrans(r, db)
			if err != nil {
				return err
			}

			if check.Status != types.StatusRunning {
				check.Status = types.StatusCancelled
				if err := pipeline.ToHistory(db, logger, check); err != nil {
					return err
				}
			} else {
				pipeline.Signals.SendSignal(check.ID, model.SignalCancel)
			}

			r.URL.Path = "/api/history"
			w.Header().Set("Location", location(r.URL, fmt.Sprint(check.ID)))
			w.WriteHeader(http.StatusAccepted)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func resumeTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			check, err := getTrans(r, db)
			if err != nil {
				return err
			}

			if check.IsServer {
				return badRequest("only the client can restart a transfer")
			}

			if check.Status != types.StatusPaused && check.Status != types.StatusInterrupted &&
				check.Status != types.StatusError {
				return badRequest("cannot resume an already running transfer")
			}

			agent := &model.RemoteAgent{ID: check.AgentID}
			if err := db.Get(agent); err != nil {
				return fmt.Errorf("failed to retrieve partner: %s", err.Error())
			}
			if agent.Protocol == "sftp" {
				return badRequest("cannot restart an SFTP transfer")
			}

			check.Status = types.StatusPlanned
			check.Error = types.TransferError{}
			if err := db.Update(check); err != nil {
				return fmt.Errorf("failed to update the transfer status: %s", err)
			}

			w.WriteHeader(http.StatusAccepted)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

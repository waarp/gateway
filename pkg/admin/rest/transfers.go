package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"github.com/gorilla/mux"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// InTransfer is the JSON representation of a transfer in requests made to
// the REST interface.
type InTransfer struct {
	Rule       string    `json:"rule"`
	Partner    string    `json:"partner"`
	Account    string    `json:"account"`
	IsSend     bool      `json:"isSend"`
	SourcePath string    `json:"sourcePath"`
	DestPath   string    `json:"destPath"`
	Start      time.Time `json:"startDate"`
}

// ToModel transforms the JSON transfer into its database equivalent.
func (i *InTransfer) ToModel(db *database.DB) (*model.Transfer, error) {
	ruleID, accountID, agentID, err := getTransIDs(db, i)
	if err != nil {
		return nil, err
	}
	return &model.Transfer{
		RuleID:     ruleID,
		IsServer:   false,
		AgentID:    agentID,
		AccountID:  accountID,
		SourceFile: i.SourcePath,
		DestFile:   i.DestPath,
		Start:      i.Start,
	}, nil
}

// OutTransfer is the JSON representation of a transfer in responses sent by
// the REST interface.
type OutTransfer struct {
	ID           uint64                  `json:"id"`
	Rule         string                  `json:"rule"`
	IsServer     bool                    `json:"isServer"`
	Requested    string                  `json:"requested"`
	Requester    string                  `json:"requester"`
	TrueFilepath string                  `json:"trueFilepath"`
	SourcePath   string                  `json:"sourcePath"`
	DestPath     string                  `json:"destPath"`
	Start        time.Time               `json:"startDate"`
	Status       model.TransferStatus    `json:"status"`
	Step         model.TransferStep      `json:"step,omitempty"`
	Progress     uint64                  `json:"progress,omitempty"`
	TaskNumber   uint64                  `json:"taskNumber,omitempty"`
	ErrorCode    model.TransferErrorCode `json:"errorCode,omitempty"`
	ErrorMsg     string                  `json:"errorMsg,omitempty"`
}

// FromTransfer transforms the given database transfer into its JSON equivalent.
func FromTransfer(db *database.DB, trans *model.Transfer) (*OutTransfer, error) {
	rule, requester, requested, err := getTransNames(db, trans)
	if err != nil {
		return nil, err
	}

	return &OutTransfer{
		ID:           trans.ID,
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
func FromTransfers(db *database.DB, models []model.Transfer) ([]OutTransfer, error) {
	jsonArray := make([]OutTransfer, len(models))
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

func createTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			jsonTrans := &InTransfer{}
			if err := readJSON(r, jsonTrans); err != nil {
				return err
			}

			trans, err := jsonTrans.ToModel(db)
			if err != nil {
				return err
			}
			if err := db.Create(trans); err != nil {
				return err
			}

			w.Header().Set("Location", location(r, fmt.Sprint(trans.ID)))
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
			query, err := parseTransferListQuery(db, r)
			if err != nil {
				return err
			}

			transfers, err := execTransferListQuery(db, query)
			if err != nil {
				return err
			}

			json, err := FromTransfers(db, transfers)
			if err != nil {
				return err
			}

			resp := map[string][]OutTransfer{"transfers": json}
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

			if err := db.Get(check); err != nil {
				return err
			}

			if check.Status == model.StatusPaused || check.Status == model.StatusInterrupted {
				return badRequest("cannot pause an already interrupted transfer")
			}

			if check.Status == model.StatusPlanned {
				check.Status = model.StatusPaused
				if err := check.Update(db); err != nil {
					return err
				}
			} else {
				pipeline.Signals.SendSignal(check.ID, model.SignalPause)
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

			if err := db.Get(check); err != nil {
				return err
			}

			if check.Status != model.StatusRunning {
				check.Status = model.StatusCancelled
				if err := pipeline.ToHistory(db, logger, check); err != nil {
					return err
				}
			} else {
				pipeline.Signals.SendSignal(check.ID, model.SignalCancel)
			}

			r.URL.Path = APIPath + HistoryPath
			w.Header().Set("Location", location(r, fmt.Sprint(check.ID)))
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

			if err := db.Get(check); err != nil {
				return err
			}

			if check.IsServer {
				return badRequest("only the client can restart a transfer")
			}

			if check.Status != model.StatusPaused && check.Status != model.StatusInterrupted {
				return badRequest("cannot resume an already running transfer")
			}

			agent := &model.RemoteAgent{ID: check.AgentID}
			if err := db.Get(agent); err != nil {
				return fmt.Errorf("failed to retrieve partner: %s", err.Error())
			}
			if agent.Protocol == "sftp" {
				return badRequest("cannot restart an SFTP transfer")
			}

			check.Status = model.StatusPlanned
			if err := check.Update(db); err != nil {
				return err
			}

			w.WriteHeader(http.StatusAccepted)
			return nil
		}()
		if err != nil {
			handleErrors(w, logger, err)
		}
	}
}

package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type restServerTransfer struct {
	Rule         string         `json:"rule"`
	Server       string         `json:"server"`
	Account      string         `json:"account"`
	IsSend       *bool          `json:"isSend"`
	File         string         `json:"file"`
	DueDate      time.Time      `json:"dueDate"`
	TransferInfo map[string]any `json:"transferInfo"`
}

//nolint:err113,wrapcheck //these are base errors
func (r *restServerTransfer) UnmarshalJSON(bytes []byte) error {
	type rst restServerTransfer
	var t rst

	if err := json.Unmarshal(bytes, &t); err != nil {
		return err
	}

	if t.IsSend == nil {
		return errors.New(`field "isSend" is required`)
	}

	if t.DueDate.IsZero() {
		return errors.New(`field "DueDate is required`)
	}

	if t.DueDate.Before(time.Now()) {
		return errors.New(`transfer due date cannot be in the past`)
	}

	*r = restServerTransfer(t)

	return nil
}

func preregisterServerTransfer(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var rTrans restServerTransfer
		if err := readJSON(r, &rTrans); handleError(w, logger, err) {
			return
		}

		var (
			rule    model.Rule
			server  model.LocalAgent
			account model.LocalAccount
		)

		if handleError(w, logger, db.Get(&rule, "name=? AND is_send=?", rTrans.Rule, *rTrans.IsSend).Run()) {
			return
		}

		if handleError(w, logger, db.Get(&server, "name=?", rTrans.Server).Owner().Run()) {
			return
		}

		if handleError(w, logger, db.Get(&account, "local_agent_id=? AND login=?",
			server.ID, rTrans.Account).Run()) {
			return
		}

		dbTrans := &model.Transfer{
			RuleID:         rule.ID,
			LocalAccountID: utils.NewNullInt64(account.ID),
			Start:          rTrans.DueDate,
			Status:         types.StatusAvailable,
		}

		if *rTrans.IsSend {
			dbTrans.SrcFilename = rTrans.File
		} else {
			dbTrans.DestFilename = rTrans.File
		}

		if handleError(w, logger, db.Insert(dbTrans).Run()) {
			return
		}

		w.Header().Set("Location", location(r.URL, utils.FormatInt(dbTrans.ID)))
		w.WriteHeader(http.StatusCreated)
	}
}

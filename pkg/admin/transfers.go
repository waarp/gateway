package admin

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/sftp"
)

func transferInfo(db *database.Db, trans *model.Transfer) (*model.RemoteAgent,
	*model.RemoteAccount, *model.Cert, *model.Rule, error) {

	remote := model.RemoteAgent{ID: trans.RemoteID}
	if err := db.Get(&remote); err != nil {
		if err == database.ErrNotFound {
			return nil, nil, nil, nil, &badRequest{msg: fmt.Sprintf("The partner n°%v does not exist", trans.RemoteID)}
		}
		return nil, nil, nil, nil, err
	}
	certs, err := remote.GetCerts(db)
	if err != nil || len(certs) == 0 {
		if len(certs) == 0 {
			return nil, nil, nil, nil, database.InvalidError(
				"No certificates found for agent n°%v", remote.ID)
		}
		return nil, nil, nil, nil, err
	}
	account := model.RemoteAccount{ID: trans.AccountID}
	if err := db.Get(&account); err != nil {
		if err == database.ErrNotFound {
			return nil, nil, nil, nil, &badRequest{msg: fmt.Sprintf("The account n°%v does not exist", account.ID)}
		}
		return nil, nil, nil, nil, err
	}

	rule := model.Rule{ID: trans.RuleID}
	if err := db.Get(&rule); err != nil {
		if err == database.ErrNotFound {
			return nil, nil, nil, nil, &badRequest{msg: fmt.Sprintf("The rule n°%v does not exist", rule.ID)}
		}
		return nil, nil, nil, nil, err
	}

	return &remote, &account, &certs[0], &rule, nil
}

func addTransfer(logger *log.Logger, db *database.Db) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		trans := model.Transfer{}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			handleErrors(w, logger, err)
			return
		}
		err = json.Unmarshal(body, &trans)
		if err != nil {
			handleErrors(w, logger, err)
			return
		}

		remote, account, cert, rule, err := transferInfo(db, &trans)
		if err != nil {
			handleErrors(w, logger, err)
			return
		}

		go func() {
			conn, err := sftp.Connect(*remote, *cert, *account)
			if err == nil {
				_ = sftp.DoTransfer(conn, trans, *rule)
			}
		}()

		w.WriteHeader(http.StatusAccepted)
	}
}

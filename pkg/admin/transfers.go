package admin

import (
	"fmt"
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func addTransfer(logger *log.Logger, db *database.Db) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		trans := model.Transfer{}

		if err := restCreate(db, r, &trans); err != nil {
			handleErrors(w, logger, err)
			return
		}

		newID := fmt.Sprint(trans.ID)
		w.Header().Set("Location", APIPath+TransfersPath+"/"+newID)
		w.WriteHeader(http.StatusCreated)
	}
}

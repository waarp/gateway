package http

import (
	"fmt"
	"net/http"
	"path"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/http/httpconst"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
)

type httpHandler struct {
	running *service.TransferMap
	agent   *model.LocalAgent
	account *model.LocalAccount
	rule    model.Rule

	db     *database.DB
	logger *log.Logger
	req    *http.Request
	resp   http.ResponseWriter
}

func (h *httpHandler) getRule(isSend bool) bool {
	name := h.req.Header.Get(httpconst.RuleName)
	if name == "" {
		name = h.req.FormValue(httpconst.Rule)
		if name == "" {
			h.sendError(http.StatusBadRequest, types.TeInternal, "missing rule name")

			return false
		}
	}

	return h.getRuleFromName(name, isSend)
}

func (h *httpHandler) getRuleFromName(name string, isSend bool) bool {
	if err := h.db.Get(&h.rule, "name=? AND send=?", name, isSend).Run(); err != nil {
		if database.IsNotFound(err) {
			h.rule.IsSend = isSend
			msg := fmt.Sprintf("No %s rule with name '%s' found", h.rule.Direction(), name)

			h.logger.Warning(msg)
			h.sendError(http.StatusBadRequest, types.TeInternal, "rule not found")

			return false
		}

		h.logger.Errorf("Failed to retrieve transfer rule: %s", err)
		h.sendError(http.StatusInternalServerError, types.TeInternal, "failed to retrieve transfer rule")

		return false
	}

	return true
}

func (h *httpHandler) checkRulePermission() bool {
	isAuthorized, err := h.rule.IsAuthorized(h.db, h.account)
	if err != nil {
		h.logger.Errorf("Failed to retrieve rule permissions: %s", err)
		h.sendError(http.StatusInternalServerError, types.TeInternal, "failed to check rule permissions")

		return false
	}

	if isAuthorized {
		return true
	}

	h.logger.Warningf("Account %s is not allowed to use %s rule %s", h.account.Login,
		h.rule.Direction(), h.rule.Name)
	h.sendError(http.StatusForbidden, types.TeForbidden, "you do not have permission to use this rule")

	return false
}

func (h *httpHandler) getSizeProgress(trans *model.Transfer) bool {
	if h.rule.IsSend {
		progress, err := getRange(h.req)
		if err != nil {
			h.logger.Errorf("Failed to parse transfer file attributes: %s", err)
			h.sendError(http.StatusRequestedRangeNotSatisfiable, types.TeInternal, err.Error())

			return false
		}

		if progress < trans.Progress {
			trans.Progress = progress
		}
	} else {
		progress, filesize, err := getContentRange(h.req.Header)
		if err != nil {
			h.logger.Errorf("Failed to parse transfer file attributes: %s", err)
			h.sendError(http.StatusBadRequest, types.TeInternal, err.Error())

			return false
		}

		if progress > trans.Progress {
			h.sendError(http.StatusRequestedRangeNotSatisfiable, types.TeBadSize, "unacceptable range start")

			return false
		}

		if progress < trans.Progress {
			trans.Progress = progress
		}

		if filesize != trans.Filesize {
			trans.Filesize = filesize
		}
	}

	return true
}

func (h *httpHandler) getTransfer(isSend bool) (*model.Transfer, bool) {
	if h.req.URL.Path == "" || path.Clean(h.req.URL.Path) == "/" {
		h.sendError(http.StatusBadRequest, types.TeFileNotFound, "missing file path")

		return nil, false
	}

	if !h.getRule(isSend) {
		return nil, false
	}

	if !h.checkRulePermission() {
		return nil, false
	}

	remoteID := h.req.Header.Get(httpconst.TransferID)
	if remoteID == "" {
		remoteID = h.req.FormValue(httpconst.ID)
	}

	trans := &model.Transfer{
		RemoteTransferID: remoteID,
		IsServer:         true,
		RuleID:           h.rule.ID,
		AgentID:          h.agent.ID,
		AccountID:        h.account.ID,
		LocalPath:        path.Base(h.req.URL.Path),
		RemotePath:       path.Base(h.req.URL.Path),
		Filesize:         model.UnknownSize,
		Start:            time.Now(),
		Status:           types.StatusPlanned,
	}

	var err *types.TransferError

	trans, err = pipeline.GetOldTransfer(h.db, h.logger, trans)
	if err != nil {
		h.sendError(http.StatusInternalServerError, err.Code, err.Details)

		return nil, false
	}

	if !h.getSizeProgress(trans) {
		return nil, false
	}

	return trans, true
}

func (h *httpHandler) handleHead() {
	remoteID := h.req.Header.Get(httpconst.TransferID)
	if remoteID == "" {
		remoteID = h.req.FormValue(httpconst.ID)
		if remoteID == "" {
			h.sendError(http.StatusBadRequest, types.TeInternal, "missing transfer ID")

			return
		}
	}

	var trans model.Transfer

	if err := h.db.Get(&trans, "is_server=? AND remote_transfer_id=? AND account_id=?",
		true, remoteID, h.account.ID).Run(); err != nil {
		if !database.IsNotFound(err) {
			h.sendError(http.StatusBadRequest, types.TeInternal, "unknown transfer ID")

			return
		}

		h.sendError(http.StatusInternalServerError, types.TeInternal, "database error")

		return
	}

	var rule model.Rule
	if err := h.db.Get(&rule, "id=?", trans.RuleID).Run(); err != nil {
		h.sendError(http.StatusInternalServerError, types.TeInternal, "database error")

		return
	}

	if ok, err := rule.IsAuthorized(h.db, h.account); err != nil {
		h.sendError(http.StatusInternalServerError, types.TeInternal, "database error")

		return
	} else if !ok {
		h.sendError(http.StatusForbidden, types.TeForbidden, "you do not have permission to see this transfer")
	}

	h.resp.Header().Set(httpconst.TransferID, trans.RemoteTransferID)
	h.resp.Header().Set(httpconst.Rule, rule.Name)
	h.resp.Header().Set(httpconst.TransferStatus, string(trans.Status))
	h.resp.Header().Set(httpconst.ErrorCode, trans.Error.Code.String())
	h.resp.Header().Set(httpconst.ErrorMessage, trans.Error.Details)
	makeContentRange(h.resp.Header(), &trans)
	h.resp.WriteHeader(http.StatusNoContent)
}

func (h *httpHandler) handle(isSend bool) {
	trans, canContinue := h.getTransfer(isSend)
	if !canContinue {
		return
	}

	op := "Upload"
	if isSend {
		op = "Download"
	}

	h.logger.Debugf("%s of file %s requested by %s using rule %s, transfer "+
		"was given ID n°%d", op, path.Base(h.req.URL.Path), h.account.Login,
		h.rule.Name, trans.ID)

	pip, err := pipeline.NewServerPipeline(h.db, trans)
	if err != nil {
		http.Error(h.resp, err.Error(), http.StatusInternalServerError)

		return
	}

	if isSend {
		runDownload(h.req, h.resp, h.running, pip)
	} else {
		runUpload(h.req, h.resp, h.running, pip)
	}

	h.logger.Debugf("File transfer done")
}

func (h *httpHandler) sendError(status int, code types.TransferErrorCode, msg string) {
	h.resp.Header().Set(httpconst.TransferStatus, string(types.StatusError))
	h.resp.Header().Set(httpconst.ErrorCode, code.String())
	h.resp.Header().Set(httpconst.ErrorMessage, msg)
	h.resp.WriteHeader(status)
	fmt.Fprint(h.resp, msg)
}

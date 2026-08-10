package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http/httpconst"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
)

type httpHandler struct {
	agent   *model.LocalAgent
	account *model.LocalAccount
	rule    *model.Rule

	tracer func() pipeline.Trace
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

	var err error
	if h.rule, err = protoutils.GetRule(h.db, h.account, name, isSend); err != nil {
		switch {
		case errors.Is(err, protoutils.ErrRuleNotFound):
			h.logger.Warningf("User %q requested unknown rule %q", h.account.Login, name)
			h.sendError(http.StatusNotFound, types.TeInternal, "rule not found")
		case errors.Is(err, protoutils.ErrPermissionDenied):
			h.logger.Warningf("User %q requested unauthorized rule %q", h.account.Login, name)
			h.sendError(http.StatusForbidden, types.TeForbidden, "you do not have permission to use this rule")

		default:
			h.logger.Errorf("Failed to retrieve rule: %v", err)
			h.sendError(http.StatusInternalServerError, types.TeInternal, "failed to retrieve rule")
		}

		return false
	}

	return true
}

func (h *httpHandler) getSizeProgress(trans *model.Transfer) bool {
	if h.rule.IsSend {
		progress, err := getRange(h.req)
		if err != nil {
			h.logger.Errorf("Failed to parse transfer file attributes: %v", err)
			h.sendError(http.StatusRequestedRangeNotSatisfiable, types.TeInternal, err.Error())

			return false
		}

		if progress < trans.Progress {
			trans.Progress = progress
		}
	} else {
		progress, filesize, err := getContentRange(h.req.Header)
		if err != nil {
			h.logger.Errorf("Failed to parse transfer file attributes: %v", err)
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

	filepath := strings.TrimPrefix(h.req.URL.Path, "/")
	remoteID := h.req.Header.Get(httpconst.TransferID)

	if remoteID == "" {
		remoteID = h.req.FormValue(httpconst.ID)
	}

	trans, tErr := h.mkTransfer(remoteID, filepath)
	if tErr != nil {
		h.sendError(http.StatusInternalServerError, tErr.Code(), tErr.Redacted())

		return nil, false
	}

	if !h.getSizeProgress(trans) {
		return nil, false
	}

	return trans, true
}

func (h *httpHandler) mkTransfer(remoteID, filepath string) (*model.Transfer, *pipeline.Error) {
	if trans, err := pipeline.GetOldTransferByRemoteID(h.db, remoteID, h.account,
		h.rule); err == nil {
		return trans, nil
	} else if !database.IsNotFound(err) {
		return nil, err
	}

	if trans, err := pipeline.GetAvailableTransferByFilename(h.db, filepath, remoteID,
		h.account, h.rule); err == nil {
		return trans, nil
	} else if !database.IsNotFound(err) {
		return nil, err
	}

	return pipeline.MakeServerTransfer(remoteID, filepath, h.account, h.rule), nil
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

	if err := h.db.Get(&trans, "remote_transfer_id=? AND local_account_id=?",
		remoteID, h.account.ID).OrderBy("start", false).Run(); database.IsNotFound(err) {
		h.sendError(http.StatusBadRequest, types.TeInternal, "unknown transfer ID")

		return
	} else if err != nil {
		h.logger.Errorf("Failed to retrieve transfer: %v", err)
		h.sendError(http.StatusInternalServerError, types.TeInternal, "database error")

		return
	}

	var rule model.Rule
	if err := h.db.Get(&rule, "id=?", trans.RuleID).Run(); database.IsNotFound(err) {
		h.sendError(http.StatusBadRequest, types.TeInternal, "unknown rule ID")

		return
	} else if err != nil {
		h.logger.Errorf("Failed to retrieve rule: %v", err)
		h.sendError(http.StatusInternalServerError, types.TeInternal, "database error")

		return
	}

	if !rule.IsAuthorized(h.account) {
		h.sendError(http.StatusForbidden, types.TeForbidden, "you do not have permission to see this transfer")
	}

	h.resp.Header().Set(httpconst.TransferID, trans.RemoteTransferID)
	h.resp.Header().Set(httpconst.Rule, rule.Name)
	h.resp.Header().Set(httpconst.TransferStatus, string(trans.Status))
	h.resp.Header().Set(httpconst.ErrorCode, trans.ErrCode.String())
	h.resp.Header().Set(httpconst.ErrorMessage, trans.ErrDetails)
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

	pip, err := pipeline.NewServerPipeline(h.db, h.logger, trans, h, snmp.GlobalService)
	if err != nil {
		h.sendError(http.StatusInternalServerError, err.Code(), err.Redacted())

		return
	}

	if h.tracer != nil {
		pip.Trace = h.tracer()
	}

	h.logger.Infof("%s of file %s requested by %s using rule %s, transfer "+
		"was given ID n°%d", op, path.Base(h.req.URL.Path), h.account.Login,
		h.rule.Name, trans.ID)

	var handler interface {
		Pause(ctx context.Context) error
		Interrupt(ctx context.Context) error
		Cancel(ctx context.Context) error
		run()
	}

	if isSend {
		handler = &downloadHandler{
			pip:  pip,
			req:  h.req,
			resp: h.resp,
		}
	} else {
		handler = &uploadHandler{
			pip:     pip,
			req:     h.req,
			reqBody: &postBody{src: h.req.Body, closed: make(chan struct{})},
			resp:    h.resp,
		}
	}

	pip.SetInterruptionHandlers(handler.Pause, handler.Interrupt, handler.Cancel)
	handler.run()
}

func (h *httpHandler) sendError(status int, code types.TransferErrorCode, msg string) {
	h.resp.Header().Set(httpconst.TransferStatus, string(types.StatusError))
	h.resp.Header().Set(httpconst.ErrorCode, code.String())
	h.resp.Header().Set(httpconst.ErrorMessage, msg)
	h.resp.WriteHeader(status)
	fmt.Fprint(h.resp, msg)
}

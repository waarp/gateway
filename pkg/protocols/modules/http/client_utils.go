package http

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

func getResponseStatus(resp *http.Response, pip *pipeline.Pipeline) *pipeline.Error {
	if stErr := getRemoteStatus(resp.Header, resp.Body, pip); stErr != nil {
		return stErr
	}

	// Pull
	if method := resp.Request.Method; method == http.MethodGet {
		switch resp.StatusCode {
		case http.StatusOK, http.StatusPartialContent:
			return nil
		default:
			return pipeline.NewErrorf(statusToErrCode(resp.StatusCode),
				"unexpected status code %d for %s: %s",
				resp.StatusCode, method, readResponseMsg(resp))
		}
	}

	// Push
	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	default:
		return pipeline.NewErrorf(statusToErrCode(resp.StatusCode),
			"unexpected status code %d for %s: %s",
			resp.StatusCode, resp.Request.Method, readResponseMsg(resp))
	}
}

func statusToErrCode(status int) types.TransferErrorCode {
	switch status {
	case http.StatusUnauthorized:
		return types.TeBadAuthentication
	case http.StatusForbidden:
		return types.TeForbidden
	case http.StatusNotFound:
		return types.TeFileNotFound
	default:
		return types.TeUnknown
	}
}

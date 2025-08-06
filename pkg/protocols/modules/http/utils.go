package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http/httpconst"
)

const (
	crBase    = 10
	crBitSize = 64
)

type contentRangeError struct{ msg string }

func (e *contentRangeError) Error() string {
	return fmt.Sprintf("could not parse Content-Range header: %s", e.msg)
}

func unauthorized(w http.ResponseWriter, msg string) {
	http.Error(w, msg, http.StatusUnauthorized)
	w.Header().Add("WWW-Authenticate", "Basic")
	w.Header().Add("WWW-Authenticate", `Transport mode="tls-client-certificate"`)
}

func getRemoteError(headers http.Header, body io.ReadCloser) *pipeline.Error {
	return parseRemoteError(headers, body, types.TeUnknownRemote, "unknown error on remote")
}

func parseRemoteError(headers http.Header, body io.ReadCloser,
	defaultCode types.TransferErrorCode, defaultMsg string,
) *pipeline.Error {
	code := defaultCode
	if c := headers.Get(httpconst.ErrorCode); c != "" {
		code = types.TecFromString(c)
	}

	msg := defaultMsg
	if m := headers.Get(httpconst.ErrorMessage); m != "" {
		msg = m
	} else {
		if bodMsg, err := io.ReadAll(body); msg == "" && err == nil {
			msg = string(bodMsg)
		}
	}

	return pipeline.NewErrorf(code, "Error on remote partner: %v", msg)
}

const haltTimeout = 5 * time.Second

func getRemoteStatus(headers http.Header, body io.ReadCloser, pip *pipeline.Pipeline) *pipeline.Error {
	status := types.StatusDone

	if st := headers.Get(httpconst.TransferStatus); st != "" {
		var ok bool
		if status, ok = types.StatusFromString(st); !ok {
			status = types.StatusError
		}
	}

	switch status {
	case types.StatusDone:
		return nil
	case types.StatusPaused:
		ctx, cancel := context.WithTimeout(context.Background(), haltTimeout)
		defer cancel()

		if err := pip.Pause(ctx); err != nil {
			return pipeline.NewErrorWith(types.TeInternal, "failed to pause transfer", err)
		}

		return errPause
	case types.StatusInterrupted:
		return errShutdown
	case types.StatusCancelled:
		ctx, cancel := context.WithTimeout(context.Background(), haltTimeout)
		defer cancel()

		if err := pip.Cancel(ctx); err != nil {
			return pipeline.NewErrorWith(types.TeInternal, "failed to cancel transfer", err)
		}

		return errCancel
	case types.StatusError:
		return getRemoteError(headers, body)
	default:
		return pipeline.NewError(types.TeUnknownRemote, "unknown error on remote")
	}
}

func makeRange(req *http.Request, trans *model.Transfer) {
	if trans.Progress == 0 {
		return
	}

	head := fmt.Sprintf("bytes %d-", trans.Progress)
	req.Header.Set("Range", head)
}

func getRange(req *http.Request) (progress int64, err error) {
	head := req.Header.Get("Range")
	if head == "" {
		return 0, nil
	}

	reg := regexp.MustCompile(`^bytes (\d+)-$`)

	matches := reg.FindAllStringSubmatch(head, -1)
	if matches == nil {
		return -1, &contentRangeError{fmt.Sprintf("invalid Range value '%s' "+
			"(only a single range-start is allowed)", head)}
	}

	progress, err = strconv.ParseInt(matches[0][1], crBase, crBitSize)
	if err != nil {
		return -1, &contentRangeError{fmt.Sprintf("invalid range-start value '%s'", head)}
	}

	return progress, nil
}

func makeContentRange(headers http.Header, trans *model.Transfer) {
	if sizeUnknown := trans.Filesize < 0; sizeUnknown {
		headers.Set("Content-Range", "bytes */*")

		return
	}

	headers.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", trans.Progress,
		trans.Filesize, trans.Filesize))
}

func getContentRange(headers http.Header) (progress, filesize int64, err error) {
	progress = 0
	filesize = model.UnknownSize

	head := headers.Get("Content-Range")
	if head == "" {
		return progress, filesize, nil // no content-range to parse
	}

	reg := regexp.MustCompile(`^bytes (\d+-\d+|\*)/(\d+|\*)$`)

	matches := reg.FindAllStringSubmatch(head, -1)
	if matches == nil {
		err = &contentRangeError{fmt.Sprintf("malformed header value '%s'", head)}

		return 0, 0, err
	}

	if contRange := matches[0][1]; contRange != "*" {
		startEnd := strings.Split(contRange, "-")

		progress, err = strconv.ParseInt(startEnd[0], crBase, crBitSize)
		if err != nil {
			err = &contentRangeError{fmt.Sprintf("invalid range-start value '%s'", matches[0][1])}

			return 0, 0, err
		}
	}

	if size := matches[0][2]; size != "*" {
		var s int64

		s, err = strconv.ParseInt(size, crBase, crBitSize)
		if err != nil {
			err = &contentRangeError{fmt.Sprintf("invalid size value '%s'", matches[0][2])}
		}

		if s >= 0 {
			filesize = s
		}
	}

	return progress, filesize, err
}

func setServerTransferInfo(pip *pipeline.Pipeline, headers http.Header,
	sendError func(int, *pipeline.Error),
) bool {
	if err := setTransferInfo(pip, headers); err != nil {
		sendError(http.StatusInternalServerError, err)

		return false
	}

	return true
}

/*
func setServerFileInfo(pip *pipeline.Pipeline, headers http.Header,
	sendError func(int, *types.TransferError)) bool {

	if err := setFileInfo(pip, headers); err != nil {
		sendError(http.StatusInternalServerError, err)
		return false
	}

	return true
}
*/

func setTransferInfo(pip *pipeline.Pipeline, headers http.Header) *pipeline.Error {
	return setInfo(pip, headers, httpconst.TransferInfo)
}

func setInfo(pip *pipeline.Pipeline, headers http.Header, key string) *pipeline.Error {
	info := pip.TransCtx.TransInfo
	const headerParts = 2

	for _, text := range headers.Values(key) {
		subStr := strings.SplitN(text, "=", headerParts)
		if len(subStr) < headerParts {
			pip.Logger.Errorf("Invalid transfer info header format %q", text)

			return pipeline.NewError(types.TeUnimplemented, "invalid transfer info header")
		}

		name := subStr[0]
		strVal := subStr[1]

		var value any
		if err := json.Unmarshal([]byte(strVal), &value); err != nil {
			pip.Logger.Errorf("Failed to unmarshall transfer info value %q: %s", strVal, err)

			return pipeline.NewErrorWith(types.TeInternal, "failed to parse transfer info value", err)
		}

		info[name] = value
	}

	if err := pip.TransCtx.Transfer.SetTransferInfo(pip.DB, info); err != nil {
		pip.Logger.Errorf("Failed to set transfer info: %v", err)
		pip.SetError(types.TeInternal, "failed to set transfer info")

		return pipeline.NewError(types.TeInternal, "database error")
	}

	pip.TransCtx.TransInfo = info

	return nil
}

func makeInfo(headers http.Header, pip *pipeline.Pipeline, key string,
	info map[string]any,
) *pipeline.Error {
	for name, val := range info {
		jVal, err := json.Marshal(val)
		if err != nil {
			pip.Logger.Errorf("Failed to encode transfer info %q: %v", name, err)

			return pipeline.NewErrorWith(types.TeInternal, "failed to encode transfer info", err)
		}

		headers.Add(key, fmt.Sprintf("%s=%s", name, string(jVal)))
	}

	return nil
}

func makeTransferInfo(headers http.Header, pip *pipeline.Pipeline) *pipeline.Error {
	return makeInfo(headers, pip, httpconst.TransferInfo, pip.TransCtx.TransInfo)
}

/*
func makeFileInfo(headers http.Header, pip *pipeline.Pipeline) *types.TransferError {
	return makeInfo(headers, pip, httpconst.FileInfo, pip.TransCtx.FileInfo)
}
*/

func sendServerError(pip *pipeline.Pipeline, req *http.Request, resp http.ResponseWriter,
	once *sync.Once, status int, err *pipeline.Error,
) {
	once.Do(func() {
		select {
		case <-req.Context().Done():
			err = pipeline.NewError(types.TeConnectionReset, "connection closed by remote host")
		default:
		}

		pip.SetError(err.Code(), err.Details())
		resp.Header().Set(httpconst.TransferStatus, string(types.StatusError))
		resp.Header().Set(httpconst.ErrorCode, err.Code().String())
		resp.Header().Set(httpconst.ErrorMessage, err.Redacted())
		resp.WriteHeader(status)
	})
}

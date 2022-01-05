package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/http/httpconst"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
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

func getRemoteError(headers http.Header, body io.ReadCloser) *types.TransferError {
	return parseRemoteError(headers, body, types.TeUnknownRemote, "unknown error on remote")
}

func parseRemoteError(headers http.Header, body io.ReadCloser,
	defaultCode types.TransferErrorCode, defaultMsg string,
) *types.TransferError {
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

	return types.NewTransferError(code, msg)
}

func getRemoteStatus(headers http.Header, body io.ReadCloser, pip *pipeline.Pipeline) *types.TransferError {
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
		pip.Pause()

		return errPause
	case types.StatusInterrupted:
		return errShutdown
	case types.StatusCancelled:
		pip.Cancel()

		return errCancel
	case types.StatusError:
		return getRemoteError(headers, body)
	default:
		return types.NewTransferError(types.TeUnknownRemote, "unknown error on remote")
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
	progress = 0

	head := req.Header.Get("Range")
	if head == "" {
		return // no range to parse
	}

	reg := regexp.MustCompile(`^bytes (\d+)-$`)

	matches := reg.FindAllStringSubmatch(head, -1)
	if matches == nil {
		err = &contentRangeError{fmt.Sprintf("invalid Range value '%s' "+
			"(only a single range-start is allowed)", head)}

		return
	}

	progress, err = strconv.ParseInt(matches[0][1], crBase, crBitSize)
	if err != nil {
		err = &contentRangeError{fmt.Sprintf("invalid range-start value '%s'", head)}

		return
	}

	return
}

func makeContentRange(headers http.Header, trans *model.Transfer) {
	head := fmt.Sprintf("bytes */%d", trans.Filesize)
	if trans.Progress != 0 {
		head = fmt.Sprintf("bytes %d-%d/%d", trans.Progress, trans.Filesize, trans.Filesize)
	}

	headers.Set("Content-Range", head)
}

func getContentRange(headers http.Header) (progress, filesize int64, err error) {
	progress = 0
	filesize = model.UnknownSize

	head := headers.Get("Content-Range")
	if head == "" {
		return // no content-range to parse
	}

	reg := regexp.MustCompile(`^bytes (\d+-\d+|\*)/(\d+|\*)$`)

	matches := reg.FindAllStringSubmatch(head, -1)
	if matches == nil {
		err = &contentRangeError{fmt.Sprintf("malformed header value '%s'", head)}

		return
	}

	if contRange := matches[0][1]; contRange != "*" {
		startEnd := strings.Split(contRange, "-")

		progress, err = strconv.ParseInt(startEnd[0], crBase, crBitSize)
		if err != nil {
			err = &contentRangeError{fmt.Sprintf("invalid range-start value '%s'", matches[0][1])}

			return
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
	sendError func(int, *types.TransferError),
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

func setTransferInfo(pip *pipeline.Pipeline, headers http.Header) *types.TransferError {
	return setInfo(pip, headers, httpconst.TransferInfo, pip.TransCtx.Transfer.SetTransferInfo)
}

/*
func setFileInfo(pip *pipeline.Pipeline, headers http.Header) *types.TransferError {
	return setInfo(pip, headers, httpconst.FileInfo, pip.TransCtx.Transfer.SetFileInfo)
}
*/

func setInfo(pip *pipeline.Pipeline, headers http.Header, key string,
	set func(database.Access, map[string]interface{}) database.Error,
) *types.TransferError {
	info := map[string]interface{}{}

	for _, text := range headers.Values(key) {
		subStr := strings.SplitN(text, "=", 2) //nolint:gomnd //necessary here
		if len(subStr) < 2 {                   //nolint:gomnd //necessary here
			pip.Logger.Error("Invalid transfer info header format '%s'", text)

			return types.NewTransferError(types.TeUnimplemented, "invalid transfer info header")
		}

		name := subStr[0]
		strVal := subStr[1]

		var value interface{}
		if err := json.Unmarshal([]byte(strVal), &value); err != nil {
			pip.Logger.Error("Failed to unmarshall transfer info value '%s': %s", strVal, err)

			return types.NewTransferError(types.TeInternal, "failed to parse transfer info value")
		}

		info[name] = value
	}

	if err := set(pip.DB, info); err != nil {
		pip.Logger.Error("Failed to set transfer info: %s", err)
		pip.SetError(types.NewTransferError(types.TeInternal, "failed to set transfer info"))

		return types.NewTransferError(types.TeInternal, "database error")
	}

	return nil
}

func makeInfo(headers http.Header, pip *pipeline.Pipeline, key string,
	info map[string]interface{},
) *types.TransferError {
	for name, val := range info {
		jVal, err := json.Marshal(val)
		if err != nil {
			pip.Logger.Error("Failed to encode transfer info '%s': %s", name, err)

			return types.NewTransferError(types.TeInternal, "failed to encode transfer info")
		}

		headers.Add(key, fmt.Sprintf("%s=%s", name, string(jVal)))
	}

	return nil
}

func makeTransferInfo(headers http.Header, pip *pipeline.Pipeline) *types.TransferError {
	return makeInfo(headers, pip, httpconst.TransferInfo, pip.TransCtx.TransInfo)
}

/*
func makeFileInfo(headers http.Header, pip *pipeline.Pipeline) *types.TransferError {
	return makeInfo(headers, pip, httpconst.FileInfo, pip.TransCtx.FileInfo)
}
*/

func sendServerError(pip *pipeline.Pipeline, req *http.Request, resp http.ResponseWriter,
	once *sync.Once, status int, err *types.TransferError,
) {
	once.Do(func() {
		select {
		case <-req.Context().Done():
			err = types.NewTransferError(types.TeConnectionReset, "connection closed by remote host")
		default:
		}

		pip.SetError(err)
		resp.Header().Set(httpconst.TransferStatus, string(types.StatusError))
		resp.Header().Set(httpconst.ErrorCode, err.Code.String())
		resp.Header().Set(httpconst.ErrorMessage, err.Details)
		resp.WriteHeader(status)
	})
}

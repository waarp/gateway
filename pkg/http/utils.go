package http

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/http/httpconst"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

const (
	crBase    = 10
	crBitSize = 64
)

type errContentRange struct{ msg string }

func (e *errContentRange) Error() string {
	return fmt.Sprintf("could not parse Content-Range header: %s", e.msg)
}

func unauthorized(w http.ResponseWriter, msg string) {
	http.Error(w, msg, http.StatusUnauthorized)
	w.Header().Add("WWW-Authenticate", "Basic")
	w.Header().Add("WWW-Authenticate", `Transport mode="tls-client-certificate"`)
}

func getRemoteError(headers http.Header) *types.TransferError {
	code := types.TeUnknownRemote
	if c := headers.Get(httpconst.ErrorCode); c != "" {
		code = types.TecFromString(c)
	}

	msg := "unknown error on remote"
	if m := headers.Get(httpconst.ErrorMessage); m != "" {
		msg = m
	}

	return types.NewTransferError(code, msg)
}

func getRemoteStatus(headers http.Header, pip *pipeline.Pipeline) *types.TransferError {
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
		return getRemoteError(headers)
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

func getRange(req *http.Request) (progress uint64, err error) {
	progress = 0

	head := req.Header.Get("Range")
	if head == "" {
		return // no range to parse
	}

	reg := regexp.MustCompile(`^bytes (\d+)-$`)

	matches := reg.FindAllStringSubmatch(head, -1)
	if matches == nil {
		err = &errContentRange{fmt.Sprintf("invalid Range value '%s' "+
			"(only a single range-start is allowed)", head)}

		return
	}

	progress, err = strconv.ParseUint(matches[0][1], crBase, crBitSize)
	if err != nil {
		err = &errContentRange{fmt.Sprintf("invalid range-start value '%s'", head)}

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

func getContentRange(headers http.Header) (progress uint64, filesize int64, err error) {
	progress = 0
	filesize = model.UnknownSize

	head := headers.Get("Content-Range")
	if head == "" {
		return // no content-range to parse
	}

	reg := regexp.MustCompile(`^bytes (\d+-\d+|\*)/(\d+|\*)$`)

	matches := reg.FindAllStringSubmatch(head, -1)
	if matches == nil {
		err = &errContentRange{fmt.Sprintf("malformed header value '%s'", head)}

		return
	}

	if contRange := matches[0][1]; contRange != "*" {
		startEnd := strings.Split(contRange, "-")

		progress, err = strconv.ParseUint(startEnd[0], crBase, crBitSize)
		if err != nil {
			err = &errContentRange{fmt.Sprintf("invalid range-start value '%s'", matches[0][1])}

			return
		}
	}

	if size := matches[0][2]; size != "*" {
		var s int64

		s, err = strconv.ParseInt(size, crBase, crBitSize)
		if err != nil {
			err = &errContentRange{fmt.Sprintf("invalid size value '%s'", matches[0][2])}
		}

		if s >= 0 {
			filesize = s
		}
	}

	return progress, filesize, err
}

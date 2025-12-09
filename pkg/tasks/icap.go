package tasks

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"code.waarp.fr/lib/log"
	"github.com/pbnjay/memory"
	ic "github.com/solidwall/icap-client"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrIcapMissingUploadURL       = errors.New("ICAP upload URL is missing")
	ErrIcapMissingErrorMovePath   = errors.New("ICAP error move path is missing")
	ErrIcapFileRefused            = errors.New("ICAP server refused file extension")
	ErrIcapUnexpectedResponseCode = errors.New("ICAP server returned an unexpected response code")
	ErrIcapFileTooBig             = errors.New("file is too voluminous for ICAP task (see documentation)")
	ErrIcapInvalidErrorAction     = fmt.Errorf("invalid ICAP error action (must be %q or %q)",
		IcapOnErrorDelete, IcapOnErrorMove)
)

const (
	IcapOnErrorDelete = "delete"
	IcapOnErrorMove   = "move"
)

type icapTask struct {
	UploadURL              string       `json:"uploadURL"`
	Timeout                jsonDuration `json:"timeout"`
	AllowFileModifications jsonBool     `json:"allowFileModifications"`
	OnError                string       `json:"onError"`
	OnErrorMovePath        string       `json:"onErrorMovePath"`

	deleteOnError, moveOnError bool
	client                     *ic.Client
}

func (i *icapTask) parseParams(params map[string]string) error {
	*i = icapTask{}
	if err := utils.JSONConvert(params, i); err != nil {
		return fmt.Errorf("failed to parse icap task arguments: %w", err)
	}

	if i.UploadURL == "" {
		return ErrIcapMissingUploadURL
	}

	if !strings.HasPrefix(i.UploadURL, "icap://") {
		i.UploadURL = "icap://" + i.UploadURL
	}

	switch i.OnError {
	case IcapOnErrorDelete:
		i.deleteOnError = true
	case IcapOnErrorMove:
		if i.OnErrorMovePath == "" {
			return ErrIcapMissingErrorMovePath
		}

		i.moveOnError = true
	case "":
		// do nothing on error
	default:
		return ErrIcapInvalidErrorAction
	}

	i.client = &ic.Client{
		Timeout:        i.Timeout.Duration,
		SetAbsoluteUrl: true,
	}

	return nil
}

func (i *icapTask) Validate(params map[string]string) error {
	return i.parseParams(params)
}

func (i *icapTask) Run(_ context.Context, params map[string]string, _ *database.DB,
	logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	if err := i.parseParams(params); err != nil {
		return err
	}

	previewSize, optErr := i.options(transCtx.Transfer.LocalPath)
	if optErr != nil {
		logger.Errorf("Failed to get preview size: %v", optErr)

		return optErr
	}

	runErr := i.run(logger, transCtx, previewSize)
	if runErr == nil {
		return nil // no error
	}

	logger.Errorf("Failed to run ICAP task: %v", runErr)

	filepath := transCtx.Transfer.LocalPath

	switch {
	case i.deleteOnError:
		if rmErr := fs.Remove(filepath); rmErr != nil {
			logger.Errorf("Failed to delete file after error: %v", rmErr)
		}
	case i.moveOnError:
		if mvErr := fs.MoveFile(filepath, i.OnErrorMovePath); mvErr != nil {
			logger.Errorf("Failed to move file after error: %v", mvErr)
		}
	}

	return fmt.Errorf("failed to run ICAP task: %w", runErr)
}

func (i *icapTask) checkFileSize(file fs.File) (int64, error) {
	info, err := file.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to stat file: %w", err)
	}

	if uint64(info.Size()) > memory.FreeMemory() {
		return 0, ErrIcapFileTooBig
	}

	return info.Size(), nil
}

func (i *icapTask) run(logger *log.Logger, transCtx *model.TransferContext, previewSize int64) error {
	flags := fs.FlagReadOnly
	if i.AllowFileModifications {
		flags = fs.FlagReadWrite
	}

	filePerms := conf.GlobalConfig.Paths.FilePerms

	file, opErr := fs.OpenFile(transCtx.Transfer.LocalPath, flags, filePerms)
	if opErr != nil {
		logger.Errorf("Failed to open transfer file: %v", opErr)

		return fmt.Errorf("failed to open transfer file: %w", opErr)
	}
	defer file.Close() //nolint:errcheck //Close() never returns errors on read-only files

	fileSize, sizErr := i.checkFileSize(file)
	if sizErr != nil {
		logger.Errorf("%v", sizErr)

		return sizErr
	}

	if err := i.makeRequest(file, transCtx, fileSize, previewSize); err != nil {
		logger.Errorf("Failed to make icap request: %v", err)

		return err
	}

	return nil
}

func (i *icapTask) options(filepath string) (int64, error) {
	fileExt := path.Ext(filepath)

	req, reqErr := ic.NewRequest(ic.MethodOPTIONS, i.UploadURL, nil, nil)
	if reqErr != nil {
		return 0, fmt.Errorf("failed to create icap OPTIONS request: %w", reqErr)
	}

	optClient := &ic.Client{
		Timeout:        i.Timeout.Duration,
		SetAbsoluteUrl: true,
	}

	resp, reqErr := optClient.Do(req)
	if reqErr != nil {
		return 0, fmt.Errorf("failed to execute icap OPTIONS request: %w", reqErr)
	}

	if containsExt(resp.Header, ic.TransferIgnoreHeader, fileExt) {
		return 0, ErrIcapFileRefused
	}

	if containsExt(resp.Header, ic.TransferCompleteHeader, fileExt) {
		// Server has indicated that this file type should be sent in full, with no preview.
		return -1, nil
	}

	previewSize := int64(-1)

	if containsExt(resp.Header, ic.TransferPreviewHeader, fileExt) {
		previewSize = int64(resp.PreviewBytes)
	}

	return previewSize, nil
}

func (i *icapTask) overwriteFile(resp *ic.Response, transCtx *model.TransferContext) error {
	isSend := transCtx.Rule.IsSend
	filepath := transCtx.Transfer.LocalPath

	var newFile io.ReadCloser

	switch {
	case isSend && resp.ContentRequest.ContentLength != 0:
		newFile = resp.ContentRequest.Body
	case !isSend && resp.ContentResponse.ContentLength != 0:
		newFile = resp.ContentResponse.Body
	default:
		return nil
	}

	file, reopErr := fs.Create(filepath)
	if reopErr != nil {
		return fmt.Errorf("failed to re-open transfer file: %w", reopErr)
	}

	n, copErr := io.Copy(file, newFile)
	if copErr != nil {
		return fmt.Errorf("failed to write to transfer file: %w", copErr)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close transfer file: %w", err)
	}

	transCtx.Transfer.Filesize = n

	return nil
}

func (i *icapTask) makeRequest(file fs.File, transCtx *model.TransferContext,
	fileSize, previewSize int64,
) error {
	var (
		req    *ic.Request
		reqErr error
	)

	if transCtx.Rule.IsSend {
		req, reqErr = i.makeReqmodRequest(file, getOriginAddress(transCtx), fileSize)
	} else {
		req, reqErr = i.makeRespmodRequest(file, fileSize)
	}

	if reqErr != nil {
		return fmt.Errorf("failed to create icap request: %w", reqErr)
	}

	if previewSize >= 0 && previewSize < fileSize {
		if err := req.SetPreview(int(previewSize)); err != nil {
			return fmt.Errorf("failed to prepare preview request: %w", err)
		}
	}

	resp, respErr := i.client.Do(req)
	if respErr != nil {
		return fmt.Errorf("failed to execute icap request: %w", respErr)
	}

	//nolint:mnd //too specific
	if resp.StatusCode > 299 {
		return fmt.Errorf("%w: %d", ErrIcapUnexpectedResponseCode, resp.StatusCode)
	}

	if i.AllowFileModifications {
		if err := i.overwriteFile(resp, transCtx); err != nil {
			return err
		}
	}

	return nil
}

func (i *icapTask) makeReqmodRequest(file io.Reader, originServer string, length int64,
) (*ic.Request, error) {
	//nolint:noctx //not a real HTTP request, just a payload inside the actual icap request
	httpReq, reqErr := http.NewRequest(http.MethodPost, "http://"+originServer, file)
	if reqErr != nil {
		return nil, fmt.Errorf("failed to create http request: %w", reqErr)
	}

	httpReq.ContentLength = length

	req, err := ic.NewRequest(ic.MethodREQMOD, i.UploadURL, httpReq, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create icap request: %w", err)
	}

	return req, nil
}

func (i *icapTask) makeRespmodRequest(file io.Reader, length int64) (*ic.Request, error) {
	httpResp := &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: http.Header{
			"Content-Type":   []string{"application/octet-stream"},
			"Content-Length": []string{utils.FormatInt(length)},
		},
		Body:          io.NopCloser(file),
		ContentLength: length,
	}

	req, err := ic.NewRequest(ic.MethodRESPMOD, i.UploadURL, nil, httpResp)
	if err != nil {
		return nil, fmt.Errorf("failed to create icap request: %w", err)
	}

	return req, nil
}

func containsExt(headers http.Header, header, fileExt string) bool {
	val := headers.Get(header)

	for _, ext := range utils.TrimSplit(val, ",") {
		if ext == "*" || strings.EqualFold(ext, fileExt) {
			return true
		}
	}

	return false
}

func getOriginAddress(ctx *model.TransferContext) string {
	if ctx.RemoteAgent != nil {
		return ctx.RemoteAgent.Address.String()
	}

	return ctx.LocalAgent.Address.String()
}

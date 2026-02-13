package webdav

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"code.waarp.fr/lib/log"
	"golang.org/x/net/webdav"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type webdavFS struct {
	db     *database.DB
	logger *log.Logger
	tracer func() pipeline.Trace

	req     *http.Request
	server  *model.LocalAgent
	account *model.LocalAccount
}

func (w *webdavFS) getRule(path string) (*model.Rule, error) {
	//nolint:wrapcheck //no need to wrap here
	return protoutils.GetClosestRule(w.db, w.logger, w.server, w.account, path, true)
}

func (w *webdavFS) getRealPath(path string) (string, error) {
	//nolint:wrapcheck //no need to wrap here
	return protoutils.GetRealPath(false, w.db, w.logger, w.server, w.account, path)
}

func (w *webdavFS) Mkdir(_ context.Context, name string, _ os.FileMode) error {
	realPath, err := w.getRealPath(name)
	if err != nil {
		return err
	}

	//nolint:wrapcheck //no need to wrap here
	return fs.MkdirAll(realPath)
}

func (w *webdavFS) OpenFile(ctx context.Context, name string, flag int, _ os.FileMode) (webdav.File, error) {
	w.logger.Debugf(`Received "OpenFile" request on %q`, name)

	if w.req.Method == propfindMethod {
		return w.propfind(name)
	}

	rule, ruleErr := w.getRule(name)
	if ruleErr != nil {
		return nil, ruleErr
	}

	if m := w.req.Method; m != http.MethodPut && m != http.MethodGet && m != http.MethodPost {
		return protoutils.FakeFile(name), nil
	}

	// Special case for Windows explorer which first sends an empty PUT request
	// before uploading a file.
	if strings.HasPrefix(w.req.Header.Get("User-Agent"), "Microsoft-WebDAV-MiniRedir/") &&
		w.req.Method == http.MethodPut && w.req.ContentLength == 0 {
		return protoutils.FakeFile(name), nil
	}

	if err := checkFileFlags(rule, flag); err != nil {
		return nil, err
	}

	filePath := strings.TrimPrefix(name, "/")
	filePath = strings.TrimPrefix(filePath, rule.Path)
	filePath = strings.TrimPrefix(filePath, "/")

	if rule.IsSend {
		w.logger.Infof("Download of file %q requested by %q using rule %q",
			filePath, w.account.Login, rule.Name)
	} else {
		w.logger.Infof("Upload of file %q requested by %q using rule %q",
			filePath, w.account.Login, rule.Name)
	}

	return w.getFile(ctx, filePath, rule)
}

func (w *webdavFS) RemoveAll(_ context.Context, _ string) error {
	return webdav.ErrNotImplemented
}

func (w *webdavFS) Rename(_ context.Context, _, _ string) error {
	return webdav.ErrNotImplemented
}

func (w *webdavFS) Stat(_ context.Context, name string) (os.FileInfo, error) {
	if name == "/" || name == "." || name == "" {
		return protoutils.FakeDirInfo(name), nil
	}

	realPath, err := w.getRealPath(name)
	if err != nil {
		return nil, err
	}

	//nolint:wrapcheck //no need to wrap here
	return fs.Stat(realPath)
}

var (
	ErrDatabase = errors.New("database error")
	ErrInternal = errors.New("internal error")
)

func (w *webdavFS) getTrans(filepath string, rule *model.Rule) (*model.Transfer, error) {
	if trans, err := pipeline.GetOldTransferByFilename(w.db, filepath, 0, w.account,
		rule); err == nil {
		return trans, nil
	} else if !database.IsNotFound(err) {
		w.logger.Errorf("Failed to retrieve transfer by name: %v", err)

		return nil, ErrDatabase
	}

	if trans, err := pipeline.GetAvailableTransferByFilename(w.db, filepath, "",
		w.account, rule); err == nil {
		return trans, nil
	} else if !database.IsNotFound(err) {
		w.logger.Errorf("Failed to retrieve transfer by name: %v", err)

		return nil, ErrDatabase
	}

	return pipeline.MakeServerTransfer("", filepath, w.account, rule), nil
}

func (w *webdavFS) getFile(ctx context.Context, filepath string, rule *model.Rule,
) (webdav.File, error) {
	trans, tErr := w.getTrans(filepath, rule)
	if tErr != nil {
		return nil, tErr
	}

	pip, pErr := pipeline.NewServerPipeline(w.db, w.logger, trans, snmp.GlobalService)
	if pErr != nil {
		w.logger.Errorf("Failed to create transfer pipeline: %v", pErr)

		return nil, ErrInternal
	}

	if w.tracer != nil {
		pip.Trace = w.tracer()
	}

	ctx, cancel := context.WithCancelCause(ctx)

	servPip := &serverPipeline{
		pipeline: pip,
		ctx:      ctx,
		cancel:   cancel,
	}

	pip.SetInterruptionHandlers(servPip.Pause, servPip.Interrupt, servPip.Cancel)

	if err := utils.RunWithCtx(ctx, servPip.init); err != nil {
		return nil, err
	}

	return servPip, nil
}

func checkFileFlags(rule *model.Rule, flags int) error {
	switch {
	case fs.HasFlag(flags, fs.FlagReadWrite):
		return nil
	case fs.HasFlag(flags, fs.FlagWriteOnly):
		if rule.IsSend {
			return fs.ErrPermission
		}
	default:
		if !rule.IsSend {
			return fs.ErrPermission
		}
	}

	return nil
}

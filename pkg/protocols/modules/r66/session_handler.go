package r66

import (
	"context"
	"path"
	"strings"
	"time"

	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type sessionHandler struct {
	*authHandler

	account *model.LocalAccount
	conf    *r66.Authent
}

func (s *sessionHandler) ValidRequest(req *r66.Request) (r66.TransferHandler, error) {
	if err := s.checkRequest(req); err != nil {
		return nil, err
	}

	rule, err := s.getRule(req.Rule, req.IsRecv)
	if err != nil {
		return nil, err
	}

	if !rule.IsSend {
		s.logger.Info("Upload of file %s was requested by %s, using rule %s",
			path.Base(req.Filepath), s.account.Login, req.Rule)
	} else {
		s.logger.Info("Download of file %s was requested by %s, using rule %s",
			path.Base(req.Filepath), s.account.Login, req.Rule)
	}

	trans, err := s.getTransfer(req, rule)
	if err != nil {
		return nil, err
	}

	s.setProgress(req, trans)

	pip, pErr := pipeline.NewServerPipeline(s.db, s.logger, trans, snmp.GlobalService)
	if pErr != nil {
		return nil, internal.ToR66Error(pErr)
	}

	if s.tracer != nil {
		pip.Trace = s.tracer()
	}

	s.getSize(req, rule, trans)

	if err := internal.UpdateTransferInfo(req.Infos, pip); err != nil {
		pip.SetError(err.Code(), err.Details())

		return nil, internal.ToR66Error(err)
	}

	ctx, cancel := context.WithCancelCause(context.Background())

	r66Pip := &serverTransfer{
		r66Conf: s.r66Conf,
		conf:    s.conf,
		pip:     pip,
		ctx:     ctx,
	}

	handler := transferHandler{
		sessionHandler: s,
		trans:          r66Pip,
		cancel:         cancel,
	}

	pip.SetInterruptionHandlers(handler.Pause, handler.Interrupt, handler.Cancel)

	return &handler, nil
}

func (s *sessionHandler) checkRequest(req *r66.Request) *r66.Error {
	if req.Filepath == "" {
		return internal.NewR66Error(r66.IncorrectCommand, "missing filepath")
	}

	if req.Block == 0 {
		return internal.NewR66Error(r66.IncorrectCommand, "missing block size")
	}

	if req.Rule == "" {
		return internal.NewR66Error(r66.IncorrectCommand, "missing transfer rule")
	}

	/*
		if !req.IsRecv && s.conf.Filesize && req.FileSize < 0 {
			return internal.NewR66Error(r66.IncorrectCommand, "missing file size")
		}
	*/

	return nil
}

func (s *sessionHandler) getRule(ruleName string, isSend bool) (*model.Rule, *r66.Error) {
	var rule model.Rule
	if err := s.db.Get(&rule, "name=? AND is_send=?", ruleName, isSend).Run(); err != nil {
		if database.IsNotFound(err) {
			rule.IsSend = isSend
			s.logger.Warning("Requested %s transfer rule '%s' does not exist",
				rule.Direction(), ruleName)

			return nil, internal.NewR66Error(r66.IncorrectCommand, "rule does not exist")
		}

		s.logger.Error("Failed to retrieve transfer rule: %s", err)

		return nil, internal.NewR66Error(r66.Internal, "database error")
	}

	ok, err := rule.IsAuthorized(s.db, s.account)
	if err != nil {
		s.logger.Error("Failed to check rule permissions: %s", err)

		return nil, internal.NewR66Error(r66.Internal, "database error")
	}

	if !ok {
		return nil, internal.NewR66Error(r66.FileNotAllowed, "you do not have the rights to use this transfer rule")
	}

	return &rule, nil
}

func (s *sessionHandler) getTransfer(req *r66.Request, rule *model.Rule) (*model.Transfer, *r66.Error) {
	trans := &model.Transfer{
		RemoteTransferID: utils.FormatInt(req.ID),
		RuleID:           rule.ID,
		LocalAccountID:   utils.NewNullInt64(s.account.ID),
		Start:            time.Now(),
		Status:           types.StatusPlanned,
	}

	if rule.IsSend {
		trans.SrcFilename = strings.TrimPrefix(req.Filepath, "/")
	} else {
		trans.DestFilename = strings.TrimPrefix(req.Filepath, "/")
	}

	trans, tErr := pipeline.GetOldTransfer(s.db, s.logger, trans)
	if tErr != nil {
		return nil, internal.ToR66Error(tErr)
	}

	return trans, nil
}

func (s *sessionHandler) getSize(req *r66.Request, rule *model.Rule,
	trans *model.Transfer,
) {
	if rule.IsSend {
		req.FileSize = trans.Filesize
	} else if req.FileSize >= 0 {
		trans.Filesize = req.FileSize
	}
}

func (s *sessionHandler) setProgress(req *r66.Request, trans *model.Transfer) {
	prog := int64(req.Rank) * int64(req.Block)
	if trans.Progress <= prog {
		req.Rank = uint32(trans.Progress / int64(req.Block))
		trans.Progress -= trans.Progress % int64(req.Block)
	} else {
		trans.Progress = prog
	}
}

func (s *sessionHandler) GetTransferInfo(id int64, isClient bool) (*r66.TransferInfo, error) {
	if isClient {
		return nil, &r66.Error{
			Code:   r66.IncorrectCommand,
			Detail: "requesting info on client transfers is forbidden",
		}
	}

	remoteID := utils.FormatInt(id)
	trans := model.Transfer{}

	if err := s.db.Get(&trans, "remote_transfer_id=? AND local_account_id=?",
		remoteID, s.account.ID).Run(); database.IsNotFound(err) {
		return s.getInfoFromHistory(id)
	} else if err != nil {
		s.logger.Error("Failed to retrieve transfer entry: %v", err)

		return nil, &r66.Error{Code: r66.Internal, Detail: "database error"}
	}

	return s.getInfoFromTransfer(id, &trans)
}

func (s *sessionHandler) getInfoFromTransfer(remoteID int64, trans *model.Transfer,
) (*r66.TransferInfo, error) {
	ctx, ctxErr := model.GetTransferContext(s.db, s.logger, trans)
	if ctxErr != nil {
		return nil, internal.ToR66Error(ctxErr)
	}

	var protoConf serverConfig
	if err := utils.JSONConvert(ctx.LocalAgent.ProtoConfig, &protoConf); err != nil {
		s.logger.Error("Failed to parse server configuration: %v", err)

		return nil, &r66.Error{Code: r66.Internal, Detail: "failed to parse server configuration"}
	}

	userContent, contErr := internal.MakeUserContent(s.logger, ctx.TransInfo)
	if contErr != nil {
		return nil, internal.ToR66Error(contErr)
	}

	return &r66.TransferInfo{
		ID:          remoteID,
		Client:      ctx.LocalAccount.Login,
		Server:      ctx.LocalAgent.Name,
		File:        trans.SrcFilename,
		Rule:        ctx.Rule.Name,
		IsRecv:      ctx.Rule.IsSend,
		IsMd5:       protoConf.CheckBlockHash,
		BlockSize:   protoConf.BlockSize,
		UserContent: userContent,
		Start:       trans.Start,
	}, nil
}

func (s *sessionHandler) getInfoFromHistory(transID int64) (*r66.TransferInfo, error) {
	var hist model.HistoryEntry

	dbErr := s.db.Get(&hist, "remote_transfer_id=? AND is_server=? AND account=?",
		utils.FormatInt(transID), true, s.account.Login).Run()
	if database.IsNotFound(dbErr) {
		return nil, &r66.Error{Code: r66.IncorrectCommand, Detail: "transfer not found"}
	} else if dbErr != nil {
		s.logger.Error("Failed to retrieve history entry: %v", dbErr)

		return nil, &r66.Error{Code: r66.Internal, Detail: "database error"}
	}

	transInfo, dbErr := hist.GetTransferInfo(s.db)
	if dbErr != nil {
		return nil, &r66.Error{Code: r66.Internal, Detail: "database error"}
	}

	userContent, err := internal.MakeUserContent(s.logger, transInfo)
	if err != nil {
		return nil, internal.ToR66Error(err)
	}

	mode := r66.ModeSend
	if hist.IsSend {
		mode = r66.ModeRecv
	}

	return &r66.TransferInfo{
		ID:        transID,
		Client:    hist.Account,
		Server:    hist.Agent,
		File:      hist.SrcFilename,
		Rule:      hist.Rule,
		RuleMode:  uint32(mode), // FIXME once issue #284 is implemented
		BlockSize: 0,            // FIXME once issue #284 is implemented
		Info:      userContent,
		Start:     hist.Start,
		Stop:      hist.Stop,
	}, nil
}

func (s *sessionHandler) GetFileInfo(ruleName, pat string) ([]r66.FileInfo, error) {
	var rule model.Rule

	if err := s.db.Get(&rule, "name=? AND is_send=?", ruleName, true).Run(); database.IsNotFound(err) {
		return nil, &r66.Error{Code: r66.IncorrectCommand, Detail: "rule not found"}
	} else if err != nil {
		s.logger.Error("Failed to retrieve rule: %v", err)

		return nil, &r66.Error{Code: r66.Internal, Detail: "database error"}
	}

	if ok, err := rule.IsAuthorized(s.db, s.account); err != nil {
		s.logger.Error("Failed to check rule permissions: %v", err)

		return nil, &r66.Error{Code: r66.Internal, Detail: "database error"}
	} else if !ok {
		return nil, &r66.Error{
			Code:   r66.IncorrectCommand,
			Detail: "you do not have the rights to use this transfer rule",
		}
	}

	pattern := path.Clean(pat)

	dir, err := s.makeDir(&rule)
	if err != nil {
		return nil, err
	}

	return s.listDirFiles(dir, pattern)
}

func (s *sessionHandler) listDirFiles(root *types.URL, pattern string) ([]r66.FileInfo, error) {
	filesys, fsErr := fs.GetFileSystem(s.db, root)
	if fsErr != nil {
		s.logger.Error("Failed to instantiate the file system: %v", fsErr)

		return nil, &r66.Error{Code: r66.Internal, Detail: "file system error"}
	}

	matches, globErr := fs.Glob(filesys, root.JoinPath(pattern))
	if globErr != nil {
		s.logger.Error("Failed to retrieve matching files: %v", globErr)

		return nil, &r66.Error{Code: r66.IncorrectCommand, Detail: "incorrect file pattern"}
	}

	if len(matches) == 0 {
		return nil, &r66.Error{Code: r66.FileNotFound, Detail: "no files found for the given pattern"}
	}

	var infos []r66.FileInfo

	for _, match := range matches {
		file, statErr := fs.Stat(filesys, match)
		if statErr != nil {
			s.logger.Error("Failed to retrieve file %q info: %v", match, statErr)

			continue
		}

		infos = append(infos, r66.FileInfo{
			Name:       strings.TrimLeft(strings.TrimPrefix(match.Path, root.Path), "/"),
			Size:       file.Size(),
			LastModify: file.ModTime(),
			Type:       fileMode(file),
			Permission: file.Mode().Perm().String(),
		})
	}

	return infos, nil
}

func (s *sessionHandler) makeDir(rule *model.Rule) (*types.URL, error) {
	servDir := s.agent.ReceiveDir
	defDir := conf.GlobalConfig.Paths.DefaultInDir

	if rule.IsSend {
		servDir = s.agent.SendDir
		defDir = conf.GlobalConfig.Paths.DefaultOutDir
	}

	type (
		leaf   = utils.Leaf
		branch = utils.Branch
	)

	dir, err := utils.GetPath("", leaf(rule.LocalDir), leaf(servDir),
		branch(s.agent.RootDir), leaf(defDir),
		branch(conf.GlobalConfig.Paths.GatewayHome))

	return (*types.URL)(dir), err
}

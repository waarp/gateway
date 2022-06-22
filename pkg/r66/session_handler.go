package r66

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/r66/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
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

	pip, pErr := pipeline.NewServerPipeline(s.db, trans)
	if pErr != nil {
		return nil, internal.ToR66Error(pErr)
	}

	s.getSize(req, rule, trans)

	if err := internal.UpdateTransferInfo(req.Infos, pip); err != nil {
		pip.SetError(err)

		return nil, internal.ToR66Error(err)
	}

	r66Pip := &serverTransfer{
		r66Conf: s.r66Conf,
		conf:    s.conf,
		pip:     pip,
		store:   utils.NewErrorStorage(),
	}

	s.runningTransfers.Add(trans.ID, r66Pip)

	handler := transferHandler{
		sessionHandler: s,
		trans:          r66Pip,
	}

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
	if err := s.db.Get(&rule, "name=? AND send=?", ruleName, isSend).Run(); err != nil {
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
		RemoteTransferID: fmt.Sprint(req.ID),
		RuleID:           rule.ID,
		IsServer:         true,
		AgentID:          s.agent.ID,
		AccountID:        s.account.ID,
		LocalPath:        strings.TrimPrefix(req.Filepath, "/"),
		RemotePath:       path.Base(req.Filepath),
		Start:            time.Now(),
		Status:           types.StatusPlanned,
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
	if trans.Step > types.StepData {
		return
	}

	prog := uint64(req.Rank) * uint64(req.Block)
	if trans.Progress <= prog {
		req.Rank = uint32(trans.Progress / uint64(req.Block))
		trans.Progress -= trans.Progress % uint64(req.Block)
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

	remoteID := fmt.Sprint(id)
	trans := model.Transfer{}

	err := s.db.Get(&trans, "remote_transfer_id=? AND is_server=? AND account_id=?",
		remoteID, true, s.account.ID).Run()
	if database.IsNotFound(err) {
		return s.getInfoFromHistory(id)
	} else if err != nil {
		s.logger.Error("Failed to retrieve transfer entry: %v", err)

		return nil, &r66.Error{Code: r66.Internal, Detail: "database error"}
	}

	return s.getInfoFromTransfer(id, &trans)
}

func (s *sessionHandler) getInfoFromTransfer(remoteID int64, trans *model.Transfer,
) (*r66.TransferInfo, error) {
	ctx, err := model.GetTransferContext(s.db, s.logger, trans)
	if err != nil {
		return nil, internal.ToR66Error(err)
	}

	var protoConf config.R66ProtoConfig
	if err := json.Unmarshal(ctx.LocalAgent.ProtoConfig, &protoConf); err != nil {
		s.logger.Error("Failed to parse server configuration: %v", err)

		return nil, &r66.Error{Code: r66.Internal, Detail: "failed to parse server configuration"}
	}

	userContent, err := internal.MakeUserContent(s.logger, ctx.TransInfo)
	if err != nil {
		return nil, internal.ToR66Error(err)
	}

	file, fErr := filepath.Rel(s.makeDir(ctx.Rule), trans.LocalPath)
	if fErr != nil {
		s.logger.Error("Failed to build file path: %v", err)

		return nil, &r66.Error{Code: r66.Internal, Detail: "failed to build file path"}
	}

	return &r66.TransferInfo{
		ID:        remoteID,
		Client:    ctx.LocalAccount.Login,
		Server:    ctx.LocalAgent.Name,
		File:      filepath.ToSlash(file),
		Rule:      ctx.Rule.Name,
		IsRecv:    ctx.Rule.IsSend,
		IsMd5:     protoConf.CheckBlockHash,
		BlockSize: protoConf.BlockSize,
		Info:      userContent,
		Start:     trans.Start,
	}, nil
}

func (s *sessionHandler) getInfoFromHistory(transID int64) (*r66.TransferInfo, error) {
	var hist model.HistoryEntry

	dbErr := s.db.Get(&hist, "remote_transfer_id=? AND is_server=? AND account=?",
		fmt.Sprint(transID), true, s.account.Login).Run()
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
		File:      filepath.Base(hist.LocalPath),
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

	if err := s.db.Get(&rule, "name=? AND send=?", ruleName, true).Run(); database.IsNotFound(err) {
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

	pattern := filepath.FromSlash(pat)
	dir := s.makeDir(&rule)

	return s.listDirFiles(dir, pattern)
}

func (s *sessionHandler) listDirFiles(root, pattern string) ([]r66.FileInfo, error) {
	matches, err := filepath.Glob(filepath.Join(root, pattern))
	if err != nil {
		s.logger.Error("Failed to retrieve matching files: %v", err)

		return nil, &r66.Error{Code: r66.IncorrectCommand, Detail: "incorrect file pattern"}
	}

	if len(matches) == 0 {
		return nil, &r66.Error{Code: r66.FileNotFound, Detail: "no files found for the given pattern"}
	}

	var infos []r66.FileInfo

	for _, match := range matches {
		file, err := os.Stat(match)
		if err != nil {
			s.logger.Error("Failed to retrieve file '%s' info: %v", match, err)

			continue
		}

		fp, err := filepath.Rel(root, match)
		if err != nil {
			s.logger.Error("Failed to split path '%s': %v", match, err)

			continue
		}

		fp = filepath.ToSlash(fp)

		if file.IsDir() {
			infos = append(infos, s.listSubFiles(match, fp)...)

			continue
		}

		infos = append(infos, r66.FileInfo{
			Name:       fp,
			Size:       file.Size(),
			LastModify: file.ModTime(),
			Type:       "File",
			Permission: file.Mode().Perm().String(),
		})
	}

	return infos, nil
}

func (s *sessionHandler) listSubFiles(full, dir string) []r66.FileInfo {
	entries, err := os.ReadDir(full)
	if err != nil {
		s.logger.Error("Failed to open sub-directory '%s': %v", full, err)

		return nil
	}

	infos := make([]r66.FileInfo, 0, len(entries))

	for _, entry := range entries {
		file, err := entry.Info()
		if err != nil {
			s.logger.Error("Failed to retrieve info of file '%s': %v", entry.Name(), err)

			continue
		}

		fileType := "File"
		if file.IsDir() {
			fileType = "Directory"
		}

		infos = append(infos, r66.FileInfo{
			Name:       path.Join(dir, file.Name()),
			Size:       file.Size(),
			LastModify: file.ModTime(),
			Type:       fileType,
			Permission: file.Mode().Perm().String(),
		})
	}

	return infos
}

func (s *sessionHandler) makeDir(rule *model.Rule) string {
	servDir := s.agent.ReceiveDir
	defDir := conf.GlobalConfig.Paths.DefaultInDir

	if rule.IsSend {
		servDir = s.agent.SendDir
		defDir = conf.GlobalConfig.Paths.DefaultOutDir
	}

	return utils.GetPath("", utils.Leaf(rule.LocalDir), utils.Leaf(servDir),
		utils.Branch(s.agent.RootDir), utils.Leaf(defDir),
		utils.Branch(conf.GlobalConfig.Paths.GatewayHome))
}

package r66

import (
	"path"

	"code.waarp.fr/lib/r66"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/r66/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

var errConf = types.NewTransferError(types.TeUnimplemented, "client-server configuration mismatch")

func (c *transferClient) logErrConf(msg string) {
	c.pip.Logger.Error("Client-server configuration mismatch: %s", msg)
}

func (c *transferClient) connect() *types.TransferError {
	cli, err := c.conns.Add(c.pip.TransCtx.RemoteAgent.Address, c.tlsConfig, c.pip.Logger)
	if err != nil {
		c.pip.Logger.Error("Failed to connect to remote host: %s", err)

		return types.NewTransferError(types.TeConnection, "failed to connect to remote host")
	}

	c.ses, err = cli.NewSession()
	if err != nil {
		c.pip.Logger.Error("Failed to start R66 session: %s", err)

		return types.NewTransferError(types.TeConnection, "failed to start R66 session")
	}

	return nil
}

func (c *transferClient) authenticate() (tErr *types.TransferError) {
	servHash := []byte(c.serverPassword)

	conf := &r66.Config{
		FileSize:   true,
		FinalHash:  !c.noFinalHash,
		DigestAlgo: "SHA-256",
		Proxified:  false,
	}

	auth, err := c.ses.Authent(c.pip.TransCtx.RemoteAccount.Login,
		[]byte(c.pip.TransCtx.RemoteAccount.Password), conf)
	if err != nil {
		c.ses = nil
		c.pip.Logger.Error("Client authentication failed: %s", err)

		return types.NewTransferError(types.TeBadAuthentication, "client authentication failed")
	}

	loginOK := utils.ConstantEqual(c.serverLogin, auth.Login)
	pwdErr := bcrypt.CompareHashAndPassword(servHash, auth.Password)

	if !loginOK {
		c.pip.Logger.Error("Server authentication failed: wrong login '%s'", auth.Login)

		return types.NewTransferError(types.TeBadAuthentication, "server authentication failed")
	}

	if pwdErr != nil {
		c.pip.Logger.Error("Server authentication failed: %s", pwdErr)

		return types.NewTransferError(types.TeBadAuthentication, "server authentication failed")
	}

	if auth.Filesize != conf.FileSize {
		c.logErrConf("file size verification")

		return errConf
	}

	if auth.FinalHash != conf.FinalHash {
		c.logErrConf("final hash verification")

		return errConf
	}

	if auth.Digest != conf.DigestAlgo {
		c.logErrConf("unknown digest algorithm")

		return errConf
	}

	return nil
}

func (c *transferClient) request() *types.TransferError {
	blockNB := c.pip.TransCtx.Transfer.Progress / int64(c.blockSize)
	blockRest := c.pip.TransCtx.Transfer.Progress % int64(c.blockSize)

	if c.pip.TransCtx.Transfer.Step <= types.StepData && blockRest != 0 {
		// round progress to the beginning of the block
		c.pip.TransCtx.Transfer.Progress -= blockRest
		if err := c.pip.UpdateTrans(); err != nil {
			return err
		}
	}

	transID, err := c.pip.TransCtx.Transfer.TransferID()
	if err != nil {
		return types.NewTransferError(types.TeInternal, err.Error())
	}

	userContent, tErr := internal.MakeUserContent(c.pip.Logger, c.pip.TransCtx.TransInfo)
	if tErr != nil {
		return tErr
	}

	req := &r66.Request{
		ID:       transID,
		Filepath: c.pip.TransCtx.Transfer.RemotePath,
		FileSize: c.pip.TransCtx.Transfer.Filesize,
		Rule:     c.pip.TransCtx.Rule.Name,
		Block:    c.blockSize,
		Rank:     uint32(blockNB),
		IsMD5:    c.checkBlockHash,
		Infos:    userContent,
	}

	if c.pip.TransCtx.Rule.IsSend {
		info, statErr := fs.Stat(c.pip.TransCtx.FS, &c.pip.TransCtx.Transfer.LocalPath)
		if statErr != nil {
			c.pip.Logger.Error("Failed to retrieve file size: %s", statErr)

			return types.NewTransferError(types.TeInternal, "failed to retrieve file size")
		}

		req.FileSize = info.Size()
		req.IsRecv = false
	} else {
		req.IsRecv = true
	}

	resp, err := c.ses.Request(req)
	if err != nil {
		c.ses = nil
		c.pip.Logger.Error("Transfer request failed: %s", err)

		return internal.FromR66Error(err, c.pip)
	}

	return c.checkReqResp(req, resp)
}

func (c *transferClient) checkReqResp(req, resp *r66.Request) *types.TransferError {
	if c.pip.TransCtx.Rule.IsSend {
		if resp.FileSize != req.FileSize {
			c.logErrConf("different file size")

			return errConf
		}
	} else {
		c.pip.TransCtx.Transfer.Filesize = resp.FileSize

		if err := c.pip.UpdateTrans(); err != nil {
			return err
		}
	}

	if path.Base(resp.Filepath) != path.Base(req.Filepath) {
		c.logErrConf("different filename")

		return errConf
	}

	if resp.Block != req.Block {
		c.logErrConf("different block size")

		return errConf
	}

	if resp.IsRecv != req.IsRecv || resp.IsMD5 != req.IsMD5 {
		c.logErrConf("different transfer mode")

		return errConf
	}

	if resp.Rule != req.Rule {
		c.logErrConf("different transfer rule")

		return errConf
	}

	if resp.ID != req.ID {
		c.logErrConf("different transfer ID")

		return errConf
	}

	progress := int64(resp.Rank) * int64(resp.Block)
	if progress < c.pip.TransCtx.Transfer.Progress {
		c.pip.TransCtx.Transfer.Progress = progress
		if err := c.pip.UpdateTrans(); err != nil {
			return err
		}
	}

	return nil
}

func (c *transferClient) makeHash() ([]byte, error) {
	if c.noFinalHash {
		return nil, nil
	}

	hash, err := internal.MakeHash(c.ctx, c.pip.TransCtx.FS, c.pip.Logger,
		&c.pip.TransCtx.Transfer.LocalPath)
	if err != nil {
		return nil, internal.ToR66Error(err)
	}

	return hash, nil
}

package r66

import (
	"os"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/r66/internal"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"code.waarp.fr/waarp-r66/r66"
	"golang.org/x/crypto/bcrypt"
)

var clientConns = internal.NewConnPool()

var (
	errConf = types.NewTransferError(types.TeUnimplemented, "client-server configuration mismatch")
)

func (c *client) logErrConf(msg string) {
	c.pip.Logger.Errorf("Client-server configuration mismatch: %s", msg)
}

func (c *client) connect() *types.TransferError {
	cli, err := clientConns.Add(c.pip.TransCtx.RemoteAgent.Address, c.tlsConfig, c.pip.Logger)
	if err != nil {
		c.pip.Logger.Errorf("Failed to connect to remote host: %s", err)
		return types.NewTransferError(types.TeConnection, "failed to connect to remote host")
	}

	c.ses, err = cli.NewSession()
	if err != nil {
		c.pip.Logger.Errorf("Failed to start R66 session: %s", err)
		return types.NewTransferError(types.TeConnection, "failed to start R66 session")
	}
	return nil
}

func (c *client) authenticate() *types.TransferError {
	//servHash, err := base64.StdEncoding.DecodeString(c.conf.ServerPassword)
	servHash := []byte(c.conf.ServerPassword)

	conf := &r66.Config{
		FileSize:   true,
		FinalHash:  !c.conf.NoFinalHash,
		DigestAlgo: "SHA-256",
		Proxified:  false,
	}

	auth, err := c.ses.Authent(c.pip.TransCtx.RemoteAccount.Login,
		[]byte(c.pip.TransCtx.RemoteAccount.Password), conf)
	if err != nil {
		c.pip.Logger.Errorf("Client authentication failed: %s", err)
		return types.NewTransferError(types.TeBadAuthentication, "client authentication failed")
	}

	loginOK := utils.ConstantEqual(c.conf.ServerLogin, auth.Login)
	pwdErr := bcrypt.CompareHashAndPassword(servHash, auth.Password)
	if !loginOK {
		c.pip.Logger.Errorf("Server authentication failed: wrong login '%s'", auth.Login)
		return types.NewTransferError(types.TeBadAuthentication, "server authentication failed")
	}
	if pwdErr != nil {
		c.pip.Logger.Errorf("Server authentication failed: %s", pwdErr)
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

func (c *client) request() *types.TransferError {
	req := &r66.Request{
		ID:       int64(c.pip.TransCtx.Transfer.ID),
		Filepath: c.pip.TransCtx.Transfer.RemotePath,
		FileSize: c.pip.TransCtx.Transfer.Filesize,
		Rule:     c.pip.TransCtx.Rule.Name,
		Block:    c.conf.BlockSize,
		IsMD5:    c.conf.CheckBlockHash,
		Infos:    "",
	}
	req.Rank = uint32(c.pip.TransCtx.Transfer.Progress / uint64(c.conf.BlockSize))

	if c.pip.TransCtx.Rule.IsSend {
		info, err := os.Stat(c.pip.TransCtx.Transfer.LocalPath)
		if err != nil {
			c.pip.Logger.Errorf("Failed to retrieve file size: %s", err)
			return types.NewTransferError(types.TeInternal, "failed to retrieve file size")
		}
		req.FileSize = info.Size()
		req.IsRecv = false
	} else {
		req.IsRecv = true
	}

	resp, err := c.ses.Request(req)
	if err != nil {
		c.pip.Logger.Errorf("Transfer request failed: %s", err)
		return internal.FromR66Error(err, c.pip)
	}

	return c.checkReqResp(req, resp)
}

func (c *client) checkReqResp(req, resp *r66.Request) *types.TransferError {
	if c.pip.TransCtx.Rule.IsSend {
		if resp.FileSize != req.FileSize {
			c.logErrConf("different file size")
			return errConf
		}
	} else {
		c.pip.TransCtx.Transfer.Filesize = resp.FileSize
		if err := c.pip.UpdateTrans("filesize"); err != nil {
			return err
		}
	}
	if resp.Filepath != req.Filepath {
		c.logErrConf("different file path")
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
	if resp.Rank != req.Rank {
		progress := uint64(resp.Rank) * uint64(resp.Block)
		if progress < c.pip.TransCtx.Transfer.Progress {
			c.pip.TransCtx.Transfer.Progress = progress
			if err := c.pip.UpdateTrans("progress"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *client) makeHash() ([]byte, error) {
	if c.conf.NoFinalHash {
		return nil, nil
	}

	hash, err := internal.MakeHash(c.ctx, c.pip.Logger, c.pip.TransCtx.Transfer.LocalPath)
	if err != nil {
		return nil, internal.ToR66Error(err)
	}
	return hash, nil
}

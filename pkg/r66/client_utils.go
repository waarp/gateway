package r66

import (
	"encoding/base64"
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
	servHash, err := base64.StdEncoding.DecodeString(c.conf.ServerPassword)

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
	pwdOK := bcrypt.CompareHashAndPassword(servHash, auth.Password) != nil
	if !loginOK {
		c.pip.Logger.Errorf("Server authentication failed: wrong login '%s'", auth.Login)
		return types.NewTransferError(types.TeBadAuthentication, "server authentication failed")
	}
	if !pwdOK {
		c.pip.Logger.Errorf("Server authentication failed: wrong password")
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
		Rule:     c.pip.TransCtx.Rule.Name,
		Block:    c.conf.BlockSize,
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

		if c.conf.CheckBlockHash {
			req.Mode = uint32(r66.MODE_SEND_MD5)
		} else {
			req.Mode = uint32(r66.MODE_SEND)
		}
	} else {
		if c.conf.CheckBlockHash {
			req.Mode = uint32(r66.MODE_RECV_MD5)
		} else {
			req.Mode = uint32(r66.MODE_RECV)
		}
	}

	resp, err := c.ses.Request(req)
	if err != nil {
		return internal.FromR66Error(err, c.pip)
	}

	if c.pip.TransCtx.Rule.IsSend {
		if resp.FileSize != req.FileSize {
			c.logErrConf("different file size")
			return errConf
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
	if resp.Mode != req.Mode {
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
			if err := c.pip.DB.Update(c.pip.TransCtx.Transfer).Cols("progress").Run(); err != nil {
				c.pip.Logger.Errorf("Failed to update transfer progress: %s", err)
				return types.NewTransferError(types.TeInternal, "internal database error")
			}
		}
	}

	return nil
}

func (c *client) makeHash() ([]byte, error) {
	if c.conf.NoFinalHash {
		return nil, nil
	}

	hash, err := internal.MakeHash(c.pip.Logger, c.pip.TransCtx.Transfer.LocalPath)
	if err != nil {
		return nil, internal.ToR66Error(err)
	}
	return hash, nil
}

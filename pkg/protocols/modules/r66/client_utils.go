package r66

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"path"

	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/r66auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

var errConf = pipeline.NewError(types.TeUnimplemented, "client-server configuration mismatch")

func (c *transferClient) logErrConf(msg string) {
	c.pip.Logger.Errorf("Client-server configuration mismatch: %s", msg)
}

func (c *transferClient) connect() (*ClientConn, *pipeline.Error) {
	cli, err := c.conns.Connect(c.pip)
	if err != nil {
		c.pip.Logger.Errorf("Failed to connect to remote host: %v", err)

		return nil, pipeline.NewErrorWith(err, types.TeConnection, "failed to connect to remote host")
	}

	return cli, nil
}

func (c *transferClient) authenticate(conn *ClientConn) *pipeline.Error {
	var err error
	if c.ses, err = conn.authenticate(c.pip.Logger, c.pip.TransCtx, c.noFinalHash,
		c.finalHashAlgo, c.serverLogin); err != nil {
		if tErr, ok := errors.AsType[*pipeline.Error](err); ok {
			return tErr
		}

		return pipeline.NewErrorWith(err, types.TeUnknown, "unknown error during authentication")
	}

	return nil
}

func (c *transferClient) sendRequest() *pipeline.Error {
	blockNB := c.pip.TransCtx.Transfer.Progress / int64(c.blockSize)
	blockRest := c.pip.TransCtx.Transfer.Progress % int64(c.blockSize)

	if c.pip.TransCtx.Transfer.Step <= types.StepData && blockRest != 0 {
		// round progress to the beginning of the block
		c.pip.TransCtx.Transfer.Progress -= blockRest
		if err := c.pip.UpdateTrans(); err != nil {
			return err
		}
	}

	transID, err := utils.ParseInt[int64](c.pip.TransCtx.Transfer.RemoteTransferID)
	if err != nil {
		return pipeline.NewErrorWith(err, types.TeInternal, "failed to parse transfer ID")
	}

	userContent, tErr := internal.MakeUserContent(c.pip.Logger, c.pip.TransCtx.Transfer.TransferInfo)
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
		info, statErr := fs.Stat(c.pip.TransCtx.Transfer.LocalPath)
		if statErr != nil {
			c.pip.Logger.Errorf("Failed to retrieve file size: %s", statErr)

			return pipeline.NewErrorWith(statErr, types.TeInternal, "failed to retrieve file size")
		}

		req.FileSize = info.Size()
		req.IsRecv = false
	} else {
		req.IsRecv = true
	}

	resp, err := c.ses.Request(req)
	if err != nil {
		c.ses = nil
		c.pip.Logger.Errorf("Transfer request failed: %v", err)

		return internal.FromR66Error(err, c.pip)
	}

	return c.checkReqResp(req, resp)
}

func (c *transferClient) checkReqResp(req, resp *r66.Request) *pipeline.Error {
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

func (c *transferClient) makeHash(file protocol.SendFile) func() ([]byte, error) {
	return func() ([]byte, error) {
		if c.noFinalHash {
			return nil, nil
		}

		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return nil, internal.ToR66Error(err)
		}

		hash, err := internal.ComputeHash(c.ctx, c.finalHashAlgo, c.pip.Logger, file)
		if err != nil {
			return nil, internal.ToR66Error(err)
		}

		return hash, nil
	}
}

func makeClientTLSConfig(logger *log.Logger, ctx *model.TransferContext) (*tls.Config, error) {
	tlsConf, err := protoutils.GetClientTLSConfig(ctx, logger)
	if err != nil {
		return nil, err
	}

	if !compatibility.IsLegacyR66CertificateAllowed {
		return tlsConf, nil
	}

	for _, cred := range ctx.RemoteAccountCreds {
		if cred.Type == r66auth.AuthLegacyCertificate {
			tlsConf.GetClientCertificate = func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
				return &compatibility.LegacyR66Cert, nil
			}

			break
		}
	}

	for _, cred := range ctx.RemoteAgentCreds {
		if cred.Type == r66auth.AuthLegacyCertificate {
			tlsConf.InsecureSkipVerify = true
			tlsConf.VerifyPeerCertificate = verifyLegacyCert

			break
		}
	}

	return tlsConf, nil
}

var (
	ErrMissingCertificate = errors.New("missing certificate")
	ErrBadCertificate     = errors.New("bad certificate")
)

func verifyLegacyCert(rawCerts [][]byte, _ [][]*x509.Certificate) error {
	if len(rawCerts) == 0 {
		return ErrMissingCertificate
	}

	chain, parsErr := auth.ParseRawCertChain(rawCerts)
	if parsErr != nil {
		return fmt.Errorf("failed to parse the certification chain: %w", parsErr)
	}

	if !compatibility.IsLegacyR66Cert(chain[0]) {
		return ErrBadCertificate
	}

	return nil
}

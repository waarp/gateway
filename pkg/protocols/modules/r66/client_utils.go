package r66

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"path"

	"code.waarp.fr/lib/r66"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

var errConf = pipeline.NewError(types.TeUnimplemented, "client-server configuration mismatch")

func (c *transferClient) logErrConf(msg string) {
	c.pip.Logger.Error("Client-server configuration mismatch: %s", msg)
}

func (c *transferClient) connect() *pipeline.Error {
	cli, err := c.conns.Add(c.pip.TransCtx.RemoteAgent.Address.String(),
		c.tlsConfig, c.pip.Logger)
	if err != nil {
		c.pip.Logger.Error("Failed to connect to remote host: %s", err)

		return pipeline.NewErrorWith(types.TeConnection, "failed to connect to remote host", err)
	}

	c.ses, err = cli.NewSession()
	if err != nil {
		c.pip.Logger.Error("Failed to start R66 session: %s", err)

		return pipeline.NewErrorWith(types.TeConnection, "failed to start R66 session", err)
	}

	return nil
}

//nolint:funlen //no easy way to split this
func (c *transferClient) authenticate() *pipeline.Error {
	conf := &r66.Config{
		FileSize:   true,
		FinalHash:  !c.noFinalHash,
		DigestAlgo: c.finalHashAlgo,
		Proxified:  false,
	}

	var pwd []byte

	for _, cred := range c.pip.TransCtx.RemoteAccountCreds {
		if cred.Type == auth.Password {
			pwd = []byte(cred.Value)
		}
	}

	authent, err := c.ses.Authent(c.pip.TransCtx.RemoteAccount.Login, pwd, conf)
	if err != nil {
		c.ses = nil
		c.pip.Logger.Error("Client authentication failed: %s", err)

		return pipeline.NewErrorWith(types.TeBadAuthentication, "client authentication failed", err)
	}

	// Server authentication
	pswd := &model.Credential{}

	for _, cred := range c.pip.TransCtx.RemoteAgentCreds {
		if cred.Type == auth.Password {
			pswd = cred
		}
	}

	loginOK := utils.ConstantEqual(c.serverLogin, authent.Login)
	pwdErr := bcrypt.CompareHashAndPassword([]byte(pswd.Value), authent.Password)

	if !loginOK {
		c.pip.Logger.Error("Server authentication failed: wrong login %q", authent.Login)

		return pipeline.NewError(types.TeBadAuthentication, "server authentication failed")
	}

	if pwdErr != nil {
		c.pip.Logger.Error("Server authentication failed: wrong password: %v", pwdErr)

		return pipeline.NewError(types.TeBadAuthentication, "server authentication failed")
	}

	if authent.Filesize != conf.FileSize {
		c.logErrConf("file size verification")

		return errConf
	}

	if authent.FinalHash != conf.FinalHash {
		c.logErrConf("final hash verification")

		return errConf
	}

	if authent.Digest != conf.DigestAlgo {
		c.logErrConf("unknown digest algorithm")

		return errConf
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

	transID, err := c.pip.TransCtx.Transfer.TransferID()
	if err != nil {
		return pipeline.NewErrorWith(types.TeInternal, "failed to parse transfer ID", err)
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

			return pipeline.NewErrorWith(types.TeInternal, "failed to retrieve file size", statErr)
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

func (c *transferClient) makeHash() ([]byte, error) {
	if c.noFinalHash {
		return nil, nil
	}

	hash, err := internal.MakeHash(c.ctx, c.finalHashAlgo, c.pip.TransCtx.FS, c.pip.Logger,
		&c.pip.TransCtx.Transfer.LocalPath)
	if err != nil {
		return nil, internal.ToR66Error(err)
	}

	return hash, nil
}

var (
	errMissingCertificate = errors.New("TLS server provided no certificate during handshake")
	errBadCertificate     = errors.New("tls: bad certificate")
)

//nolint:funlen //no easy way to split this
func makeClientTLSConfig(pip *pipeline.Pipeline) (*tls.Config, error) {
	conf := &tls.Config{
		ServerName:       pip.TransCtx.RemoteAgent.Address.Host,
		MinVersion:       tls.VersionTLS12,
		VerifyConnection: compatibility.LogSha1(pip.Logger),
	}

	conf.Certificates = make([]tls.Certificate, 0, len(pip.TransCtx.RemoteAccountCreds))

	for _, cred := range pip.TransCtx.RemoteAccountCreds {
		if cred.Type == AuthLegacyCertificate {
			conf.Certificates = []tls.Certificate{compatibility.LegacyR66Cert}

			break
		}

		if cred.Type != auth.TLSCertificate {
			continue
		}

		tlsCert, err := utils.X509KeyPair(cred.Value, cred.Value2)
		if err != nil {
			return nil, fmt.Errorf("failed to parse TLS certificate: %w", err)
		}

		conf.Certificates = append(conf.Certificates, tlsCert)
	}

	caPool := utils.TLSCertPool()

	for _, cred := range pip.TransCtx.RemoteAgentCreds {
		if cred.Type == AuthLegacyCertificate {
			conf.InsecureSkipVerify = true
			conf.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
				if len(rawCerts) == 0 {
					return errMissingCertificate
				}

				chain, parsErr := auth.ParseRawCertChain(rawCerts)
				if parsErr != nil {
					return fmt.Errorf("failed to parse the certification chain: %w", parsErr)
				}

				if !compatibility.IsLegacyR66Cert(chain[0]) {
					return errBadCertificate
				}

				return nil
			}

			return conf, nil
		}

		if cred.Type != auth.TLSTrustedCertificate {
			continue
		}

		certChain, parseErr := utils.ParsePEMCertChain(cred.Value)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse the certification chain: %w", parseErr)
		}

		caPool.AddCert(certChain[0])
	}

	conf.RootCAs = caPool

	if err := auth.AddTLSAuthorities(pip.DB, conf); err != nil {
		return nil, fmt.Errorf("failed to setup TLS authorities: %w", err)
	}

	return conf, nil
}

package sftp

import (
	"errors"
	"io"
	"os"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/exp/slices"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// transferClient is the SFTP implementation of the `pipeline.TransferClient`
// interface which enables the gateway to execute SFTP transfers.
type transferClient struct {
	pip    *pipeline.Pipeline
	dialer *protoutils.TraceDialer

	partnerConf *partnerConfig
	sshConf     *ssh.Config

	sshClient  *ssh.Client
	sftpClient *sftp.Client
	sftpFile   *sftp.File
}

func newTransferClient(pip *pipeline.Pipeline, dialer *protoutils.TraceDialer, sshClientConf *ssh.Config,
) (*transferClient, *pipeline.Error) {
	var partnerConf partnerConfig
	if err := utils.JSONConvert(pip.TransCtx.RemoteAgent.ProtoConfig, &partnerConf); err != nil {
		pip.Logger.Error("Failed to parse SFTP partner protocol configuration: %s", err)

		return nil, pipeline.NewErrorWith(types.TeInternal,
			"failed to parse SFTP partner protocol configuration", err)
	}

	sshPartnerConf := &ssh.Config{
		KeyExchanges: sshClientConf.KeyExchanges,
		Ciphers:      sshClientConf.Ciphers,
		MACs:         sshClientConf.MACs,
	}

	if len(partnerConf.KeyExchanges) != 0 {
		sshPartnerConf.KeyExchanges = partnerConf.KeyExchanges
	}

	if len(partnerConf.Ciphers) != 0 {
		sshPartnerConf.Ciphers = partnerConf.Ciphers
	}

	if len(partnerConf.MACs) != 0 {
		sshPartnerConf.MACs = partnerConf.MACs
	}

	return &transferClient{
		pip:         pip,
		dialer:      dialer,
		partnerConf: &partnerConf,
		sshConf:     sshPartnerConf,
	}, nil
}

// algorithmsForKeyFormat returns the supported signature algorithms for a given
// public key format (PublicKey.Type), in order of preference. See RFC 8332,
// Section 2. See also the note in sendKexInit on backwards compatibility.
func algorithmsForKeyFormat(keyFormat string) []string {
	switch keyFormat {
	case ssh.KeyAlgoRSA:
		return []string{ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSA}
	case ssh.CertAlgoRSAv01:
		return []string{ssh.CertAlgoRSASHA256v01, ssh.CertAlgoRSASHA512v01, ssh.CertAlgoRSAv01}
	default:
		return []string{keyFormat}
	}
}

func (c *transferClient) makePartnerHostKeys(creds model.Credentials,
) ([]ssh.PublicKey, []string, *pipeline.Error) {
	var (
		hostKeys []ssh.PublicKey
		algos    []string
	)

	partner := c.pip.TransCtx.RemoteAgent.Name

	for _, cred := range creds {
		key, err := ParseAuthorizedKey(cred.Value)
		if err != nil {
			c.pip.Logger.Warning("Failed to parse the SFTP partner %q's hostkey %q: %v",
				partner, cred.Name, err)

			continue
		}

		hostKeys = append(hostKeys, key)

		for _, newAlgo := range algorithmsForKeyFormat(key.Type()) {
			if !slices.Contains(algos, newAlgo) {
				algos = append(algos, newAlgo)
			}
		}
	}

	if len(hostKeys) == 0 {
		c.pip.Logger.Error("No valid hostkey found for partner %q", partner)

		return nil, nil, pipeline.NewError(types.TeInternal,
			"no valid hostkey found for partner %q", partner)
	}

	return hostKeys, algos, nil
}

func (*transferClient) makeClientAuthMethods(creds model.Credentials,
) []ssh.AuthMethod {
	var (
		signers  []ssh.Signer
		auths    []ssh.AuthMethod
		password string
	)

	for _, c := range creds {
		switch c.Type {
		case auth.Password:
			password = c.Value
		case AuthSSHPrivateKey:
			signer, err := ParsePrivateKey(c.Value)
			if err != nil {
				continue
			}

			signers = append(signers, signer)
		}
	}

	if len(signers) > 0 {
		auths = append(auths, ssh.PublicKeys(signers...))
	}

	if len(password) > 0 {
		auths = append(auths, ssh.Password(password))
	}

	return auths
}

func setDefaultClientAlgos(sshConf *ssh.ClientConfig) {
	if len(sshConf.KeyExchanges) == 0 {
		sshConf.KeyExchanges = validKeyExchanges.ClientDefaults()
	}

	if len(sshConf.Ciphers) == 0 {
		sshConf.Ciphers = validCiphers.ClientDefaults()
	}

	if len(sshConf.MACs) == 0 {
		sshConf.MACs = validMACs.ClientDefaults()
	}
}

func (c *transferClient) makeSSHClientConfig(info *model.TransferContext,
) (*ssh.ClientConfig, *pipeline.Error) {
	hostKeys, algos, err := c.makePartnerHostKeys(info.RemoteAgentCreds)
	if err != nil {
		return nil, err
	}

	authMethods := c.makeClientAuthMethods(info.RemoteAccountCreds)

	certChecker := &ssh.CertChecker{
		IsHostAuthority: isHostAuthority(c.pip.DB, c.pip.Logger),
		HostKeyFallback: makeFixedHostKeys(hostKeys),
	}

	sshConf := &ssh.ClientConfig{
		Config:            *c.sshConf,
		User:              info.RemoteAccount.Login,
		Auth:              authMethods,
		HostKeyCallback:   certChecker.CheckHostKey,
		HostKeyAlgorithms: algos,
	}

	setDefaultClientAlgos(sshConf)

	return sshConf, nil
}

func (c *transferClient) openSSHConn() *pipeline.Error {
	sshClientConf, confErr := c.makeSSHClientConfig(c.pip.TransCtx)
	if confErr != nil {
		return confErr
	}

	addr := conf.GetRealAddress(c.pip.TransCtx.RemoteAgent.Address.Host,
		utils.FormatUint(c.pip.TransCtx.RemoteAgent.Address.Port))

	conn, dialErr := c.dialer.Dial("tcp", addr)
	if dialErr != nil {
		c.pip.Logger.Error("Failed to connect to the SFTP partner: %v", dialErr)

		return pipeline.NewErrorWith(types.TeConnection,
			"failed to connect to the SFTP partner", dialErr)
	}

	sshConn, chans, reqs, sshErr := ssh.NewClientConn(conn, addr, sshClientConf)
	if sshErr != nil {
		c.pip.Logger.Error("Failed to start the SSH session: %v", sshErr)

		return pipeline.NewErrorWith(types.TeConnection,
			"failed to start the SSH session", sshErr)
	}

	c.sshClient = ssh.NewClient(sshConn, chans, reqs)

	return nil
}

func (c *transferClient) startSFTPSession() *pipeline.Error {
	var opts []sftp.ClientOption

	if !c.partnerConf.UseStat {
		opts = append(opts, sftp.UseFstat(true))
	}

	if c.partnerConf.DisableClientConcurrentReads {
		opts = append(opts, sftp.UseConcurrentReads(false))
	}

	var sftpErr error

	c.sftpClient, sftpErr = sftp.NewClient(c.sshClient, opts...)
	if sftpErr != nil {
		c.pip.Logger.Error("Failed to start SFTP session: %s", sftpErr)

		return fromSFTPErr(sftpErr, types.TeUnknownRemote, c.pip)
	}

	return nil
}

func (c *transferClient) Request() *pipeline.Error {
	if tErr := c.request(); tErr != nil {
		c.SendError(tErr.Code(), tErr.Details())

		return tErr
	}

	return nil
}

func (c *transferClient) request() *pipeline.Error {
	if err := c.openSSHConn(); err != nil {
		return err
	}

	if err := c.startSFTPSession(); err != nil {
		return err
	}

	if filepath := c.pip.TransCtx.Transfer.RemotePath; c.pip.TransCtx.Rule.IsSend {
		return c.requestSend(filepath)
	} else {
		return c.requestReceive(filepath)
	}
}

func (c *transferClient) requestSend(filepath string) *pipeline.Error {
	if c.pip.TransCtx.Transfer.Progress > 0 {
		if stat, statErr := c.sftpClient.Stat(filepath); statErr != nil {
			c.pip.Logger.Warning("Failed to retrieve the remote file's size: %s", statErr)
			c.pip.TransCtx.Transfer.Progress = 0
		} else {
			c.pip.TransCtx.Transfer.Progress = stat.Size()
		}

		if err := c.pip.UpdateTrans(); err != nil {
			return err
		}
	}

	var err error

	c.sftpFile, err = c.sftpClient.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		c.pip.Logger.Error("Failed to create remote file: %s", err)

		return fromSFTPErr(err, types.TeUnknownRemote, c.pip)
	}

	return nil
}

func (c *transferClient) requestReceive(filepath string) *pipeline.Error {
	var err error

	c.sftpFile, err = c.sftpClient.Open(filepath)
	if err != nil {
		c.pip.Logger.Error("Failed to open remote file: %s", err)

		return fromSFTPErr(err, types.TeUnknownRemote, c.pip)
	}

	return nil
}

// Send copies the content from the local source file to the remote one.
func (c *transferClient) Send(file protocol.SendFile) *pipeline.Error {
	// Check parent dir, if it doesn't exist, try to create it
	parentDir := path.Dir(c.pip.TransCtx.Transfer.RemotePath)
	if _, statErr := c.sftpClient.Stat(parentDir); errors.Is(statErr, fs.ErrNotExist) {
		if mkdirErr := c.sftpClient.MkdirAll(parentDir); mkdirErr != nil {
			c.pip.Logger.Warning("Failed to create remote parent directory: %s", mkdirErr)
		}
	} else if statErr != nil {
		c.pip.Logger.Warning("Failed to stat remote parent directory: %s", statErr)
	}

	if _, err := c.sftpFile.ReadFrom(file); err != nil {
		c.pip.Logger.Error("Failed to write to remote SFTP file: %s", err)

		return c.wrapAndSendError(err, types.TeDataTransfer)
	}

	return nil
}

func (c *transferClient) Receive(file protocol.ReceiveFile) *pipeline.Error {
	if c.pip.TransCtx.Transfer.Progress != 0 {
		if _, err := c.sftpFile.Seek(c.pip.TransCtx.Transfer.Progress, io.SeekStart); err != nil {
			c.pip.Logger.Error("Failed to seek into remote SFTP file: %s", err)

			return c.wrapAndSendError(err, types.TeUnknownRemote)
		}
	}

	if _, err := c.sftpFile.WriteTo(file); err != nil {
		c.pip.Logger.Error("Failed to read from remote SFTP file: %s", err)

		return c.wrapAndSendError(err, types.TeDataTransfer)
	}

	return nil
}

func (c *transferClient) EndTransfer() *pipeline.Error {
	if err := c.endTransfer(); err != nil {
		return err
	}

	return nil
}

func (c *transferClient) endTransfer() (tErr *pipeline.Error) {
	if c.sftpFile != nil {
		if err := c.sftpFile.Close(); err != nil {
			c.pip.Logger.Error("Failed to close remote SFTP file: %s", err)

			if cErr := c.sftpClient.Close(); cErr != nil {
				c.pip.Logger.Warning("An error occurred while closing the SFTP session: %v", cErr)
			}

			tErr = fromSFTPErr(err, types.TeFinalization, c.pip)
		}
	}

	if c.sftpClient != nil {
		if err := c.sftpClient.Close(); err != nil {
			c.pip.Logger.Error("Failed to close SFTP session: %s", err)

			if tErr == nil {
				tErr = fromSFTPErr(err, types.TeFinalization, c.pip)
			}
		}
	}

	if c.sshClient != nil {
		if err := c.sshClient.Close(); err != nil {
			c.pip.Logger.Error("Failed to close SSH session: %s", err)

			if tErr == nil {
				tErr = fromSFTPErr(err, types.TeFinalization, c.pip)
			}
		}
	}

	return tErr
}

func (c *transferClient) wrapAndSendError(err error, defaultCode types.TransferErrorCode) *pipeline.Error {
	tErr := fromSFTPErr(err, defaultCode, c.pip)
	c.SendError(tErr.Code(), tErr.Details())

	return tErr
}

func (c *transferClient) SendError(types.TransferErrorCode, string) {
	if c.sshClient != nil {
		if err := c.sshClient.Close(); err != nil {
			c.pip.Logger.Warning("An error occurred while closing the SSH session: %v", err)
		}
	}
}

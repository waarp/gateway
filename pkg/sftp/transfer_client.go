package sftp

import (
	"errors"
	"io"
	"net"
	"os"
	"regexp"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

// transferClient is the SFTP implementation of the `pipeline.TransferClient`
// interface which enables the gateway to execute SFTP transfers.
type transferClient struct {
	pip    *pipeline.Pipeline
	dialer *net.Dialer

	partnerConf *config.SftpPartnerProtoConfig
	sshConf     *ssh.Config

	sshClient  *ssh.Client
	sftpClient *sftp.Client
	sftpFile   *sftp.File
}

func newTransferClient(pip *pipeline.Pipeline, dialer *net.Dialer, sshClientConf *ssh.Config,
) (*transferClient, *types.TransferError) {
	var partnerConf config.SftpPartnerProtoConfig
	if err := utils.JSONConvert(pip.TransCtx.RemoteAgent.ProtoConfig, &partnerConf); err != nil {
		pip.Logger.Error("Failed to parse SFTP partner protocol configuration: %s", err)

		return nil, types.NewTransferError(types.TeInternal,
			"failed to parse SFTP partner protocol configuration")
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

func (c *transferClient) makePartnerHostKeys(cryptos model.Cryptos,
) ([]ssh.PublicKey, []string, *types.TransferError) {
	var (
		hostKeys []ssh.PublicKey
		algos    []string
	)

	partner := c.pip.TransCtx.RemoteAgent.Name

	for _, crypto := range cryptos {
		key, err := ParseAuthorizedKey([]byte(crypto.SSHPublicKey))
		if err != nil {
			c.pip.Logger.Warning("Failed to parse the SFTP partner %q's hostkey %q: %v",
				partner, crypto.Name, err)

			continue
		}

		hostKeys = append(hostKeys, key)

		if !utils.ContainsStrings(algos, key.Type()) {
			algos = append(algos, key.Type())
		}
	}

	if len(hostKeys) == 0 {
		c.pip.Logger.Error("No valid hostkey found for partner %q", partner)

		return nil, nil, types.NewTransferError(types.TeInternal,
			"no valid hostkey found for partner %q", partner)
	}

	return hostKeys, algos, nil
}

func (*transferClient) makeClientAuthMethods(password string, cryptos model.Cryptos,
) []ssh.AuthMethod {
	var (
		signers []ssh.Signer
		auths   []ssh.AuthMethod
	)

	for _, c := range cryptos {
		signer, err := ssh.ParsePrivateKey([]byte(c.PrivateKey))
		if err != nil {
			continue
		}

		signers = append(signers, signer)
	}

	if len(signers) > 0 {
		auths = append(auths, ssh.PublicKeys(signers...))
	}

	if len(password) > 0 {
		auths = append(auths, ssh.Password(password))
	}

	return auths
}

func setDefaultClientAlgos(conf *ssh.ClientConfig) {
	if len(conf.KeyExchanges) == 0 {
		conf.KeyExchanges = config.SFTPValidKeyExchanges.ClientDefaults()
	}

	if len(conf.Ciphers) == 0 {
		conf.Ciphers = config.SFTPValidCiphers.ClientDefaults()
	}

	if len(conf.MACs) == 0 {
		conf.MACs = config.SFTPValidMACs.ClientDefaults()
	}
}

func (c *transferClient) makeSSHClientConfig(info *model.TransferContext,
) (*ssh.ClientConfig, *types.TransferError) {
	hostKeys, algos, err := c.makePartnerHostKeys(info.RemoteAgentCryptos)
	if err != nil {
		return nil, err
	}

	authMethods := c.makeClientAuthMethods(string(info.RemoteAccount.Password),
		info.RemoteAccountCryptos)

	conf := &ssh.ClientConfig{
		Config:            *c.sshConf,
		User:              info.RemoteAccount.Login,
		Auth:              authMethods,
		HostKeyCallback:   makeFixedHostKeys(hostKeys),
		HostKeyAlgorithms: algos,
	}

	setDefaultClientAlgos(conf)

	return conf, nil
}

func (c *transferClient) openSSHConn() *types.TransferError {
	sshClientConf, confErr := c.makeSSHClientConfig(c.pip.TransCtx)
	if confErr != nil {
		return confErr
	}

	addr := c.pip.TransCtx.RemoteAgent.Address

	conn, dialErr := c.dialer.Dial("tcp", addr)
	if dialErr != nil {
		c.pip.Logger.Error("Failed to connect to the SFTP partner: %v", dialErr)

		return types.NewTransferError(types.TeConnection,
			"failed to connect to the SFTP partner: %v", dialErr)
	}

	sshConn, chans, reqs, sshErr := ssh.NewClientConn(conn, addr, sshClientConf)
	if sshErr != nil {
		c.pip.Logger.Error("Failed to start the SSH session: %v", sshErr)

		return types.NewTransferError(types.TeConnection,
			"failed to start the SSH session: %v", sshErr)
	}

	c.sshClient = ssh.NewClient(sshConn, chans, reqs)

	return nil
}

func (c *transferClient) startSFTPSession() *types.TransferError {
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

		return c.fromSFTPErr(sftpErr, types.TeUnknownRemote)
	}

	return nil
}

func (c *transferClient) Request() (tErr *types.TransferError) {
	if err := c.openSSHConn(); err != nil {
		return err
	}

	defer func() {
		if tErr != nil {
			c.SendError(tErr)
		}
	}()

	if err := c.startSFTPSession(); err != nil {
		return err
	}

	if filepath := c.pip.TransCtx.Transfer.RemotePath; c.pip.TransCtx.Rule.IsSend {
		return c.requestSend(filepath)
	} else {
		return c.requestReceive(filepath)
	}
}

func (c *transferClient) requestSend(filepath string) *types.TransferError {
	if c.pip.TransCtx.Transfer.Progress > 0 {
		if stat, statErr := c.sftpClient.Stat(filepath); statErr != nil {
			c.pip.Logger.Warning("Failed to retrieve the remote file's size: %s", statErr)
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

		return c.fromSFTPErr(err, types.TeUnknownRemote)
	}

	return nil
}

func (c *transferClient) requestReceive(filepath string) *types.TransferError {
	var err error

	c.sftpFile, err = c.sftpClient.Open(filepath)
	if err != nil {
		c.pip.Logger.Error("Failed to open remote file: %s", err)

		return c.fromSFTPErr(err, types.TeUnknownRemote)
	}

	return nil
}

// Data copies the content of the source file into the destination file.
func (c *transferClient) Data(data pipeline.DataStream) (tErr *types.TransferError) {
	defer func() {
		if tErr != nil {
			c.SendError(tErr)
		}
	}()

	if c.pip.TransCtx.Transfer.Progress != 0 {
		_, err := c.sftpFile.Seek(c.pip.TransCtx.Transfer.Progress, io.SeekStart)
		if err != nil {
			c.pip.Logger.Error("Failed to seek into remote SFTP file: %s", err)

			return c.fromSFTPErr(err, types.TeUnknownRemote)
		}
	}

	if c.pip.TransCtx.Rule.IsSend {
		_, err := c.sftpFile.ReadFrom(data)
		if err != nil {
			c.pip.Logger.Error("Failed to write to remote SFTP file: %s", err)

			return c.fromSFTPErr(err, types.TeDataTransfer)
		}
	} else {
		_, err := c.sftpFile.WriteTo(data)
		if err != nil {
			c.pip.Logger.Error("Failed to read from remote SFTP file: %s", err)

			return c.fromSFTPErr(err, types.TeDataTransfer)
		}
	}

	return nil
}

func (c *transferClient) EndTransfer() (tErr *types.TransferError) {
	if c.sftpFile != nil {
		if err := c.sftpFile.Close(); err != nil {
			c.pip.Logger.Error("Failed to close remote SFTP file: %s", err)

			if cErr := c.sftpClient.Close(); cErr != nil {
				c.pip.Logger.Warning("An error occurred while closing the SFTP session: %v", cErr)
			}

			tErr = c.fromSFTPErr(err, types.TeFinalization)
		}
	}

	if c.sftpClient != nil {
		if err := c.sftpClient.Close(); err != nil {
			c.pip.Logger.Error("Failed to close SFTP session: %s", err)

			if tErr == nil {
				tErr = c.fromSFTPErr(err, types.TeFinalization)
			}
		}
	}

	if c.sshClient != nil {
		if err := c.sshClient.Close(); err != nil {
			c.pip.Logger.Error("Failed to close SSH session: %s", err)

			if tErr == nil {
				tErr = c.fromSFTPErr(err, types.TeFinalization)
			}
		}
	}

	return tErr
}

func (c *transferClient) SendError(*types.TransferError) {
	if c.sshClient != nil {
		if err := c.sshClient.Close(); err != nil {
			c.pip.Logger.Warning("An error occurred while closing the SSH session: %v", err)
		}
	}
}

func (c *transferClient) fromSFTPErr(err error, defaults types.TransferErrorCode) *types.TransferError {
	code := defaults
	msg := err.Error()

	var sErr *sftp.StatusError
	if !errors.As(err, &sErr) {
		return types.NewTransferError(code, msg)
	}

	regex := regexp.MustCompile(`sftp: "TransferError\((Te\w*)\): (.*)" \(.*\)`)

	s := regex.FindStringSubmatch(err.Error())
	if len(s) >= 3 { //nolint:gomnd // using a const is unnecessary
		code = types.TecFromString(s[1])
		switch code {
		case types.TeStopped:
			c.pip.Pause()

		case types.TeCanceled:
			c.pip.Cancel()

		default:
		}

		msg = s[2]

		return types.NewTransferError(code, msg)
	}

	switch sErr.FxCode() {
	case sftp.ErrSSHFxOk, sftp.ErrSSHFxEOF:
		return nil
	case sftp.ErrSSHFxNoSuchFile:
		code = types.TeFileNotFound
	case sftp.ErrSSHFxPermissionDenied:
		code = types.TeForbidden
	case sftp.ErrSSHFxFailure:
		code = types.TeUnknownRemote
	case sftp.ErrSSHFxBadMessage:
		code = types.TeUnimplemented
	case sftp.ErrSSHFxNoConnection:
		code = types.TeConnection
	case sftp.ErrSSHFxConnectionLost:
		code = types.TeConnectionReset
	case sftp.ErrSSHFxOpUnsupported:
		code = types.TeUnimplemented
	}

	regex2 := regexp.MustCompile(`sftp: "(.*)" \(.*\)`)

	s2 := regex2.FindStringSubmatch(err.Error())
	if len(s2) >= 1 {
		msg = s2[1]
	}

	return types.NewTransferError(code, msg)
}

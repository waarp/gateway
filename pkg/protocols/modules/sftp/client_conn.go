package sftp

import (
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type sftpConnPool = protoutils.ConnPool[*clientConn]

type clientConn struct {
	*sftp.Client

	ssh ssh.Conn
}

//nolint:wrapcheck //no need to wrap here
func (c *clientConn) Close() error {
	if err := c.Client.Close(); err != nil {
		defer c.ssh.Close()

		return err
	}

	return c.ssh.Close()
}

func (c *client) newClientConn(pip *pipeline.Pipeline, dialer *protoutils.TraceDialer) (*clientConn, error) {
	var partnerConf partnerConfig
	if err := utils.JSONConvert(pip.TransCtx.RemoteAgent.ProtoConfig, &partnerConf); err != nil {
		pip.Logger.Errorf("Failed to parse SFTP partner protocol configuration: %v", err)

		return nil, pipeline.NewErrorWith(types.TeInternal,
			"failed to parse SFTP partner protocol configuration", err)
	}

	sshPartnerConf := &ssh.Config{
		KeyExchanges: c.sshConf.KeyExchanges,
		Ciphers:      c.sshConf.Ciphers,
		MACs:         c.sshConf.MACs,
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

	sshConn, err := openSSHConn(pip, dialer, sshPartnerConf)
	if err != nil {
		return nil, err
	}

	sftpSes, err := startSFTPSession(sshConn, &partnerConf, pip)
	if err != nil {
		return nil, err
	}

	return &clientConn{sftpSes, sshConn}, nil
}

func openSSHConn(pip *pipeline.Pipeline, dialer *protoutils.TraceDialer, sshConfig *ssh.Config,
) (*ssh.Client, *pipeline.Error) {
	sshClientConf, confErr := makeSSHClientConfig(pip, sshConfig)
	if confErr != nil {
		return nil, confErr
	}

	addr := conf.GetRealAddress(pip.TransCtx.RemoteAgent.Address.Host,
		utils.FormatUint(pip.TransCtx.RemoteAgent.Address.Port))

	conn, dialErr := dialer.Dial("tcp", addr)
	if dialErr != nil {
		pip.Logger.Errorf("Failed to connect to the SFTP partner: %v", dialErr)

		return nil, pipeline.NewErrorWith(types.TeConnection,
			"failed to connect to the SFTP partner", dialErr)
	}

	sshConn, chans, reqs, sshErr := ssh.NewClientConn(conn, addr, sshClientConf)
	if sshErr != nil {
		pip.Logger.Errorf("Failed to start the SSH session: %v", sshErr)

		return nil, pipeline.NewErrorWith(types.TeConnection,
			"failed to start the SSH session", sshErr)
	}

	return ssh.NewClient(sshConn, chans, reqs), nil
}

func startSFTPSession(sshConn *ssh.Client, partnerConf *partnerConfig, pip *pipeline.Pipeline,
) (*sftp.Client, *pipeline.Error) {
	var opts []sftp.ClientOption

	if !partnerConf.UseStat {
		opts = append(opts, sftp.UseFstat(true))
	}

	if partnerConf.DisableClientConcurrentReads {
		opts = append(opts, sftp.UseConcurrentReads(false))
	}

	sftpSes, sftpErr := sftp.NewClient(sshConn, opts...)
	if sftpErr != nil {
		pip.Logger.Errorf("Failed to start SFTP session: %v", sftpErr)

		return nil, fromSFTPErr(sftpErr, types.TeUnknownRemote, pip)
	}

	return sftpSes, nil
}

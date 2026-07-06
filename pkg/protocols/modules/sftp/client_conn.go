package sftp

import (
	"fmt"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type sftpConnPool = protoutils.ConnPool[*ClientConn]

type ClientConn struct {
	*sftp.Client

	ssh ssh.Conn
}

//nolint:wrapcheck //no need to wrap here
func (c *ClientConn) Close() error {
	if err := c.Client.Close(); err != nil {
		defer c.ssh.Close()

		return err
	}

	return c.ssh.Close()
}

func (c *client) newClientConn(pip *pipeline.Pipeline, dialer *protoutils.TraceDialer) (*ClientConn, error) {
	return OpenConn(pip.Logger, pip.TransCtx, dialer, pip.DB.Config.Overrides)
}

func OpenConn(logger *log.Logger, ctx *model.TransferContext, dialer *protoutils.TraceDialer,
	overrides *conf.ConfigOverride,
) (*ClientConn, error) {
	var clientConf ClientConfig
	if err := utils.JSONConvert(ctx.Client.ProtoConfig, &clientConf); err != nil {
		return nil, fmt.Errorf("failed to parse the SFTP client's proto config: %w", err)
	}

	sshConf := &ssh.Config{
		KeyExchanges: clientConf.KeyExchanges,
		Ciphers:      clientConf.Ciphers,
		MACs:         clientConf.MACs,
	}

	return openConn(logger, ctx, dialer, sshConf, overrides)
}

func openConn(logger *log.Logger, ctx *model.TransferContext,
	dialer *protoutils.TraceDialer, sshConf *ssh.Config, overrides *conf.ConfigOverride,
) (*ClientConn, error) {
	var partnerConf PartnerConfig
	if err := utils.JSONConvert(ctx.RemoteAgent.ProtoConfig, &partnerConf); err != nil {
		logger.Errorf("Failed to parse SFTP partner protocol configuration: %v", err)

		return nil, pipeline.NewErrorWith(err, types.TeInternal, "failed to parse SFTP partner protocol configuration")
	}

	sshPartnerConf := &ssh.Config{
		KeyExchanges: sshConf.KeyExchanges,
		Ciphers:      sshConf.Ciphers,
		MACs:         sshConf.MACs,
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

	sshConn, err := openSSHConn(logger, ctx, dialer, sshPartnerConf, overrides)
	if err != nil {
		return nil, err
	}

	sftpSes, err := startSFTPSession(logger, sshConn, &partnerConf)
	if err != nil {
		_ = sshConn.Close() //nolint:errcheck //close error is irrelevant here

		return nil, err
	}

	return &ClientConn{sftpSes, sshConn}, nil
}

func openSSHConn(logger *log.Logger, ctx *model.TransferContext,
	dialer *protoutils.TraceDialer, sshConfig *ssh.Config, overrides *conf.ConfigOverride,
) (*ssh.Client, *pipeline.Error) {
	sshClientConf, confErr := makeSSHClientConfig(logger, ctx, sshConfig)
	if confErr != nil {
		return nil, confErr
	}

	addr := overrides.GetRealAddress(ctx.RemoteAgent.Address.Host,
		utils.FormatUint(ctx.RemoteAgent.Address.Port))

	conn, dialErr := dialer.Dial("tcp", addr)
	if dialErr != nil {
		logger.Errorf("Failed to connect to the SFTP partner: %v", dialErr)

		return nil, pipeline.NewErrorWith(dialErr, types.TeConnection,
			"failed to connect to the SFTP partner")
	}

	sshConn, chans, reqs, sshErr := ssh.NewClientConn(conn, addr, sshClientConf)
	if sshErr != nil {
		logger.Errorf("Failed to start the SSH session: %v", sshErr)

		return nil, pipeline.NewErrorWith(sshErr, types.TeConnection,
			"failed to start the SSH session")
	}

	return ssh.NewClient(sshConn, chans, reqs), nil
}

func startSFTPSession(logger *log.Logger, sshConn *ssh.Client, partnerConf *PartnerConfig,
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
		logger.Errorf("Failed to start SFTP session: %v", sftpErr)

		return nil, pipeline.NewErrorWith(sftpErr, types.TeUnknownRemote, "failed to start SFTP session")
	}

	return sftpSes, nil
}

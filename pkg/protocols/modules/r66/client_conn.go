package r66

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const ClientDialTimeout = 10 * time.Second

type r66ConnPool = protoutils.ConnPool[*ClientConn]

type ClientConn struct {
	*r66.Client
}

func (c *ClientConn) Close() error {
	c.Client.Close()

	return nil
}

func makeDialer(client *model.Client) (*protoutils.TraceDialer, error) {
	dialer := &net.Dialer{Timeout: ClientDialTimeout}

	if client.LocalAddress.IsSet() {
		var err error
		if dialer.LocalAddr, err = net.ResolveTCPAddr("tcp",
			client.LocalAddress.String()); err != nil {
			return nil, fmt.Errorf("failed to parse the R66 client's local address: %w", err)
		}
	}

	return &protoutils.TraceDialer{Dialer: dialer}, nil
}

func (c *Client) openNewConn(pip *pipeline.Pipeline, dialer *protoutils.TraceDialer) (*ClientConn, error) {
	return dialClientConn(c.logger, pip.TransCtx, dialer, pip.DB.Config.Overrides)
}

func OpenConn(logger *log.Logger, ctx *model.TransferContext, dialer *protoutils.TraceDialer,
	overrides *conf.ConfigOverride,
) (*ClientConn, error) {
	var clientConfig ClientConfigTLS
	if err := utils.JSONConvert(ctx.Client.ProtoConfig, &clientConfig); err != nil {
		return nil, fmt.Errorf("failed to parse the R66 client's proto config: %w", err)
	}

	return dialClientConn(logger, ctx, dialer, overrides)
}

func dialClientConn(logger *log.Logger, ctx *model.TransferContext,
	dialer *protoutils.TraceDialer, overrides *conf.ConfigOverride,
) (*ClientConn, error) {
	var partConf PartnerConfigTLS
	if err := utils.JSONConvert(ctx.RemoteAgent.ProtoConfig, &partConf); err != nil {
		logger.Errorf("Failed to parse R66 partner proto config: %v", err)

		return nil, pipeline.NewErrorWith(err, types.TeInternal, "failed to parse R66 partner proto config")
	}

	var tlsConf *tls.Config
	if ctx.Client.Protocol == R66TLS {
		var err error
		if tlsConf, err = makeClientTLSConfig(logger, ctx); err != nil {
			logger.Errorf("Failed to parse R66 TLS config: %v", err)

			return nil, pipeline.NewErrorWith(err, types.TeInternal, "invalid R66 TLS config")
		}
	}

	addr := overrides.GetRealAddress(ctx.RemoteAgent.Address.Host,
		utils.FormatUint(ctx.RemoteAgent.Address.Port))

	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate the TCP connection: %w", err)
	}

	if tlsConf != nil {
		conn = tls.Client(conn, tlsConf)
	}

	client, err := r66.NewClient(conn, logger.AsStdLogger(log.LevelTrace))
	if err != nil {
		return nil, fmt.Errorf("failed to initiate the R66 connection: %w", err)
	}

	return &ClientConn{client}, nil
}

func (c *ClientConn) Authenticate(logger *log.Logger, ctx *model.TransferContext,
) (*r66.Session, error) {
	partnerLogin, err := utils.GetAs[string](ctx.RemoteAgent.ProtoConfig, "login")
	if err != nil {
		partnerLogin = ctx.RemoteAgent.Name
	}

	return c.authenticate(logger, ctx, false, "", partnerLogin)
}

func (c *ClientConn) authenticate(logger *log.Logger, ctx *model.TransferContext,
	noFinalHash bool, finalHashAlgo, partnerLogin string,
) (*r66.Session, error) {
	ses, sesErr := c.NewSession()
	if sesErr != nil {
		logger.Errorf("Failed to start R66 session: %s", sesErr)

		return nil, pipeline.NewErrorWith(sesErr, types.TeConnection, "failed to start R66 session")
	}

	r66Conf := &r66.Config{
		FileSize:   true,
		FinalHash:  !noFinalHash,
		DigestAlgo: finalHashAlgo,
		Proxified:  false,
	}

	var pwd []byte

	for _, cred := range ctx.RemoteAccountCreds {
		if cred.Type == auth.Password {
			pwd = []byte(cred.Value)
		}
	}

	authent, err := ses.Authent(ctx.RemoteAccount.Login, pwd, r66Conf)
	if err != nil {
		logger.Errorf("Client authentication failed: %v", err)

		return nil, pipeline.NewErrorWith(err, types.TeBadAuthentication, "client authentication failed")
	}

	// Server authentication
	pswd := &model.Credential{}
	for _, cred := range ctx.RemoteAgentCreds {
		if cred.Type == auth.Password {
			pswd = cred
			break
		}
	}

	loginOK := utils.ConstantEqual(partnerLogin, authent.Login)
	pwdOK := utils.IsHashOf(pswd.Value, string(authent.Password))

	if !loginOK {
		logger.Errorf("Server authentication failed: wrong login %q", authent.Login)

		return nil, pipeline.NewError(types.TeBadAuthentication, "server authentication failed")
	}

	if !pwdOK {
		logger.Error("Server authentication failed: wrong password")

		return nil, pipeline.NewError(types.TeBadAuthentication, "server authentication failed")
	}

	if authent.Filesize != r66Conf.FileSize {
		logErrConf(logger, "file size verification")

		return nil, errConf
	}

	if authent.FinalHash != r66Conf.FinalHash {
		logErrConf(logger, "final hash verification")

		return nil, errConf
	}

	if authent.Digest != r66Conf.DigestAlgo {
		logErrConf(logger, "unknown digest algorithm")

		return nil, errConf
	}

	return ses, nil
}

func logErrConf(logger *log.Logger, msg string) {
	logger.Errorf("Client-server configuration mismatch: %s", msg)
}

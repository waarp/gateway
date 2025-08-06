package pesit

import (
	"crypto/tls"
	"errors"
	"io"
	"net"

	"code.waarp.fr/lib/log"
	"code.waarp.fr/lib/pesit"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// Ensures Client implements the optional client interfaces.
var _ interface {
	protocol.TransferClient
	protocol.PauseHandler
	protocol.CancelHandler
} = &clientTransfer{}

type clientTransfer struct {
	isTLS      bool
	pip        *pipeline.Pipeline
	clientConf *ClientConfigTLS
	client     *pesit.Client
	dialer     *protoutils.TraceDialer
	pesitID    uint32

	pTrans *pesit.ClientTransfer
}

func (c *clientTransfer) configureClient(config *PartnerConfig) *pipeline.Error {
	// configure checkpoints
	if utils.If(config.DisableCheckpoints.Valid,
		config.DisableCheckpoints.Value,
		c.clientConf.DisableCheckpoints) {
		c.client.AllowCheckpoints(pesit.CheckpointDisabled, 0)
	} else {
		c.client.AllowCheckpoints(
			utils.If(config.CheckpointSize != 0,
				config.CheckpointSize,
				c.clientConf.CheckpointSize),
			utils.If(config.CheckpointWindow != 0,
				config.CheckpointWindow,
				c.clientConf.CheckpointWindow))

		// configure restarts
		c.client.AllowRestart(!utils.If(config.DisableRestart.Valid,
			config.DisableRestart.Valid, c.clientConf.DisableRestart))
	}

	if config.UseNSDU {
		c.client.SetNSDUUsage(true)
	}

	if config.CompatibilityMode == CompatibilityModeNonStandard {
		c.client.SetCFTCompatibilityUsage(true)
	}

	if config.DisablePreConnection {
		c.client.SetPreConnectionUsage(false)
	} else {
		c.client.SetPreConnectionUsage(true)

		for _, cred := range c.pip.TransCtx.RemoteAccountCreds {
			if cred.Type == preConnectionAuth {
				c.client.SetPreConnectLogin(cred.Value)
				c.client.SetPreConnectPassword(cred.Value2)

				break
			}
		}
	}

	return setFreetext(c.pip, clientConnFreetextKey, c.client)
}

func (c *clientTransfer) Request() *pipeline.Error {
	var fileInfo fs.FileInfo

	if c.pip.TransCtx.Rule.IsSend {
		var statErr error
		if fileInfo, statErr = fs.Stat(c.pip.TransCtx.Transfer.LocalPath); statErr != nil {
			return pipeline.FileErrToTransferErr(statErr)
		}
	}

	// parse the partner's proto config
	var partConf PartnerConfigTLS
	if err := utils.JSONConvert(c.pip.TransCtx.RemoteAgent.ProtoConfig, &partConf); err != nil {
		c.pip.Logger.Errorf("Failed to parse the pesit partner's proto config: %v", err)

		return pipeline.NewErrorWith(types.TeInternal, "failed to parse the pesit partner's proto config", err)
	}

	// connect to partner
	realAddr := conf.GetRealAddress(c.pip.TransCtx.RemoteAgent.Address.Host,
		utils.FormatUint(c.pip.TransCtx.RemoteAgent.Address.Port))

	conn, connErr := c.dialer.Dial("tcp", realAddr)
	if connErr != nil {
		c.pip.Logger.Errorf("Failed to connect to partner: %v", connErr)

		return pipeline.NewErrorWith(types.TeConnection, "failed to connect to partner", connErr)
	}

	if err := c.request(fileInfo, &partConf, conn); err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			c.pip.Logger.Warningf("Failed to close connection: %v", closeErr)
		}

		return err
	}

	return nil
}

//nolint:funlen,gocognit,gocyclo,cyclop //no easy way to split the function for now
func (c *clientTransfer) request(fileInfo fs.FileInfo, partConf *PartnerConfigTLS,
	conn net.Conn,
) *pipeline.Error {
	serverLogin := c.pip.TransCtx.RemoteAgent.Name
	if partConf.Login != "" {
		serverLogin = partConf.Login
	}

	c.client = pesit.NewClient(c.pip.TransCtx.RemoteAccount.Login,
		getPassword(c.pip.TransCtx), serverLogin)
	c.client.Logger = c.pip.Logger.AsStdLogger(log.LevelDebug)

	if err := c.configureClient(&partConf.PartnerConfig); err != nil {
		return err
	}

	if c.isTLS {
		tlsConfig, tlsErr := c.makeTLSConfig(c.pip.TransCtx.RemoteAgent.Address.Host, partConf)
		if tlsErr != nil {
			c.pip.Logger.Errorf("Failed to parse TLS config: %v", tlsErr)

			return pipeline.NewErrorWith(types.TeInternal, "failed to parse TLS config", tlsErr)
		}

		conn = tls.Client(conn, tlsConfig)
	}

	if err := c.client.Connect(conn); err != nil {
		c.pip.Logger.Errorf("Failed to open PeSIT connection: %v", err)

		return pipeline.NewErrorWith(types.TeConnection, "failed to connect to partner", err)
	}

	if err := c.authenticateServer(); err != nil {
		return err
	}

	setTransInfo(c.pip, serverConnFreetextKey, c.client.FreeText())

	// initialize transfer
	method := pesit.MethodRecv

	if c.pip.TransCtx.Rule.IsSend {
		c.client.SetAccessType(pesit.AccessWrite)
		method = pesit.MethodSend
	} else {
		c.client.SetAccessType(pesit.AccessRead)
	}

	c.pTrans = pesit.NewTransfer(method, c.pip.TransCtx.Transfer.RemotePath)

	// configure recovery if transfer is resumed
	if prog := c.pip.TransCtx.Transfer.Progress; prog != 0 ||
		c.pip.TransCtx.Transfer.Step > types.StepSetup {
		if !c.client.HasRestart() {
			return pipeline.NewError(types.TeForbidden,
				"cannot resume transfer, server does not allow restarts")
		}

		c.pTrans.SetRecovered(true)

		if c.pip.TransCtx.Rule.IsSend && c.client.HasCheckpoints() {
			checkpointNb := prog / int64(c.client.CheckpointSize())
			c.pTrans.SetRecoveryPoint(uint32(checkpointNb))
		}
	}

	// c.pTrans.UseClientLogin(true)
	c.pTrans.SetTransferID(c.pesitID)
	c.pTrans.StopReceived = stopReceived(c.pip)
	c.pTrans.ConnectionAborted = connectionAborted(c.pip)
	c.pTrans.RestartReceived = restartReceived(c.pip)
	c.pTrans.CheckpointRequestReceived = checkpointRequestReceived(c.pip)
	c.pTrans.SetMessageSize(partConf.MaxMessageSize)
	c.pTrans.SetArticleSize(defaultArticleSize)

	if err := setFreetext(c.pip, clientTransFreetextKey, c.pTrans); err != nil {
		return err
	}

	if err := setBankID(c.pip, c.pTrans); err != nil {
		return err
	}

	if err := setCustomerID(c.pip, c.pTrans); err != nil {
		return err
	}

	if c.pip.TransCtx.Rule.IsSend {
		c.pTrans.SetCreationDate(fileInfo.ModTime())
		c.pTrans.SetReservationSpace(makeReservationSpaceKB(fileInfo), pesit.UnitKB)

		if err := setFileType(c.pip, c.pTrans); err != nil {
			return err
		}

		if err := setFileOrganization(c.pip, c.pTrans); err != nil {
			return nil
		}

		if err := setFileEncoding(c.pip, c.pTrans); err != nil {
			return err
		}
	}

	if c.client.UseCFTCompatibility() {
		c.pTrans.SetFilenamePI12(c.pip.TransCtx.Rule.Name)
	}

	// request transfer
	if err := c.client.SelectFile(c.pTrans); err != nil {
		c.pip.Logger.Errorf("Failed to make transfer request: %v", err)

		return toPipErr(types.TeForbidden, "failed to make transfer request", err)
	}

	if !c.pip.TransCtx.Rule.IsSend {
		c.pip.TransCtx.Transfer.RemoteTransferID = utils.FormatUint(c.pTrans.TransferID())
		c.pip.TransCtx.Transfer.Filesize = model.UnknownSize

		setTransInfo(c.pip, fileEncodingKey, c.pTrans.DataCoding().String())
		setTransInfo(c.pip, fileTypeKey, c.pTrans.FileType())
		setTransInfo(c.pip, organizationKey, c.pTrans.FileOrganization().String())
	}

	setTransInfo(c.pip, serverTransFreetextKey, c.pTrans.FreeText())

	return nil
}

func (c *clientTransfer) authenticateServer() *pipeline.Error {
	if c.client.NewServerPassword() != "" {
		c.pip.Logger.Error("Server is not allowed to change its password")
		c.SendError(types.TeForbidden, "changing password is not allowed")

		return pipeline.NewError(types.TeForbidden, "changing password is not allowed")
	}

	var servPwd model.Credential
	if err := c.pip.DB.Get(&servPwd, "remote_agent_id=? AND type=?",
		c.pip.TransCtx.RemoteAgent.ID, auth.Password).Run(); err == nil {
		if bcrypt.CompareHashAndPassword([]byte(servPwd.Value), []byte(c.client.ServerPassword())) != nil {
			c.pip.Logger.Error("Server authentication failed: bad password")
			c.SendError(types.TeBadAuthentication, "server authentication failed")

			return pipeline.NewError(types.TeBadAuthentication, "server authentication failed: bad password")
		}
	} else if !database.IsNotFound(err) {
		c.pip.Logger.Errorf("Failed to retrieve partner password: %v", err)
		c.SendError(types.TeInternal, "database error")

		return pipeline.NewErrorWith(types.TeInternal, "failed to retrieve partner password", err)
	}

	return nil
}

func (c *clientTransfer) Send(fullFile protocol.SendFile) *pipeline.Error {
	copyArticle := func(article io.Writer, file io.Reader) *pipeline.Error {
		if _, err := io.Copy(article, file); err != nil {
			c.pip.Logger.Errorf("Failed to send data: %v", err)

			pErr := toPesitErr(pesit.CodeOtherTransferError, err)
			if hErr := c.halt(pesit.StopError, pErr); hErr != nil {
				c.pip.Logger.Warningf("Failed to send error to partner: %v", hErr)
			}

			return toPipErr(types.TeDataTransfer, "failed to send data", err)
		}

		return nil
	}

	return c.dataTransfer(func() *pipeline.Error {
		articleLengths, isMArt := isMultiArticles(c.pip)
		if !isMArt {
			return copyArticle(c.pTrans, fullFile)
		}

		c.pTrans.SetArticleFormat(pesit.FormatVariable)
		c.pTrans.SetManualArticleHandling(true)

		for _, length := range articleLengths {
			article, aErr := c.pTrans.StartNextSendArticle()
			if aErr != nil {
				c.pip.Logger.Errorf("Failed to start next article: %v", aErr)

				return toPipErr(types.TeDataTransfer, "failed to start next article", aErr)
			}

			file := io.LimitReader(fullFile, length)

			if err := copyArticle(article, file); err != nil {
				return err
			}
		}

		return nil
	}, fullFile)
}

func (c *clientTransfer) Receive(file protocol.ReceiveFile) *pipeline.Error {
	return c.dataTransfer(func() *pipeline.Error {
		var articleLengths []int64

		for {
			article, aErr := c.pTrans.GetNextRecvArticle()
			if errors.Is(aErr, pesit.ErrNoMoreArticle) {
				return nil
			} else if aErr != nil {
				c.pip.Logger.Errorf("Failed to retrieve next article: %v", aErr)

				return toPipErr(types.TeDataTransfer, "failed to retrieve next article", aErr)
			}

			start := c.pip.TransCtx.Transfer.Progress

			if _, err := io.Copy(file, article); err != nil {
				c.pip.Logger.Errorf("Failed to retrieve data: %v", err)

				pErr := toPesitErr(pesit.CodeOtherTransferError, err)
				if hErr := c.halt(pesit.StopError, pErr); hErr != nil {
					c.pip.Logger.Warningf("Failed to send error to partner: %v", hErr)
				}

				return toPipErr(types.TeDataTransfer, "failed to retrieve data", err)
			}

			end := c.pip.TransCtx.Transfer.Progress
			articleLengths = append(articleLengths, end-start)
			c.pip.TransCtx.TransInfo[articlesLengthsKey] = articleLengths
		}
	}, file)
}

func (c *clientTransfer) dataTransfer(doTransfer func() *pipeline.Error,
	file io.Seeker,
) *pipeline.Error {
	if err := c.pTrans.OpenFile(); err != nil {
		c.pip.Logger.Errorf("Failed to open transfer file: %v", err)

		return toPipErr(types.TeInternal, "failed to open transfer file", err)
	}

	if err := c.pTrans.StartDataTransfer(); err != nil {
		c.pip.Logger.Errorf("Failed to start data transfer: %v", err)

		return toPipErr(types.TeInternal, "failed to start data transfer", err)
	}

	if c.pTrans.IsRecovered() {
		ckptSize := int64(c.pTrans.CheckpointSize())
		ckptNumber := int64(c.pTrans.RecoveryPoint())
		offset := ckptSize * ckptNumber

		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			c.pip.Logger.Errorf("Failed to seek to offset: %v", err)

			return toPipErr(types.TeInternal, "failed to seek to offset", err)
		}
	}

	if err := doTransfer(); err != nil {
		return err
	}

	if err := c.pTrans.EndDataTransfer(); err != nil {
		c.pip.Logger.Errorf("Failed to end data transfer: %v", err)

		return toPipErr(types.TeInternal, "failed to end data transfer", err)
	}

	if err := c.pTrans.CloseFile(nil); err != nil {
		c.pip.Logger.Errorf("Failed to close transfer file: %v", err)

		return toPipErr(types.TeInternal, "failed to close transfer file", err)
	}

	return nil
}

func (c *clientTransfer) EndTransfer() *pipeline.Error {
	if err := c.pTrans.DeselectFile(nil); err != nil {
		c.pip.Logger.Errorf("Failed to end transfer: %v", err)

		return toPipErr(types.TeFinalization, "failed to end transfer", err)
	}

	if err := c.client.Close(nil); err != nil {
		c.pip.Logger.Warningf("failed to close client: %v", err)
	}

	return nil
}

func (c *clientTransfer) SendError(code types.TransferErrorCode, details string) {
	pErr := transErrToPesitErr(pipeline.NewError(code, details))

	if err := c.halt(pesit.StopError, pErr); err != nil {
		c.pip.Logger.Warning(err.Details())
	}
}

var (
	errClientPause  = pesit.NewDiagnostic(pesit.CodeTryLater, "transfer was paused by user")
	errClientCancel = pesit.NewDiagnostic(pesit.CodeVolontaryTermination, "transfer was canceled by user")
)

func (c *clientTransfer) Pause() *pipeline.Error {
	if err := c.halt(pesit.StopError, errClientPause); err != nil {
		c.pip.Logger.Error(err.Details())

		return nil
	}

	return nil
}

func (c *clientTransfer) Cancel() *pipeline.Error {
	if err := c.halt(pesit.StopCancel, errClientCancel); err != nil {
		c.pip.Logger.Errorf("Failed to halt transfer: %v", err)

		return nil
	}

	return nil
}

func (c *clientTransfer) halt(cause pesit.StopCause, pErr pesit.Diagnostic) *pipeline.Error {
	defer func() {
		if err := c.client.Close(pErr); err != nil {
			c.pip.Logger.Warningf("failed to close connection: %v", err)
		}
	}()

	var retErr *pipeline.Error

	//nolint:nestif // no easy way to reduce complexity here
	if c.pTrans != nil {
		if err := c.pTrans.Stop(cause, pErr); err != nil {
			var diag pesit.Diagnostic
			if !errors.As(err, &diag) || diag.GetCode() != pesit.CodeVolontaryTermination {
				retErr = toPipErr(types.TeUnknownRemote, "failed to send error to partner", err)
			}
		}

		if c.pTrans.IsFileOpened() {
			if err := c.pTrans.CloseFile(pErr); err != nil {
				retErr = toPipErr(types.TeUnknownRemote, "failed to close transfer file", err)
			}
		}

		if err := c.pTrans.DeselectFile(pErr); err != nil {
			retErr = toPipErr(types.TeUnknownRemote, "failed to deselect transfer file", err)
		}
	}

	return retErr
}

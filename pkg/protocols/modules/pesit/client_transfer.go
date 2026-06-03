package pesit

import (
	"errors"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/lib/pesit"
)

// Ensures Client implements the optional client interfaces.

var _ interface {
	protocol.TransferClient

	protocol.PauseHandler

	protocol.CancelHandler
} = &clientTransfer{}

type clientTransfer struct {
	isTLS bool

	pip *pipeline.Pipeline

	clientConf *ClientConfigTLS

	conn *pesitClientConn // connection (pooled or standalone)

	conns *pesitConnPool // reference to pool for Connect/CloseConn

	pooled bool // true if conn came from pool, false if standalone

	pesitID uint32

	pTrans *pesit.ClientTransfer
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

	// Get a PeSIT connection from the pool (or create a standalone one

	// if the pool is in exclusive mode and the existing conn is busy).

	pooledConn, connErr := c.conns.Connect(c.pip)

	if connErr != nil {

		c.pip.Logger.Errorf("Failed to get PeSIT connection: %v", connErr)

		return pipeline.NewErrorWith(types.TeConnection, "failed to get PeSIT connection", connErr)

	}

	c.conn = pooledConn

	c.pooled = c.conns.Exists(c.pip.TransCtx.RemoteAccount)

	if err := c.request(fileInfo, &partConf); err != nil {

		c.releaseConn()

		return err

	}

	return nil
}

//nolint:funlen,gocognit,gocyclo,cyclop //no easy way to split the function for now
func (c *clientTransfer) request(fileInfo fs.FileInfo, partConf *PartnerConfigTLS,
) *pipeline.Error {
	// Connection is already established via ConnPool (c.conn)

	c.injectReplyTo(partConf.ReplyTo)

	setTransInfo(c.pip, serverConnFreetextKey, c.conn.FreeText())

	// initialize transfer

	method := pesit.MethodRecv

	if c.pip.TransCtx.Rule.IsSend {

		c.conn.SetAccessType(pesit.AccessWrite)

		method = pesit.MethodSend

	} else {
		c.conn.SetAccessType(pesit.AccessRead)
	}

	c.pTrans = pesit.NewTransfer(method, c.pip.TransCtx.Transfer.RemotePath)

	// configure recovery if transfer is resumed

	if prog := c.pip.TransCtx.Transfer.Progress; prog != 0 ||

		c.pip.TransCtx.Transfer.Step > types.StepSetup {

		if !c.conn.HasRestart() {
			return pipeline.NewError(types.TeForbidden,

				"cannot resume transfer, server does not allow restarts")
		}

		c.pTrans.SetRecovered(true)

		if c.pip.TransCtx.Rule.IsSend && c.conn.HasCheckpoints() {

			checkpointNb := prog / int64(c.conn.CheckpointSize())

			c.pTrans.SetRecoveryPoint(uint32(checkpointNb))

		}

	}

	c.pTrans.SetTransferID(c.pesitID)

	c.pTrans.SetMessageSize(partConf.MaxMessageSize)

	c.pTrans.SetArticleFormat(resolveArticleFormat(partConf.ArticleFormat))

	c.pTrans.SetArticleSize(partConf.ArticleSize)

	c.pTrans.SetCompression(partConf.Compression.ToPeSIT())

	c.pTrans.StopReceived = stopReceived(c.pip)

	c.pTrans.ConnectionAborted = connectionAborted(c.pip)

	c.pTrans.RestartReceived = restartReceived(c.pip)

	c.pTrans.CheckpointRequestReceived = checkpointRequestReceived(c.pip)

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

		c.pTrans.SetFilename(c.pip.TransCtx.Transfer.RemotePath)

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

	if c.conn.UseHistoriqueMode() {
		c.pTrans.SetFilenamePI12(c.pip.TransCtx.Rule.Name)
	}

	// request transfer

	if err := c.conn.SelectFile(c.pTrans); err != nil {

		c.pip.Logger.Errorf("Failed to make transfer request: %v", err)

		return toPipErr(types.TeForbidden, "failed to make transfer request", err)

	}

	if !c.pip.TransCtx.Rule.IsSend {

		c.pip.TransCtx.Transfer.RemoteTransferID = utils.FormatUint(c.pTrans.TransferID())

		c.pip.TransCtx.Transfer.Filesize = model.UnknownSize

		// When the server resolved a glob pattern (e.g. "data-*" -> "data-003.txt"),

		// update the local transfer filenames and reset the local path so the

		// pipeline creates the file with the resolved name instead of the pattern.

		resolvedName := c.pTrans.Filename()

		if resolvedName != "" && !pipeline.ContainsWildcard(resolvedName) {

			resolvedBase := path.Base(resolvedName)

			if pipeline.ContainsWildcard(c.pip.TransCtx.Transfer.DestFilename) ||

				pipeline.ContainsWildcard(c.pip.TransCtx.Transfer.SrcFilename) {

				c.pip.Logger.Infof("Pattern resolved by server: %q -> %q",

					c.pip.TransCtx.Transfer.DestFilename, resolvedBase)

				c.pip.TransCtx.Transfer.DestFilename = resolvedBase

				c.pip.TransCtx.Transfer.SrcFilename = resolvedName

				c.pip.TransCtx.Transfer.RemotePath = resolvedName

				// Reset LocalPath so it will be recomputed with the resolved name.

				c.pip.TransCtx.Transfer.LocalPath = ""

			}

		}

		setTransInfo(c.pip, fileEncodingKey, c.pTrans.DataCoding().String())

		setTransInfo(c.pip, fileTypeKey, c.pTrans.FileType())

		setTransInfo(c.pip, organizationKey, c.pTrans.FileOrganization().String())

	}

	setTransInfo(c.pip, serverTransFreetextKey, c.pTrans.FreeText())

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

			c.pip.TransCtx.Transfer.TransferInfo[articlesLengthsKey] = articleLengths

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

// releaseConn returns the connection to the pool (if pooled) or closes it

// directly (if standalone, i.e. created because the pooled one was busy).

func (c *clientTransfer) releaseConn() {
	if c.conn == nil {
		return
	}

	if c.pooled {
		c.conns.CloseConn(c.pip)
	} else {
		if err := c.conn.Close(); err != nil {
			c.pip.Logger.Warningf("failed to close standalone PeSIT connection: %v", err)
		}
	}

	c.conn = nil
}

// injectReplyTo adds "REPLY=partner:account" to the connection freetext

// (PI 99) if the partner config has a replyTo value. This tells the

// receiver where to send F.MESSAGE ACKs.

func (c *clientTransfer) injectReplyTo(replyTo string) {
	if replyTo == "" {
		return
	}

	freetext := c.conn.FreeText()

	if freetext != "" {
		freetext += " "
	}

	freetext += "REPLY=" + replyTo

	c.conn.SetFreeText(freetext)
}

func (c *clientTransfer) EndTransfer() *pipeline.Error {
	if err := c.pTrans.DeselectFile(nil); err != nil {

		c.pip.Logger.Errorf("Failed to end transfer: %v", err)

		return toPipErr(types.TeFinalization, "failed to end transfer", err)

	}

	// Return the connection to the pool (or close standalone connection).

	c.releaseConn()

	return nil
}

func (c *clientTransfer) SendError(code types.TransferErrorCode, details string) {
	pErr := transErrToPesitErr(pipeline.NewError(code, details))

	if err := c.halt(pesit.StopError, pErr); err != nil {
		c.pip.Logger.Warning(err.Details())
	}
}

var (
	errClientPause = pesit.NewDiagnostic(pesit.CodeTryLater, "transfer was paused by user")

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
		if c.conn == nil {
			return
		}

		// Send ABORT/RELEASE to the partner

		if err := c.conn.Client.Close(pErr); err != nil {
			c.pip.Logger.Warningf("failed to close connection: %v", err)
		}

		// Remove from pool without re-closing (already closed above).

		// Use Evict, not CloseConn, to avoid grace period keeping a dead conn.

		if c.pooled {
			c.conns.Evict(c.pip.TransCtx.RemoteAccount)
		}

		c.conn = nil
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

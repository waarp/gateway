package as2

import (
	"context"
	"crypto/x509"
	"net/http"
	"slices"
	"strings"

	"code.waarp.fr/lib/as2"
	"code.waarp.fr/lib/log/v2"
	"github.com/dustin/go-humanize"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const dummyMDNAddress = "foo@bar.org"

type clientTransfer struct {
	ctx    context.Context
	cancel context.CancelCauseFunc
	logger *log.Logger

	client  *as2.Client
	partner *as2.Partner
	opts    *as2.SendOptions

	bufLen    uint64
	asyncChan utils.NullableChan[*as2.MDN]
	done      func()
}

func (c *client) newClientTransfer(parent context.Context, pip *pipeline.Pipeline,
	partConf partnerProtoConfigTLS,
) (*clientTransfer, *pipeline.Error) {
	maxSize := int64(c.protoConfig.MaxFileSize)
	size := pip.TransCtx.Transfer.Filesize

	if size == model.UnknownSize {
		size = maxSize
	}

	if size > maxSize {
		return nil, pipeline.NewErrorf(types.TeExceededLimit,
			"file size exceeds the maximum allowed size of %s",
			humanize.Bytes(uint64(maxSize)))
	}

	as2.SetPKCS7ContentEncryptionAlgorithm(partConf.EncryptionAlgorithm.PKCS7())

	sendURL := pip.TransCtx.RemoteAgent.Address.String()
	if proto := pip.TransCtx.RemoteAgent.Protocol; proto == AS2 && !strings.HasPrefix(sendURL, "http://") {
		sendURL = "http://" + sendURL
	} else if proto == AS2TLS && !strings.HasPrefix(sendURL, "https://") {
		sendURL = "https://" + sendURL
	}

	signAlgo := partConf.SignatureAlgorithm.as2()
	encryptAlgo := partConf.EncryptionAlgorithm.PKCS7()
	isSigned := signAlgo != ""
	isEncrypted := encryptAlgo >= 0
	filename := pip.TransCtx.Transfer.RemotePath

	withSigner, err := setSigner(isSigned, pip)
	if err != nil {
		return nil, err
	}

	partnerCerts, err := getPartnerCert(isEncrypted, pip)
	if err != nil {
		return nil, err
	}

	transport, err := c.getTransport(pip)
	if err != nil {
		return nil, err
	}

	var asyncURL string
	if partConf.AsyncMDNAddress != "" {
		asyncURL = "http://" + partConf.AsyncMDNAddress
		if pip.TransCtx.RemoteAgent.Protocol == AS2TLS {
			asyncURL = "https://" + partConf.AsyncMDNAddress
		}
	}

	ctx, cancel := context.WithCancelCause(parent)

	return &clientTransfer{
		ctx:    ctx,
		cancel: cancel,
		client: as2.NewClient(
			as2.WithHTTPClient(&http.Client{Transport: transport}),
			as2.WithClientLogger(protoutils.LibSLogger(c.logger, log.LevelInfo, log.LevelDebug)),
			as2.WithLocalClientAS2ID(pip.TransCtx.Client.Name),
			as2.WithMDNAddress(dummyMDNAddress),
			withSigner,
		),
		partner: &as2.Partner{
			Name:      pip.TransCtx.RemoteAgent.Name,
			AS2ID:     pip.TransCtx.RemoteAgent.Name,
			SendUrl:   sendURL,
			CertChain: partnerCerts,
		},
		opts: as2.NewSendOptions(
			as2.WithFileName(filename),
			as2.WithSignedMDN(isSigned),
			setMDNSignOpts(isSigned, signAlgo),
			setEncryptOpts(isEncrypted, encryptAlgo),
			as2.WithAsyncReturnURL(asyncURL),
			as2.WithMessageID(pip.TransCtx.Transfer.RemoteTransferID),
		),
		bufLen: uint64(size),
		done:   func() {},
	}, nil
}

func (c *clientTransfer) Request() *pipeline.Error {
	return nil
}

func (c *clientTransfer) Send(file protocol.SendFile) *pipeline.Error {
	cont, sizErr := getFileContent(file, c.bufLen)
	if sizErr != nil {
		return sizErr
	}

	res, err := c.client.Send(c.ctx, c.partner, cont, c.opts)
	if err != nil {
		return pipeline.NewErrorWith(err, types.TeConnection, "failed to connect to partner")
	}

	if res.StatusCode != http.StatusOK {
		return pipeline.NewErrorf(types.TeConnection,
			"unexpected status code %d: %s", res.StatusCode, res.MDN.Raw)
	}

	return c.handleMDN(res.MDN)
}

func (c *clientTransfer) handleMDN(mdn *as2.MDN) *pipeline.Error {
	if mdn == nil {
		return nil
	}

	switch mdn.Status {
	case as2.MDNStatusSuccess:
		return nil
	case as2.MDNStatusWarning:
		c.logger.Warningf("server returned MDN warning: %s", mdn.HumanText)

		return nil
	case as2.MDNStatusError:
		return pipeline.NewErrorf(types.TeConnection,
			"server returned MDN error: %s", mdn.HumanText)
	default:
		return pipeline.NewErrorf(types.TeInternal,
			"unknown MDN status %q: %s", mdn.Status, mdn.HumanText)
	}
}

func (c *clientTransfer) Receive(protocol.ReceiveFile) *pipeline.Error {
	return pipeline.NewError(types.TeInternal, ErrTransferPull.Error())
}

func (c *clientTransfer) EndTransfer() *pipeline.Error {
	if mdn, ok := c.asyncChan.Recv(); ok {
		return c.handleMDN(mdn)
	}

	return nil
}

func (c *clientTransfer) close() {
	c.asyncChan.Close()
	c.done()
}

func (c *clientTransfer) SendError(code types.TransferErrorCode, msg string) {
	c.close()
	c.cancel(pipeline.NewError(code, msg))
}

func sNoop(*as2.SendOptions) {}
func cNoop(*as2.Client)      {}

func setSigner(isSigned bool, pip *pipeline.Pipeline) (as2.ClientOption, *pipeline.Error) {
	if !isSigned {
		return cNoop, nil
	}

	for _, cred := range pip.TransCtx.RemoteAccountCreds {
		if cred.Type != auth.TLSCertificate {
			continue
		}

		cert, err := utils.X509KeyPair(cred.Value, cred.Value2)
		if err != nil {
			pip.Logger.Warningf("failed to parse x509 certificate %q: %v", cred.Name, err)

			continue
		}

		der := slices.Concat(cert.Certificate...)
		certs, err := x509.ParseCertificates(der)
		if err != nil {
			pip.Logger.Warningf("failed to parse x509 certificate chain %q: %v", cred.Name, err)

			continue
		}

		return as2.WithSigner(cert.PrivateKey, certs), nil
	}

	return nil, pipeline.NewErrorf(types.TeBadAuthentication,
		"no valid x509 certificate found for account %q", pip.TransCtx.RemoteAccount.Login)
}

func setMDNSignOpts(isSigned bool, signAlgo as2.MICAlg) as2.SendOption {
	if !isSigned {
		return sNoop
	}

	return func(opts *as2.SendOptions) {
		as2.WithSign(as2.DefaultSignPKCS7, signAlgo)(opts)
		as2.WithMICAlg(string(signAlgo))(opts)
	}
}

func setEncryptOpts(isEncrypted bool, encryptAlgo as2.PKCS7EncryptionAlgorithm) as2.SendOption {
	if !isEncrypted {
		return sNoop
	}

	return as2.WithEncryptFunc(as2.NewPKCS7Encrypter(encryptAlgo))
}

func getPartnerCert(isEncrypted bool, pip *pipeline.Pipeline) ([]*x509.Certificate, *pipeline.Error) {
	if !isEncrypted {
		return []*x509.Certificate{}, nil
	}

	for _, cred := range pip.TransCtx.RemoteAgentCreds {
		if cred.Type == auth.TLSTrustedCertificate {
			cert, err := utils.ParsePEMCertChain(cred.Value)
			if err != nil {
				pip.Logger.Warningf("failed to parse x509 certificate %q: %v", cred.Name, err)

				continue
			}

			return cert, nil
		}
	}

	return nil, pipeline.NewErrorf(types.TeBadAuthentication,
		"no valid x509 certificate found for partner %q", pip.TransCtx.RemoteAgent.Name)
}

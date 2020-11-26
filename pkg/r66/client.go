package r66

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"os"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/executor"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"code.waarp.fr/waarp-r66/r66"
	r66utils "code.waarp.fr/waarp-r66/r66/utils"
)

func init() {
	executor.ClientsConstructors["r66"] = NewClient
}

type client struct {
	r66Client *r66.Client
	info      model.OutTransferInfo
	signals   <-chan model.Signal

	conf    config.R66ProtoConfig
	tlsConf *tls.Config

	remote  *r66.Remote
	session *r66.ClientSession

	stream  pipeline.DataStream
	hasData bool
}

// NewClient creates and returns a new r66 client using the given transfer info.
func NewClient(info model.OutTransferInfo, signals <-chan model.Signal) (pipeline.Client, error) {
	pswd, err := utils.DecryptPassword(info.Account.Password)
	if err != nil {
		return nil, err
	}

	r66Client := r66.NewClient(info.Account.Login, pswd)

	var conf config.R66ProtoConfig
	if err := json.Unmarshal(info.Agent.ProtoConfig, &conf); err != nil {
		return nil, err
	}

	var tlsConf *tls.Config
	if conf.IsTLS {
		var err error
		tlsConf, err = makeClientTLSConfig(&info)
		if err != nil {
			return nil, model.NewPipelineError(types.TeInternal, "invalid R66 TLS config")
		}
	}

	//TODO: configure r66 client
	c := &client{
		r66Client: r66Client,
		info:      info,
		signals:   signals,
		conf:      conf,
		tlsConf:   tlsConf,
	}
	c.r66Client.FinalHash = false //TODO: remove once implemented
	c.r66Client.AuthentHandler = &clientAuthHandler{
		getFile: func() r66utils.ReadWriterAt { return c.stream },
		info:    &info,
	}

	return c, nil
}

func makeClientTLSConfig(info *model.OutTransferInfo) (*tls.Config, error) {
	tlsCerts := make([]tls.Certificate, len(info.ClientCerts))
	for i, cert := range info.ClientCerts {
		var err error
		tlsCerts[i], err = tls.X509KeyPair(cert.Certificate, cert.PrivateKey)
		if err != nil {
			return nil, err
		}
	}

	var caPool *x509.CertPool
	for _, cert := range info.ServerCerts {
		if caPool == nil {
			caPool = x509.NewCertPool()
		}
		caPool.AppendCertsFromPEM(cert.Certificate)
	}

	return &tls.Config{
		Certificates: tlsCerts,
		MinVersion:   tls.VersionTLS12,
		RootCAs:      caPool,
	}, nil
}

func (c *client) Connect() *model.PipelineError {
	var remote *r66.Remote
	var err error
	if c.tlsConf != nil {
		remote, err = c.r66Client.DialTLS(c.info.Agent.Address, c.tlsConf)
	} else {
		remote, err = c.r66Client.Dial(c.info.Agent.Address)
	}

	if err != nil {
		if r66Err, ok := err.(*r66.Error); ok {
			return model.NewPipelineError(types.FromR66Code(r66Err.Code), r66Err.Detail)
		}
		return model.NewPipelineError(types.TeConnection, err.Error())
	}
	c.remote = remote
	return nil
}

func (c *client) Authenticate() *model.PipelineError {
	ses, err := c.remote.Authent()
	if err != nil {
		if r66Err, ok := err.(*r66.Error); ok {
			return model.NewPipelineError(types.FromR66Code(r66Err.Code), r66Err.Detail)
		}
		return model.NewPipelineError(types.TeBadAuthentication, err.Error())
	}
	c.session = ses
	return nil
}

func (c *client) Request() *model.PipelineError {
	file := c.info.Transfer.SourceFile
	var size uint64
	if c.info.Rule.IsSend {
		file = c.info.Transfer.DestFile

		stats, err := os.Stat(utils.DenormalizePath(c.info.Transfer.TrueFilepath))
		if err != nil {
			return model.NewPipelineError(types.TeInternal, err.Error())
		}
		size = uint64(stats.Size())
	}

	var blockSize uint32 = 65536
	if c.conf.BlockSize != 0 {
		blockSize = c.conf.BlockSize
	}

	trans := &r66.Transfer{
		ID:    c.info.Transfer.ID,
		Get:   !c.info.Rule.IsSend,
		File:  file,
		Rule:  c.info.Rule.Name,
		Block: blockSize,
		Rank:  uint32(c.info.Transfer.Progress / uint64(c.r66Client.Block)),
		Size:  size,
	}

	if err := c.session.Request(trans); err != nil {
		if r66Err, ok := err.(*r66.Error); ok {
			return model.NewPipelineError(types.FromR66Code(r66Err.Code), r66Err.Detail)
		}
		return model.NewPipelineError(types.TeConnection, err.Error())
	}
	return nil
}

func (c *client) Data(file pipeline.DataStream) *model.PipelineError {
	c.hasData = true
	c.stream = file

	if err := c.session.Data(); err != nil {
		if e, ok := err.(*r66.Error); ok {
			return model.NewPipelineError(types.FromR66Code(e.Code), e.Detail)
		}
		return model.NewPipelineError(types.TeDataTransfer, err.Error())
	}
	if err := c.session.EndTransfer(); err != nil {
		if e, ok := err.(*r66.Error); ok {
			return model.NewPipelineError(types.FromR66Code(e.Code), e.Detail)
		}
		return model.NewPipelineError(types.TeDataTransfer, err.Error())
	}
	return nil
}

func (c *client) Close(err *model.PipelineError) *model.PipelineError {
	if c.remote == nil {
		return nil
	}
	defer c.remote.Close()

	if c.session == nil {
		return nil
	}
	defer c.session.Close()

	if !c.hasData && err == nil {
		if err1 := c.session.EndTransfer(); err1 != nil {
			if e, ok := err1.(*r66.Error); ok {
				return model.NewPipelineError(types.FromR66Code(e.Code), e.Detail)
			}
			return model.NewPipelineError(types.TeDataTransfer, err1.Error())
		}
	}

	if err == nil {
		err1 := c.session.EndRequest()
		if err1 == nil {
			return nil
		}
		if e, ok := err1.(*r66.Error); ok {
			return model.NewPipelineError(types.FromR66Code(e.Code), e.Detail)
		}
		return model.NewPipelineError(types.TeUnknownRemote, err1.Error())
	}

	r66Err := &r66.Error{Code: err.Cause.Code.R66Code(), Detail: err.Cause.Details}
	_ = c.session.SendError(r66Err)
	return nil
}

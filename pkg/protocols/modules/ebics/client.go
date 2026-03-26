package ebics

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	libebicsclient "code.waarp.fr/lib/ebics/ebics/client"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var errMissingEndpointServerName = errors.New("endpointURL does not contain a server name")

type Client struct {
	db      *database.DB
	logger  *log.Logger
	client  *model.Client
	config  *clientConfig
	profile *libebicsclient.ProductionProfile
	state   utils.State
}

func NewClient(db *database.DB, dbClient *model.Client) *Client {
	return &Client{db: db, client: dbClient}
}

func (c *Client) Start() error {
	if c.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	c.logger = logging.NewLogger(c.client.Name)
	cfg := defaultClientConfig()
	if err := utils.JSONConvert(c.client.ProtoConfig, cfg); err != nil {
		err = wrapConfigError(err)
		c.state.Set(utils.StateError, err.Error())

		return err
	}

	if err := cfg.ValidClient(); err != nil {
		err = wrapConfigError(err)
		c.state.Set(utils.StateError, err.Error())

		return err
	}

	c.config = cfg
	profile, err := c.newLibraryProfile(cfg)
	if err != nil {
		err = wrapConfigError(err)
		c.state.Set(utils.StateError, err.Error())

		return err
	}

	c.profile = profile
	c.state.Set(utils.StateRunning, "")
	c.logger.Info("EBICS client bootstrap completed with lib-ebics production profile")

	return nil
}

func (c *Client) Stop(ctx context.Context) error {
	if !c.state.IsRunning() {
		return utils.ErrNotRunning
	}
	_ = ctx
	c.profile = nil
	c.state.Set(utils.StateOffline, "")

	return nil
}

func (c *Client) State() (utils.StateCode, string) {
	return c.state.Get()
}

func (c *Client) InitTransfer(_ *pipeline.Pipeline) (protocol.TransferClient, *pipeline.Error) {
	return nil, pipeline.NewErrorWith(
		types.TeUnimplemented,
		"EBICS transfer bootstrap is not implemented yet",
		fmt.Errorf("%w", ErrNotImplemented),
	)
}

func (c *Client) newLibraryProfile(cfg *clientConfig) (*libebicsclient.ProductionProfile, error) {
	parsedURL, err := url.Parse(cfg.EndpointURL)
	if err != nil {
		return nil, fmt.Errorf("parse endpointURL for lib-ebics bootstrap: %w", err)
	}

	serverName := parsedURL.Hostname()
	if serverName == "" {
		return nil, fmt.Errorf("%w: %q", errMissingEndpointServerName, cfg.EndpointURL)
	}

	profile, err := libebicsclient.NewProductionProfile(
		libebicsclient.ProductionHTTPClientRequired{
			ServerName: serverName,
		},
		libebicsclient.ProductionHTTPClientOptional{
			MinTLSVersion:  cfg.MinTLSVersion.TLS(),
			RequestTimeout: time.Duration(cfg.RequestTimeout) * time.Second,
		},
		libebicsclient.ProductionRetryPolicyOptional{},
		libebicsclient.ProductionRecoveryOptional{},
	)
	if err != nil {
		return nil, fmt.Errorf("initialize lib-ebics production profile: %w", err)
	}

	return profile, nil
}

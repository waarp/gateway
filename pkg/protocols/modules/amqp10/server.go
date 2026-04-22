package amqp10

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	amqp "github.com/Azure/go-amqp"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	errMissingServerResources = errors.New("the AMQP 1.0 server has not been initialized")
	errEmptyResolvedFilename  = errors.New("the AMQP 1.0 message filename resolved to an empty value")
	errCloseServerResources   = errors.New("failed to close the AMQP 1.0 server resources")
)

const (
	serverReceiveRetryDelay = 250 * time.Millisecond
	serverCloseTimeout      = 5 * time.Second
)

type server struct {
	db     *database.DB
	agent  *model.LocalAgent
	logger *log.Logger
	state  utils.State
	conf   *serverConfig
	dialer amqpDialer

	conn     amqpConnection
	session  amqpSession
	receiver amqpReceiver

	localAccount *model.LocalAccount
	rule         *model.Rule

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newServer(db *database.DB, agent *model.LocalAgent) *server {
	return &server{
		db:     db,
		agent:  agent,
		dialer: defaultDialer{},
	}
}

func (s *server) Start() error {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	s.logger = logging.NewLogger(s.agent.Name)
	if err := s.start(); err != nil {
		s.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(s.agent.Name, err)

		return err
	}

	s.state.Set(utils.StateRunning, "")

	return nil
}

func (s *server) start() error {
	s.conf = defaultServerConfig()
	if err := utils.JSONConvert(s.agent.ProtoConfig, s.conf); err != nil {
		return fmt.Errorf("invalid AMQP 1.0 server config: %w", err)
	}
	if err := s.conf.ValidServer(); err != nil {
		return fmt.Errorf("invalid AMQP 1.0 server config: %w", err)
	}
	if err := s.loadExecutionScope(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.conf.IdleTimeoutSeconds)*time.Second)
	defer cancel()
	conn, err := s.dialer.Dial(ctx, s.conf.Endpoint, &amqp.ConnOptions{
		ContainerID: connectionName(s.agent.Name, s.conf.SourceAddress),
		IdleTimeout: time.Duration(s.conf.IdleTimeoutSeconds) * time.Second,
	})
	if err != nil {
		return fmt.Errorf("connect the AMQP 1.0 server broker: %w", err)
	}
	session, err := conn.NewSession(ctx, nil)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("open the AMQP 1.0 server session: %w", err)
	}
	receiver, err := session.NewReceiver(ctx, s.conf.SourceAddress, &amqp.ReceiverOptions{
		Name:   s.conf.ReceiverLinkName,
		Credit: int32(s.conf.Credit),
	})
	if err != nil {
		_ = session.Close(ctx)
		_ = conn.Close()

		return fmt.Errorf("open the AMQP 1.0 server receiver: %w", err)
	}

	s.conn = conn
	s.session = session
	s.receiver = receiver

	loopCtx, loopCancel := context.WithCancel(context.Background())
	s.cancel = loopCancel
	s.wg.Add(1)
	go s.consumeLoop(loopCtx)

	return nil
}

func (s *server) loadExecutionScope() error {
	account := &model.LocalAccount{}
	if err := s.db.Get(account, "local_agent_id=? AND login=?", s.agent.ID, s.conf.LocalAccount).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"no AMQP 1.0 server local account %q found on server %q",
				s.conf.LocalAccount, s.agent.Name)
		}

		return fmt.Errorf("load AMQP 1.0 server local account: %w", err)
	}

	rule := &model.Rule{}
	if err := s.db.Get(rule, "name=?", s.conf.RuleName).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf("no AMQP 1.0 rule %q found", s.conf.RuleName)
		}

		return fmt.Errorf("load AMQP 1.0 server rule: %w", err)
	}
	if rule.IsSend {
		return database.NewValidationErrorf(
			"the AMQP 1.0 server requires a receive rule, got send rule %q",
			rule.Name)
	}

	authorized, err := rule.IsAuthorized(s.db, account)
	if err != nil {
		return fmt.Errorf("check AMQP 1.0 server rule permissions: %w", err)
	}
	if !authorized {
		return database.NewValidationErrorf(
			"the local account %q is not authorized to use the receive rule %q",
			account.Login, rule.Name)
	}

	s.localAccount = account
	s.rule = rule

	return nil
}

func (s *server) consumeLoop(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msg, err := s.receiver.Receive(ctx, nil)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return
			}
			s.logger.Errorf("Failed to receive the AMQP 1.0 message: %v", err)
			time.Sleep(serverReceiveRetryDelay)

			continue
		}

		s.handleMessage(msg)
	}
}

func (s *server) handleMessage(msg *amqp.Message) {
	if err := s.processMessage(msg); err != nil {
		s.logger.Errorf("Failed to process the AMQP 1.0 message: %v", err)
		releaseCtx, cancel := context.WithTimeout(context.Background(), serverCloseTimeout)
		defer cancel()
		if releaseErr := s.receiver.ReleaseMessage(releaseCtx, msg); releaseErr != nil {
			s.logger.Errorf("Failed to release the AMQP 1.0 message: %v", releaseErr)
		}

		return
	}

	ackCtx, cancel := context.WithTimeout(context.Background(), serverCloseTimeout)
	defer cancel()
	if err := s.receiver.AcceptMessage(ackCtx, msg); err != nil {
		s.logger.Errorf("Failed to acknowledge the AMQP 1.0 message: %v", err)
	}
}

func (s *server) processMessage(msg *amqp.Message) error {
	filename := renderServerFilename(s.conf, msg)
	if filename == "" {
		return errEmptyResolvedFilename
	}

	remoteID := stringifyAMQPMessageID(msg)
	if strings.TrimSpace(remoteID) == "" || !isNumericString(remoteID) {
		remoteID = fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	}

	transfer := pipeline.MakeServerTransfer(remoteID, filename, s.localAccount, s.rule)
	transfer.TransferInfo = map[string]any{
		"amqp10SourceAddress": s.conf.SourceAddress,
		"amqp10MessageID":     remoteID,
	}
	if msg.Properties != nil {
		transfer.TransferInfo["amqp10CorrelationID"] = stringifyAMQPValue(msg.Properties.CorrelationID)
	}

	pip, err := pipeline.NewServerPipeline(s.db, s.logger, transfer, snmp.GlobalService)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	serverPip := &serverPipeline{
		pipeline: pip,
		ctx:      ctx,
		cancel:   cancel,
	}
	pip.SetInterruptionHandlers(serverPip.Pause, serverPip.Interrupt, serverPip.Cancel)

	if err := serverPip.init(); err != nil {
		return err
	}

	if _, err := serverPip.Write(msg.GetData()); err != nil {
		return fmt.Errorf("write AMQP 1.0 message payload: %w", err)
	}

	return serverPip.Close()
}

func (s *server) Stop(ctx context.Context) error {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	if s.cancel != nil {
		s.cancel()
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.wg.Wait()
	}()

	select {
	case <-ctx.Done():
		s.state.Set(utils.StateError, ctx.Err().Error())
		snmp.ReportServiceFailure(s.agent.Name, ctx.Err())

		return fmt.Errorf("stop the AMQP 1.0 server: %w", ctx.Err())
	case <-done:
	}

	if err := s.closeResources(ctx); err != nil {
		s.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(s.agent.Name, err)

		return err
	}

	s.state.Set(utils.StateOffline, "")

	return nil
}

func (s *server) closeResources(ctx context.Context) error {
	if s.receiver == nil && s.session == nil && s.conn == nil {
		return errMissingServerResources
	}

	var errs []string
	closeCtx, cancel := context.WithTimeout(ctx, serverCloseTimeout)
	defer cancel()

	if s.receiver != nil {
		if err := s.receiver.Close(closeCtx); err != nil {
			errs = append(errs, err.Error())
		}
		s.receiver = nil
	}
	if s.session != nil {
		if err := s.session.Close(closeCtx); err != nil {
			errs = append(errs, err.Error())
		}
		s.session = nil
	}
	if s.conn != nil {
		if err := s.conn.Close(); err != nil {
			errs = append(errs, err.Error())
		}
		s.conn = nil
	}
	if len(errs) != 0 {
		return fmt.Errorf("%w: %s", errCloseServerResources, strings.Join(errs, "; "))
	}

	return nil
}

func (s *server) State() (utils.StateCode, string) { return s.state.Get() }

func stringifyAMQPMessageID(msg *amqp.Message) string {
	if msg == nil || msg.Properties == nil {
		return ""
	}

	return stringifyAMQPValue(msg.Properties.MessageID)
}

func isNumericString(raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return false
	}
	for _, r := range raw {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

func renderServerFilename(conf *serverConfig, msg *amqp.Message) string {
	value := conf.FilenameTemplate
	replacements := map[string]string{
		"${messageID}":     stringifyAMQPMessageID(msg),
		"${correlationID}": "",
		"${sourceAddress}": conf.SourceAddress,
	}
	if msg != nil && msg.Properties != nil {
		replacements["${correlationID}"] = stringifyAMQPValue(msg.Properties.CorrelationID)
	}
	for key, replacement := range replacements {
		value = strings.ReplaceAll(value, key, replacement)
	}

	return strings.TrimSpace(value)
}

package amqp091

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	errMissingServerResources = errors.New("the AMQP 0.9.1 server has not been initialized")
	errEmptyResolvedFilename  = errors.New("the AMQP 0.9.1 message filename resolved to an empty value")
	errCloseServerResources   = errors.New("failed to close the AMQP 0.9.1 server resources")
)

type server struct {
	db     *database.DB
	agent  *model.LocalAgent
	logger *log.Logger
	state  utils.State
	conf   *serverConfig
	dialer amqpDialer

	conn       amqpConnection
	channel    amqpChannel
	deliveries <-chan amqp.Delivery

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
		return fmt.Errorf("invalid AMQP 0.9.1 server config: %w", err)
	}

	if err := s.loadExecutionScope(); err != nil {
		return err
	}

	cfg := amqp.Config{
		Heartbeat: time.Duration(s.conf.HeartbeatSeconds) * time.Second,
		Properties: amqp.Table{
			"connection_name": connectionName(s.conf.ConnectionName, s.agent.Name),
		},
	}

	conn, err := s.dialer.DialConfig(s.conf.URI, &cfg)
	if err != nil {
		return fmt.Errorf("connect the AMQP 0.9.1 server broker: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("open the AMQP 0.9.1 server channel: %w", err)
	}

	prepareErr := s.prepareChannel(channel)
	if prepareErr != nil {
		_ = channel.Close()
		_ = conn.Close()

		return prepareErr
	}

	deliveries, err := channel.Consume(
		s.conf.Queue,
		s.conf.ConsumerTag,
		s.conf.AutoAck,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()

		return fmt.Errorf("start AMQP 0.9.1 server consumer: %w", err)
	}

	s.conn = conn
	s.channel = channel
	s.deliveries = deliveries

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.wg.Add(1)
	go s.consumeLoop(ctx)

	return nil
}

func (s *server) loadExecutionScope() error {
	account := &model.LocalAccount{}
	if err := s.db.Get(account, "local_agent_id=? AND login=?", s.agent.ID, s.conf.LocalAccount).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"no AMQP 0.9.1 server local account %q found on server %q",
				s.conf.LocalAccount, s.agent.Name)
		}

		return fmt.Errorf("load AMQP 0.9.1 server local account: %w", err)
	}

	rule := &model.Rule{}
	if err := s.db.Get(rule, "name=?", s.conf.RuleName).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"no AMQP 0.9.1 rule %q found",
				s.conf.RuleName)
		}

		return fmt.Errorf("load AMQP 0.9.1 server rule: %w", err)
	}
	if rule.IsSend {
		return database.NewValidationErrorf(
			"the AMQP 0.9.1 server requires a receive rule, got send rule %q",
			rule.Name)
	}

	authorized, err := rule.IsAuthorized(s.db, account)
	if err != nil {
		return fmt.Errorf("check AMQP 0.9.1 server rule permissions: %w", err)
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

func (s *server) prepareChannel(channel amqpChannel) error {
	if s.conf.Exchange != "" {
		exchangeType := firstNonEmpty(s.conf.ExchangeType, amqp.ExchangeDirect)
		if err := channel.ExchangeDeclare(
			s.conf.Exchange,
			exchangeType,
			true,
			false,
			false,
			false,
			nil,
		); err != nil {
			return fmt.Errorf("declare AMQP 0.9.1 exchange: %w", err)
		}
	}

	if _, err := channel.QueueDeclare(
		s.conf.Queue,
		s.conf.QueueDurable,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare AMQP 0.9.1 queue: %w", err)
	}

	if s.conf.Exchange != "" {
		bindingKeys := s.conf.BindingKeys
		if len(bindingKeys) == 0 {
			bindingKeys = []string{s.conf.Queue}
		}

		for _, bindingKey := range bindingKeys {
			if err := channel.QueueBind(s.conf.Queue, bindingKey, s.conf.Exchange, false, nil); err != nil {
				return fmt.Errorf("bind AMQP 0.9.1 queue: %w", err)
			}
		}
	}

	if s.conf.PrefetchCount > 0 {
		if err := channel.Qos(s.conf.PrefetchCount, 0, false); err != nil {
			return fmt.Errorf("configure AMQP 0.9.1 server prefetch: %w", err)
		}
	}

	return nil
}

func (s *server) consumeLoop(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case delivery, ok := <-s.deliveries:
			if !ok {
				return
			}

			s.handleDelivery(&delivery)
		}
	}
}

func (s *server) handleDelivery(delivery *amqp.Delivery) {
	if err := s.processDelivery(delivery); err != nil {
		s.logger.Errorf("Failed to process the AMQP 0.9.1 message: %v", err)
		if !s.conf.AutoAck {
			if nackErr := delivery.Nack(false, true); nackErr != nil {
				s.logger.Errorf("Failed to negatively acknowledge the AMQP 0.9.1 message: %v", nackErr)
			}
		}

		return
	}

	if !s.conf.AutoAck {
		if err := delivery.Ack(false); err != nil {
			s.logger.Errorf("Failed to acknowledge the AMQP 0.9.1 message: %v", err)
		}
	}
}

func (s *server) processDelivery(delivery *amqp.Delivery) error {
	filename := renderServerFilename(s.conf, delivery)
	if filename == "" {
		return errEmptyResolvedFilename
	}

	remoteID := strconv.FormatUint(delivery.DeliveryTag, 10)
	if delivery.DeliveryTag == 0 {
		remoteID = strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	}

	transfer := pipeline.MakeServerTransfer(remoteID, filename, s.localAccount, s.rule)
	transfer.TransferInfo = map[string]any{
		"amqpExchange":    delivery.Exchange,
		"amqpRoutingKey":  delivery.RoutingKey,
		"amqpMessageID":   delivery.MessageId,
		"amqpCorrelation": delivery.CorrelationId,
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

	if _, err := serverPip.Write(delivery.Body); err != nil {
		return fmt.Errorf("write AMQP 0.9.1 message payload: %w", err)
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

	if s.channel != nil {
		if err := s.channel.Cancel(s.conf.ConsumerTag, false); err != nil && !errors.Is(err, amqp.ErrClosed) {
			s.logger.Warningf("Failed to cancel the AMQP 0.9.1 consumer cleanly: %v", err)
		}
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

		return fmt.Errorf("stop the AMQP 0.9.1 server: %w", ctx.Err())
	case <-done:
	}

	if err := s.closeResources(); err != nil {
		s.state.Set(utils.StateError, err.Error())
		snmp.ReportServiceFailure(s.agent.Name, err)

		return err
	}

	s.state.Set(utils.StateOffline, "")

	return nil
}

func (s *server) closeResources() error {
	if s.channel == nil && s.conn == nil {
		return errMissingServerResources
	}

	var errs []string
	if s.channel != nil {
		if err := s.channel.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
			errs = append(errs, err.Error())
		}
		s.channel = nil
	}

	if s.conn != nil {
		if err := s.conn.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
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

func renderServerFilename(conf *serverConfig, delivery *amqp.Delivery) string {
	if conf != nil {
		if headerName := strings.TrimSpace(conf.FilenameHeader); headerName != "" {
			if filename, ok := amqpHeaderString(delivery.Headers, headerName); ok {
				return normalizeServerFilename(filename)
			}
		}

		template := strings.TrimSpace(conf.FilenameTemplate)
		if template != "" {
			filename := template
			replacements := map[string]string{
				"${messageID}":     firstNonEmpty(delivery.MessageId, fmt.Sprintf("delivery-%d", delivery.DeliveryTag)),
				"${correlationID}": delivery.CorrelationId,
				"${routingKey}":    delivery.RoutingKey,
				"${exchange}":      delivery.Exchange,
				"${consumerTag}":   firstNonEmpty(conf.ConsumerTag, conf.Queue),
				"${timestamp}":     delivery.Timestamp.UTC().Format("20060102T150405"),
			}
			for key, value := range replacements {
				filename = strings.ReplaceAll(filename, key, value)
			}

			return normalizeServerFilename(filename)
		}
	}

	return normalizeServerFilename(firstNonEmpty(delivery.MessageId, fmt.Sprintf("delivery-%d", delivery.DeliveryTag)))
}

func amqpHeaderString(headers amqp.Table, key string) (string, bool) {
	if headers == nil {
		return "", false
	}

	raw, ok := headers[key]
	if !ok {
		return "", false
	}

	switch value := raw.(type) {
	case string:
		return strings.TrimSpace(value), strings.TrimSpace(value) != ""
	case []byte:
		out := strings.TrimSpace(string(value))
		return out, out != ""
	default:
		return "", false
	}
}

func normalizeServerFilename(raw string) string {
	filename := strings.TrimSpace(raw)
	filename = strings.ReplaceAll(filename, "\\", "/")
	filename = strings.TrimLeft(filename, "/")
	filename = strings.TrimPrefix(filename, "./")
	filename = strings.ReplaceAll(filename, "../", "")
	filename = strings.ReplaceAll(filename, "..", "")

	return filename
}

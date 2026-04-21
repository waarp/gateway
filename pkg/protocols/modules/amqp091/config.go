package amqp091

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	defaultHeartbeatSeconds = 30
	defaultConsumeTimeout   = 30
)

var (
	errNegativeRetryPolicy     = errors.New("the AMQP 0.9.1 retry policy values cannot be negative")
	errRetryDelayOrder         = errors.New("the AMQP 0.9.1 retry policy initial delay cannot exceed the max delay")
	errMissingExchangeOrQueue  = errors.New("the AMQP 0.9.1 partner must define at least an exchange or a queue")
	errNegativePrefetch        = errors.New("the AMQP 0.9.1 partner prefetchCount cannot be negative")
	errMissingServerQueue      = errors.New("the AMQP 0.9.1 server queue is missing")
	errNegativeServerPrefetch  = errors.New("the AMQP 0.9.1 server prefetchCount cannot be negative")
	errMissingServerAccount    = errors.New("the AMQP 0.9.1 server localAccount is missing")
	errMissingServerRule       = errors.New("the AMQP 0.9.1 server ruleName is missing")
	errMissingAMQPURI          = errors.New("the AMQP 0.9.1 URI is missing")
	errUnsupportedAMQPScheme   = errors.New("the AMQP 0.9.1 URI must use amqp or amqps")
	errMissingAMQPURIHost      = errors.New("the AMQP 0.9.1 URI host is missing")
	errUnsupportedExchangeType = errors.New("is not a supported AMQP 0.9.1 exchange type")
)

type retryPolicy struct {
	InitialDelaySeconds int `json:"initialDelaySeconds" yaml:"initialDelaySeconds"`
	MaxDelaySeconds     int `json:"maxDelaySeconds" yaml:"maxDelaySeconds"`
	MaxAttempts         int `json:"maxAttempts" yaml:"maxAttempts"`
}

type clientConfig struct {
	URI                string      `json:"uri" yaml:"uri"`
	Exchange           string      `json:"exchange" yaml:"exchange"`
	ExchangeType       string      `json:"exchangeType" yaml:"exchangeType"`
	RoutingKeyTemplate string      `json:"routingKeyTemplate" yaml:"routingKeyTemplate"`
	Mandatory          bool        `json:"mandatory" yaml:"mandatory"`
	PersistentMessages bool        `json:"persistentMessages" yaml:"persistentMessages"`
	PublisherConfirms  bool        `json:"publisherConfirms" yaml:"publisherConfirms"`
	HeartbeatSeconds   int         `json:"heartbeatSeconds" yaml:"heartbeatSeconds"`
	ConnectionName     string      `json:"connectionName" yaml:"connectionName"`
	ConsumeTimeout     int         `json:"consumeTimeout" yaml:"consumeTimeout"`
	RetryPolicy        retryPolicy `json:"retryPolicy" yaml:"retryPolicy"`
}

type partnerConfig struct {
	VirtualHost   string   `json:"virtualHost" yaml:"virtualHost"`
	Exchange      string   `json:"exchange" yaml:"exchange"`
	Queue         string   `json:"queue" yaml:"queue"`
	QueueDurable  bool     `json:"queueDurable" yaml:"queueDurable"`
	BindingKeys   []string `json:"bindingKeys" yaml:"bindingKeys"`
	ConsumerTag   string   `json:"consumerTag" yaml:"consumerTag"`
	PrefetchCount int      `json:"prefetchCount" yaml:"prefetchCount"`
	AutoAck       bool     `json:"autoAck" yaml:"autoAck"`
}

type serverConfig struct {
	URI              string   `json:"uri" yaml:"uri"`
	Exchange         string   `json:"exchange" yaml:"exchange"`
	ExchangeType     string   `json:"exchangeType" yaml:"exchangeType"`
	Queue            string   `json:"queue" yaml:"queue"`
	QueueDurable     bool     `json:"queueDurable" yaml:"queueDurable"`
	BindingKeys      []string `json:"bindingKeys" yaml:"bindingKeys"`
	ConsumerTag      string   `json:"consumerTag" yaml:"consumerTag"`
	PrefetchCount    int      `json:"prefetchCount" yaml:"prefetchCount"`
	AutoAck          bool     `json:"autoAck" yaml:"autoAck"`
	HeartbeatSeconds int      `json:"heartbeatSeconds" yaml:"heartbeatSeconds"`
	ConnectionName   string   `json:"connectionName" yaml:"connectionName"`
	LocalAccount     string   `json:"localAccount" yaml:"localAccount"`
	RuleName         string   `json:"ruleName" yaml:"ruleName"`
	FilenameHeader   string   `json:"filenameHeader" yaml:"filenameHeader"`
	FilenameTemplate string   `json:"filenameTemplate" yaml:"filenameTemplate"`
}

func defaultClientConfig() *clientConfig {
	return &clientConfig{
		HeartbeatSeconds: defaultHeartbeatSeconds,
		ConsumeTimeout:   defaultConsumeTimeout,
	}
}

func defaultPartnerConfig() *partnerConfig { return &partnerConfig{} }

func defaultServerConfig() *serverConfig {
	return &serverConfig{
		HeartbeatSeconds: defaultHeartbeatSeconds,
		FilenameHeader:   "filename",
		FilenameTemplate: "${messageID}",
	}
}

func (c *clientConfig) ValidClient() error {
	c.URI = strings.TrimSpace(c.URI)
	c.Exchange = strings.TrimSpace(c.Exchange)
	c.ExchangeType = strings.TrimSpace(c.ExchangeType)
	c.RoutingKeyTemplate = strings.TrimSpace(c.RoutingKeyTemplate)
	c.ConnectionName = strings.TrimSpace(c.ConnectionName)

	if err := validateAMQPURI(c.URI); err != nil {
		return err
	}
	if c.HeartbeatSeconds <= 0 {
		c.HeartbeatSeconds = defaultHeartbeatSeconds
	}
	if c.ConsumeTimeout <= 0 {
		c.ConsumeTimeout = defaultConsumeTimeout
	}
	if c.ExchangeType != "" && !isValidExchangeType(c.ExchangeType) {
		return fmt.Errorf("%w %q", errUnsupportedExchangeType, c.ExchangeType)
	}
	if c.RetryPolicy.InitialDelaySeconds < 0 || c.RetryPolicy.MaxDelaySeconds < 0 || c.RetryPolicy.MaxAttempts < 0 {
		return errNegativeRetryPolicy
	}
	if c.RetryPolicy.MaxDelaySeconds != 0 &&
		c.RetryPolicy.InitialDelaySeconds > c.RetryPolicy.MaxDelaySeconds {
		return errRetryDelayOrder
	}

	return nil
}

func (p *partnerConfig) ValidPartner() error {
	p.VirtualHost = strings.TrimSpace(p.VirtualHost)
	p.Exchange = strings.TrimSpace(p.Exchange)
	p.Queue = strings.TrimSpace(p.Queue)
	p.ConsumerTag = strings.TrimSpace(p.ConsumerTag)
	for i, key := range p.BindingKeys {
		p.BindingKeys[i] = strings.TrimSpace(key)
	}
	if p.Exchange == "" && p.Queue == "" {
		return errMissingExchangeOrQueue
	}
	if p.PrefetchCount < 0 {
		return errNegativePrefetch
	}

	return nil
}

func (s *serverConfig) ValidServer() error {
	s.URI = strings.TrimSpace(s.URI)
	s.Exchange = strings.TrimSpace(s.Exchange)
	s.ExchangeType = strings.TrimSpace(s.ExchangeType)
	s.Queue = strings.TrimSpace(s.Queue)
	s.ConsumerTag = strings.TrimSpace(s.ConsumerTag)
	s.ConnectionName = strings.TrimSpace(s.ConnectionName)
	s.LocalAccount = strings.TrimSpace(s.LocalAccount)
	s.RuleName = strings.TrimSpace(s.RuleName)
	s.FilenameHeader = strings.TrimSpace(s.FilenameHeader)
	s.FilenameTemplate = strings.TrimSpace(s.FilenameTemplate)
	for i, key := range s.BindingKeys {
		s.BindingKeys[i] = strings.TrimSpace(key)
	}

	if err := validateAMQPURI(s.URI); err != nil {
		return err
	}
	if s.ExchangeType != "" && !isValidExchangeType(s.ExchangeType) {
		return fmt.Errorf("%w %q", errUnsupportedExchangeType, s.ExchangeType)
	}
	if s.Queue == "" {
		return errMissingServerQueue
	}
	if s.LocalAccount == "" {
		return errMissingServerAccount
	}
	if s.RuleName == "" {
		return errMissingServerRule
	}
	if s.PrefetchCount < 0 {
		return errNegativeServerPrefetch
	}
	if s.HeartbeatSeconds <= 0 {
		s.HeartbeatSeconds = defaultHeartbeatSeconds
	}

	return nil
}

func validateAMQPURI(raw string) error {
	if raw == "" {
		return errMissingAMQPURI
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid AMQP 0.9.1 URI: %w", err)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "amqp", "amqps":
	default:
		return errUnsupportedAMQPScheme
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return errMissingAMQPURIHost
	}

	return nil
}

func isValidExchangeType(raw string) bool {
	switch raw {
	case "", amqp.ExchangeDirect, amqp.ExchangeFanout, amqp.ExchangeTopic, amqp.ExchangeHeaders:
		return true
	default:
		return false
	}
}

package amqp10

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const (
	defaultIdleTimeoutSeconds = 60
	defaultConsumeTimeout     = 30
	defaultReceiverCredit     = 1
)

var (
	errMissingAMQPEndpoint          = errors.New("the AMQP 1.0 endpoint is missing")
	errUnsupportedAMQPScheme        = errors.New("the AMQP 1.0 endpoint must use amqp or amqps")
	errMissingAMQPEndpointHost      = errors.New("the AMQP 1.0 endpoint host is missing")
	errMissingTargetAddress         = errors.New("the AMQP 1.0 target address is missing")
	errMissingSourceAddress         = errors.New("the AMQP 1.0 source address is missing")
	errNegativeRetryPolicy          = errors.New("the AMQP 1.0 retry policy values cannot be negative")
	errRetryDelayOrder              = errors.New("the AMQP 1.0 retry policy initial delay cannot exceed the max delay")
	errNegativeMaxInFlight          = errors.New("the AMQP 1.0 maxInFlight cannot be negative")
	errNegativeCredit               = errors.New("the AMQP 1.0 receiver credit cannot be negative")
	errUnsupportedSettlementMode    = errors.New("is not a supported AMQP 1.0 settlement mode")
	errMissingServerSourceAddress   = errors.New("the AMQP 1.0 server sourceAddress is missing")
	errMissingServerLocalAccount    = errors.New("the AMQP 1.0 server localAccount is missing")
	errMissingServerRule            = errors.New("the AMQP 1.0 server ruleName is missing")
	errNegativeServerCredit         = errors.New("the AMQP 1.0 server credit cannot be negative")
	errMissingServerFilenamePattern = errors.New("the AMQP 1.0 server filenameTemplate is missing")
)

type retryPolicy struct {
	InitialDelaySeconds int `json:"initialDelaySeconds" yaml:"initialDelaySeconds"`
	MaxDelaySeconds     int `json:"maxDelaySeconds" yaml:"maxDelaySeconds"`
	MaxAttempts         int `json:"maxAttempts" yaml:"maxAttempts"`
}

type clientConfig struct {
	Endpoint           string      `json:"endpoint" yaml:"endpoint"`
	TargetAddress      string      `json:"targetAddress" yaml:"targetAddress"`
	SenderLinkName     string      `json:"senderLinkName" yaml:"senderLinkName"`
	SettlementMode     string      `json:"settlementMode" yaml:"settlementMode"`
	Durable            bool        `json:"durable" yaml:"durable"`
	IdleTimeoutSeconds int         `json:"idleTimeoutSeconds" yaml:"idleTimeoutSeconds"`
	MaxInFlight        int         `json:"maxInFlight" yaml:"maxInFlight"`
	ConnectionName     string      `json:"connectionName" yaml:"connectionName"`
	ConsumeTimeout     int         `json:"consumeTimeout" yaml:"consumeTimeout"`
	RetryPolicy        retryPolicy `json:"retryPolicy" yaml:"retryPolicy"`
}

type partnerConfig struct {
	SourceAddress      string `json:"sourceAddress" yaml:"sourceAddress"`
	ReceiverLinkName   string `json:"receiverLinkName" yaml:"receiverLinkName"`
	Credit             int    `json:"credit" yaml:"credit"`
	SettlementMode     string `json:"settlementMode" yaml:"settlementMode"`
	IdleTimeoutSeconds int    `json:"idleTimeoutSeconds" yaml:"idleTimeoutSeconds"`
}

type serverConfig struct {
	Endpoint           string `json:"endpoint" yaml:"endpoint"`
	SourceAddress      string `json:"sourceAddress" yaml:"sourceAddress"`
	ReceiverLinkName   string `json:"receiverLinkName" yaml:"receiverLinkName"`
	Credit             int    `json:"credit" yaml:"credit"`
	IdleTimeoutSeconds int    `json:"idleTimeoutSeconds" yaml:"idleTimeoutSeconds"`
	LocalAccount       string `json:"localAccount" yaml:"localAccount"`
	RuleName           string `json:"ruleName" yaml:"ruleName"`
	FilenameTemplate   string `json:"filenameTemplate" yaml:"filenameTemplate"`
}

func defaultClientConfig() *clientConfig {
	return &clientConfig{
		IdleTimeoutSeconds: defaultIdleTimeoutSeconds,
		ConsumeTimeout:     defaultConsumeTimeout,
		MaxInFlight:        1,
	}
}

func defaultPartnerConfig() *partnerConfig {
	return &partnerConfig{
		Credit:             defaultReceiverCredit,
		IdleTimeoutSeconds: defaultIdleTimeoutSeconds,
	}
}

func defaultServerConfig() *serverConfig {
	return &serverConfig{
		Credit:             defaultReceiverCredit,
		IdleTimeoutSeconds: defaultIdleTimeoutSeconds,
		FilenameTemplate:   "${messageID}",
	}
}

func (c *clientConfig) ValidClient() error {
	c.Endpoint = strings.TrimSpace(c.Endpoint)
	c.TargetAddress = strings.TrimSpace(c.TargetAddress)
	c.SenderLinkName = strings.TrimSpace(c.SenderLinkName)
	c.SettlementMode = strings.TrimSpace(strings.ToLower(c.SettlementMode))
	c.ConnectionName = strings.TrimSpace(c.ConnectionName)

	if err := validateEndpoint(c.Endpoint); err != nil {
		return err
	}
	if c.TargetAddress == "" {
		return errMissingTargetAddress
	}
	if c.IdleTimeoutSeconds <= 0 {
		c.IdleTimeoutSeconds = defaultIdleTimeoutSeconds
	}
	if c.ConsumeTimeout <= 0 {
		c.ConsumeTimeout = defaultConsumeTimeout
	}
	if c.MaxInFlight < 0 {
		return errNegativeMaxInFlight
	}
	if c.MaxInFlight == 0 {
		c.MaxInFlight = 1
	}
	if !isValidSettlementMode(c.SettlementMode) {
		return fmt.Errorf("%w %q", errUnsupportedSettlementMode, c.SettlementMode)
	}
	if c.RetryPolicy.InitialDelaySeconds < 0 || c.RetryPolicy.MaxDelaySeconds < 0 || c.RetryPolicy.MaxAttempts < 0 {
		return errNegativeRetryPolicy
	}
	if c.RetryPolicy.MaxDelaySeconds != 0 && c.RetryPolicy.InitialDelaySeconds > c.RetryPolicy.MaxDelaySeconds {
		return errRetryDelayOrder
	}

	return nil
}

func (p *partnerConfig) ValidPartner() error {
	p.SourceAddress = strings.TrimSpace(p.SourceAddress)
	p.ReceiverLinkName = strings.TrimSpace(p.ReceiverLinkName)
	p.SettlementMode = strings.TrimSpace(strings.ToLower(p.SettlementMode))

	if p.SourceAddress == "" {
		return errMissingSourceAddress
	}
	if p.Credit < 0 {
		return errNegativeCredit
	}
	if p.Credit == 0 {
		p.Credit = defaultReceiverCredit
	}
	if p.IdleTimeoutSeconds <= 0 {
		p.IdleTimeoutSeconds = defaultIdleTimeoutSeconds
	}
	if !isValidSettlementMode(p.SettlementMode) {
		return fmt.Errorf("%w %q", errUnsupportedSettlementMode, p.SettlementMode)
	}

	return nil
}

func (s *serverConfig) ValidServer() error {
	s.Endpoint = strings.TrimSpace(s.Endpoint)
	s.SourceAddress = strings.TrimSpace(s.SourceAddress)
	s.ReceiverLinkName = strings.TrimSpace(s.ReceiverLinkName)
	s.LocalAccount = strings.TrimSpace(s.LocalAccount)
	s.RuleName = strings.TrimSpace(s.RuleName)
	s.FilenameTemplate = strings.TrimSpace(s.FilenameTemplate)

	if err := validateEndpoint(s.Endpoint); err != nil {
		return err
	}
	if s.SourceAddress == "" {
		return errMissingServerSourceAddress
	}
	if s.LocalAccount == "" {
		return errMissingServerLocalAccount
	}
	if s.RuleName == "" {
		return errMissingServerRule
	}
	if s.Credit < 0 {
		return errNegativeServerCredit
	}
	if s.Credit == 0 {
		s.Credit = defaultReceiverCredit
	}
	if s.IdleTimeoutSeconds <= 0 {
		s.IdleTimeoutSeconds = defaultIdleTimeoutSeconds
	}
	if s.FilenameTemplate == "" {
		return errMissingServerFilenamePattern
	}

	return nil
}

func validateEndpoint(raw string) error {
	if raw == "" {
		return errMissingAMQPEndpoint
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid AMQP 1.0 endpoint: %w", err)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "amqp", "amqps":
	default:
		return errUnsupportedAMQPScheme
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return errMissingAMQPEndpointHost
	}

	return nil
}

func isValidSettlementMode(raw string) bool {
	switch raw {
	case "", "mixed", "settled", "unsettled", "peek-lock":
		return true
	default:
		return false
	}
}

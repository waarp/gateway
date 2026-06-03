package pesit

import (
	"fmt"
	"math"
	"strings"

	libpesit "code.waarp.fr/lib/pesit"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

const (
	ArticleFormatVariable = "variable"
	ArticleFormatFixed    = "fixed"
)

// resolveArticleFormat converts a config string to a lib-pesit ArticleFormat.
// Returns FormatVariable by default (standard PeSIT behavior).
func resolveArticleFormat(configValue string) libpesit.ArticleFormat {
	if strings.EqualFold(configValue, ArticleFormatFixed) {
		return libpesit.FormatFixed
	}

	return libpesit.FormatVariable
}

const (
	// DefaultCheckpointSize defines the default checkpoint size (in bytes) if
	// omitted by the user in the proto config.
	DefaultCheckpointSize uint16 = math.MaxUint16
	// DefaultCheckpointWindow defines the default checkpoint window if omitted
	// by the user in the proto config. Default is 2.
	DefaultCheckpointWindow uint8 = 2
	// DefaultMessageSize defines the default PeSIT message size (in bytes) if
	// omitted by the user in the proto config.
	DefaultMessageSize uint16 = math.MaxUint16

	defaultArticleSize uint16 = 4096 // Matches Axway CFT default for interop
)

// CompressionMode defines the PeSIT compression type for PI 21.
type CompressionMode string

const (
	CompressionNone       CompressionMode = "none"
	CompressionHorizontal CompressionMode = "horizontal"
	CompressionVertical   CompressionMode = "vertical"
	CompressionBoth       CompressionMode = "both"
)

// ToPeSIT converts a CompressionMode to the lib/pesit Compression type.
func (m CompressionMode) ToPeSIT() libpesit.Compression {
	switch m {
	case CompressionHorizontal:
		return libpesit.Horizontal
	case CompressionVertical:
		return libpesit.Vertical
	case CompressionBoth:
		return libpesit.HorizontalVertical
	default:
		return libpesit.NoCompression
	}
}

type CompatibilityMode string

const (
	CompatibilityModeStandard    = "standard"
	CompatibilityModeNonStandard = "non-standard" // Deprecated: use CompatibilityModeHistorique
	CompatibilityModeHistorique  = "historique"
)

func (m *CompatibilityMode) UnmarshalJSON(b []byte) error {
	val := strings.Trim(string(b), "\"")

	switch val {
	case CompatibilityModeStandard, CompatibilityModeHistorique, CompatibilityModeNonStandard:
		// Accept "non-standard" as an alias for "historique" for backward compatibility
		if val == CompatibilityModeNonStandard {
			val = CompatibilityModeHistorique
		}

		*m = CompatibilityMode(val)

		return nil
	default:
		//nolint:err113 // this is a base error
		return fmt.Errorf("unknown compatibility mode %q (accepted: \"standard\", \"historique\")", val)
	}
}

type CheckPointConfig struct {
	// DisableRestart will disable restarts on this agent if set to true.
	// By default, restarts are enabled.
	DisableRestart bool `json:"disableRestart"`
	// DisableCheckpoints will disable checkpoints on this agent if set to true.
	// By default, checkpoints are enabled.
	DisableCheckpoints bool `json:"disableCheckpoints"`
	// CheckpointSize defines the size (in bytes) of the checkpoint set for the
	// connections to this agent. Default is 65535.
	CheckpointSize uint16 `json:"checkpointSize,omitempty"`
	// CheckpointWindow defines the number of checkpoints that can go unacknowledged
	// without stopping the transfer. Default is 2.
	CheckpointWindow uint8 `json:"checkpointWindow,omitempty"`
}

func (c *CheckPointConfig) validCheckpoints() {
	if c.DisableCheckpoints {
		c.DisableRestart = true
		c.CheckpointSize = 0
		c.CheckpointWindow = 0
	} else {
		if c.CheckpointSize == 0 {
			c.CheckpointSize = DefaultCheckpointSize
		}

		if c.CheckpointWindow == 0 {
			c.CheckpointWindow = DefaultCheckpointWindow
		}
	}
}

// ServerConfig defines the JSON object representing the configuration of
// a pesit local server.
type ServerConfig struct {
	CheckPointConfig

	// The PeSIT compatibility mode to use when communicating with this server.
	// Accepted values are: "standard" or "historique". Default is "standard".
	// The "historique" mode uses the market convention inherited from the SIT
	// banking era: PI 12 = logical flow ID (rule name), PI 37 = physical filename.
	// "non-standard" is accepted as a deprecated alias for "historique".
	CompatibilityMode CompatibilityMode `json:"compatibilityMode,omitempty"`
	// DisablePreConnection disables the pre-connection authentication when
	// connecting to this server. By default, the pre-connection authentication
	// is activated.
	//
	// Deprecated: This is no longer useful as the library detects the preconnection stage.
	DisablePreConnection bool `json:"disablePreConnection,omitempty"`
	// MaxMessageSize defines the maximum allowed size for PeSIT packages sent to
	// this server. Default is 65535.
	MaxMessageSize uint16 `json:"maxMessageSize,omitempty"`
	// ArticleSize defines the article size (PI 32) announced in the protocol
	// negotiation. Default is 4096 (matching Axway CFT default).
	ArticleSize uint16 `json:"articleSize,omitempty"`
	// ProtocolTimeout defines the protocol surveillance timeout in seconds
	// (PI 26). When set, the server will abort the connection if no FPDU is
	// received within this delay during active protocol phases. Default is 0
	// (no timeout).
	ProtocolTimeout uint16 `json:"protocolTimeout,omitempty"`
	// ArticleFormat defines the article format used for outgoing transfers.
	// Accepted values are: "variable" (default) and "fixed".
	ArticleFormat string `json:"articleFormat,omitempty"`
	// Compression defines the compression mode proposed during the file open
	// phase (PI 21). Accepted values: "none", "horizontal", "vertical", "both".
	// Default is "none".
	Compression CompressionMode `json:"compression,omitempty"`
	// RelayMessages enables automatic Store & Forward relay of incoming
	// F.MESSAGE. When a partner sends an ACK message, the server looks up
	// the original transfer chain (__followID__) and relays the message
	// upstream to the originator. Default is false.
	RelayMessages bool `json:"relayMessages,omitempty"`
}

func (s *ServerConfig) ValidServer() error {
	s.validCheckpoints()

	if s.MaxMessageSize == 0 {
		s.MaxMessageSize = DefaultMessageSize
	}

	if s.ArticleSize == 0 {
		s.ArticleSize = defaultArticleSize
	}

	return nil
}

type ClientConfig struct {
	CheckPointConfig
}

func (c *ClientConfig) ValidClient() error {
	c.validCheckpoints()

	return nil
}

// PartnerConfig defines the JSON object representing the configuration of
// a pesit remote partner.
type PartnerConfig struct {
	// The partner's login for authentication purposes. If left empty, the partner
	// will not be authenticated.
	Login string `json:"login,omitempty"`

	// DisableRestart will disable restarts on this agent if set to true.
	// By default, restarts are enabled.
	DisableRestart api.Nullable[bool] `json:"disableRestart"`
	// DisableRestart will disable checkpoints on this agent if set to true.
	// By default, checkpoints are enabled.
	DisableCheckpoints api.Nullable[bool] `json:"disableCheckpoints"`
	// CheckpointSize defines the size (in bytes) of the checkpoint set for the
	// connections to this agent. Default is 65535.
	CheckpointSize uint16 `json:"checkpointSize,omitempty"`
	// CheckpointWindow defines the number of checkpoints that can go unacknowledged
	// without stopping the transfer. Default is 2.
	CheckpointWindow uint8 `json:"checkpointWindow,omitempty"`

	// UseNSDU specifies whether NSDU meta-packets should be used when making
	// transfers with that partner. By default, NSDU packets are used.
	UseNSDU api.Nullable[bool] `json:"useNSDU"` //nolint:tagliatelle //does not recognize NSDU as an acronym
	// The PeSIT compatibility mode to use when communicating with this partner.
	// Accepted values are: "standard" or "historique". Default is "standard".
	// The "historique" mode uses the market convention inherited from the SIT
	// banking era: PI 12 = logical flow ID (rule name), PI 37 = physical filename.
	// "non-standard" is accepted as a deprecated alias for "historique".
	CompatibilityMode CompatibilityMode `json:"compatibilityMode,omitempty"`
	// MaxMessageSize defines the maximum allowed size for PeSIT packages sent to
	// this partner. Default is 65535.
	MaxMessageSize uint16 `json:"maxMessageSize,omitempty"`
	// DisablePreConnection disables the pre-connection authentication when
	// connecting to this partner. By default, the pre-connection authentication
	// is activated.
	DisablePreConnection bool `json:"disablePreConnection,omitempty"`
	// ArticleSize defines the article size (PI 32) used when communicating
	// with this partner. Default is 4096 (matching Axway CFT default).
	ArticleSize uint16 `json:"articleSize,omitempty"`
	// ProtocolTimeout defines the protocol surveillance timeout in seconds
	// (PI 26) proposed to the server during connection. When set, both sides
	// will abort if no FPDU is received within this delay. Default is 0
	// (no timeout).
	ProtocolTimeout uint16 `json:"protocolTimeout,omitempty"`
	// MaxConnections limits the number of concurrent PeSIT connections to this
	// partner. When the limit is reached, new transfers wait until a connection
	// is released. Default is 0 (unlimited).
	MaxConnections uint16 `json:"maxConnections,omitempty"`
	// ArticleFormat defines the article format used for outgoing transfers to
	// this partner. Accepted values are: "variable" (default) and "fixed".
	// Overrides the server/client configuration if set.
	ArticleFormat string `json:"articleFormat,omitempty"`
	// Compression defines the compression mode proposed during the file open
	// phase (PI 21). Accepted values: "none", "horizontal", "vertical", "both".
	// Default is "none".
	Compression CompressionMode `json:"compression,omitempty"`
	// ReplyTo specifies the return address for Store & Forward ACKs.
	// When set, the Gateway automatically adds "REPLY=<value>" in PI 99
	// (connection freetext) for every transfer to this partner.
	// Format: "partner-name:account-login" or just "partner-name".
	// This tells the receiver where to send the F.MESSAGE acknowledgment.
	ReplyTo string `json:"replyTo,omitempty"`
}

func (p *PartnerConfig) ValidPartner() error {
	if p.DisableCheckpoints.Valid && p.DisableCheckpoints.Value {
		p.DisableRestart = api.Nullable[bool]{Value: true, Valid: true}
	}

	if !p.UseNSDU.Valid {
		p.UseNSDU = api.NewNullable(true)
	}

	if p.CompatibilityMode == "" {
		p.CompatibilityMode = CompatibilityModeStandard
	}

	if p.MaxMessageSize == 0 {
		p.MaxMessageSize = DefaultMessageSize
	}

	if p.ArticleSize == 0 {
		p.ArticleSize = defaultArticleSize
	}

	return nil
}

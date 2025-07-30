package pesit

import (
	"fmt"
	"math"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

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
)

const (
	CompatibilityModeNone  = "none"
	CompatibilityModeAxway = "axway"
)

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

	// DisablePreConnection disables the pre-connection authentication when
	// connecting to this server. By default, the pre-connection authentication
	// is activated.
	DisablePreConnection bool `json:"disablePreConnection,omitempty"`
	// MaxMessageSize defines the maximum allowed size for PeSIT packages sent to
	// this server. Default is 65535.
	MaxMessageSize uint16 `json:"maxMessageSize,omitempty"`
}

func (s *ServerConfig) ValidServer() error {
	s.validCheckpoints()

	if s.MaxMessageSize == 0 {
		s.MaxMessageSize = DefaultMessageSize
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
	// transfers with that partner. By default, NSDU packets are not used.
	UseNSDU bool `json:"useNSDU"` //nolint:tagliatelle //does not recognize NSDU as an acronym
	// The PeSIT compatibility mode to use when communicating with this partner.
	// Accepted values are: "none" or "axway". Default is "none".
	CompatibilityMode string `json:"compatibilityMode,omitempty"`
	// MaxMessageSize defines the maximum allowed size for PeSIT packages sent to
	// this partner. Default is 65535.
	MaxMessageSize uint16 `json:"maxMessageSize,omitempty"`
	// DisablePreConnection disables the pre-connection authentication when
	// connecting to this partner. By default, the pre-connection authentication
	// is activated.
	DisablePreConnection bool `json:"disablePreConnection,omitempty"`
}

func (p *PartnerConfig) ValidPartner() error {
	if p.DisableCheckpoints.Valid && !p.DisableRestart.Value {
		p.DisableRestart = api.Nullable[bool]{Value: true, Valid: true}
	}

	if p.CompatibilityMode == "" {
		p.CompatibilityMode = CompatibilityModeNone
	}

	if p.MaxMessageSize == 0 {
		p.MaxMessageSize = DefaultMessageSize
	}

	return checkCompatibilityMode(p.CompatibilityMode)
}

func checkCompatibilityMode(mode string) error {
	switch mode {
	case CompatibilityModeNone, CompatibilityModeAxway:
		return nil
	default:
		//nolint:err113 // this is a base error
		return fmt.Errorf("unknown compatibility mode %q", mode)
	}
}

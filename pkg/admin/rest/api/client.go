package api

// InClient is the JSON object representing a local client in POST requests
// made to the gateway's REST server.
type InClient struct {
	Name                 Nullable[string]  `json:"name,omitzero" yaml:"name,omitempty"`
	Protocol             Nullable[string]  `json:"protocol,omitzero" yaml:"protocol,omitempty"`
	Disabled             Nullable[bool]    `json:"disabled,omitzero" yaml:"disabled,omitempty"`
	LocalAddress         Nullable[string]  `json:"localAddress,omitzero" yaml:"localAddress,omitempty"`
	NbOfAttempts         Nullable[int8]    `json:"nbOfAttempts,omitzero" yaml:"nbOfAttempts,omitempty"`
	FirstRetryDelay      Nullable[int32]   `json:"firstRetryDelay,omitzero" yaml:"firstRetryDelay,omitempty"`
	RetryIncrementFactor Nullable[float32] `json:"retryIncrementFactor,omitzero" yaml:"retryIncrementFactor,omitempty"`
	ProtoConfig          UpdateObject[any] `json:"protoConfig,omitempty" yaml:"protoConfig,omitempty"`
}

// OutClient is the JSON object representing a local client in responses to GET
// requests made to the gateway's REST server.
type OutClient struct {
	Name                 string         `json:"name" yaml:"name"`
	Enabled              bool           `json:"enabled" yaml:"enabled"`
	Protocol             string         `json:"protocol" yaml:"protocol"`
	LocalAddress         string         `json:"localAddress" yaml:"localAddress"`
	NbOfAttempts         int8           `json:"nbOfAttempts" yaml:"nbOfAttempts"`
	FirstRetryDelay      int32          `json:"firstRetryDelay" yaml:"firstRetryDelay"`
	RetryIncrementFactor float32        `json:"retryIncrementFactor" yaml:"retryIncrementFactor"`
	ProtoConfig          map[string]any `json:"protoConfig" yaml:"protoConfig"`
}

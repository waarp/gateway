package api

// InClient is the JSON object representing a local client in POST requests
// made to the gateway's REST server.
type InClient struct {
	Name                 Nullable[string]  `json:"name,omitempty"`
	Protocol             Nullable[string]  `json:"protocol,omitempty"`
	Disabled             Nullable[bool]    `json:"disabled,omitempty"`
	LocalAddress         Nullable[string]  `json:"localAddress,omitempty"`
	NbOfAttempts         Nullable[int8]    `json:"nbOfAttempts,omitempty"`
	FirstRetryDelay      Nullable[int32]   `json:"firstRetryDelay,omitempty"`
	RetryIncrementFactor Nullable[float32] `json:"retryIncrementFactor,omitempty"`
	ProtoConfig          UpdateObject[any] `json:"protoConfig,omitempty"`
}

// OutClient is the JSON object representing a local client in responses to GET
// requests made to the gateway's REST server.
type OutClient struct {
	Name                 string         `json:"name"`
	Enabled              bool           `json:"enabled"`
	Protocol             string         `json:"protocol"`
	LocalAddress         string         `json:"localAddress"`
	NbOfAttempts         int8           `json:"nbOfAttempts"`
	FirstRetryDelay      int32          `json:"firstRetryDelay"`
	RetryIncrementFactor float32        `json:"retryIncrementFactor"`
	ProtoConfig          map[string]any `json:"protoConfig"`
}

package api

// InServer is the JSON representation of a local agent in requests
// made to the REST interface.
//
//nolint:lll // JSON tags can be long
type InServer struct {
	Name          Nullable[string]  `json:"name,omitzero" yaml:"name,omitempty"`
	Protocol      Nullable[string]  `json:"protocol,omitzero" yaml:"protocol,omitempty"`
	Address       Nullable[string]  `json:"address,omitzero" yaml:"address,omitempty"`
	RootDir       Nullable[string]  `json:"rootDir,omitzero" yaml:"rootDir,omitempty"`
	ReceiveDir    Nullable[string]  `json:"receiveDir,omitzero" yaml:"receiveDir,omitempty"`
	SendDir       Nullable[string]  `json:"sendDir,omitzero" yaml:"sendDir,omitempty"`
	TmpReceiveDir Nullable[string]  `json:"tmpReceiveDir,omitzero" yaml:"tmpReceiveDir,omitempty"`
	ProtoConfig   UpdateObject[any] `json:"protoConfig,omitempty" yaml:"protoConfig,omitempty"`

	// Deprecated fields
	Root    Nullable[string] `json:"root,omitzero"`    // Deprecated: replaced by RootDir
	InDir   Nullable[string] `json:"inDir,omitzero"`   // Deprecated: replaced by ReceiveDir & SendDir
	OutDir  Nullable[string] `json:"outDir,omitzero"`  // Deprecated: replaced by ReceiveDir & SendDir
	WorkDir Nullable[string] `json:"workDir,omitzero"` // Deprecated: replaced by TmpReceiveDir
}

// OutServer is the JSON representation of a local server in responses sent by
// the REST interface.
//
//nolint:lll // JSON tags can be long
type OutServer struct {
	Name            string          `json:"name" yaml:"name"`
	Protocol        string          `json:"protocol" yaml:"protocol"`
	Enabled         bool            `json:"enabled" yaml:"enabled"`
	Address         string          `json:"address" yaml:"address"`
	RootDir         string          `json:"rootDir,omitempty" yaml:"rootDir,omitempty"`
	ReceiveDir      string          `json:"receiveDir,omitempty" yaml:"receiveDir,omitempty"`
	SendDir         string          `json:"sendDir,omitempty" yaml:"sendDir,omitempty"`
	TmpReceiveDir   string          `json:"tmpReceiveDir,omitempty" yaml:"tmpReceiveDir,omitempty"`
	Credentials     []string        `json:"credentials,omitempty" yaml:"credentials,omitempty"`
	ProtoConfig     map[string]any  `json:"protoConfig" yaml:"protoConfig"`
	AuthorizedRules AuthorizedRules `json:"authorizedRules" yaml:"authorizedRules"`

	// Deprecated fields
	Root    string `json:"root,omitempty"`    // Deprecated: replaced by RootDir
	InDir   string `json:"inDir,omitempty"`   // Deprecated: replaced by ReceiveDir & SendDir
	OutDir  string `json:"outDir,omitempty"`  // Deprecated: replaced by ReceiveDir & SendDir
	WorkDir string `json:"workDir,omitempty"` // Deprecated: replaced by TmpReceiveDir
}

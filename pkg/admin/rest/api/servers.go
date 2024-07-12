package api

// InServer is the JSON representation of a local agent in requests
// made to the REST interface.
//
//nolint:lll // JSON tags can be long
type InServer struct {
	Name          Nullable[string] `json:"name,omitempty"`
	Protocol      Nullable[string] `json:"protocol,omitempty"`
	Address       Nullable[string] `json:"address,omitempty"`
	RootDir       Nullable[string] `json:"rootDir,omitempty"`
	ReceiveDir    Nullable[string] `json:"receiveDir,omitempty"`
	SendDir       Nullable[string] `json:"sendDir,omitempty"`
	TmpReceiveDir Nullable[string] `json:"tmpReceiveDir,omitempty"`
	ProtoConfig   UpdateObject     `json:"protoConfig,omitempty"`

	// Deprecated fields
	Root    Nullable[string] `json:"root,omitempty"`    // Deprecated: replaced by RootDir
	InDir   Nullable[string] `json:"inDir,omitempty"`   // Deprecated: replaced by ReceiveDir & SendDir
	OutDir  Nullable[string] `json:"outDir,omitempty"`  // Deprecated: replaced by ReceiveDir & SendDir
	WorkDir Nullable[string] `json:"workDir,omitempty"` // Deprecated: replaced by TmpReceiveDir
}

// OutServer is the JSON representation of a local server in responses sent by
// the REST interface.
//
//nolint:lll // JSON tags can be long
type OutServer struct {
	Name            string          `json:"name"`
	Protocol        string          `json:"protocol"`
	Enabled         bool            `json:"enabled"`
	Address         string          `json:"address"`
	RootDir         string          `json:"rootDir,omitempty"`
	ReceiveDir      string          `json:"receiveDir,omitempty"`
	SendDir         string          `json:"sendDir,omitempty"`
	TmpReceiveDir   string          `json:"tmpReceiveDir,omitempty"`
	Credentials     []string        `json:"credentials,omitempty"`
	ProtoConfig     map[string]any  `json:"protoConfig"`
	AuthorizedRules AuthorizedRules `json:"authorizedRules"`

	// Deprecated fields
	Root    string `json:"root,omitempty"`    // Deprecated: replaced by RootDir
	InDir   string `json:"inDir,omitempty"`   // Deprecated: replaced by ReceiveDir & SendDir
	OutDir  string `json:"outDir,omitempty"`  // Deprecated: replaced by ReceiveDir & SendDir
	WorkDir string `json:"workDir,omitempty"` // Deprecated: replaced by TmpReceiveDir
}

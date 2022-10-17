package api

import (
	"encoding/json"
)

// InServer is the JSON representation of a local agent in requests
// made to the REST interface.
//
//nolint:lll // JSON tags can be long
type InServer struct {
	Name          *string         `json:"name,omitempty"`
	Protocol      *string         `json:"protocol,omitempty"`
	Address       *string         `json:"address,omitempty"`
	RootDir       *string         `json:"rootDir,omitempty"`
	ReceiveDir    *string         `json:"receiveDir,omitempty"`
	SendDir       *string         `json:"sendDir,omitempty"`
	TmpReceiveDir *string         `json:"tmpReceiveDir,omitempty"`
	ProtoConfig   json.RawMessage `json:"protoConfig,omitempty"`

	// Deprecated fields
	Root    *string `json:"root,omitempty"`    // Deprecated: replaced by RootDir
	InDir   *string `json:"inDir,omitempty"`   // Deprecated: replaced by ReceiveDir & SendDir
	OutDir  *string `json:"outDir,omitempty"`  // Deprecated: replaced by ReceiveDir & SendDir
	WorkDir *string `json:"workDir,omitempty"` // Deprecated: replaced by TmpReceiveDir
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
	ProtoConfig     json.RawMessage `json:"protoConfig"`
	AuthorizedRules AuthorizedRules `json:"authorizedRules"`

	// Deprecated fields
	Root    string `json:"root,omitempty"`    // Deprecated: replaced by RootDir
	InDir   string `json:"inDir,omitempty"`   // Deprecated: replaced by ReceiveDir & SendDir
	OutDir  string `json:"outDir,omitempty"`  // Deprecated: replaced by ReceiveDir & SendDir
	WorkDir string `json:"workDir,omitempty"` // Deprecated: replaced by TmpReceiveDir
}

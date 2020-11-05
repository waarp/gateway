package models

import (
	"encoding/json"
)

// InServer is the JSON representation of a local agent in requests
// made to the REST interface.
type InServer struct {
	Name        *string         `json:"name,omitempty"`
	Protocol    *string         `json:"protocol,omitempty"`
	Address     *string         `json:"address,omitempty"`
	Root        *string         `json:"root,omitempty"`
	InDir       *string         `json:"inDir,omitempty"`
	OutDir      *string         `json:"outDir,omitempty"`
	WorkDir     *string         `json:"workDir,omitempty"`
	ProtoConfig json.RawMessage `json:"protoConfig,omitempty"`
}

// OutServer is the JSON representation of a local server in responses sent by
// the REST interface.
type OutServer struct {
	Name            string          `json:"name"`
	Protocol        string          `json:"protocol"`
	Address         string          `json:"address"`
	Root            string          `json:"root,omitempty"`
	InDir           string          `json:"inDir,omitempty"`
	OutDir          string          `json:"outDir,omitempty"`
	WorkDir         string          `json:"workDir,omitempty"`
	ProtoConfig     json.RawMessage `json:"protoConfig"`
	AuthorizedRules AuthorizedRules `json:"authorizedRules"`
}

// Package file contains the declaration of the struct describing the JSON
// object used in backup files.
package file

import (
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// Data is the top-level structure of the dump file.
type Data struct {
	Locals  []LocalAgent  `json:"locals,omitempty"`
	Clients []Client      `json:"clients,omitempty"`
	Remotes []RemoteAgent `json:"remotes,omitempty"`
	Rules   []Rule        `json:"rules,omitempty"`
	Users   []User        `json:"users,omitempty"`
}

// LocalAgent is the JSON struct representing a local server along with its
// accounts.
type LocalAgent struct {
	Name          string         `json:"name"`
	Protocol      string         `json:"protocol"`
	Disabled      bool           `json:"disabled"`
	RootDir       string         `json:"rootDir,omitempty"`
	ReceiveDir    string         `json:"receiveDir,omitempty"`
	SendDir       string         `json:"sendDir,omitempty"`
	TmpReceiveDir string         `json:"tmpReceiveDir,omitempty"`
	Address       string         `json:"address"`
	Configuration map[string]any `json:"configuration"`
	Accounts      []LocalAccount `json:"accounts"`
	Certs         []Certificate  `json:"certificates"` //nolint:tagliatelle // doesn't matter

	// Deprecated fields.
	Root    string `json:"root,omitempty"`    // Deprecated: replaced by Root
	InDir   string `json:"inDir,omitempty"`   // Deprecated: replaced by ReceiveDir
	OutDir  string `json:"outDir,omitempty"`  // Deprecated: replaced by SendDir
	WorkDir string `json:"workDir,omitempty"` // Deprecated: replaced by TmpReceiveDir
}

// LocalAccount is the JSON struct representing a local account.
type LocalAccount struct {
	Login        string        `json:"login"`
	Password     string        `json:"password,omitempty"`
	PasswordHash string        `json:"passwordHash,omitempty"`
	Certs        []Certificate `json:"certificates,omitempty"` //nolint:tagliatelle // doesn't matter
}

type Client struct {
	Name         string         `json:"name"`
	Protocol     string         `json:"protocol"`
	Disabled     bool           `json:"disabled"`
	LocalAddress string         `json:"localAddress,omitempty"`
	ProtoConfig  map[string]any `json:"protoConfig"`
}

// RemoteAgent is the JSON struct representing a remote partner along with its
// accounts.
type RemoteAgent struct {
	Name          string          `json:"name"`
	Address       string          `json:"address"`
	Protocol      string          `json:"protocol"`
	Configuration map[string]any  `json:"configuration"`
	Accounts      []RemoteAccount `json:"accounts"`
	Certs         []Certificate   `json:"certificates"` //nolint:tagliatelle // doesn't matter
}

// RemoteAccount is the JSON struct representing a local account.
type RemoteAccount struct {
	Login    string        `json:"login"`
	Password string        `json:"password,omitempty"`
	Certs    []Certificate `json:"certificates,omitempty"` //nolint:tagliatelle // doesn't matter
}

// Certificate is the JSON struct representing a certificate.
type Certificate struct {
	Name        string `json:"name"`
	PublicKey   string `json:"publicKey,omitempty"`
	PrivateKey  string `json:"privateKey,omitempty"`
	Certificate string `json:"Certificate,omitempty"` //nolint:tagliatelle // doesn't matter
}

// Rule is the JSON struct representing a transfer rule.
type Rule struct {
	Name           string   `json:"name"`
	IsSend         bool     `json:"isSend"`
	Path           string   `json:"path"`
	LocalDir       string   `json:"localDir,omitempty"`
	RemoteDir      string   `json:"remoteDir,omitempty"`
	TmpLocalRcvDir string   `json:"tmpLocalRcvDir,omitempty"`
	Accesses       []string `json:"auth,omitempty"` //nolint:tagliatelle // doesn't matter
	Pre            []Task   `json:"pre,omitempty"`
	Post           []Task   `json:"post,omitempty"`
	Error          []Task   `json:"error,omitempty"`

	// Deprecated fields.
	InPath   string `json:"inPath,omitempty"`   // Deprecated: replaced by LocalDir & RemoteDir
	OutPath  string `json:"outPath,omitempty"`  // Deprecated: replaced by LocalDir & RemoteDir
	WorkPath string `json:"workPath,omitempty"` // Deprecated: replaced by TmpLocalRcvDir
}

// Task is the JSON struct representing a rule task.
type Task struct {
	Type string            `json:"type"`
	Args map[string]string `json:"args"`
}

// User is the JSON struct representing a gateway user.
type User struct {
	Username     string      `json:"username"`
	Password     string      `json:"password,omitempty"`
	PasswordHash string      `json:"passwordHash,omitempty"`
	Permissions  Permissions `json:"permissions"`
}

// Permissions if the JSON struct representing a gateway user's permissions.
// Each attribute represents a permission target, and its value defines the read,
// write & deletion permissions for that target in a chmod-like ('rwd') format.
type Permissions struct {
	Transfers string `json:"transfers"`
	Servers   string `json:"servers"`
	Partners  string `json:"partners"`
	Rules     string `json:"rules"`
	Users     string `json:"users"`
}

// Transfer is the JSON struct representing a transfer history entry.
type Transfer struct {
	ID             int64                   `json:"id"`
	RemoteID       string                  `json:"remoteId,omitempty"`
	Rule           string                  `json:"rule"`
	IsSend         bool                    `json:"isSend"`
	IsServer       bool                    `json:"isServer"`
	Client         string                  `json:"client,omitempty"`
	Requester      string                  `json:"requester"`
	Requested      string                  `json:"requested"`
	Protocol       string                  `json:"protocol"`
	SrcFilename    string                  `json:"srcFilename,omitempty"`
	DestFilename   string                  `json:"destFilename,omitempty"`
	LocalFilepath  string                  `json:"localFilepath,omitempty"`
	RemoteFilepath string                  `json:"remoteFilepath,omitempty"`
	Filesize       int64                   `json:"filesize"`
	Start          time.Time               `json:"start"`
	Stop           time.Time               `json:"stop,omitempty"`
	Status         types.TransferStatus    `json:"status"`
	Step           types.TransferStep      `json:"step,omitempty"`
	Progress       int64                   `json:"progress,omitempty"`
	TaskNumber     int8                    `json:"taskNumber,omitempty"`
	ErrorCode      types.TransferErrorCode `json:"errorCode,omitempty"`
	ErrorMsg       string                  `json:"errorMsg,omitempty"`
	TransferInfo   map[string]any          `json:"transferInfo,omitempty"`
}

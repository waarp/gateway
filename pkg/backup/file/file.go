// Package file contains the declaration of the struct describing the JSON
// object used in backup files.
package file

import (
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// Data is the top-level structure of the dump file.
type Data struct {
	Locals      []LocalAgent  `json:"locals,omitempty"`
	Clients     []Client      `json:"clients,omitempty"`
	Remotes     []RemoteAgent `json:"remotes,omitempty"`
	Rules       []Rule        `json:"rules,omitempty"`
	Users       []User        `json:"users,omitempty"`
	Clouds      []Cloud       `json:"clouds,omitempty"`
	SNMPConfig  *SNMPConfig   `json:"snmpConfig,omitempty"`
	Authorities []*Authority  `json:"authorities,omitempty"`
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
	Credentials   []Credential   `json:"credentials,omitempty"`

	// Deprecated fields.

	Root    string `json:"root,omitempty"`    // Deprecated: replaced by Root
	InDir   string `json:"inDir,omitempty"`   // Deprecated: replaced by ReceiveDir
	OutDir  string `json:"outDir,omitempty"`  // Deprecated: replaced by SendDir
	WorkDir string `json:"workDir,omitempty"` // Deprecated: replaced by TmpReceiveDir
	//nolint:tagliatelle // doesn't matter
	Certs []Certificate `json:"certificates"` // Deprecated: use Credentials instead.
}

// LocalAccount is the JSON struct representing a local account.
type LocalAccount struct {
	Login       string       `json:"login"`
	Password    string       `json:"password,omitempty"`
	Credentials []Credential `json:"credentials,omitempty"`

	// Deprecated fields.

	PasswordHash string `json:"passwordHash,omitempty"` // Deprecated: use Credentials instead.
	//nolint:tagliatelle // doesn't matter
	Certs []Certificate `json:"certificates,omitempty"` // Deprecated: use Credentials instead.
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
	Credentials   []Credential    `json:"credentials,omitempty"`

	// Deprecated fields.

	//nolint:tagliatelle // doesn't matter
	Certs []Certificate `json:"certificates"` // Deprecated: use Credentials instead.
}

// RemoteAccount is the JSON struct representing a local account.
type RemoteAccount struct {
	Login       string       `json:"login"`
	Password    string       `json:"password,omitempty"`
	Credentials []Credential `json:"credentials,omitempty"`

	// Deprecated fields.

	//nolint:tagliatelle // doesn't matter
	Certs []Certificate `json:"certificates,omitempty"` // Deprecated: use Credentials instead.
}

// Certificate is the JSON struct representing a certificate.
// Deprecated: replaced by Credential.
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
	Transfers      string `json:"transfers"`
	Servers        string `json:"servers"`
	Partners       string `json:"partners"`
	Rules          string `json:"rules"`
	Users          string `json:"users"`
	Administration string `json:"administration"`
}

// Transfer is the JSON struct representing a transfer history entry.
type Transfer struct {
	ID             int64                   `json:"id"`
	RemoteID       string                  `json:"remoteId,omitempty"` //nolint:tagliatelle //can't change
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

type Credential struct {
	Name   string `json:"name,omitempty"`
	Type   string `json:"type"`
	Value  string `json:"value"`
	Value2 string `json:"value2"`
}

type Cloud struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Key     string            `json:"key"`
	Secret  string            `json:"secret"`
	Options map[string]string `json:"options"`
}

type SNMPConfig struct {
	Server   *SNMPServer    `json:"server,omitempty"`
	Monitors []*SNMPMonitor `json:"monitors,omitempty"`
}

type SNMPMonitor struct {
	Name                string `json:"name"`
	SNMPVersion         string `json:"snmpVersion"`
	UDPAddress          string `json:"udpAddress"`
	Community           string `json:"community,omitempty"`
	UseInforms          bool   `json:"useInforms"`
	V3ContextName       string `json:"v3ContextName"`
	V3ContextEngineID   string `json:"v3ContextEngineID"`
	V3Security          string `json:"v3Security,omitempty"`
	V3AuthEngineID      string `json:"v3AuthEngineID,omitempty"`
	V3AuthUsername      string `json:"v3AuthUsername,omitempty"`
	V3AuthProtocol      string `json:"v3AuthProtocol,omitempty"`
	V3AuthPassphrase    string `json:"v3AuthPassphrase,omitempty"`
	V3PrivacyProtocol   string `json:"v3PrivacyProtocol,omitempty"`
	V3PrivacyPassphrase string `json:"v3PrivacyPassphrase,omitempty"`
}

type SNMPServer struct {
	LocalUDPAddress     string `json:"localUDPAddress"`
	Community           string `json:"community,omitempty"`
	V3Only              bool   `json:"v3Only"`
	V3Username          string `json:"v3Username,omitempty"`
	V3AuthProtocol      string `json:"v3AuthProtocol,omitempty"`
	V3AuthPassphrase    string `json:"v3AuthPassphrase,omitempty"`
	V3PrivacyProtocol   string `json:"v3PrivacyProtocol,omitempty"`
	V3PrivacyPassphrase string `json:"v3PrivacyPassphrase,omitempty"`
}

type Authority struct {
	Name           string   `json:"name"`
	Type           string   `json:"type"`
	PublicIdentity string   `json:"publicIdentity"`
	ValidHosts     []string `json:"validHosts,omitempty"`
}

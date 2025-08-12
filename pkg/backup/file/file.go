// Package file contains the declaration of the struct describing the JSON
// object used in backup files.
package file

import (
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// Data is the top-level structure of the dump file.
type Data struct {
	Locals      []LocalAgent  `json:"locals,omitempty" yaml:"locals,omitempty"`
	Clients     []Client      `json:"clients,omitempty" yaml:"clients,omitempty"`
	Remotes     []RemoteAgent `json:"remotes,omitempty" yaml:"remotes,omitempty"`
	Rules       []Rule        `json:"rules,omitempty" yaml:"rules,omitempty"`
	Users       []User        `json:"users,omitempty" yaml:"users,omitempty"`
	Clouds      []Cloud       `json:"clouds,omitempty" yaml:"clouds,omitempty"`
	SNMPConfig  *SNMPConfig   `json:"snmpConfig,omitempty" yaml:"snmpConfig,omitempty"`
	Authorities []*Authority  `json:"authorities,omitempty" yaml:"authorities,omitempty"`
	CryptoKeys  []*CryptoKey  `json:"cryptoKeys,omitempty" yaml:"cryptoKeys,omitempty"`
	EmailConfig *EmailConfig  `json:"emailConfig,omitempty" yaml:"emailConfig,omitempty"`
}

// LocalAgent is the JSON struct representing a local server along with its
// accounts.
//
//nolint:lll //tags are long
type LocalAgent struct {
	Name          string         `json:"name" yaml:"name"`
	Protocol      string         `json:"protocol" yaml:"protocol"`
	Disabled      bool           `json:"disabled" yaml:"disabled"`
	Address       string         `json:"address" yaml:"address"`
	RootDir       string         `json:"rootDir,omitempty" yaml:"rootDir,omitempty"`
	ReceiveDir    string         `json:"receiveDir,omitempty" yaml:"receiveDir,omitempty"`
	SendDir       string         `json:"sendDir,omitempty" yaml:"sendDir,omitempty"`
	TmpReceiveDir string         `json:"tmpReceiveDir,omitempty" yaml:"tmpReceiveDir,omitempty"`
	Configuration map[string]any `json:"configuration" yaml:"configuration"`
	Accounts      []LocalAccount `json:"accounts" yaml:"accounts"`
	Credentials   []Credential   `json:"credentials" yaml:"credentials"`

	// Deprecated fields.
	Root         string        `json:"root,omitempty" yaml:"root,omitempty"`                 // Deprecated: replaced by Root
	InDir        string        `json:"inDir,omitempty" yaml:"inDir,omitempty"`               // Deprecated: replaced by ReceiveDir
	OutDir       string        `json:"outDir,omitempty" yaml:"outDir,omitempty"`             // Deprecated: replaced by SendDir
	WorkDir      string        `json:"workDir,omitempty" yaml:"workDir,omitempty"`           // Deprecated: replaced by TmpReceiveDir
	Certificates []Certificate `json:"certificates,omitempty" yaml:"certificates,omitempty"` // Deprecated: use Credentials instead.
}

// LocalAccount is the JSON struct representing a local account.
//
//nolint:lll //tags are long
type LocalAccount struct {
	Login       string       `json:"login" yaml:"login"`
	Password    string       `json:"password,omitempty" yaml:"password,omitempty"`
	Credentials []Credential `json:"credentials" yaml:"credentials"`

	// Deprecated fields.
	PasswordHash string        `json:"passwordHash,omitempty" yaml:"passwordHash,omitempty"` // Deprecated: use Credentials instead.
	Certificates []Certificate `json:"certificates,omitempty" yaml:"certificates,omitempty"` // Deprecated: use Credentials instead.
}

type Client struct {
	Name                 string         `json:"name" yaml:"name"`
	Protocol             string         `json:"protocol" yaml:"protocol"`
	Disabled             bool           `json:"disabled" yaml:"disabled"`
	LocalAddress         string         `json:"localAddress,omitempty" yaml:"localAddress,omitempty"`
	ProtoConfig          map[string]any `json:"protoConfig" yaml:"protoConfig"`
	NbOfAttempts         int8           `json:"nbOfAttempts" yaml:"nbOfAttempts"`
	FirstRetryDelay      int32          `json:"firstRetryDelay" yaml:"firstRetryDelay"`
	RetryIncrementFactor float32        `json:"retryIncrementFactor" yaml:"retryIncrementFactor"`
}

// RemoteAgent is the JSON struct representing a remote partner along with its
// accounts.
//
//nolint:lll //tags are long
type RemoteAgent struct {
	Name          string          `json:"name" yaml:"name"`
	Address       string          `json:"address" yaml:"address"`
	Protocol      string          `json:"protocol" yaml:"protocol"`
	Configuration map[string]any  `json:"configuration" yaml:"configuration"`
	Accounts      []RemoteAccount `json:"accounts" yaml:"accounts"`
	Credentials   []Credential    `json:"credentials" yaml:"credentials"`

	// Deprecated fields.
	Certificates []Certificate `json:"certificates,omitempty" yaml:"certificates,omitempty"` // Deprecated: use Credentials instead.
}

// RemoteAccount is the JSON struct representing a local account.
//
//nolint:lll //tags are long
type RemoteAccount struct {
	Login       string       `json:"login" yaml:"login"`
	Password    string       `json:"password" yaml:"password"`
	Credentials []Credential `json:"credentials" yaml:"credentials"`

	// Deprecated fields.
	Certificates []Certificate `json:"certificates,omitempty" yaml:"certificates,omitempty"` // Deprecated: use Credentials instead.
}

// Certificate is the JSON struct representing a certificate.
// Deprecated: replaced by Credential.
type Certificate struct {
	Name       string `json:"name" yaml:"name"`
	PublicKey  string `json:"publicKey,omitempty" yaml:"publicKey,omitempty"`
	PrivateKey string `json:"privateKey,omitempty" yaml:"privateKey,omitempty"`
	//nolint:tagliatelle //can't change
	Certificate string `json:"Certificate,omitempty" yaml:"Certificate,omitempty"`
}

// Rule is the JSON struct representing a transfer rule.
type Rule struct {
	Name           string   `json:"name" yaml:"name"`
	IsSend         bool     `json:"isSend" yaml:"isSend"`
	Path           string   `json:"path" yaml:"path"`
	LocalDir       string   `json:"localDir,omitempty" yaml:"localDir,omitempty"`
	RemoteDir      string   `json:"remoteDir,omitempty" yaml:"remoteDir,omitempty"`
	TmpLocalRcvDir string   `json:"tmpLocalRcvDir,omitempty" yaml:"tmpLocalRcvDir,omitempty"`
	Accesses       []string `json:"auth,omitempty" yaml:"auth,omitempty"` //nolint:tagliatelle // doesn't matter
	Pre            []Task   `json:"pre,omitempty" yaml:"pre,omitempty"`
	Post           []Task   `json:"post,omitempty" yaml:"post,omitempty"`
	Error          []Task   `json:"error,omitempty" yaml:"error,omitempty"`

	// Deprecated fields.
	InPath   string `json:"inPath,omitempty" yaml:"inPath,omitempty"`     // Deprecated: replaced by LocalDir & RemoteDir
	OutPath  string `json:"outPath,omitempty" yaml:"outPath,omitempty"`   // Deprecated: replaced by LocalDir & RemoteDir
	WorkPath string `json:"workPath,omitempty" yaml:"workPath,omitempty"` // Deprecated: replaced by TmpLocalRcvDir
}

// Task is the JSON struct representing a rule task.
type Task struct {
	Type string            `json:"type" yaml:"type"`
	Args map[string]string `json:"args" yaml:"args"`
}

// User is the JSON struct representing a gateway user.
type User struct {
	Username     string      `json:"username" yaml:"username"`
	Password     string      `json:"password,omitempty" yaml:"password,omitempty"`
	PasswordHash string      `json:"passwordHash,omitempty" yaml:"passwordHash,omitempty"`
	Permissions  Permissions `json:"permissions" yaml:"permissions"`
}

// Permissions if the JSON struct representing a gateway user's permissions.
// Each attribute represents a permission target, and its value defines the read,
// write & deletion permissions for that target in a chmod-like ('rwd') format.
type Permissions struct {
	Transfers      string `json:"transfers" yaml:"transfers"`
	Servers        string `json:"servers" yaml:"servers"`
	Partners       string `json:"partners" yaml:"partners"`
	Rules          string `json:"rules" yaml:"rules"`
	Users          string `json:"users" yaml:"users"`
	Administration string `json:"administration" yaml:"administration"`
}

// Transfer is the JSON struct representing a transfer history entry.
type Transfer struct {
	ID             int64                   `json:"id" yaml:"id"`
	RemoteID       string                  `json:"remoteId,omitempty" yaml:"remoteId"` //nolint:tagliatelle //can't change
	Rule           string                  `json:"rule" yaml:"rule"`
	IsSend         bool                    `json:"isSend" yaml:"isSend"`
	IsServer       bool                    `json:"isServer" yaml:"isServer"`
	Client         string                  `json:"client,omitempty" yaml:"client,omitempty"`
	Requester      string                  `json:"requester" yaml:"requester"`
	Requested      string                  `json:"requested" yaml:"requested"`
	Protocol       string                  `json:"protocol" yaml:"protocol"`
	SrcFilename    string                  `json:"srcFilename,omitempty" yaml:"srcFilename,omitempty"`
	DestFilename   string                  `json:"destFilename,omitempty" yaml:"destFilename,omitempty"`
	LocalFilepath  string                  `json:"localFilepath,omitempty" yaml:"localFilepath,omitempty"`
	RemoteFilepath string                  `json:"remoteFilepath,omitempty" yaml:"remoteFilepath,omitempty"`
	Filesize       int64                   `json:"filesize" yaml:"filesize"`
	Start          time.Time               `json:"start" yaml:"start"`
	Stop           time.Time               `json:"stop,omitzero" yaml:"stop,omitempty"`
	Status         types.TransferStatus    `json:"status" yaml:"status"`
	Step           types.TransferStep      `json:"step,omitempty" yaml:"step,omitempty"`
	Progress       int64                   `json:"progress,omitempty" yaml:"progress,omitempty"`
	TaskNumber     int8                    `json:"taskNumber,omitempty" yaml:"taskNumber,omitempty"`
	ErrorCode      types.TransferErrorCode `json:"errorCode,omitempty" yaml:"errorCode,omitempty"`
	ErrorMsg       string                  `json:"errorMsg,omitempty" yaml:"errorMsg,omitempty"`
	TransferInfo   map[string]any          `json:"transferInfo,omitempty" yaml:"transferInfo,omitempty"`
}

type Credential struct {
	Name   string `json:"name" yaml:"name"`
	Type   string `json:"type" yaml:"type"`
	Value  string `json:"value" yaml:"value"`
	Value2 string `json:"value2" yaml:"value2"`
}

type Cloud struct {
	Name    string            `json:"name" yaml:"name"`
	Type    string            `json:"type" yaml:"type"`
	Key     string            `json:"key" yaml:"key"`
	Secret  string            `json:"secret" yaml:"secret"`
	Options map[string]string `json:"options" yaml:"options"`
}

type SNMPConfig struct {
	Server   *SNMPServer    `json:"server,omitempty" yaml:"server,omitempty"`
	Monitors []*SNMPMonitor `json:"monitors,omitempty" yaml:"monitors,omitempty"`
}

type SNMPMonitor struct {
	Name                string `json:"name" yaml:"name"`
	SNMPVersion         string `json:"snmpVersion" yaml:"snmpVersion"`
	UDPAddress          string `json:"udpAddress" yaml:"udpAddress"`
	Community           string `json:"community,omitempty" yaml:"community,omitempty"`
	UseInforms          bool   `json:"useInforms" yaml:"useInforms"`
	V3ContextName       string `json:"v3ContextName" yaml:"v3ContextName"`
	V3ContextEngineID   string `json:"v3ContextEngineID" yaml:"v3ContextEngineID"`
	V3Security          string `json:"v3Security,omitempty" yaml:"v3Security,omitempty"`
	V3AuthEngineID      string `json:"v3AuthEngineID,omitempty" yaml:"v3AuthEngineID,omitempty"`
	V3AuthUsername      string `json:"v3AuthUsername,omitempty" yaml:"v3AuthUsername,omitempty"`
	V3AuthProtocol      string `json:"v3AuthProtocol,omitempty" yaml:"v3AuthProtocol,omitempty"`
	V3AuthPassphrase    string `json:"v3AuthPassphrase,omitempty" yaml:"v3AuthPassphrase,omitempty"`
	V3PrivacyProtocol   string `json:"v3PrivacyProtocol,omitempty" yaml:"v3PrivacyProtocol,omitempty"`
	V3PrivacyPassphrase string `json:"v3PrivacyPassphrase,omitempty" yaml:"v3PrivacyPassphrase,omitempty"`
}

type SNMPServer struct {
	LocalUDPAddress     string `json:"localUDPAddress" yaml:"localUDPAddress"`
	Community           string `json:"community,omitempty" yaml:"community,omitempty"`
	V3Only              bool   `json:"v3Only" yaml:"v3Only"`
	V3Username          string `json:"v3Username,omitempty" yaml:"v3Username,omitempty"`
	V3AuthProtocol      string `json:"v3AuthProtocol,omitempty" yaml:"v3AuthProtocol,omitempty"`
	V3AuthPassphrase    string `json:"v3AuthPassphrase,omitempty" yaml:"v3AuthPassphrase,omitempty"`
	V3PrivacyProtocol   string `json:"v3PrivacyProtocol,omitempty" yaml:"v3PrivacyProtocol,omitempty"`
	V3PrivacyPassphrase string `json:"v3PrivacyPassphrase,omitempty" yaml:"v3PrivacyPassphrase,omitempty"`
}

type Authority struct {
	Name           string   `json:"name" yaml:"name"`
	Type           string   `json:"type" yaml:"type"`
	PublicIdentity string   `json:"publicIdentity" yaml:"publicIdentity"`
	ValidHosts     []string `json:"validHosts,omitempty" yaml:"validHosts,omitempty"`
}

type CryptoKey struct {
	Name string `json:"name" yaml:"name"`
	Type string `json:"type" yaml:"type"`
	Key  string `json:"key" yaml:"key"`
}

type EmailConfig struct {
	Credentials []*SMTPCredential `json:"credentials,omitempty" yaml:"credentials,omitempty"`
	Templates   []*EmailTemplate  `json:"templates,omitempty" yaml:"templates,omitempty"`
}

type SMTPCredential struct {
	EmailAddress  string `json:"emailAddress" yaml:"emailAddress"`
	ServerAddress string `json:"serverAddress" yaml:"serverAddress"`
	Login         string `json:"login" yaml:"login"`
	Password      string `json:"password" yaml:"password"`
}

type EmailTemplate struct {
	Name        string   `json:"name" yaml:"name"`
	Subject     string   `json:"subject" yaml:"subject"`
	MIMEType    string   `json:"mimeType" yaml:"mimeType"`
	Body        string   `json:"body" yaml:"body"`
	Attachments []string `json:"attachments,omitempty" yaml:"attachments,omitempty"`
}

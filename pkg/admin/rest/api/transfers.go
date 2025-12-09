package api

import (
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// InTransfer is the JSON representation of a transfer in requests made to
// the REST interface.
type InTransfer struct {
	Rule                 string         `json:"rule,omitempty" yaml:"rule,omitempty"`
	Client               string         `json:"client,omitempty" yaml:"client,omitempty"`
	Partner              string         `json:"partner,omitempty" yaml:"partner,omitempty"`
	Account              string         `json:"account,omitempty" yaml:"account,omitempty"`
	IsSend               Nullable[bool] `json:"isSend,omitzero" yaml:"isSend,omitempty"`
	File                 string         `json:"file,omitempty" yaml:"file,omitempty"`
	Output               string         `json:"output,omitempty" yaml:"output,omitempty"`
	Start                time.Time      `json:"start,omitzero" yaml:"start,omitempty"`
	TransferInfo         map[string]any `json:"transferInfo,omitempty" yaml:"transferInfo,omitempty"`
	NbOfAttempts         int8           `json:"nbOfAttempts" yaml:"nbOfAttempts"`
	FirstRetryDelay      int32          `json:"firstRetryDelay" yaml:"firstRetryDelay"`
	RetryIncrementFactor float32        `json:"retryIncrementFactor" yaml:"retryIncrementFactor"`

	// Deprecated fields
	SourcePath string              `json:"sourcePath,omitempty"` // Deprecated: replaced by File
	DestPath   string              `json:"destPath,omitempty"`   // Deprecated: replaced by File
	StartDate  Nullable[time.Time] `json:"startDate,omitzero"`   // Deprecated: replaced by Start
}

// OutTransfer is the JSON representation of a transfer in responses sent by
// the REST interface.
type OutTransfer struct {
	ID                   int64                `json:"id" yaml:"id"`
	RemoteID             string               `json:"remoteID,omitempty" yaml:"remoteID,omitempty"`
	Rule                 string               `json:"rule" yaml:"rule"`
	IsServer             bool                 `json:"isServer" yaml:"isServer"`
	IsSend               bool                 `json:"isSend" yaml:"isSend"`
	Client               string               `json:"client" yaml:"client"`
	Requested            string               `json:"requested" yaml:"requested"`
	Requester            string               `json:"requester" yaml:"requester"`
	Protocol             string               `json:"protocol" yaml:"protocol"`
	SrcFilename          string               `json:"srcFilename" yaml:"srcFilename"`
	DestFilename         string               `json:"destFilename" yaml:"destFilename"`
	LocalFilepath        string               `json:"localFilepath,omitempty" yaml:"localFilepath,omitempty"`
	RemoteFilepath       string               `json:"remoteFilepath,omitempty" yaml:"remoteFilepath,omitempty"`
	Filesize             int64                `json:"filesize" yaml:"filesize"`
	Start                time.Time            `json:"start" yaml:"start"`
	Stop                 Nullable[time.Time]  `json:"stop,omitzero" yaml:"stop,omitempty"`
	Status               types.TransferStatus `json:"status" yaml:"status"`
	Step                 string               `json:"step,omitempty" yaml:"step,omitempty"`
	Progress             int64                `json:"progress,omitempty" yaml:"progress,omitempty"`
	TaskNumber           int8                 `json:"taskNumber,omitempty" yaml:"taskNumber,omitempty"`
	ErrorCode            string               `json:"errorCode,omitempty" yaml:"errorCode,omitempty"`
	ErrorMsg             string               `json:"errorMsg,omitempty" yaml:"errorMsg,omitempty"`
	TransferInfo         map[string]any       `json:"transferInfo,omitempty" yaml:"transferInfo,omitempty"`
	RemainingAttempts    int8                 `json:"remainingAttempts,omitempty" yaml:"remainingAttempts,omitempty"`
	NextAttempt          time.Time            `json:"nextAttempt,omitzero" yaml:"nextAttempt,omitzero"`
	NextRetryDelay       int32                `json:"nextRetryDelay,omitempty" yaml:"nextRetryDelay,omitempty"`
	RetryIncrementFactor float32              `json:"retryIncrementFactor,omitempty" yaml:"retryIncrementFactor,omitempty"`

	// Deprecated fields
	TrueFilepath string    `json:"trueFilepath"` // Deprecated: replaced by LocalFilepath & RemoteFilepath
	SourcePath   string    `json:"sourcePath"`   // Deprecated: replaced by LocalFilepath & RemoteFilepath
	DestPath     string    `json:"destPath"`     // Deprecated: replaced by LocalFilepath & RemoteFilepath
	StartDate    time.Time `json:"startDate"`    // Deprecated: replaced by Start
}

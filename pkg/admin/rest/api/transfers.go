package api

import (
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// InTransfer is the JSON representation of a transfer in requests made to
// the REST interface.
type InTransfer struct {
	Rule                 string         `json:"rule,omitempty"`
	Client               string         `json:"client,omitempty"`
	Partner              string         `json:"partner,omitempty"`
	Account              string         `json:"account,omitempty"`
	IsSend               Nullable[bool] `json:"isSend,omitempty"`
	File                 string         `json:"file,omitempty"`
	Output               string         `json:"output,omitempty"`
	Start                time.Time      `json:"start,omitempty"`
	TransferInfo         map[string]any `json:"transferInfo,omitempty"`
	NumberOfTries        int8           `json:"numberOfTries"`
	FirstRetryDelay      int32          `json:"firstRetryDelay"`
	RetryIncrementFactor float32        `json:"retryIncrementFactor"`

	// Deprecated fields
	SourcePath string              `json:"sourcePath,omitempty"` // Deprecated: replaced by File
	DestPath   string              `json:"destPath,omitempty"`   // Deprecated: replaced by File
	StartDate  Nullable[time.Time] `json:"startDate,omitempty"`  // Deprecated: replaced by Start
}

// OutTransfer is the JSON representation of a transfer in responses sent by
// the REST interface.
type OutTransfer struct {
	ID                   int64                `json:"id"`
	RemoteID             string               `json:"remoteID,omitempty"` //nolint:tagliatelle // FIXME too late to change that
	Rule                 string               `json:"rule"`
	IsServer             bool                 `json:"isServer"`
	IsSend               bool                 `json:"isSend"`
	Client               string               `json:"client"`
	Requested            string               `json:"requested"`
	Requester            string               `json:"requester"`
	Protocol             string               `json:"protocol"`
	SrcFilename          string               `json:"srcFilename"`
	DestFilename         string               `json:"destFilename"`
	LocalFilepath        string               `json:"localFilepath,omitempty"`
	RemoteFilepath       string               `json:"remoteFilepath,omitempty"`
	Filesize             int64                `json:"filesize"`
	Start                time.Time            `json:"start"`
	Stop                 Nullable[time.Time]  `json:"stop,omitempty"`
	Status               types.TransferStatus `json:"status"`
	Step                 string               `json:"step,omitempty"`
	Progress             int64                `json:"progress,omitempty"`
	TaskNumber           int8                 `json:"taskNumber,omitempty"`
	ErrorCode            string               `json:"errorCode,omitempty"`
	ErrorMsg             string               `json:"errorMsg,omitempty"`
	TransferInfo         map[string]any       `json:"transferInfo,omitempty"`
	RemainingTries       int8                 `json:"remainingTries,omitempty"`
	NextAttempt          time.Time            `json:"nextAttempt,omitzero"`
	NextRetryDelay       int32                `json:"nextRetryDelay,omitempty"`
	RetryIncrementFactor float32              `json:"retryIncrementFactor,omitempty"`

	// Deprecated fields
	TrueFilepath string    `json:"trueFilepath"` // Deprecated: replaced by LocalFilepath & RemoteFilepath
	SourcePath   string    `json:"sourcePath"`   // Deprecated: replaced by LocalFilepath & RemoteFilepath
	DestPath     string    `json:"destPath"`     // Deprecated: replaced by LocalFilepath & RemoteFilepath
	StartDate    time.Time `json:"startDate"`    // Deprecated: replaced by Start
}

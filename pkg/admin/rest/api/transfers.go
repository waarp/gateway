package api

import (
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// InTransfer is the JSON representation of a transfer in requests made to
// the REST interface.
type InTransfer struct {
	Rule    string    `json:"rule"`
	Partner string    `json:"partner"`
	Account string    `json:"account"`
	IsSend  *bool     `json:"isSend"`
	File    string    `json:"file"`
	Output  *string   `json:"output"`
	Start   time.Time `json:"start"`

	// Deprecated fields
	SourcePath string    `json:"sourcePath,omitempty"` // Deprecated: replaced by File
	DestPath   string    `json:"destPath,omitempty"`   // Deprecated: replaced by File
	StartDate  time.Time `json:"startDate"`            // Deprecated: replaced by Start
}

// OutTransfer is the JSON representation of a transfer in responses sent by
// the REST interface.
type OutTransfer struct {
	ID             uint64               `json:"id"`
	RemoteID       string               `json:"remoteID,omitempty"` //nolint:tagliatelle // FIXME too late to change that
	Rule           string               `json:"rule"`
	IsServer       bool                 `json:"isServer"`
	IsSend         bool                 `json:"isSend"`
	Requested      string               `json:"requested"`
	Requester      string               `json:"requester"`
	Protocol       string               `json:"protocol"`
	LocalFilepath  string               `json:"localFilepath"`
	RemoteFilepath string               `json:"remoteFilepath"`
	Filesize       int64                `json:"filesize"`
	Start          time.Time            `json:"start"`
	Status         types.TransferStatus `json:"status"`
	Step           string               `json:"step,omitempty"`
	Progress       uint64               `json:"progress,omitempty"`
	TaskNumber     uint64               `json:"taskNumber,omitempty"`
	ErrorCode      string               `json:"errorCode,omitempty"`
	ErrorMsg       string               `json:"errorMsg,omitempty"`

	// Deprecated fields
	TrueFilepath string    `json:"trueFilepath"` // Deprecated: replaced by LocalFilepath & RemoteFilepath
	SourcePath   string    `json:"sourcePath"`   // Deprecated: replaced by LocalFilepath & RemoteFilepath
	DestPath     string    `json:"destPath"`     // Deprecated: replaced by LocalFilepath & RemoteFilepath
	StartDate    time.Time `json:"startDate"`    // Deprecated: replaced by Start
}

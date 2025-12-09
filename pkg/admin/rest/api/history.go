package api

import (
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// OutHistory is the JSON representation of a history entry in responses sent by
// the REST interface.
//
// Deprecated: replaced by OutTransfer.
type OutHistory struct {
	ID             int64                   `json:"id"`
	RemoteID       string                  `json:"remoteID,omitempty"`
	IsServer       bool                    `json:"isServer"`
	IsSend         bool                    `json:"isSend"`
	Requester      string                  `json:"requester"`
	Requested      string                  `json:"requested"`
	Protocol       string                  `json:"protocol"`
	LocalFilepath  string                  `json:"localFilepath"`
	RemoteFilepath string                  `json:"remoteFilepath"`
	Filesize       int64                   `json:"filesize"`
	Rule           string                  `json:"rule"`
	Start          time.Time               `json:"start"`
	Stop           Nullable[time.Time]     `json:"stop,omitzero"`
	TransferInfo   map[string]any          `json:"transferInfo,omitempty"`
	Status         types.TransferStatus    `json:"status"`
	ErrorCode      types.TransferErrorCode `json:"errorCode,omitempty"`
	ErrorMsg       string                  `json:"errorMsg,omitempty"`
	Step           types.TransferStep      `json:"step,omitempty"`
	Progress       int64                   `json:"progress,omitempty"`
	TaskNumber     int8                    `json:"taskNumber,omitempty"`

	// Deprecated fields
	SourceFilename string `json:"sourceFilename"` // Deprecated: replaced by LocalFilepath & RemoteFilepath
	DestFilename   string `json:"destFilename"`   // Deprecated: replaced by LocalFilepath & RemoteFilepath
}

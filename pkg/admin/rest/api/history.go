package api

import (
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
)

// OutHistory is the JSON representation of a history entry in responses sent by
// the REST interface.
type OutHistory struct {
	ID             uint64                  `json:"id"`
	RemoteID       string                  `json:"remoteID,omitempty"`
	IsServer       bool                    `json:"isServer"`
	IsSend         bool                    `json:"isSend"`
	Requester      string                  `json:"requester"`
	Requested      string                  `json:"requested"`
	Protocol       string                  `json:"protocol"`
	SourceFilename string                  `json:"sourceFilename"`
	DestFilename   string                  `json:"destFilename"`
	Rule           string                  `json:"rule"`
	Start          time.Time               `json:"start"`
	Stop           *time.Time              `json:"stop"`
	Status         types.TransferStatus    `json:"status"`
	ErrorCode      types.TransferErrorCode `json:"errorCode,omitempty"`
	ErrorMsg       string                  `json:"errorMsg,omitempty"`
	Step           types.TransferStep      `json:"step,omitempty"`
	Progress       uint64                  `json:"progress,omitempty"`
	TaskNumber     uint64                  `json:"taskNumber,omitempty"`
}

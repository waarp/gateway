package api

import (
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// InTransfer is the JSON representation of a transfer in requests made to
// the REST interface.
type InTransfer struct {
	Rule       string    `json:"rule"`
	Partner    string    `json:"partner"`
	Account    string    `json:"account"`
	IsSend     *bool     `json:"isSend"`
	SourcePath string    `json:"sourcePath"`
	DestPath   string    `json:"destPath"`
	Start      time.Time `json:"startDate"`
}

// OutTransfer is the JSON representation of a transfer in responses sent by
// the REST interface.
type OutTransfer struct {
	ID           uint64               `json:"id"`
	RemoteID     string               `json:"remoteID,omitempty"`
	Rule         string               `json:"rule"`
	IsServer     bool                 `json:"isServer"`
	IsSend       bool                 `json:"isSend"`
	Requested    string               `json:"requested"`
	Requester    string               `json:"requester"`
	TrueFilepath string               `json:"trueFilepath"`
	SourcePath   string               `json:"sourcePath"`
	DestPath     string               `json:"destPath"`
	Start        time.Time            `json:"startDate"`
	Status       types.TransferStatus `json:"status"`
	Step         string               `json:"step,omitempty"`
	Progress     uint64               `json:"progress,omitempty"`
	TaskNumber   uint64               `json:"taskNumber,omitempty"`
	ErrorCode    string               `json:"errorCode,omitempty"`
	ErrorMsg     string               `json:"errorMsg,omitempty"`
}

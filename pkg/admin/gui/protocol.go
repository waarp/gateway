package gui

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

type Protocols struct {
	R66      string
	R66TLS   string
	SFTP     string
	HTTP     string
	HTTPS    string
	FTP      string
	FTPS     string
	PeSIT    string
	PeSITTLS string
}

//nolint:gochecknoglobals // Constant
var (
	TLSVersions            = []string{protoutils.TLSv10, protoutils.TLSv11, protoutils.TLSv12, protoutils.TLSv13}
	CompatibilityModePeSIT = []string{pesit.CompatibilityModeAxway, pesit.CompatibilityModeNone}
)

package protocols

import (
	"iter"
	"slices"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/as2"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/webdav"
)

var ErrUnknownProtocol = model.ErrUnknownProtocol

//nolint:gochecknoglobals //global var is used by design here
var list []string

//nolint:gochecknoinits //init is required here
func init() {
	// AS2
	Register(as2.AS2, as2.Module{})
	Register(as2.AS2TLS, as2.ModuleTLS{})
	// FTP
	Register(ftp.FTP, ftp.Module{})
	Register(ftp.FTPS, ftp.ModuleFTPS{})
	// HTTP
	Register(http.HTTP, http.Module{})
	Register(http.HTTPS, http.ModuleHTTPS{})
	// PeSIT
	Register(pesit.Pesit, pesit.Module{})
	Register(pesit.PesitTLS, pesit.ModuleTLS{})
	// R66
	Register(r66.R66, r66.Module{})
	Register(r66.R66TLS, r66.ModuleTLS{})
	// SFTP
	Register(sftp.SFTP, sftp.Module{})
	// WebDAV
	Register(webdav.Webdav, webdav.Module{})
	Register(webdav.WebdavTLS, webdav.ModuleTLS{})
}

// Register registers a new protocol module.
func Register(name string, p Module) {
	model.Protocols[name] = p
	clientMakers[name] = p.NewClient
	serverMakers[name] = p.NewServer
	list = append(list, name)
}

func List() iter.Seq[string] {
	return slices.Values(list)
}

func Exists(name string) bool {
	_, ok := model.Protocols[name]

	return ok
}

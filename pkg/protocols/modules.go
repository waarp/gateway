package protocols

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/amqp091"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/webdav"
)

// Register registers a new protocol module.
func Register(name string, p Module) { List[name] = p }

// Get returns the protocol implementation with the given name.
func Get(name string) Module { return List[name] }

// IsValid returns whether the given protocol is implemented.
func IsValid(name string) bool { return Get(name) != nil }

// List is the list of all protocols implemented by the gateway.
//
//nolint:gochecknoglobals //global var is required here
var List = map[string]Module{
	amqp091.AMQP091:  &amqp091.Module{},   // AMQP 0.9.1
	ebics.EBICS:      &ebics.Module{},     // EBICS
	sftp.SFTP:        &sftp.Module{},      // SFTP
	r66.R66:          &r66.Module{},       // R66
	r66.R66TLS:       &r66.ModuleTLS{},    // R66-TLS
	http.HTTP:        &http.Module{},      // HTTP
	http.HTTPS:       &http.ModuleHTTPS{}, // HTTPS
	ftp.FTP:          &ftp.Module{},       // FTP
	ftp.FTPS:         &ftp.ModuleFTPS{},   // FTPS
	pesit.Pesit:      &pesit.Module{},     // Pesit
	pesit.PesitTLS:   &pesit.ModuleTLS{},  // Pesit-TLS
	webdav.Webdav:    &webdav.Module{},    // WebDAV
	webdav.WebdavTLS: &webdav.ModuleTLS{}, // WebDAV over HTTPS
}

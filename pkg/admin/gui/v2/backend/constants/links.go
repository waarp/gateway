package constants

import (
	"path"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

//nolint:gochecknoglobals //cannot be constant: must be changed at runtime
var (
	AppName     = "Waarp Gateway"
	DocLinkHome = "https://doc.waarp.org/waarp-gateway/"
)

const (
	WebuiPrefix  = "/webui"
	StaticPrefix = "/static_v2/"
)

//nolint:staticcheck //remove once the doc has been translated
func DocLink(language string) string {
	language = "fr"

	num := version.Num
	if num == "dev" {
		num = "latest"
	}

	return DocLinkHome + path.Join(num, language) + "/"
}

func DocPage(language, page string) string {
	url := path.Join(DocLink(language), page)
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}

	return url
}

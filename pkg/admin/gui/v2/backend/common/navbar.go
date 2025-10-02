package common

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
)

const (
	NavbarTemplateFile     = "navbar/navbar.gohtml"
	navbarLocalizationFile = "navbar/navbar.yaml"
)

type NavbarData struct {
	AppName      string
	InstanceName string
	Username     string
	Language     string
	Version      string
	CompileDate  string
	Revision     string
	Text         locale.Dictionary
}

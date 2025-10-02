package common

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
)

const (
	SidebarTemplateFile     = "sidebar/sidebar.gohtml"
	sidebarLocalizationFile = "sidebar/sidebar.yaml"
)

type SidebarData struct {
	DocLink       string
	ActiveSection string
	ActiveMenu    string
	UserRights    *Permissions
	Text          locale.Dictionary
}

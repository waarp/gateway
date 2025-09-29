package common

import (
	"fmt"
	"html/template"
	"io/fs"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/webfs"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

const (
	PageTemplateName = "page"

	baseLocalizationFilePattern = "base.yaml"
	basePageTemplateFile        = "base.gohtml"
)

type BaseLocalizationData struct {
	Navbar  locale.LocalizationData
	Sidebar locale.LocalizationData
	Common  locale.LocalizationData
}

func InitBaseLocalization() *BaseLocalizationData {
	return &BaseLocalizationData{
		Sidebar: locale.ParseLocalizationFile(sidebarLocalizationFile),
		Navbar:  locale.ParseLocalizationFile(navbarLocalizationFile),
		Common:  locale.ParseLocalizationFile(baseLocalizationFilePattern),
	}
}

type GeneralData struct {
	AppName      string
	InstanceName string
	Language     string
	NavbarData   *NavbarData
	SidebarData  *SidebarData
	Text         locale.Dictionary

	PageData any
}

func InitBasePageTemplate(pageFile string) *template.Template {
	base := ParseTemplate(
		basePageTemplateFile,
		NavbarTemplateFile,
		SidebarTemplateFile,
	)

	pageContent, err := fs.ReadFile(webfs.WebFS, pageFile)
	if err != nil {
		panic(fmt.Sprintf("failed to read page template file: %v", err))
	}

	template.Must(
		base.New(PageTemplateName).Parse(string(pageContent)),
	)

	return base
}

func MakeBasePageData(user *model.User, language, activeSection, activeMenu string,
	localization *BaseLocalizationData, pageData any,
) *GeneralData {
	return &GeneralData{
		AppName:      constants.AppName,
		InstanceName: conf.GlobalConfig.GatewayName,
		Language:     language,
		NavbarData: &NavbarData{
			AppName:      constants.AppName,
			InstanceName: conf.GlobalConfig.GatewayName,
			Username:     user.Username,
			Language:     language,
			Version:      version.Num,
			CompileDate:  version.Date,
			Revision:     version.Commit,
			Text:         locale.MakeLocalText(language, localization.Navbar),
		},
		SidebarData: &SidebarData{
			DocLink:       constants.DocLink(language),
			ActiveSection: activeSection,
			ActiveMenu:    activeMenu,
			UserRights:    ParsePermissions(user.Permissions),
			Text:          locale.MakeLocalText(language, localization.Sidebar),
		},
		Text:     locale.MakeLocalText(language, localization.Common),
		PageData: pageData,
	}
}

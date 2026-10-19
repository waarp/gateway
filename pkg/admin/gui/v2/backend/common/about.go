package common

import (
	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

// The editor and its public services, linked from the "About" dialog, as
// the historic header listed them.
const (
	editorURL    = "https://www.waarp.fr"
	downloadURL  = "https://dl.waarp.org/dist/waarp-gateway/"
	docSiteURL   = "https://doc.waarp.org"
	forumsURL    = "https://discuss.waarp.org/"
	contactEmail = "info@waarp.fr"
)

// About is the content of the "About" dialog of the account menu, and of
// the "About" page that stands in for it without JavaScript: the build of the
// gateway, the support and the licence sentences, the useful links.
func About(translate gui.Translator) *gui.About {
	return &gui.About{
		Version:   version.Num,
		Date:      version.Date,
		Revision:  version.Commit,
		EditorURL: editorURL,
		Support:   translate("about.support", "product", constants.AppName),
		License:   translate("about.license", "product", constants.AppName),
		Links: []gui.Link{
			{Label: translate("about.link_download", "product", constants.AppName), URL: downloadURL},
			{Label: translate("about.link_doc"), URL: docSiteURL},
			{Label: translate("about.link_forums"), URL: forumsURL},
			{Label: translate("about.link_contact", "email", contactEmail), URL: "mailto:" + contactEmail},
		},
	}
}

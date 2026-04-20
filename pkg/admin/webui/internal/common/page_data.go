package common

import (
	"net/http"
	"runtime"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/webui/internal/session"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/webui/internal/translations"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

type AppInfo struct {
	Name      string
	Version   string
	BuildDate time.Time
	Runtime   string
}

type SidebarState struct {
	Collapsed bool
}

type PageData struct {
	*translations.Printer
	App     *AppInfo
	Sidebar *SidebarState
	User    *model.User
}

func getSidebarState(r *http.Request) *SidebarState {
	state := &SidebarState{}

	if colCookie, err := r.Cookie("sidebar_state"); err == nil && colCookie.Value == "false" {
		state.Collapsed = true
	}

	return state
}

func GetPageData(r *http.Request) *PageData {
	buildDate, _ := time.Parse(time.RFC3339, version.Date)

	return &PageData{
		Printer: translations.GetPrinter(r),
		App: &AppInfo{
			Name:      conf.GlobalConfig.GatewayName,
			Version:   version.Num,
			BuildDate: buildDate,
			Runtime:   runtime.Version(),
		},
		Sidebar: getSidebarState(r),
		User:    session.GetUser(r),
	}
}

//nolint:gochecknoglobals // template
package gui

import (
	"encoding/json"
	"html/template"
	"reflect"
	"strings"

	"github.com/Masterminds/sprig/v3"
	"github.com/dustin/go-humanize"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	index              = "front-end/html/index.html"
	header             = "front-end/html/header.html"
	multiLanguage      = "front-end/html/multi_language.html"
	addProtoConfig     = "front-end/html/addProtoConfig.html"
	editProtoConfig    = "front-end/html/editProtoConfig.html"
	displayProtoConfig = "front-end/html/displayProtoConfig.html"
	displayFormAuth    = "front-end/html/typeCredential.html"
	addTasks           = "front-end/html/addTasks.html"
	editTasks          = "front-end/html/editTasks.html"
	displayTasks       = "front-end/html/displayTasks.html"
)

var funcs = template.FuncMap{
	"contains": strings.Contains,
	"isArray": func(value any) bool {
		return reflect.TypeOf(value).Kind() == reflect.Slice
	},
	"isBool": func(value any) bool {
		return reflect.TypeOf(value).Kind() == reflect.Bool
	},
	"add": func(a, b int) int {
		return a + b
	},
	"dict":  sprig.TxtFuncMap()["dict"],
	"split": strings.Split,
	"splitTime": func(timeout, part string) string {
		i := strings.Index(timeout, part)
		if i != -1 {
			start := i - 1
			for start >= 0 && timeout[start] >= '0' && timeout[start] <= '9' {
				start--
			}

			return timeout[start+1 : i]
		}

		return ""
	},
	"splitExtensions": func(fileName, part string) string {
		i := strings.Index(fileName, ".")
		if i != -1 {
			switch part {
			case "before":
				return fileName[:i]
			case "after":
				return fileName[i:]
			}
		}

		return ""
	},
	"marshalJSON": func(v any) template.JS {
		b, err := json.Marshal(v)
		if err != nil {
			return "null"
		}

		return template.JS(b) //nolint:gosec // template.JS is necessary
	},
	"formatDateTime": translateDateTime,
	"div":            sprig.TxtFuncMap()["div"],
	"mul":            sprig.TxtFuncMap()["mul"],
	"displayOrDefault": func(val any, def string) any {
		if val == nil {
			return def
		}
		if s, ok := val.(string); ok && s == "" {
			return def
		}

		return val
	},
	"humanizeSize": func(size int64) string {
		return humanize.Bytes(uint64(size))
	},
}

func CombinedFuncMap(db *database.DB) template.FuncMap {
	funcMap := template.FuncMap{}
	for k, v := range funcs {
		funcMap[k] = v
	}

	for k, v := range NewFuncMap(db) {
		funcMap[k] = v
	}

	return funcMap
}

func NewFuncMap(db *database.DB) template.FuncMap {
	return template.FuncMap{
		"getServerName": func(id int64) string {
			server, err := internal.GetServerByID(db, id)
			if err != nil {
				return ""
			}

			return server.Name
		},
		"getPartnerName": func(id int64) string {
			partner, err := internal.GetPartnerByID(db, id)
			if err != nil {
				return ""
			}

			return partner.Name
		},
	}
}

var (
	homeTemplate = template.Must(
		template.ParseFS(webFS, index, header, multiLanguage, "front-end/html/home_page.html"),
	)
	loginTemplate = template.Must(
		template.ParseFS(webFS, multiLanguage, "front-end/html/login_page.html"),
	)
	userManagementTemplate = template.Must(
		template.New("user_management_page.html").
			Funcs(funcs).
			ParseFS(webFS, index, header, multiLanguage, "front-end/html/user_management_page.html"),
	)
	partnerManagementTemplate = template.Must(
		template.New("partner_management_page.html").
			Funcs(funcs).
			ParseFS(webFS, index, header, multiLanguage, addProtoConfig, editProtoConfig, displayProtoConfig,
				"front-end/html/partner_management_page.html"),
	)
	partnerAuthenticationTemplate = template.Must(
		template.New("partner_management_page.html").
			Funcs(funcs).
			ParseFS(webFS, index, header, multiLanguage, displayFormAuth, "front-end/html/partner_authentication_page.html"),
	)
	remoteAccountTemplate = template.Must(
		template.New("remote_account_management_page.html").
			Funcs(funcs).
			ParseFS(webFS, index, header, multiLanguage, "front-end/html/remote_account_management_page.html"),
	)
	remoteAccountAuthenticationTemplate = template.Must(
		template.New("remote_account_authentication_page.html").
			Funcs(funcs).
			ParseFS(webFS, index, header, multiLanguage, displayFormAuth,
				"front-end/html/remote_account_authentication_page.html"),
	)
	serverManagementTemplate = template.Must(
		template.New("server_management_page.html").
			Funcs(funcs).
			ParseFS(webFS, index, header, multiLanguage, addProtoConfig, editProtoConfig, displayProtoConfig,
				"front-end/html/server_management_page.html"),
	)
	serverAuthenticationTemplate = template.Must(
		template.New("server_authentication_page.html").
			Funcs(funcs).
			ParseFS(webFS, index, header, multiLanguage, displayFormAuth, "front-end/html/server_authentication_page.html"),
	)
	localAccountTemplate = template.Must(
		template.New("local_account_management_page.html").
			Funcs(funcs).
			ParseFS(webFS, index, header, multiLanguage, "front-end/html/local_account_management_page.html"),
	)
	localAccountAuthenticationTemplate = template.Must(
		template.New("local_account_authentication_page.html").
			Funcs(funcs).
			ParseFS(webFS, index, header, multiLanguage, displayFormAuth,
				"front-end/html/local_account_authentication_page.html"),
	)
	localClientManagementTemplate = template.Must(
		template.New("local_client_management_page.html").
			Funcs(funcs).
			ParseFS(webFS, index, header, multiLanguage, addProtoConfig, editProtoConfig, displayProtoConfig,
				"front-end/html/local_client_management_page.html"),
	)
	ruleManagementTemplate = template.Must(
		template.New("transfer_rules_management_page.html").
			Funcs(funcs).
			ParseFS(webFS, index, header, multiLanguage, "front-end/html/transfer_rules_management_page.html"),
	)
	tasksTransferRulesTemplate = template.Must(
		template.New("tasks_transfer_rules_page.html").
			Funcs(funcs).
			ParseFS(webFS, index, header, multiLanguage, addTasks, editTasks, displayTasks,
				"front-end/html/tasks_transfer_rules_page.html"),
	)
	// ManagementUsageRightsRulesTemplate in .go, for dynamics template (with db).
	transferMonitoringTemplate = template.Must(
		template.New("transfer_monitoring_page.html").
			Funcs(funcs).
			ParseFS(webFS, index, header, multiLanguage, "front-end/html/transfer_monitoring_page.html"),
	)
	statusServicesTemplate = template.Must(
		template.New("status_services_page.html").
			Funcs(funcs).
			ParseFS(webFS, index, header, multiLanguage, "front-end/html/status_services_page.html"),
	)
)

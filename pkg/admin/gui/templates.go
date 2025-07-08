//nolint:gochecknoglobals // template
package gui

import (
	"html/template"
	"reflect"
	"strings"

	"github.com/Masterminds/sprig/v3"
)

const (
	index              = "front-end/html/index.html"
	header             = "front-end/html/header.html"
	multiLanguage      = "front-end/html/multi_language.html"
	addProtoConfig     = "front-end/html/addProtoConfig.html"
	editProtoConfig    = "front-end/html/editProtoConfig.html"
	displayProtoConfig = "front-end/html/displayProtoConfig.html"
	displayFormAuth    = "front-end/html/typeCredential.html"
)

var funcs = template.FuncMap{
	"contains": strings.Contains,
	"isArray": func(value interface{}) bool {
		return reflect.TypeOf(value).Kind() == reflect.Slice
	},
	"isBool": func(value interface{}) bool {
		return reflect.TypeOf(value).Kind() == reflect.Bool
	},
	"add": func(a, b int) int {
		return a + b
	},
	"dict": sprig.TxtFuncMap()["dict"],
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
	accountAuthenticationTemplate = template.Must(
		template.New("account_authentication_page.html").
			Funcs(funcs).
			ParseFS(webFS, index, header, multiLanguage, displayFormAuth, "front-end/html/account_authentication_page.html"),
	)
)

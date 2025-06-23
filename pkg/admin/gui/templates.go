//nolint:gochecknoglobals // template
package gui

import (
	"html/template"
	"reflect"
	"strings"

	"github.com/Masterminds/sprig/v3"
)

const (
	index           = "front_end/html/index.html"
	header          = "front_end/html/header.html"
	multiLanguage   = "front_end/html/multi_language.html"
	addProtoConfig  = "front_end/html/addProtoConfig.html"
	editProtoConfig = "front_end/html/editProtoConfig.html"
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
		template.ParseFS(webFS, index, header, multiLanguage, "front_end/html/home_page.html"),
	)
	loginTemplate = template.Must(
		template.ParseFS(webFS, multiLanguage, "front_end/html/login_page.html"),
	)
	userManagementTemplate = template.Must(
		template.New("user_management_page.html").
			Funcs(funcs).
			ParseFS(webFS, index, header, multiLanguage, "front_end/html/user_management_page.html"),
	)
	partnerManagementTemplate = template.Must(
		template.New("partner_management_page.html").
			Funcs(funcs).
			ParseFS(webFS, index, header, multiLanguage, addProtoConfig, editProtoConfig,
				"front_end/html/partner_management_page.html"),
	)
)

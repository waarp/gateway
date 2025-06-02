//nolint:gochecknoglobals // template
package gui

import (
	"html/template"
	"strings"
)

const (
	index         = "front_end/html/index.html"
	header        = "front_end/html/header.html"
	multiLanguage = "front_end/html/multi_language.html"
)

var funcs = template.FuncMap{
	"contains": strings.Contains,
}

var (
	homeTemplate = template.Must(
		template.ParseFS(webFS, index, header, multiLanguage, "front_end/html/home_page.html"),
	)
	userManagementTemplate = template.Must(
		template.New("user_management_page.html").
		Funcs(funcs).
		ParseFS(webFS, index, header, multiLanguage, "front_end/html/user_management_page.html"),
	)
	loginTemplate = template.Must(
		template.ParseFS(webFS, multiLanguage, "front_end/html/login_page.html"),
	)
)
